package store

import (
	"encoding/json"
	"github.com/DingGGu/cloudflare-access-controller/internal/option"
	"github.com/DingGGu/cloudflare-access-controller/internal/provider/cf"
	"github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"strings"
	"testing"
)

type MockClient struct {
	mock.Mock
}

func (c *MockClient) ZoneIDByName(zoneName string) (string, error) {
	args := c.Called(zoneName)
	return args.String(0), args.Error(1)
}

func (c *MockClient) AccessApplications(zoneID string, pageOpts cloudflare.PaginationOptions) ([]cloudflare.AccessApplication, cloudflare.ResultInfo, error) {
	args := c.Called(zoneID, pageOpts)
	return args.Get(0).([]cloudflare.AccessApplication), args.Get(1).(cloudflare.ResultInfo), args.Error(2)
}

func (c *MockClient) CreateAccessApplication(zoneID string, accessApplication cloudflare.AccessApplication) (cloudflare.AccessApplication, error) {
	args := c.Called(zoneID, accessApplication)
	return args.Get(0).(cloudflare.AccessApplication), args.Error(1)
}

func (c *MockClient) UpdateAccessApplication(zoneID string, accessApplication cloudflare.AccessApplication) (cloudflare.AccessApplication, error) {
	args := c.Called(zoneID, accessApplication)
	return args.Get(0).(cloudflare.AccessApplication), args.Error(1)
}

func (c *MockClient) DeleteAccessApplication(zoneID, applicationID string) error {
	args := c.Called(zoneID, applicationID)
	return args.Error(0)
}

func (c *MockClient) AccessPolicies(zoneID, applicationID string, pageOpts cloudflare.PaginationOptions) ([]cloudflare.AccessPolicy, cloudflare.ResultInfo, error) {
	args := c.Called(zoneID, applicationID, pageOpts)
	return args.Get(0).([]cloudflare.AccessPolicy), args.Get(1).(cloudflare.ResultInfo), args.Error(2)
}

func (c *MockClient) CreateAccessPolicy(zoneID, applicationID string, accessPolicy cloudflare.AccessPolicy) (cloudflare.AccessPolicy, error) {
	args := c.Called(zoneID, applicationID, accessPolicy)
	return args.Get(0).(cloudflare.AccessPolicy), args.Error(1)
}

func (c *MockClient) UpdateAccessPolicy(zoneID, applicationID string, accessPolicy cloudflare.AccessPolicy) (cloudflare.AccessPolicy, error) {
	args := c.Called(zoneID, applicationID, accessPolicy)
	return args.Get(0).(cloudflare.AccessPolicy), args.Error(1)
}

func (c *MockClient) DeleteAccessPolicy(zoneID, applicationID, accessPolicyID string) error {
	args := c.Called(zoneID, applicationID, accessPolicyID)
	return args.Error(0)

}

var (
	clusterUID = "testCluster"
	Namespace  = "testNamespace"
	zoneName   = "test.zone.name"
)

func TestCheckProvidersDeleteRemote(t *testing.T) {
	mockClient := MockClient{}
	mockClient.On("ZoneIDByName", mock.Anything).
		Return(zoneName, nil)
	mockClient.On("AccessApplications", mock.Anything, mock.Anything).
		Return([]cloudflare.AccessApplication{
			{
				ID:              "testUUID1",
				Name:            strings.Join([]string{clusterUID, Namespace, "testIngress1"}, "-"),
				Domain:          "subdomain.test.zone.name/path",
				SessionDuration: "24h",
			},
			{
				ID:              "testUUID2",
				Name:            strings.Join([]string{clusterUID, Namespace, "testIngress2"}, "-"),
				Domain:          "subdomain2.test.zone.name/path2",
				SessionDuration: "24h",
			},
			{
				ID:              "testUUID3",
				Name:            "DummyAccessApp",
				Domain:          "subdomain3.test.zone.name/path3",
				SessionDuration: "24h",
			},
		}, cloudflare.ResultInfo{}, nil)
	mockClient.On("AccessPolicies", mock.Anything, mock.Anything, mock.Anything).
		Return([]cloudflare.AccessPolicy{}, cloudflare.ResultInfo{}, nil)

	s := Store{
		ClusterUID: clusterUID,
		CloudFlareProvider: &cf.CloudFlareProvider{
			Api: &mockClient,
		},
	}

	planApp, _ := s.CheckProviders([]option.AccessApp{}, []string{zoneName})

	assert.Equal(t, planApp.Deletes[0].App.ID, "testUUID1")
	assert.Equal(t, planApp.Deletes[0].ZoneName, zoneName)
	assert.Equal(t, planApp.Deletes[1].App.ID, "testUUID2")
	assert.Equal(t, planApp.Deletes[1].ZoneName, zoneName)
	assert.Empty(t, planApp.Creates)
	assert.Empty(t, planApp.Updates)
}

