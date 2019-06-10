package store

import (
	"github.com/DingGGu/cloudflare-access-controller/internal/option"
	"github.com/DingGGu/cloudflare-access-controller/internal/provider/cf"
	"github.com/cloudflare/cloudflare-go"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"strconv"
	"strings"
)

type AppEndPoint struct {
	ZoneName string
	App      cloudflare.AccessApplication
	Policies []cloudflare.AccessPolicy // for create
}

type PolicyEndpoint struct {
	ZoneName string
	AppId    string
	Policy   cloudflare.AccessPolicy
	PolicyId string
}

type PlanApp struct {
	Creates []AppEndPoint
	Updates []AppEndPoint
	Deletes []AppEndPoint
}

type PlanPolicy struct {
	Creates []PolicyEndpoint
	Updates []PolicyEndpoint
	Deletes []PolicyEndpoint
}

type Store struct {
	ClusterUID         string
	CloudFlareProvider *cf.CloudFlareProvider
	KubeClient         *kubernetes.Clientset
}

func (s Store) Run(apps []option.AccessApp, cfZoneNames []string) {
	planApp, planPolicy := s.CheckProviders(apps, cfZoneNames)
	s.ApplyAppChanges(planApp, planPolicy)
	s.ApplyPolicyChanges(planPolicy)
}

func (s Store) CheckProviders(sourceApps []option.AccessApp, cfZoneNames []string) (*PlanApp, *PlanPolicy) {
	planApp := PlanApp{}
	planPolicy := PlanPolicy{}

	for _, zoneName := range cfZoneNames {
		remoteApps := s.CloudFlareProvider.GetAccessApplications(zoneName, s.ClusterUID)
		for _, remoteApp := range remoteApps {
			// Check belong to Cluster
			if !s.IsClusterAccessApp(remoteApp) {
				continue
			}

			remotePolicies := s.CloudFlareProvider.GetAccessPolicies(remoteApp.CfZoneName, remoteApp.RemoteID)
			remoteApp.AccessAppPolicies = remotePolicies

			if sourceApp, ok := FindApp(sourceApps, remoteApp.GetName()); ok {
				sourceApp.RemoteExisted = true
				// Compare Policy
				s.validatePolicy(&planPolicy, *sourceApp, remoteApp, remoteApp.RemoteID)
				// Compare App
				if s.checkAppDifference(*sourceApp, remoteApp) {
					planApp.Updates = append(planApp.Updates, AppEndPoint{
						ZoneName: sourceApp.CfZoneName,
						App: cloudflare.AccessApplication{
							ID:              remoteApp.RemoteID,
							Name:            sourceApp.GetName(),
							Domain:          sourceApp.AccessAppOption.Domain,
							SessionDuration: sourceApp.AccessAppOption.SessionDuration,
						},
					})
				} else {
					logrus.WithFields(logrus.Fields{
						"zoneName": zoneName,
						"appName":  remoteApp.GetName(),
					}).Info("Skip already exist access application")
				}
			} else {
				planApp.Deletes = append(planApp.Deletes, AppEndPoint{
					ZoneName: zoneName,
					App: cloudflare.AccessApplication{
						ID: remoteApp.RemoteID,
					},
				})
			}
		}
	}
	for _, sourceApp := range sourceApps {

		if sourceApp.RemoteExisted {
			continue
		}

		planApp.Creates = append(planApp.Creates, AppEndPoint{
			ZoneName: sourceApp.CfZoneName,
			App: cloudflare.AccessApplication{
				Name:            sourceApp.GetName(),
				Domain:          sourceApp.AccessAppOption.Domain,
				SessionDuration: sourceApp.AccessAppOption.SessionDuration,
			},
			Policies: sourceApp.AccessAppPolicies,
		})
	}

	return &planApp, &planPolicy
}

func (s Store) ApplyAppChanges(planApp *PlanApp, planPolicy *PlanPolicy) {
	for _, update := range planApp.Updates {
		updatedAccessApplication, err := s.CloudFlareProvider.UpdateAccessApplication(update.ZoneName, update.App)

		if err != nil {
			logrus.Error(err)
		} else {
			// todo: Need to added k8s status field
			logrus.WithFields(logrus.Fields{
				"zoneName": update.ZoneName,
				"appName":  updatedAccessApplication.Name,
			}).Info("Successfully updated access application")
		}
	}
	for _, del := range planApp.Deletes {
		err := s.CloudFlareProvider.DeleteAccessApplication(del.ZoneName, del.App.ID)

		if err != nil {
			logrus.Error(err)
		} else {
			logrus.WithFields(logrus.Fields{
				"zoneName": del.ZoneName,
				"appName":  del.App.Name,
			}).Info("Successfully deleted access application")
		}
	}
	for _, create := range planApp.Creates {
		createdAccessApplication, err := s.CloudFlareProvider.CreateAccessApplication(create.ZoneName, create.App)

		if err != nil {
			logrus.Error(err)
		} else {
			//ing := create.Ingress
			//ingClient := s.KubeClient.ExtensionsV1beta1().Ingresses(ing.Namespace)

			logrus.WithFields(logrus.Fields{
				"zoneName": create.ZoneName,
				"domain":   create.App.Domain,
				"aud":      create.App.AUD,
				"appName":  create.App.Name,
			}).Info("Successfully created application")

			// Also create policies for first App
			for idx, policy := range create.Policies {
				policy.Name = generatePolicyName(idx)
				planPolicy.Creates = append(planPolicy.Creates, PolicyEndpoint{
					ZoneName: create.ZoneName,
					AppId:    createdAccessApplication.ID,
					Policy:   policy,
				})
			}
		}
	}
}

