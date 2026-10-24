package credential

import (
	"context"
	"errors"
	"testing"

	"github.com/viant/scy"
	"github.com/viant/scy/cred/secret"
)

type freshProvider struct {
	SecretLookup
	seen  secret.Resource
	calls int
	fail  bool
}

func (p *freshProvider) Lookup(context.Context, secret.Resource) (*scy.Secret, error) {
	panic("fresh errors must not fall back to cached lookup")
}
func (p *freshProvider) LookupFresh(_ context.Context, r secret.Resource) (*scy.Secret, error) {
	p.calls++
	p.seen = r
	if p.fail {
		return nil, errors.New("fixture unavailable")
	}
	return scy.NewSecret("fixture", &scy.Resource{URL: "memory://fixture"}), nil
}

func TestLookupFreshResolvesAliasAndPreservesKey(t *testing.T) {
	svc := NewService()
	provider := &freshProvider{}
	svc.inner = provider
	svc.SetCredentialMap(map[string]string{"rotating": "file:///fixture/bearer.token"})
	if _, err := svc.LookupFresh(context.Background(), secret.Resource("rotating|blowfish://fixture")); err != nil {
		t.Fatal(err)
	}
	if provider.seen != "file:///fixture/bearer.token|blowfish://fixture" || provider.calls != 1 {
		t.Fatal("fresh provider did not receive the resolved keyed resource")
	}
	provider.fail = true
	if _, err := svc.LookupFresh(context.Background(), secret.Resource("rotating")); err == nil {
		t.Fatal("fresh provider failure did not propagate")
	}
}