func TestCheckProvidersCreateRemote(t *testing.T) {
	mockClient := MockClient{}
	mockClient.On("ZoneIDByName", mock.Anything).
		Return(zoneName, nil)
	mockClient.On("AccessApplications", mock.Anything, mock.Anything).
		Return([]cloudflare.AccessApplication{}, cloudflare.ResultInfo{}, nil)
	mockClient.On("AccessPolicies", mock.Anything, mock.Anything, mock.Anything).
		Return([]cloudflare.AccessPolicy{}, cloudflare.ResultInfo{}, nil)

	s := Store{
		ClusterUID: clusterUID,
		CloudFlareProvider: &cf.CloudFlareProvider{
			Api: &mockClient,
		},
	}

	planApp, _ := s.CheckProviders([]option.AccessApp{
		{
			ClusterUID:  clusterUID,
			Namespace:   Namespace,
			IngressName: "testIngress1",
			AccessAppOption: option.AccessAppOption{
				Domain:          "subdomain.test.zone.name/path",
				SessionDuration: "12h",
			},
			CfZoneName: zoneName,
			Source:     "k8s",
		},
		{
			ClusterUID:  clusterUID,
			Namespace:   Namespace,
			IngressName: "testIngress2",
			AccessAppOption: option.AccessAppOption{
				Domain:          "subdomain2.test.zone.name/",
				SessionDuration: "24h",
			},
			CfZoneName: zoneName,
			Source:     "k8s",
		},
	}, []string{zoneName})

	assert.Equal(t, planApp.Creates[0].ZoneName, zoneName)
	assert.Equal(t, planApp.Creates[0].App.Name, "testCluster-testNamespace-testIngress1")
	assert.Equal(t, planApp.Creates[0].App.Domain, "subdomain.test.zone.name/path")
	assert.Equal(t, planApp.Creates[0].App.SessionDuration, "12h")
	assert.Equal(t, planApp.Creates[1].ZoneName, zoneName)
	assert.Equal(t, planApp.Creates[1].App.Name, "testCluster-testNamespace-testIngress2")
	assert.Equal(t, planApp.Creates[1].App.Domain, "subdomain2.test.zone.name/")
	assert.Equal(t, planApp.Creates[1].App.SessionDuration, "24h")
	assert.Empty(t, planApp.Updates)
	assert.Empty(t, planApp.Deletes)
}

func TestCheckProvidersUpdateRemote(t *testing.T) {
	mockClient := MockClient{}
	mockClient.On("ZoneIDByName", mock.Anything).
		Return(zoneName, nil)
	mockClient.On("AccessApplications", mock.Anything, mock.Anything).
		Return([]cloudflare.AccessApplication{
			{ // Same
				ID:              "testUUID1",
				Name:            strings.Join([]string{clusterUID, Namespace, "testIngress1"}, "-"),
				Domain:          "subdomain.test.zone.name/path",
				SessionDuration: "24h",
			},
			{ // Delete
				ID:              "testUUID2",
				Name:            strings.Join([]string{clusterUID, Namespace, "testIngress2"}, "-"),
				Domain:          "subdomain2.test.zone.name/path2",
				SessionDuration: "12h",
			},
			{ // Update
				ID:              "testUUID3",
				Name:            strings.Join([]string{clusterUID, Namespace, "testIngress3"}, "-"),
				Domain:          "change.subdomain3.test.zone.name/path3",
				SessionDuration: "12h",
			},
			{ // Delete
				ID:              "testUUID4",
				Name:            strings.Join([]string{clusterUID, Namespace, "testIngress4"}, "-"),
				Domain:          "subdomain4.test.zone.name/path4",
				SessionDuration: "30m",
			},
		}, cloudflare.ResultInfo{}, nil)
	mockClient.On("AccessPolicies", mock.Anything, mock.Anything, mock.Anything).
		Return([]cloudflare.AccessPolicy{}, cloudflare.ResultInfo{}, nil)

	s := Store{
		ClusterUID: clusterUID,
		CloudFlareProvider: &cf.CloudFlareProvider{
			Api: &mockClient,
		},
	}

	planApp, _ := s.CheckProviders([]option.AccessApp{
		{
			ClusterUID:  clusterUID,
			Namespace:   Namespace,
			IngressName: "testIngress1",
			AccessAppOption: option.AccessAppOption{
				Domain:          "subdomain.test.zone.name/path",
				SessionDuration: "24h",
			},
			CfZoneName: zoneName,
			Source:     "k8s",
		},
		{
			ClusterUID:  clusterUID,
			Namespace:   Namespace,
			IngressName: "testIngress3",
			AccessAppOption: option.AccessAppOption{
				Domain:          "subdomain3.test.zone.name/",
				SessionDuration: "24h",
			},
			CfZoneName: zoneName,
			Source:     "k8s",
		},
		{ // Create
			ClusterUID:  clusterUID,
			Namespace:   Namespace,
			IngressName: "testIngress5",
			AccessAppOption: option.AccessAppOption{
				Domain:          "subdomain5.test.zone.name/",
				SessionDuration: "60m",
			},
			CfZoneName: zoneName,
			Source:     "k8s",
		},
	}, []string{zoneName})

	assert.Equal(t, planApp.Updates[0].ZoneName, zoneName)
	assert.Equal(t, planApp.Creates[0].ZoneName, zoneName)
	assert.Equal(t, planApp.Creates[0].App.Name, "testCluster-testNamespace-testIngress5")
	assert.Equal(t, planApp.Deletes[0].ZoneName, zoneName)
	assert.Equal(t, planApp.Deletes[0].App.ID, "testUUID2")
	assert.Equal(t, planApp.Deletes[1].ZoneName, zoneName)
	assert.Equal(t, planApp.Deletes[1].App.ID, "testUUID4")
}

