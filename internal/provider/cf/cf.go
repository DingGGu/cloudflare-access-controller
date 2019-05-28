package cf

import (
	"github.com/DingGGu/cloudflare-access-controller/internal/option"
	"github.com/cloudflare/cloudflare-go"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

type CloudFlareProvider struct {
	api             *cloudflare.API
	cachedZoneIdMap map[string]string
}

func GetCloudFlareClient(cfApiKey string, cfApiEmail string) *CloudFlareProvider {
	api, err := cloudflare.New(cfApiKey, cfApiEmail)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	return &CloudFlareProvider{
		api: api,
	}
}

func (cf CloudFlareProvider) getZoneIDByName(zoneName string) string {
	if cf.cachedZoneIdMap == nil {
		cf.cachedZoneIdMap = make(map[string]string)
	}

	if val, ok := cf.cachedZoneIdMap[zoneName]; ok {
		logrus.Debugf("Using Cached zoneName %s: %s", zoneName, cf.cachedZoneIdMap[zoneName])
		return val
	}

	zoneId, err := cf.api.ZoneIDByName(zoneName) // Assuming example.com exists in your Cloudflare account already
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	cf.cachedZoneIdMap[zoneName] = zoneId

	return zoneId
}

func (cf CloudFlareProvider) GetAccessApplications(zoneName string) map[string]option.AccessApp {
	zoneId := cf.getZoneIDByName(zoneName)
	applications, _, err := cf.api.AccessApplications(zoneId, cloudflare.PaginationOptions{})

	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	var m = make(map[string]option.AccessApp)

	for _, app := range applications {
		split := strings.Split(app.Name, "-")
		if len(split) == 3 {
			clusterUID, namespace, ingressName := split[0], split[1], split[2]
			cfOpt := option.AccessAppOption{
				Domain:          app.Domain,
				SessionDuration: app.SessionDuration,
			}
			m[app.ID] = option.AccessApp{
				ClusterUID:      clusterUID,
				Namespace:       namespace,
				IngressName:     ingressName,
				CfZoneName:      zoneName,
				AccessAppOption: cfOpt,
				Source:          "cf",
			}
		}
	}

	return m
}

func (cf CloudFlareProvider) UpdateAccessApplication(zoneName string, accessApp cloudflare.AccessApplication) (cloudflare.AccessApplication, error) {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.api.UpdateAccessApplication(zoneId, accessApp)
}

func (cf CloudFlareProvider) DeleteAccessApplication(zoneName string, appId string) error {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.api.DeleteAccessApplication(zoneId, appId)
}

func (cf CloudFlareProvider) CreateAccessApplication(zoneName string, accessApp cloudflare.AccessApplication) (cloudflare.AccessApplication, error) {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.api.CreateAccessApplication(zoneId, accessApp)
}

func (cf CloudFlareProvider) GetAccessPolicies(zoneName string, accessAppId string) []cloudflare.AccessPolicy {
	zoneId := cf.getZoneIDByName(zoneName)
	policies, _, err := cf.api.AccessPolicies(zoneId, accessAppId, cloudflare.PaginationOptions{})

	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	return policies
}

func (cf CloudFlareProvider) CreateAccessPolicy(zoneName string, accessAppId string, accessPolicy cloudflare.AccessPolicy) (cloudflare.AccessPolicy, error) {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.api.CreateAccessPolicy(zoneId, accessAppId, accessPolicy)
}

func (cf CloudFlareProvider) UpdateAccessPolicy(zoneName string, accessAppId string, accessPolicy cloudflare.AccessPolicy) (cloudflare.AccessPolicy, error) {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.api.UpdateAccessPolicy(zoneId, accessAppId, accessPolicy)
}

func (cf CloudFlareProvider) DeleteAccessPolicy(zoneName string, accessAppId string, accessPolicyId string) error {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.api.DeleteAccessPolicy(zoneId, accessAppId, accessPolicyId)
}
