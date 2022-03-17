package providers

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DingGGu/cloudflare-access-controller/v2/internal/store"
	"github.com/DingGGu/cloudflare-access-controller/v2/internal/types"
	"github.com/cloudflare/cloudflare-go"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/record"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

type Resource struct {
	Name              string
	Domain            string
	SessionDuration   string
	AccessApplication cloudflare.AccessApplication
	Policies          []cloudflare.AccessPolicy
}

func (r *Resource) stripSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s[1:]
	}
	return s
}

func (r *Resource) Equal(o cloudflare.AccessApplication) bool {
	return r.Name == o.Name &&
		r.Domain == o.Domain &&
		r.SessionDuration == r.SessionDuration
}

func (r *Resource) PolicyEqual(index int, o cloudflare.AccessPolicy) bool {
	p := r.Policies[index]
	return p.Decision == o.Decision &&
		p.Name == o.Name &&
		cmp.Equal(p.Include, o.Include, cmpopts.EquateEmpty()) &&
		cmp.Equal(p.Require, o.Require, cmpopts.EquateEmpty()) &&
		cmp.Equal(p.Exclude, o.Exclude, cmpopts.EquateEmpty())
}

func (r *Resource) New(ingress *networkingv1.Ingress, resourceName string, zoneName string) (*Resource, error) {
	r.Name = resourceName
	r.Domain = strings.Join([]string{
		ingress.Annotations[types.AnnotationApplicationSubDomain],
		".",
		zoneName,
		"/",
		r.stripSlash(ingress.Annotations[types.AnnotationApplicationPath]),
	}, "")
	r.SessionDuration = ingress.Annotations[types.AnnotationSessionDuration]

	r.AccessApplication = cloudflare.AccessApplication{
		Name:            r.Name,
		Domain:          r.Domain,
		SessionDuration: r.SessionDuration,
	}

	if a, ok := ingress.Annotations[types.AnnotationSessionPolicies]; ok {
		var policies []cloudflare.AccessPolicy
		if err := json.Unmarshal([]byte(a), &policies); err != nil {
			return nil, err
		}

		for i := range policies {
			policies[i].Name = fmt.Sprintf("cac-policy-%d", i+1)
		}
		r.Policies = policies
	}

	return r, nil
}

type Cloudflare struct {
	client     *cloudflare.API
	log        logr.Logger
	zoneName   string
	clusterUid string
	zoneId     string
	store      *store.Store
}

