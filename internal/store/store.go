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

func (s Store) CheckProviders(apps []option.AccessApp, cfZoneNames []string) (*PlanApp, *PlanPolicy) {
	planApp := PlanApp{}
	planPolicy := PlanPolicy{}
	appMap := createMapAppName(apps)

	for _, zoneName := range cfZoneNames {
		cfAppMap := s.CloudFlareProvider.GetAccessApplications(zoneName)
		for cfAppId, cfApp := range cfAppMap {
			if cfApp.CfZoneName != zoneName &&
				s.checkManagedAccess(cfApp) {
				continue
			}

			cfPolicies := s.CloudFlareProvider.GetAccessPolicies(cfApp.CfZoneName, cfAppId)
			cfApp.AccessAppPolicies = cfPolicies

			existCf := false

			for k8sAppName, k8sApp := range appMap {
				if k8sAppName == cfApp.GetName() {
					s.validatePolicy(&planPolicy, k8sApp, cfApp, cfAppId)
					// Start compare
					if k8sApp.AccessAppOption.Domain != cfApp.AccessAppOption.Domain ||
						k8sApp.AccessAppOption.SessionDuration != cfApp.AccessAppOption.SessionDuration {
						// Need Update
						planApp.Updates = append(planApp.Updates, AppEndPoint{
							ZoneName: k8sApp.CfZoneName,
							App: cloudflare.AccessApplication{
								ID:              cfAppId,
								Name:            k8sApp.GetName(),
								Domain:          k8sApp.AccessAppOption.Domain,
								SessionDuration: k8sApp.AccessAppOption.SessionDuration,
							},
						})
					} else {
						logrus.Printf("Skip already app: %s/%s", k8sApp.CfZoneName, k8sApp.GetName())
					}
					delete(appMap, k8sAppName)
					existCf = true
					break
				}
			}

			if !existCf {
				// Delete App
				planApp.Deletes = append(planApp.Deletes, AppEndPoint{
					ZoneName: zoneName,
					App: cloudflare.AccessApplication{
						ID: cfAppId,
					},
				})
			}
		}
	}

	for k8sAppName, k8sApp := range appMap {
		// Create App via remains
		planApp.Creates = append(planApp.Creates, AppEndPoint{
			ZoneName: k8sApp.CfZoneName,
			App: cloudflare.AccessApplication{
				Name:            k8sApp.GetName(),
				Domain:          k8sApp.AccessAppOption.Domain,
				SessionDuration: k8sApp.AccessAppOption.SessionDuration,
			},
			Policies: k8sApp.AccessAppPolicies,
		})
		delete(appMap, k8sAppName) // Noting to do. maybe usage for double check? (If operation properly, Map will be empty)
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
			logrus.Printf("Successfully updated application %s", updatedAccessApplication)
		}
	}
	for _, del := range planApp.Deletes {
		err := s.CloudFlareProvider.DeleteAccessApplication(del.ZoneName, del.App.ID)

		if err != nil {
			logrus.Error(err)
		} else {
			logrus.Printf("Successfully deleted application %s", del.App)
		}
	}
	for _, create := range planApp.Creates {
		createdAccessApplication, err := s.CloudFlareProvider.CreateAccessApplication(create.ZoneName, create.App)

		if err != nil {
			logrus.Error(err)
		} else {
			//ing := create.Ingress
			//ingClient := s.KubeClient.ExtensionsV1beta1().Ingresses(ing.Namespace)

			// todo: Need to added k8s status field
			logrus.WithFields(logrus.Fields{
				"zoneName": create.ZoneName,
				"domain":   create.App.Domain,
				"aud":      create.App.AUD,
				"name":     create.App.Name,
			}).Infof("Successfully created application")

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
			logrus.Printf("Successfully updated policy (%s) %v", update.AppId, update.Policy)
		}
	}

	for _, del := range policyPlan.Deletes {
		err := s.CloudFlareProvider.DeleteAccessPolicy(del.ZoneName, del.AppId, del.PolicyId)
		if err != nil {
			logrus.Error(err)
		} else {
			logrus.Printf("Successfully deleted policy (%s) %v", del.AppId, del.Policy)
		}
	}

	for _, create := range policyPlan.Creates {
		_, err := s.CloudFlareProvider.CreateAccessPolicy(create.ZoneName, create.AppId, create.Policy)
		if err != nil {
			logrus.Error(err)
		} else {
			logrus.Printf("Successfully created policy (%s) %v", create.AppId, create.Policy)
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
		if remote, ok := Index(remotePolicies, idx); ok {
			if s.checkPolicyDifference(source, *remote) {
				source.Name = remote.Name // Overwrite original policy name
				policyPlan.Updates = append(policyPlan.Updates, PolicyEndpoint{
					ZoneName: remoteApp.CfZoneName,
					AppId:    remoteAppId,
					Policy:   source,
					PolicyId: remote.ID,
				})
			} else {
				logrus.Printf("Skip already exist policy: %s/%s/%s", remoteApp.CfZoneName, remoteApp.GetName(), remote.Name)
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

func (s Store) checkManagedAccess(app option.AccessApp) bool {
	if app.ClusterUID == s.ClusterUID {
		return true
	}
	return false
}

func createMapAppName(apps []option.AccessApp) map[string]option.AccessApp {
	var appMap = make(map[string]option.AccessApp)
	for _, app := range apps {
		appMap[app.GetName()] = app
	}

	return appMap
}

func Equal(a, b []interface{}) bool {
	return cmp.Equal(a, b, cmpopts.EquateEmpty())
}

func Index(a []cloudflare.AccessPolicy, index int) (*cloudflare.AccessPolicy, bool) {
	if len(a) > index {
		return &a[index], true
	} else {
		return nil, false
	}
}

func generatePolicyName(idx int) string {
	return strings.Join([]string{"policy", strconv.Itoa(idx)}, "-")
}
