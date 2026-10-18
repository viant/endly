package mcp

import (
	"encoding/json"
	"github.com/viant/endly"
	"github.com/viant/mcp-protocol/schema"
	"net/http"
	"strings"
	"testing"
)

func TestNativeResourceAndSkillActions(t *testing.T) {
	skill := map[string]interface{}{"uri": "skill://namespace/guide/SKILL.md", "frontmatter": map[string]interface{}{"name": "guide", "description": "Namespace guide"}, "resources": []interface{}{map[string]interface{}{"uri": "skill://namespace/guide/SKILL.md", "digest": "sha256:fixture", "size": 42}}}
	calls := map[string]int{}
	server := mockServer(t, func(w http.ResponseWriter, r *http.Request, method string, params map[string]interface{}) {
		calls[method]++
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("verified secret was not attached")
		}
		rw := w.(responseWriter)
		switch method {
		case schema.MethodResourcesList:
			if params["cursor"] != "next" {
				t.Error("resource pagination cursor lost")
			}
			rw.result(map[string]interface{}{"resources": []interface{}{map[string]interface{}{"uri": "mcp://ns/sales", "name": "sales", "mimeType": "application/yaml"}}, "resultType": "complete", "cacheScope": "private", "ttlMs": 0})
		case schema.MethodResourcesRead:
			if r.Header.Get(schema.HeaderName) != "mcp://ns/sales" || params["uri"] != "mcp://ns/sales" {
				t.Error("resource routing identity lost")
			}
			rw.result(map[string]interface{}{"contents": []interface{}{map[string]interface{}{"uri": "mcp://ns/sales", "mimeType": "application/yaml", "text": "name: sales\ntransport:\n  type: streamable\n"}}, "resultType": "complete"})
		case schema.MethodSkillsList:
			rw.result(map[string]interface{}{"skills": []interface{}{skill, skill}, "nextCursor": "more", "resultType": "complete", "cacheScope": "private", "ttlMs": 0})
		case schema.MethodSkillsGet:
			if r.Header.Get(schema.HeaderName) != skill["uri"] || params["uri"] != skill["uri"] {
				t.Error("skill routing identity lost")
			}
			rw.result(map[string]interface{}{"skill": skill, "resultType": "complete", "cacheScope": "private", "ttlMs": 0})
		default:
			t.Errorf("unexpected method %q", method)
		}
	})
	defer server.Close()
	s := New().(*service)
	ctx := &endly.Context{Secrets: testSecrets{token: "test-token"}}
	request := Request{URL: server.URL, BearerTokenSecret: "alias"}
	cursor := "next"
	resources, err := s.listResources(ctx, &ListResourcesRequest{Request: request, Cursor: &cursor})
	if err != nil || len(resources.Resources) != 1 {
		t.Fatalf("resources: %+v %v", resources, err)
	}
	content, err := s.readResource(ctx, &ReadResourceRequest{Request: request, URI: "mcp://ns/sales"})
	if err != nil || len(content.Contents) != 1 || content.Contents[0].Text != "name: sales\ntransport:\n  type: streamable\n" {
		t.Fatalf("content: %+v %v", content, err)
	}
	skills, err := s.listSkills(ctx, &ListSkillsRequest{Request: request})
	if err != nil || len(skills.Skills) != 2 || skills.NextCursor == nil || *skills.NextCursor != "more" {
		t.Fatalf("skills: %+v %v", skills, err)
	}
	got, err := s.getSkill(ctx, &GetSkillRequest{Request: request, URI: skill["uri"].(string)})
	if err != nil || got.Skill.Frontmatter["name"] != "guide" || len(got.Skill.Resources.Files) != 1 {
		t.Fatalf("skill: %+v %v", got, err)
	}
	for _, method := range []string{schema.MethodResourcesList, schema.MethodResourcesRead, schema.MethodSkillsList, schema.MethodSkillsGet} {
		if calls[method] != 1 {
			t.Fatalf("method %s was prefetched or replayed: %d", method, calls[method])
		}
	}
}

func TestResourceProtocolDenialIsOptInAndRedacted(t *testing.T) {
	server := mockServer(t, func(w http.ResponseWriter, _ *http.Request, _ string, _ map[string]interface{}) {
		rw := w.(responseWriter)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": rw.id, "error": map[string]interface{}{"code": -32600, "message": "private-token", "data": "private-secret"}})
	})
	defer server.Close()
	s := New().(*service)
	ctx := &endly.Context{}
	request := ReadResourceRequest{Request: Request{URL: server.URL}, URI: "mcp://ns/private"}
	if _, err := s.readResource(ctx, &request); err == nil {
		t.Fatal("default swallowed protocol error")
	}
	request.AllowProtocolError = true
	result, err := s.readResource(ctx, &request)
	if err != nil || !result.IsError || result.ProtocolErrorCode == nil || *result.ProtocolErrorCode != -32600 || len(result.Contents) != 0 {
		t.Fatalf("denial: %+v %v", result, err)
	}
	skill, err := s.getSkill(ctx, &GetSkillRequest{Request: request.Request, URI: "skill://private", AllowProtocolError: true})
	if err != nil || !skill.IsError || skill.ProtocolErrorCode == nil || *skill.ProtocolErrorCode != -32600 {
		t.Fatalf("skill denial: %+v %v", skill, err)
	}
	encoded, err := json.Marshal(skill)
	if err != nil || strings.Contains(string(encoded), "private-token") || strings.Contains(string(encoded), "private-secret") {
		t.Fatalf("unsafe serialized denial: %s %v", encoded, err)
	}

}
