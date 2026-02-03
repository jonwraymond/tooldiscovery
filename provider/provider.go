package provider

import (
	"errors"
	"sort"
	"sync"

	"github.com/jonwraymond/toolfoundation/adapter"
)

// Error values for consistent error handling by callers.
var (
	ErrNotFound          = errors.New("provider not found")
	ErrInvalidProvider   = errors.New("invalid provider")
	ErrInvalidProviderID = errors.New("invalid provider id")
)

// Store defines provider discovery operations.
type Store interface {
	// RegisterProvider registers a provider and returns its resolved ID.
	RegisterProvider(id string, provider adapter.CanonicalProvider) (string, error)
	// DescribeProvider returns a provider by ID.
	DescribeProvider(id string) (adapter.CanonicalProvider, error)
	// ListProviders returns all registered providers in stable order.
	ListProviders() ([]adapter.CanonicalProvider, error)
}

// InMemoryStore stores providers in memory.
type InMemoryStore struct {
	mu        sync.RWMutex
	providers map[string]adapter.CanonicalProvider
}

// NewInMemoryStore creates a new provider store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		providers: make(map[string]adapter.CanonicalProvider),
	}
}

// ProviderID returns a stable provider ID from name/version.
func ProviderID(name, version string) string {
	if name == "" {
		return ""
	}
	if version == "" {
		return name
	}
	return name + ":" + version
}

// RegisterProvider registers a provider and returns its resolved ID.
func (s *InMemoryStore) RegisterProvider(id string, provider adapter.CanonicalProvider) (string, error) {
	if provider.Name == "" {
		return "", ErrInvalidProvider
	}
	if id == "" {
		id = ProviderID(provider.Name, provider.Version)
	}
	if id == "" {
		return "", ErrInvalidProviderID
	}

	s.mu.Lock()
	s.providers[id] = provider
	s.mu.Unlock()

	return id, nil
}

// DescribeProvider returns a provider by ID.
func (s *InMemoryStore) DescribeProvider(id string) (adapter.CanonicalProvider, error) {
	if id == "" {
		return adapter.CanonicalProvider{}, ErrInvalidProviderID
	}

	s.mu.RLock()
	provider, ok := s.providers[id]
	s.mu.RUnlock()

	if !ok {
		return adapter.CanonicalProvider{}, ErrNotFound
	}
	return provider, nil
}

// ListProviders returns all registered providers in stable order.
func (s *InMemoryStore) ListProviders() ([]adapter.CanonicalProvider, error) {
	s.mu.RLock()
	ids := make([]string, 0, len(s.providers))
	for id := range s.providers {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	sort.Strings(ids)

	result := make([]adapter.CanonicalProvider, 0, len(ids))
	s.mu.RLock()
	for _, id := range ids {
		result = append(result, s.providers[id])
	}
	s.mu.RUnlock()

	return result, nil
}
