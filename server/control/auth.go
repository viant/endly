package control

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/viant/scy"
	scyjwt "github.com/viant/scy/auth/jwt"
	"github.com/viant/scy/auth/jwt/verifier"
)

type Authenticator interface {
	Authenticate(context.Context, string) (*scyjwt.Claims, error)
}

type JWTAuthenticator struct {
	verifier *verifier.Service
	issuer   string
	audience string
	scope    string
}

func NewJWTAuthenticator(ctx context.Context, publicKeyResource, issuer, audience, scope string) (*JWTAuthenticator, error) {
	if strings.TrimSpace(publicKeyResource) == "" {
		return nil, errors.New("JWT public key resource was empty")
	}
	resource := scy.EncodedResource(publicKeyResource).Decode(ctx, nil)
	return NewJWTAuthenticatorWithResource(ctx, resource, issuer, audience, scope)
}

func NewJWTAuthenticatorWithResource(ctx context.Context, resource *scy.Resource, issuer, audience, scope string) (*JWTAuthenticator, error) {
	jwtVerifier := verifier.New(&verifier.Config{RSA: []*scy.Resource{resource}})
	if err := jwtVerifier.Init(ctx); err != nil {
		return nil, fmt.Errorf("initialize scy JWT verifier: %w", err)
	}
	return &JWTAuthenticator{verifier: jwtVerifier, issuer: issuer, audience: audience, scope: scope}, nil
}

func (a *JWTAuthenticator) Authenticate(ctx context.Context, token string) (*scyjwt.Claims, error) {
	if token == "" {
		return nil, errors.New("JWT was empty")
	}
	claims, err := a.verifier.VerifyClaims(ctx, token)
	if err != nil {
		return nil, err
	}
	if a.issuer != "" && claims.Issuer != a.issuer {
		return nil, fmt.Errorf("unexpected JWT issuer %q", claims.Issuer)
	}
	if a.audience != "" && !claims.VerifyAudience(a.audience, true) {
		return nil, fmt.Errorf("JWT audience %q was required", a.audience)
	}
	if a.scope != "" && !containsScope(claims.Scope, a.scope) {
		return nil, fmt.Errorf("JWT scope %q was required", a.scope)
	}
	if claims.Subject == "" {
		return nil, errors.New("JWT subject was empty")
	}
	return claims, nil
}

func containsScope(actual, required string) bool {
	for _, candidate := range strings.Fields(actual) {
		if candidate == required {
			return true
		}
	}
	return false
}

type principalContextKey struct{}

func principalFromContext(ctx context.Context) *scyjwt.Claims {
	claims, _ := ctx.Value(principalContextKey{}).(*scyjwt.Claims)
	return claims
}