func (s Store) ApplyPolicyChanges(policyPlan *PlanPolicy) {
	for _, update := range policyPlan.Updates {
		update.Policy.ID = update.PolicyId
		_, err := s.CloudFlareProvider.UpdateAccessPolicy(update.ZoneName, update.AppId, update.Policy)
		if err != nil {
			logrus.Error(err)
		} else {
			logrus.WithFields(logrus.Fields{
				"zoneName":   update.ZoneName,
				"appName":    update.AppId,
				"policyName": update.Policy.Name,
			}).Info("Successfully updated policy")
		}
	}

	for _, del := range policyPlan.Deletes {
		err := s.CloudFlareProvider.DeleteAccessPolicy(del.ZoneName, del.AppId, del.PolicyId)
		if err != nil {
			logrus.Error(err)
		} else {
			logrus.WithFields(logrus.Fields{
				"zoneName":   del.ZoneName,
				"appName":    del.AppId,
				"policyName": del.Policy.Name,
			}).Info("Successfully deleted policy")
		}
	}

	for _, create := range policyPlan.Creates {
		_, err := s.CloudFlareProvider.CreateAccessPolicy(create.ZoneName, create.AppId, create.Policy)
		if err != nil {
			logrus.Error(err)
		} else {
			logrus.WithFields(logrus.Fields{
				"zoneName":   create.ZoneName,
				"appName":    create.AppId,
				"policyName": create.Policy.Name,
			}).Info("Successfully created policy")
		}
	}
}

func (s Store) validatePolicy(policyPlan *PlanPolicy, sourceApp option.AccessApp, remoteApp option.AccessApp, remoteAppId string) {
	var sourcePolicies = make([]cloudflare.AccessPolicy, len(sourceApp.AccessAppPolicies))
	var remotePolicies = make([]cloudflare.AccessPolicy, len(remoteApp.AccessAppPolicies))
	copy(sourcePolicies, sourceApp.AccessAppPolicies)
	copy(remotePolicies, remoteApp.AccessAppPolicies)

	for idx := 0; idx < len(sourcePolicies); idx++ {
		source := sourcePolicies[idx]
		if remote, ok := IndexPolicy(remotePolicies, idx); ok {
			if s.checkPolicyDifference(source, *remote) {
				source.Name = remote.Name // Overwrite original policy name
				policyPlan.Updates = append(policyPlan.Updates, PolicyEndpoint{
					ZoneName: remoteApp.CfZoneName,
					AppId:    remoteAppId,
					Policy:   source,
					PolicyId: remote.ID,
				})
			} else {
				logrus.WithFields(logrus.Fields{
					"zoneName":   remoteApp.CfZoneName,
					"appName":    remoteApp.GetName(),
					"policyName": remote.Name,
				}).Info("Skip already exist policy")
			}
			sourcePolicies = append(sourcePolicies[:idx], sourcePolicies[idx+1:]...)
			remotePolicies = append(remotePolicies[:idx], remotePolicies[idx+1:]...)
			idx--
		}
	}

	for idx, source := range sourcePolicies {
		source.Name = generatePolicyName(idx)
		policyPlan.Creates = append(policyPlan.Creates, PolicyEndpoint{
			ZoneName: remoteApp.CfZoneName,
			AppId:    remoteAppId,
			Policy:   source,
		})
	}

	for _, remote := range remotePolicies {
		policyPlan.Deletes = append(policyPlan.Deletes, PolicyEndpoint{
			ZoneName: remoteApp.CfZoneName,
			AppId:    remoteAppId,
			PolicyId: remote.ID,
		})
	}
}

func (s Store) checkAppDifference(source, remote option.AccessApp) bool {
	if source.AccessAppOption.Domain != remote.AccessAppOption.Domain ||
		source.AccessAppOption.SessionDuration != remote.AccessAppOption.SessionDuration {
		return true
	}

	return false
}

func (s Store) checkPolicyDifference(source cloudflare.AccessPolicy, remote cloudflare.AccessPolicy) bool {
	if source.Decision != remote.Decision {
		return true
	}

	if Equal(source.Include, remote.Include) &&
		Equal(source.Require, remote.Require) &&
		Equal(source.Exclude, remote.Exclude) {
		return false
	}

	return true
}

func (s Store) IsClusterAccessApp(app option.AccessApp) bool {
	if app.ClusterUID == s.ClusterUID {
		return true
	}
	return false
}

func Equal(a, b []interface{}) bool {
	return cmp.Equal(a, b, cmpopts.EquateEmpty())
}

func FindApp(a []option.AccessApp, appName string) (*option.AccessApp, bool) {
	for idx, b := range a {
		if b.GetName() == appName {
			return &a[idx], true
		}
	}
	return nil, false
}

func IndexPolicy(a []cloudflare.AccessPolicy, index int) (*cloudflare.AccessPolicy, bool) {
	if len(a) > index {
		return &a[index], true
	} else {
		return nil, false
	}
}

func generatePolicyName(idx int) string {
	return strings.Join([]string{"policy", strconv.Itoa(idx)}, "-")
}
