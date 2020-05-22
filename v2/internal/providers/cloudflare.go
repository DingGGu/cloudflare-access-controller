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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
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

func (r *Resource) New(ingress *v1beta1.Ingress, resourceName string, zoneName string) (*Resource, error) {
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

	var policies []cloudflare.AccessPolicy
	if err := json.Unmarshal([]byte(ingress.Annotations[types.AnnotationSessionPolicies]), &policies); err != nil {
		return nil, err
	}
	r.Policies = policies

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

func (p *Cloudflare) Reconcile(ctx context.Context, req reconcile.Request, ingress *v1beta1.Ingress, recorder record.EventRecorder) error {
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

	if app, err := p.store.GetApplication(r.Name); err != nil {
		if errors.Is(err, store.ApplicationNotFoundError) {
			log.Info("Create access application")

			app, err = p.client.CreateAccessApplication(p.zoneId, r.AccessApplication)
			if err != nil {
				log.Error(err, "Cannot create access application")
				recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot create access application: %s", err.Error()))
				return err
			}

			recorder.Event(ingress, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created Access Application %s audience tag: %s", app.Name, app.AUD))

			for _, policy := range r.Policies {
				_, err := p.client.CreateAccessPolicy(p.zoneId, app.ID, policy)
				if err != nil {
					log.Error(err, "Cannot create access policy", "policy", policy)
					recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot create access policy: %v: %s", policy, err.Error()))
					return err
				}
			}
			return nil
		}

		log.Error(err, "Error from getApplication")
		recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Error from getApplication: %s", err.Error()))
		return err
	} else {
		if !r.Equal(app) { // Check if AccessApplication need update
			if app, err = p.client.UpdateAccessApplication(p.zoneId, r.AccessApplication); err != nil {
				log.Error(err, "Cannot update access application")
				recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot update access application: %s", err.Error()))
				return err
			}
		}

		_, _, err := p.client.AccessPolicies(p.zoneId, app.ID, cloudflare.PaginationOptions{}) // Get Policies
		if err != nil {
			log.Error(err, "Cannot get access policies")
			recorder.Event(ingress, corev1.EventTypeWarning, "Error", fmt.Sprintf("Cannot get access policy: %s", err.Error()))
			return err
		}

		// Check if AccessPolicy need update
		//for i, policy := range r.Policies {
		//
		//}

	}

	return nil
}

func (p *Cloudflare) Delete(ctx context.Context, req reconcile.Request, ingress *v1beta1.Ingress) error {
	log := p.log.WithValues("ingress", req.NamespacedName)

	if _, ok := ingress.Annotations[types.AnnotationApplicationSubDomain]; !ok {
		return nil
	}

	resourceName := p.ResourceName(req)
	if app, err := p.store.GetApplication(resourceName); err != nil {
		if errors.Is(err, store.ApplicationNotFoundError) {
			log.Info("Cannot find access application", "resourceName", resourceName)
			return nil
		}
		log.Error(err, "Error from getApplication", "resourceName", resourceName)
		return err
	} else {
		return p.client.DeleteAccessApplication(p.zoneId, app.ID)
	}
}

func (p *Cloudflare) ResourceName(req reconcile.Request) string {
	return strings.Join([]string{p.clusterUid, req.Namespace, req.Name}, ".")
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
	if nil != err {
		log.Error(err, "cannot find zoneId", "zoneName", zoneName)
		os.Exit(1)
	}

	return &Cloudflare{client, log, zoneName, clusterUid, zoneId, store.New(client, zoneId)}
}
