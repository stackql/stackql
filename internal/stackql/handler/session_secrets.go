package handler

import (
	"context"
	"slices"
	"sync"

	"github.com/stackql-labs/omnisdk/public/secrets"
)

// sessionSecretStore belongs to one client session and never accesses persistent storage.
type sessionSecretStore struct {
	mu     sync.Mutex
	values map[string][]byte
}

func newSessionSecretStore() secrets.Store {
	return &sessionSecretStore{values: make(map[string][]byte)}
}

func (s *sessionSecretStore) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	value, exists := s.values[key]
	if !exists {
		return nil, secrets.ErrNotFound
	}
	return slices.Clone(value), nil
}

func (s *sessionSecretStore) Set(ctx context.Context, key string, value []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = slices.Clone(value)
	return nil
}

func (s *sessionSecretStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}
