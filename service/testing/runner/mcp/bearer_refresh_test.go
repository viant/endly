package mcp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/endly/service/credential"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy"
	"github.com/viant/scy/cred/secret"
)

// Both the real published Scy provider and Endly's alias wrapper must expose
// fresh lookup. This regression must not silently fall back to an older Scy pin.
var _ freshSecretLookup = (*secret.Service)(nil)
var _ freshSecretLookup = (*credential.Service)(nil)

func fixtureJWT(expiry int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]interface{}{"sub": "fixture-only", "exp": expiry})
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte("test-owned-signing-key"))
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func validFixtureJWT(token string, now int64) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte("test-owned-signing-key"))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return false
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(data, &claims) != nil {
		return false
	}
	return claims.Exp > now
}
func TestNativeMCPRealScyCredentialAliasRotationAfterExpiry(t *testing.T) {
	dir := t.TempDir()
	file, key := filepath.Join(dir, "bearer.token"), filepath.Join(dir, "signing.key")
	var now atomic.Int64
	now.Store(1000)
	first := fixtureJWT(1001)
	second := fixtureJWT(1900)
	if err := os.WriteFile(file, []byte(first), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("static-fixture-key"), 0600); err != nil {
		t.Fatal(err)
	}
	provider := credential.NewService()
	provider.SetCredentialMap(map[string]string{"rotating-bearer": file})
	old, err := provider.Lookup(context.Background(), secret.Resource("rotating-bearer"))
	if err != nil {
		t.Fatal(err)
	}
	static, err := provider.Lookup(context.Background(), secret.Resource(key))
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request, method string, _ map[string]interface{}) {
		if !validFixtureJWT(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), now.Load()) {
			http.Error(w, "expired fixture credential", http.StatusUnauthorized)
			return
		}
		if method != schema.MethodToolsCall {
			t.Errorf("unexpected native method")
		}
		calls.Add(1)
		w.(responseWriter).result(map[string]interface{}{"content": []interface{}{}, "structuredContent": map[string]interface{}{"accepted": true}, "resultType": "complete"})
	})
	defer server.Close()
	manager := endly.New()
	manager.Register(New())
	ctx := manager.NewContext(nil)
	defer ctx.Close()
	ctx.Secrets = provider
	request := &CallRequest{Request: Request{URL: server.URL, BearerTokenSecret: "rotating-bearer"}, Name: "fixture_read"}
	var response CallResponse
	if err = endly.Run(ctx, request, &response); err != nil {
		t.Fatal("first native operation failed")
	}
	now.Store(1002)
	if validFixtureJWT(old.String(), now.Load()) {
		t.Fatal("test did not advance beyond old expiry")
	}
	temp := file + ".next"
	if err = os.WriteFile(temp, []byte(second), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(temp, file); err != nil {
		t.Fatal(err)
	}
	if err = endly.Run(ctx, request, &response); err != nil {
		t.Fatal("same-session native operation failed after rotation")
	}
	if calls.Load() != 2 {
		t.Fatal("native operations did not both reach the authenticated producer")
	}
	stillCached, err := provider.Lookup(context.Background(), secret.Resource("rotating-bearer"))
	if err != nil || stillCached != old {
		t.Fatal("bearer refresh reset the ordinary cache")
	}
	stillStatic, err := provider.Lookup(context.Background(), secret.Resource(key))
	if err != nil || stillStatic != static {
		t.Fatal("bearer refresh reset a static key")
	}
	if err = os.Remove(file); err != nil {
		t.Fatal(err)
	}
	err = endly.Run(ctx, request, &response)
	if err == nil {
		t.Fatal("missing rotated resource reused a stale bearer")
	}
	if strings.Contains(err.Error(), first) || strings.Contains(err.Error(), second) {
		t.Fatal("credential escaped into native lookup error")
	}
	if calls.Load() != 2 {
		t.Fatal("failed fresh lookup reached the producer")
	}
}

// A provider may include sensitive data in its internal error. The native MCP
// boundary must not disclose it or retry using a cached, expired bearer.
type failingFreshProvider struct{ testSecrets }

func (p failingFreshProvider) LookupFresh(context.Context, secret.Resource) (*scy.Secret, error) {
	return nil, errors.New(p.token)
}
func TestFreshBearerFailureIsRedactedAndNeverUsesCache(t *testing.T) {
	token := "fixture-private-bearer-must-not-escape"
	ctx := &endly.Context{Secrets: failingFreshProvider{testSecrets{token: token}}}
	_, cancel, value, err := New().(*service).operationContext(ctx, &Request{BearerTokenSecret: "fixture"})
	if cancel != nil {
		cancel()
	}
	if err == nil || value != "" || strings.Contains(err.Error(), token) {
		t.Fatal("fresh lookup failure was not fail-closed and redacted")
	}
}
