package store

import (
	"errors"
	"github.com/cloudflare/cloudflare-go"
	"github.com/go-logr/logr"
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
		s.log.Error(err, "Cannot get accessApplications")
		return err
	}

	s.apps = res
	return nil
}

func New(client *cloudflare.API, zoneId string) *Store {
	return &Store{
		client: client,
		zoneId: zoneId,
	}
}
