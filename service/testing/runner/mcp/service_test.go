package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/endly"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/authorizer"
	"github.com/viant/scy/cred"
	"github.com/viant/scy/cred/secret"
	"github.com/viant/toolbox"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

type testSecrets struct{ token string }

func (s testSecrets) Lookup(_ context.Context, _ secret.Resource) (*scy.Secret, error) {
	return scy.NewSecret(s.token, &scy.Resource{URL: "memory://token"}), nil
}
func (testSecrets) Expand(_ context.Context, input string, _ map[secret.Key]secret.Resource) (string, error) {
	return input, nil
}
func (testSecrets) GetCredentials(context.Context, string) (*cred.Generic, error) { return nil, nil }
func (testSecrets) GeyKey(context.Context, string) (*cred.SecretKey, error)       { return nil, nil }

func mockServer(t *testing.T, handler func(http.ResponseWriter, *http.Request, string, map[string]interface{})) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     json.RawMessage        `json:"id"`
			Method string                 `json:"method"`
			Params map[string]interface{} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if request.Method == schema.MethodServerDiscover {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": request.ID, "result": map[string]interface{}{
				"supportedVersions": []string{schema.LatestProtocolVersion}, "capabilities": map[string]interface{}{}, "resultType": "complete", "cacheScope": "public", "ttlMs": 0,
			}})
			return
		}
		writer := responseWriter{ResponseWriter: w, id: request.ID}
		handler(writer, r, request.Method, request.Params)
	}))
}

type responseWriter struct {
	http.ResponseWriter
	id json.RawMessage
}

func (w responseWriter) result(value interface{}) {
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": w.id, "result": value})
}

func TestListToolsAndCall(t *testing.T) {
	var calls atomic.Int32
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request, method string, params map[string]interface{}) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get(schema.HeaderProtocolVersion); got != schema.LatestProtocolVersion {
			t.Errorf("protocol = %q", got)
		}
		rw := w.(responseWriter)
		switch method {
		case schema.MethodToolsList:
			rw.result(map[string]interface{}{"tools": []interface{}{map[string]interface{}{"name": "echo", "inputSchema": map[string]interface{}{"type": "object"}}}, "nextCursor": "page-2", "resultType": "complete", "cacheScope": "private", "ttlMs": 0})
		case schema.MethodToolsCall:
			calls.Add(1)
			if params["name"] != "echo" {
				t.Errorf("tool = %v", params["name"])
			}
			arguments, _ := params["arguments"].(map[string]interface{})
			if arguments["value"] != "hello" {
				t.Errorf("arguments = %#v", arguments)
			}
			rw.result(map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "hello"}}, "structuredContent": map[string]interface{}{"value": "hello"}, "resultType": "complete"})
		default:
			t.Errorf("unexpected method %q", method)
		}
	})
	defer server.Close()
	ctx := &endly.Context{Secrets: testSecrets{token: "test-token"}}
	s := New().(*service)
	list, err := s.listTools(ctx, &ListToolsRequest{Request: Request{URL: server.URL, BearerTokenSecret: "test-alias"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 1 || list.Tools[0].Name != "echo" || list.NextCursor == nil || *list.NextCursor != "page-2" {
		t.Fatalf("list = %#v", list)
	}
	response, err := s.call(ctx, &CallRequest{Request: Request{URL: server.URL, BearerTokenSecret: "test-alias"}, Name: "echo", Arguments: map[string]interface{}{"value": "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	structured, ok := response.StructuredContent.(map[string]interface{})
	if !ok || structured["value"] != "hello" || response.IsError || calls.Load() != 1 {
		t.Fatalf("call = %#v, count = %d", response, calls.Load())
	}
}

func TestCallIsErrorPreservesResult(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, _ *http.Request, method string, _ map[string]interface{}) {
		if method != schema.MethodToolsCall {
			t.Errorf("method = %q", method)
		}
		w.(responseWriter).result(map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": "failed"}}, "isError": true, "resultType": "complete"})
	})
	defer server.Close()
	response, err := New().(*service).call(&endly.Context{}, &CallRequest{Request: Request{URL: server.URL}, Name: "fail"})
	if err == nil || response == nil || !response.IsError || len(response.Content) != 1 {
		t.Fatalf("response = %#v, error = %v", response, err)
	}
	allowed, err := New().(*service).call(&endly.Context{}, &CallRequest{Request: Request{URL: server.URL}, Name: "fail", AllowToolError: true})
	if err != nil || allowed == nil || !allowed.IsError || len(allowed.Content) != 1 {
		t.Fatalf("allowed response = %#v, error = %v", allowed, err)
	}
}

func TestCallTimeout(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request, _ string, _ map[string]interface{}) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(time.Second):
			w.(responseWriter).result(map[string]interface{}{"content": []interface{}{}, "resultType": "complete"})
		}
	})
	defer server.Close()
	start := time.Now()
	_, err := New().(*service).call(&endly.Context{}, &CallRequest{Request: Request{URL: server.URL, TimeoutMs: 25}, Name: "slow"})
	if err == nil || time.Since(start) > 500*time.Millisecond {
		t.Fatalf("timeout error = %v, elapsed = %v", err, time.Since(start))
	}
}