func TestCheckProvidersSame(t *testing.T) {
	mockClient := MockClient{}
	mockClient.On("ZoneIDByName", mock.Anything).
		Return(zoneName, nil)
	mockClient.On("AccessApplications", mock.Anything, mock.Anything).
		Return([]cloudflare.AccessApplication{
			{
				ID:              "testUUID1",
				Name:            strings.Join([]string{clusterUID, Namespace, "testIngress1"}, "-"),
				Domain:          "subdomain.test.zone.name/path",
				SessionDuration: "24h",
			},
			{
				ID:              "testUUID2",
				Name:            strings.Join([]string{clusterUID, Namespace, "testIngress2"}, "-"),
				Domain:          "subdomain2.test.zone.name/path2",
				SessionDuration: "12h",
			},
		}, cloudflare.ResultInfo{}, nil)
	mockClient.On("AccessPolicies", mock.Anything, mock.Anything, mock.Anything).
		Return([]cloudflare.AccessPolicy{}, cloudflare.ResultInfo{}, nil)

	s := Store{
		ClusterUID: clusterUID,
		CloudFlareProvider: &cf.CloudFlareProvider{
			Api: &mockClient,
		},
	}

	planApp, _ := s.CheckProviders([]option.AccessApp{
		{
			ClusterUID:  clusterUID,
			Namespace:   Namespace,
			IngressName: "testIngress1",
			AccessAppOption: option.AccessAppOption{
				Domain:          "subdomain.test.zone.name/path",
				SessionDuration: "24h",
			},
			CfZoneName: zoneName,
			Source:     "k8s",
		},
		{
			ClusterUID:  clusterUID,
			Namespace:   Namespace,
			IngressName: "testIngress2",
			AccessAppOption: option.AccessAppOption{
				Domain:          "subdomain2.test.zone.name/path2",
				SessionDuration: "12h",
			},
			CfZoneName: zoneName,
			Source:     "k8s",
		},
	}, []string{zoneName})

	assert.Empty(t, planApp.Creates)
	assert.Empty(t, planApp.Updates)
	assert.Empty(t, planApp.Deletes)
}

func TestEqual(t *testing.T) {
	assert.Equal(t, Equal(nil, make([]interface{}, 0)), true)
	assert.Equal(t, Equal(nil, nil), true)
	assert.Equal(t, Equal(make([]interface{}, 0), make([]interface{}, 0)), true)

	var a1, b1, a2, b2, a3, b3 []interface{}
	var err error
	err = json.Unmarshal([]byte("[]"), &a1)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("{}"), &b1)
	assert.Error(t, err)
	assert.Equal(t, Equal(a1, b1), true)

	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"a.hye\"}}]"), &a2)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}}]"), &b2)
	assert.Empty(t, err)
	assert.Equal(t, Equal(a2, b2), false)

	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.123\"}}]"), &a3)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.122\"}}]"), &b3)
	assert.Empty(t, err)
	assert.Equal(t, Equal(a3, b3), false)

	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.123\"}}]"), &a3)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.122\"}}]"), &b3)
	assert.Empty(t, err)
	assert.Equal(t, Equal(a3, b3), false)

	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.123\"}}]"), &a3)
	assert.Empty(t, err)
	err = json.Unmarshal([]byte("[{\"email\": {\"domain\": \"ma.hye\"}},{\"ip\":{\"ip\":\"123.123.123.123\"}}]"), &b3)
	assert.Empty(t, err)
	assert.Equal(t, Equal(a3, b3), true)
}
