package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/viant/scy"
	"github.com/viant/scy/cred"
	"github.com/viant/scy/cred/secret"
)

type bearerTestSecrets struct {
	token   string
	failure error
	lookups int
}

func (s *bearerTestSecrets) Lookup(_ context.Context, _ secret.Resource) (*scy.Secret, error) {
	s.lookups++
	if s.failure != nil {
		return nil, s.failure
	}
	return scy.NewSecret(s.token, &scy.Resource{URL: "memory://token"}), nil
}
func (*bearerTestSecrets) Expand(_ context.Context, input string, _ map[secret.Key]secret.Resource) (string, error) {
	return input, nil
}
func (*bearerTestSecrets) GetCredentials(context.Context, string) (*cred.Generic, error) {
	return nil, nil
}
func (*bearerTestSecrets) GeyKey(context.Context, string) (*cred.SecretKey, error) { return nil, nil }
func TestHTTPBearerSecretIsOutboundOnly(t *testing.T) {
	ctx := newContextWithState(nil)
	secrets := &bearerTestSecrets{token: "  private-test-token  "}
	ctx.Secrets = secrets
	r := &Request{Method: "PATCH", URL: "http://127.0.0.1:12345/resource", BearerTokenSecret: "owned-token-file", Header: http.Header{"Content-Type": []string{"application/json"}}, JSONBody: map[string]any{"enabled": false}}
	outbound, _, err := r.Build(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outbound.Header.Get("Authorization") != "Bearer private-test-token" || secrets.lookups != 1 {
		t.Fatal("outbound credential was not resolved")
	}
	if r.Header.Get("Authorization") != "" {
		t.Fatal("credential escaped into recorded request")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private-test-token") {
		t.Fatal("credential escaped into serialized workflow request")
	}
	if r.Clone(ctx).BearerTokenSecret != "owned-token-file" {
		t.Fatal("cloning dropped the secret reference")
	}
}
func TestHTTPBearerSecretFailureDoesNotLeakCredential(t *testing.T) {
	for _, tc := range []struct {
		name, token string
		failure     error
	}{{"lookup", "", errors.New("private-test-token must not leak")}, {"empty", "  ", nil}, {"control", "private-test-token\r\n", nil}, {"internal control", "private-test-token\r\ninjected", nil}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := newContextWithState(nil)
			ctx.Secrets = &bearerTestSecrets{token: tc.token, failure: tc.failure}
			r := &Request{Method: "GET", URL: "https://example.test/resource", BearerTokenSecret: "reference"}
			_, _, err := r.Build(ctx, nil)
			// Leading/trailing whitespace follows existing MCP secret trimming.
			if tc.name == "control" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || strings.Contains(err.Error(), "private-test-token") {
				t.Fatalf("unsafe lookup failure: %v", err)
			}
		})
	}
}
func TestHTTPBearerSecretRejectsAmbiguousOrInsecureDestinationsBeforeLookup(t *testing.T) {
	for _, tc := range []struct {
		url    string
		header http.Header
	}{{"http://external.test/resource", nil}, {"https://user:password@example.test/resource", nil}, {"https://example.test/resource", http.Header{"authorization": []string{"Basic value"}}}} {
		ctx := newContextWithState(nil)
		secrets := &bearerTestSecrets{token: "private-test-token"}
		ctx.Secrets = secrets
		r := &Request{Method: "GET", URL: tc.url, Header: tc.header, BearerTokenSecret: "reference"}
		if _, _, err := r.Build(ctx, nil); err == nil || secrets.lookups != 0 {
			t.Fatal("invalid request resolved credentials")
		}
	}
}

func TestHTTPBearerSecretSendRetainsNativeDenialStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer private-test-token" {
			t.Error("verified credential was not sent")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status":"error","message":"permission denied"}`))
	}))
	defer server.Close()
	ctx := newContextWithState(nil)
	ctx.Secrets = &bearerTestSecrets{token: "private-test-token"}
	input := &SendRequest{Requests: []*Request{{Method: "PATCH", URL: server.URL, BearerTokenSecret: "owned-reference", JSONBody: map[string]any{"enabled": false}, Expect: map[string]interface{}{"Code": 403, "JSONBody": map[string]interface{}{"status": "error"}}}}}
	if err := input.Init(); err != nil {
		t.Fatal(err)
	}
	response, err := New().(*service).send(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Responses) != 1 || response.Responses[0].Code != 403 || response.Assert == nil || response.Assert.FailedCount != 0 {
		t.Fatal("native HTTP status proof failed")
	}
	for _, value := range []any{input, response} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "private-test-token") {
			t.Fatal("credential entered native operation evidence")
		}
	}
}