func (p *Cloudflare) Reconcile(ctx context.Context, req reconcile.Request, ingress *networkingv1.Ingress, recorder record.EventRecorder) error {
	log := p.log.WithValues("ingress", req.NamespacedName)

	if _, ok := ingress.Annotations[types.AnnotationApplicationSubDomain]; !ok {
		return nil
	}

	r, err := (&Resource{}).New(ingress, p.ResourceName(req), p.zoneName)
	if err != nil {
		log.Error(err, "Cannot create resource")
		recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot create resource: %s", err.Error()))
		return err
	}

	log = log.WithValues("resourceName", r.Name)

	if app, err := p.store.GetApplication(ctx, r.Name); err != nil {
		if errors.Is(err, store.ApplicationNotFoundError) {
			log.Info("Create access application")

			app, err = p.client.CreateZoneLevelAccessApplication(ctx, p.zoneId, r.AccessApplication)
			if err != nil {
				log.Error(err, "Cannot create access application", "domain", r.Domain)
				recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot create access application (%s): %s", r.Domain, err.Error()))
				return err
			}

			recorder.Event(ingress, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created Access Application %s (%s) audience tag: %s", app.Name, r.Domain, app.AUD))

			for i, policy := range r.Policies {
				_, err := p.client.CreateZoneLevelAccessPolicy(ctx, p.zoneId, app.ID, policy)
				if err != nil {
					log.Error(err, "Cannot create access policy", "policy", policy)
					recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot create access policy: %v: %s", policy, err.Error()))
					return err
				}
				recorder.Event(ingress, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created Access Policy[%d] %s (%s): %v", i, app.Name, r.Domain, policy))
			}
			return nil
		}

		log.Error(err, "Error from getApplication")
		recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Error from getApplication: %s", err.Error()))
		return err
	} else {
		if !r.Equal(app) { // Check if AccessApplication need update
			r.AccessApplication.ID = app.ID
			if app, err = p.client.UpdateZoneLevelAccessApplication(ctx, p.zoneId, r.AccessApplication); err != nil {
				log.Error(err, "Cannot update access application")
				recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot update access application: %s", err.Error()))
				return err
			}
			recorder.Event(ingress, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Update Access Application %s (%s): %s", app.Name, r.Domain, app.AUD))
		}

		originPolicies, err := p.store.GetPolicies(ctx, app.ID) // Get Policies
		if err != nil {
			log.Error(err, "Cannot get access policies")
			recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot get access policy: %s", err.Error()))
			return err
		}

		length := len(originPolicies)

		// Check if AccessPolicy need update
		for i, policy := range r.Policies {
			if i <= length-1 {
				if !r.PolicyEqual(i, originPolicies[i]) {
					policy.ID = originPolicies[i].ID
					if _, err := p.client.UpdateZoneLevelAccessPolicy(ctx, p.zoneId, app.ID, policy); err != nil { // Update
						log.Error(err, "Cannot update access policies[%d]", i)
						recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot update access policy[%d]: %s", i, err.Error()))
						return err
					}

					log.Info(fmt.Sprintf("Updated Access Policy[%d]: %+v", i, policy))
					recorder.Event(ingress, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated Access Policy[%d] %s (%s): %v", i, app.Name, r.Domain, policy))
					continue
				} else {
					continue
				}
			}
			if _, err := p.client.CreateZoneLevelAccessPolicy(ctx, p.zoneId, app.ID, policy); err != nil { // Create
				log.Error(err, "Cannot create access policies[%d]", i)
				recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot create access policy[%d]: %s", i, err.Error()))
				return err
			}

			log.Info(fmt.Sprintf("Created Access Policy[%d]: %+v", i, policy))
			recorder.Event(ingress, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created Access Policy[%d] %s (%s): %v", i, app.Name, r.Domain, policy))
		}

		removeRange := len(r.Policies)
		if removeRange < length {
			for i, policy := range originPolicies[removeRange:] {
				idx := length - 1 + i
				if err := p.client.DeleteZoneLevelAccessPolicy(ctx, p.zoneId, app.ID, policy.ID); err != nil { // Delete
					log.Error(err, "Cannot delete access policies[%d]", idx)
					recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot delete access policy[%d]: %s", idx, err.Error()))
					return err
				}

				log.Info(fmt.Sprintf("Deleted Access Policy[%d]: %+v", idx, policy))
				recorder.Event(ingress, corev1.EventTypeNormal, "Deleted", fmt.Sprintf("Deleted Access Policy[%d] %s (%s): %v", idx, app.Name, r.Domain, policy))
			}
		}

	}

	return nil
}

func (p *Cloudflare) Delete(ctx context.Context, req reconcile.Request, ingress *networkingv1.Ingress) error {
	log := p.log.WithValues("ingress", req.NamespacedName)

	if _, ok := ingress.Annotations[types.AnnotationApplicationSubDomain]; !ok {
		return nil
	}

	resourceName := p.ResourceName(req)
	if app, err := p.store.GetApplication(ctx, resourceName); err != nil {
		if errors.Is(err, store.ApplicationNotFoundError) {
			log.Info("Cannot find access application", "resourceName", resourceName)
			return nil
		}
		log.Error(err, "Error from getApplication", "resourceName", resourceName)
		return err
	} else {
		return p.client.DeleteZoneLevelAccessApplication(ctx, p.zoneId, app.ID)
	}
}

func (p *Cloudflare) ResourceName(req reconcile.Request) string {
	return strings.Join([]string{p.clusterUid, req.Namespace, req.Name}, "-")
}

func NewCloudflare(apiToken string, log logr.Logger, zoneName, clusterName string) Provider {
	client, err := cloudflare.NewWithAPIToken(apiToken)

	if err != nil {
		log.Error(err, "Cannot initialize cloudflare client")
		os.Exit(1)
	}

	if "" == clusterName {
		log.Error(nil, "provide ClusterName with '-c' option")
		os.Exit(1)
	}

	clusterHash := md5.Sum([]byte(clusterName))
	clusterUid := hex.EncodeToString(clusterHash[:])[:8]

	if "" == zoneName {
		log.Error(nil, "provide ZoneName with '-z' option")
		os.Exit(1)
	}

	zoneId, err := client.ZoneIDByName(zoneName)
	log.Info(fmt.Sprintf("Get zoneId: %s", zoneId))
	if nil != err {
		log.Error(err, "cannot find zoneId", "zoneName", zoneName)
		os.Exit(1)
	}

	return &Cloudflare{client, log, zoneName, clusterUid, zoneId,
		store.New(client, zoneId, log.WithName("store"))}
}
