package cf

import (
	"github.com/DingGGu/cloudflare-access-controller/internal/option"
	"github.com/cloudflare/cloudflare-go"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

type CloudflareInterface interface {
	ZoneIDByName(zoneName string) (string, error)
	AccessApplications(zoneID string, pageOpts cloudflare.PaginationOptions) ([]cloudflare.AccessApplication, cloudflare.ResultInfo, error)
	CreateAccessApplication(zoneID string, accessApplication cloudflare.AccessApplication) (cloudflare.AccessApplication, error)
	UpdateAccessApplication(zoneID string, accessApplication cloudflare.AccessApplication) (cloudflare.AccessApplication, error)
	DeleteAccessApplication(zoneID, applicationID string) error
	AccessPolicies(zoneID, applicationID string, pageOpts cloudflare.PaginationOptions) ([]cloudflare.AccessPolicy, cloudflare.ResultInfo, error)
	CreateAccessPolicy(zoneID, applicationID string, accessPolicy cloudflare.AccessPolicy) (cloudflare.AccessPolicy, error)
	UpdateAccessPolicy(zoneID, applicationID string, accessPolicy cloudflare.AccessPolicy) (cloudflare.AccessPolicy, error)
	DeleteAccessPolicy(zoneID, applicationID, accessPolicyID string) error
}

type CloudFlareProvider struct {
	Api             CloudflareInterface // *cloudflare.API
	cachedZoneIdMap map[string]string
}

func GetCloudFlareClient(cfApiKey string, cfApiEmail string) *CloudFlareProvider {
	api, err := cloudflare.New(cfApiKey, cfApiEmail)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	return &CloudFlareProvider{
		Api: api,
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

	zoneId, err := cf.Api.ZoneIDByName(zoneName) // Assuming example.com exists in your Cloudflare account already
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	cf.cachedZoneIdMap[zoneName] = zoneId

	return zoneId
}

func (cf CloudFlareProvider) GetAccessApplications(zoneName string) map[string]option.AccessApp {
	zoneId := cf.getZoneIDByName(zoneName)
	applications, _, err := cf.Api.AccessApplications(zoneId, cloudflare.PaginationOptions{})

	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	var m = make(map[string]option.AccessApp)

	for _, app := range applications {
		split := strings.Split(app.Name, "-")
		// Get Managed Cloudflare Accesses
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
	return cf.Api.UpdateAccessApplication(zoneId, accessApp)
}

func (cf CloudFlareProvider) DeleteAccessApplication(zoneName string, appId string) error {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.Api.DeleteAccessApplication(zoneId, appId)
}

func (cf CloudFlareProvider) CreateAccessApplication(zoneName string, accessApp cloudflare.AccessApplication) (cloudflare.AccessApplication, error) {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.Api.CreateAccessApplication(zoneId, accessApp)
}

func (cf CloudFlareProvider) GetAccessPolicies(zoneName string, accessAppId string) []cloudflare.AccessPolicy {
	zoneId := cf.getZoneIDByName(zoneName)
	policies, _, err := cf.Api.AccessPolicies(zoneId, accessAppId, cloudflare.PaginationOptions{})

	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	return policies
}

func (cf CloudFlareProvider) CreateAccessPolicy(zoneName string, accessAppId string, accessPolicy cloudflare.AccessPolicy) (cloudflare.AccessPolicy, error) {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.Api.CreateAccessPolicy(zoneId, accessAppId, accessPolicy)
}

func (cf CloudFlareProvider) UpdateAccessPolicy(zoneName string, accessAppId string, accessPolicy cloudflare.AccessPolicy) (cloudflare.AccessPolicy, error) {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.Api.UpdateAccessPolicy(zoneId, accessAppId, accessPolicy)
}

func (cf CloudFlareProvider) DeleteAccessPolicy(zoneName string, accessAppId string, accessPolicyId string) error {
	zoneId := cf.getZoneIDByName(zoneName)
	return cf.Api.DeleteAccessPolicy(zoneId, accessAppId, accessPolicyId)
}
