package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/signer"
)

type TokenProvider func(context.Context) (string, error)

func NewJWTTokenProvider(ctx context.Context, privateKeyResource, issuer, audience, scope, subject string, ttl time.Duration) (TokenProvider, error) {
	if strings.TrimSpace(privateKeyResource) == "" {
		return nil, errors.New("JWT private key resource was empty")
	}
	if subject == "" {
		return nil, errors.New("JWT subject was empty")
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	resource := scy.EncodedResource(privateKeyResource).Decode(ctx, nil)
	return NewJWTTokenProviderWithResource(ctx, resource, issuer, audience, scope, subject, ttl)
}

func NewJWTTokenProviderWithResource(ctx context.Context, resource *scy.Resource, issuer, audience, scope, subject string, ttl time.Duration) (TokenProvider, error) {
	jwtSigner := signer.New(&signer.Config{RSA: resource})
	if err := jwtSigner.Init(ctx); err != nil {
		return nil, fmt.Errorf("initialize scy JWT signer: %w", err)
	}
	return func(_ context.Context) (string, error) {
		claims := map[string]interface{}{
			"iss":   issuer,
			"sub":   subject,
			"aud":   []string{audience},
			"scope": scope,
		}
		return jwtSigner.Create(ttl, claims)
	}, nil
}
