package store

import (
	"context"
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

func (s *Store) GetApplication(ctx context.Context, name string) (cloudflare.AccessApplication, error) {
	if err := s.getApplications(ctx); err != nil {
		return cloudflare.AccessApplication{}, err
	}

	for _, app := range s.apps {
		if app.Name == name {
			return app, nil
		}
	}

	return cloudflare.AccessApplication{}, ApplicationNotFoundError
}

func (s *Store) getApplications(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res, _, err := s.client.AccessApplications(ctx, s.zoneId, cloudflare.PaginationOptions{})
	if err != nil {
		s.log.Error(err, "Cannot get access applications")
		return err
	}

	s.apps = res
	return nil
}

func (s *Store) GetPolicies(ctx context.Context, appId string) ([]cloudflare.AccessPolicy, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res, _, err := s.client.AccessPolicies(ctx, s.zoneId, appId, cloudflare.PaginationOptions{})
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

func New(client *cloudflare.API, zoneId string, log logr.Logger) *Store {
	return &Store{
		client: client,
		zoneId: zoneId,
		log:    log,
	}
}
