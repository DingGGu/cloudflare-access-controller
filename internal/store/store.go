package store

import (
	"errors"
	"github.com/cloudflare/cloudflare-go"
	"github.com/go-logr/logr"
	"strings"
	"sync"
)

var ApplicationNotFoundError = errors.New("ApplicationNotFound")

type Store struct {
	client *cloudflare.API
	apps   []cloudflare.AccessApplication
	mutex  sync.Mutex
	zoneId string
	log    logr.Logger
}

func (s *Store) GetApplication(name string) (cloudflare.AccessApplication, error) {
	if err := s.getApplications(); err != nil {
		return cloudflare.AccessApplication{}, err
	}

	for _, app := range s.apps {
		if app.Name == name {
			return app, nil
		}
	}

	return cloudflare.AccessApplication{}, ApplicationNotFoundError
}

func (s *Store) getApplications() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res, _, err := s.client.AccessApplications(s.zoneId, cloudflare.PaginationOptions{})
	if err != nil {
		s.log.Error(err, "Cannot get access applications")
		return err
	}

	s.apps = res
	return nil
}

func (s *Store) GetPolicies(appId string) ([]cloudflare.AccessPolicy, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res, _, err := s.client.AccessPolicies(s.zoneId, appId, cloudflare.PaginationOptions{})
	if err != nil {
		return nil, err
	}

	var policies []cloudflare.AccessPolicy

	for _, item := range res {
		if strings.HasPrefix(item.Name, "cac-policy-") {
			policies = append(policies, item)
		}
	}

	return policies, nil
}

func New(client *cloudflare.API, zoneId string) *Store {
	return &Store{
		client: client,
		zoneId: zoneId,
	}
}