func TestErrorRedactsToken(t *testing.T) {
	token := "sensitive-token"
	server := mockServer(t, func(w http.ResponseWriter, _ *http.Request, _ string, _ map[string]interface{}) {
		writer := w.(responseWriter)
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"%s"}}`, writer.id, token)
	})
	defer server.Close()
	ctx := &endly.Context{Secrets: testSecrets{token: token}}
	_, err := New().(*service).call(ctx, &CallRequest{Request: Request{URL: server.URL, BearerTokenSecret: "alias"}, Name: "fail"})
	if err == nil || strings.Contains(err.Error(), token) {
		t.Fatalf("unredacted error: %v", err)
	}
}

func TestOAuthUsesScyReferencesAndAccessToken(t *testing.T) {
	var authorizationCalls atomic.Int32
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request, _ string, _ map[string]interface{}) {
		if got := r.Header.Get("Authorization"); got != "Bearer oauth-access-token" {
			t.Errorf("Authorization = %q", got)
		}
		w.(responseWriter).result(map[string]interface{}{"content": []interface{}{}, "resultType": "complete"})
	})
	defer server.Close()
	s := New().(*service)
	s.authorizeOAuth = func(_ context.Context, command *authorizer.Command) (*oauth2.Token, error) {
		authorizationCalls.Add(1)
		if command.ConfigURL != "oauth-config|blowfish://default" || command.SecretsURL != "basic-secret|blowfish://default" || command.AuthFlow != "OOB" || !command.UsePKCE || len(command.Scopes) != 1 || command.Scopes[0] != "read" {
			t.Errorf("unexpected OAuth command: config=%q secrets=%q flow=%q scopes=%v PKCE=%v", command.ConfigURL, command.SecretsURL, command.AuthFlow, command.Scopes, command.UsePKCE)
		}
		return &oauth2.Token{AccessToken: "oauth-access-token"}, nil
	}
	_, err := s.call(&endly.Context{}, &CallRequest{Request: Request{URL: server.URL, OAuth: &OAuthRequest{ConfigURL: "oauth-config|blowfish://default", SecretsURL: "basic-secret|blowfish://default", AuthFlow: "OOB", Scopes: []string{"read"}, UsePKCE: true}}, Name: "secured"})
	if err != nil || authorizationCalls.Load() != 1 {
		t.Fatalf("error = %v, authorization calls = %d", err, authorizationCalls.Load())
	}
}

func TestToolCallIsNeverRetried(t *testing.T) {
	var calls atomic.Int32
	server := mockServer(t, func(w http.ResponseWriter, _ *http.Request, _ string, _ map[string]interface{}) {
		calls.Add(1)
		http.Error(w, "session 'abc' not found", http.StatusNotFound)
	})
	defer server.Close()
	_, err := New().(*service).call(&endly.Context{}, &CallRequest{Request: Request{URL: server.URL}, Name: "mutate"})
	if err == nil || calls.Load() != 1 {
		t.Fatalf("error = %v, calls = %d", err, calls.Load())
	}
}

func TestAuthenticatedRedirectIsRejected(t *testing.T) {
	var targetCalls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetCalls.Add(1) }))
	defer target.Close()
	server := mockServer(t, func(w http.ResponseWriter, _ *http.Request, _ string, _ map[string]interface{}) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
	})
	defer server.Close()
	ctx := &endly.Context{Secrets: testSecrets{token: "test-token"}}
	_, err := New().(*service).call(ctx, &CallRequest{Request: Request{URL: server.URL, BearerTokenSecret: "alias"}, Name: "mutate"})
	if err == nil || targetCalls.Load() != 0 {
		t.Fatalf("error = %v, target calls = %d", err, targetCalls.Load())
	}
}

func TestRequestRejectsInsecureRemoteAuthAndMixedMethods(t *testing.T) {
	requests := []Request{
		{URL: "http://example.com/mcp", BearerTokenSecret: "alias"},
		{URL: "http://example.com/mcp", OAuth: &OAuthRequest{ConfigURL: "config", SecretsURL: "basic"}},
		{URL: "https://example.com/mcp", BearerTokenSecret: "alias", OAuth: &OAuthRequest{ConfigURL: "config", SecretsURL: "basic"}},
		{URL: "https://example.com/mcp", TimeoutMs: 24*60*60*1000 + 1},
	}
	for _, request := range requests {
		if err := request.Validate(); err == nil {
			t.Errorf("expected validation error for %#v", request)
		}
	}
}

func TestEndlyRouteRunsValidation(t *testing.T) {
	response := New().Run(&endly.Context{}, &CallRequest{
		Request: Request{URL: "http://example.com/mcp", BearerTokenSecret: "alias"},
		Name:    "mutate",
	})
	if response.Err == nil || !strings.Contains(response.Error, "HTTPS or loopback HTTP") {
		t.Fatalf("route response = %#v", response)
	}
}

func TestWorkflowRequestConversion(t *testing.T) {
	request := &CallRequest{}
	err := toolbox.DefaultConverter.AssignConverted(request, map[string]interface{}{
		"url": "https://example.com/mcp",
		"oauth": map[string]interface{}{
			"configURL":  "oauth-config|blowfish://default",
			"secretsURL": "basic-secret|blowfish://default",
			"authFlow":   "OOB",
			"scopes":     []interface{}{"read"},
			"usePKCE":    true,
		},
		"name":      "describe",
		"arguments": map[string]interface{}{"entity": "sample"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.URL != "https://example.com/mcp" || request.OAuth == nil || request.OAuth.ConfigURL != "oauth-config|blowfish://default" || request.OAuth.SecretsURL != "basic-secret|blowfish://default" || request.OAuth.AuthFlow != "OOB" || !request.OAuth.UsePKCE || request.Name != "describe" {
		t.Fatalf("workflow request conversion = %#v", request)
	}
}

func TestEndlyAsRequestPreservesNestedJSONSource(t *testing.T) {
	const source = `{"version":1,"job":{"name":"sample","tasks":[{"id":"inspect","options":{}}]}}`
	workflowYAML := `url: http://127.0.0.1:8080/mcp
name: validate_source
arguments:
  format: json
  source: '` + source + `'`
	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(workflowYAML), &raw); err != nil {
		t.Fatal(err)
	}
	ctx := endly.New().NewContext(toolbox.NewContext())
	converted, err := ctx.AsRequest(ServiceID, "call", raw)
	if err != nil {
		t.Fatal(err)
	}
	request, ok := converted.(*CallRequest)
	if !ok {
		t.Fatalf("request type = %T", converted)
	}
	actual, ok := request.Arguments["source"].(string)
	if !ok || actual != source {
		t.Fatalf("source was altered: type %T, value %v", request.Arguments["source"], request.Arguments["source"])
	}
}
