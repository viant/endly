package credential

import (
	"context"

	"github.com/viant/scy"
	"github.com/viant/scy/cred"
	"github.com/viant/scy/cred/secret"
)

// SecretLookup is the secret surface used by endly workflow context.
type SecretLookup interface {
	Lookup(ctx context.Context, resource secret.Resource) (*scy.Secret, error)
	Expand(ctx context.Context, input string, secrets map[secret.Key]secret.Resource) (string, error)
	GetCredentials(ctx context.Context, resource string) (*cred.Generic, error)
	GeyKey(ctx context.Context, resource string) (*cred.SecretKey, error)
}

// Service wraps scy secret lookup with e2e alias resolution.
type Service struct {
	inner    *secret.Service
	registry *Registry
}

// NewService creates a secret service that resolves e2e credential aliases before lookup.
func NewService(opts ...secret.Option) *Service {
	return &Service{
		inner:    secret.New(opts...),
		registry: NewRegistry(),
	}
}

func (s *Service) resolveResource(resource secret.Resource) (secret.Resource, error) {
	resolved, err := s.registry.Resolve(resource.URL())
	if err != nil {
		return "", err
	}
	if resolved == resource.URL() {
		return resource, nil
	}
	if key := resource.Key(); key != "" {
		return secret.Resource(resolved + "|" + key), nil
	}
	return secret.Resource(resolved), nil
}

// Lookup loads a secret after resolving e2e aliases.
func (s *Service) Lookup(ctx context.Context, resource secret.Resource) (*scy.Secret, error) {
	resolved, err := s.resolveResource(resource)
	if err != nil {
		return nil, err
	}
	return s.inner.Lookup(ctx, resolved)
}

// GetCredentials returns credentials for resource after alias resolution.
func (s *Service) GetCredentials(ctx context.Context, resource string) (*cred.Generic, error) {
	resolved, err := s.resolveResource(secret.Resource(resource))
	if err != nil {
		return nil, err
	}
	return s.inner.GetCredentials(ctx, resolved.String())
}

// GeyKey returns secret key for resource after alias resolution.
func (s *Service) GeyKey(ctx context.Context, resource string) (*cred.SecretKey, error) {
	resolved, err := s.resolveResource(secret.Resource(resource))
	if err != nil {
		return nil, err
	}
	return s.inner.GeyKey(ctx, resolved.String())
}

// Expand expands secret placeholders after resolving configured aliases.
func (s *Service) Expand(ctx context.Context, input string, secrets map[secret.Key]secret.Resource) (string, error) {
	if len(secrets) == 0 {
		return input, nil
	}
	resolved := make(map[secret.Key]secret.Resource, len(secrets))
	for key, resource := range secrets {
		value, err := s.resolveResource(resource)
		if err != nil {
			return "", err
		}
		resolved[key] = value
	}
	return s.inner.Expand(ctx, input, resolved)
}
