package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/viant/endly"
	manager "github.com/viant/endly/service/manager"
)

func TestNativeToolSchemasHaveUniqueRequiredProperties(t *testing.T) {
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	registry := endly.New()
	handler, err := NewHandler(context.Background(), runtime, func() endly.Manager { return registry })
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := handler.ToolRegistry.Get("endly_dsunit_init"); !ok {
		t.Fatal("missing datastore initialization schema")
	}
	checked := 0
	for _, id := range extendedServices {
		service, err := registry.Service(id)
		if err != nil {
			continue
		}
		for _, action := range service.Actions() {
			name := "endly_" + strings.ReplaceAll(id, "/", "_") + "_" + action
			entry, ok := handler.ToolRegistry.Get(name)
			if !ok {
				t.Fatalf("missing native tool %s", name)
			}
			raw, err := json.Marshal(entry.Metadata.InputSchema)
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err = json.Unmarshal(raw, &document); err != nil {
				t.Fatal(err)
			}
			checkRequiredProperties(t, document, name)
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no native schemas checked")
	}
}

func checkRequiredProperties(t *testing.T, document map[string]any, path string) {
	t.Helper()
	seen := map[string]bool{}
	if values, ok := document["required"].([]any); ok {
		for _, value := range values {
			name, ok := value.(string)
			if !ok || seen[name] {
				t.Errorf("%s: invalid/duplicate required property %v", path, value)
			}
			seen[name] = true
		}
	}
	for _, keyword := range []string{"properties", "$defs", "definitions"} {
		if children, ok := document[keyword].(map[string]any); ok {
			for name, child := range children {
				if object, ok := child.(map[string]any); ok {
					checkRequiredProperties(t, object, path+"/"+keyword+"/"+name)
				}
			}
		}
	}
	for _, keyword := range []string{"items", "additionalProperties", "not"} {
		if child, ok := document[keyword].(map[string]any); ok {
			checkRequiredProperties(t, child, path+"/"+keyword)
		}
	}
	for _, keyword := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
		if children, ok := document[keyword].([]any); ok {
			for _, child := range children {
				if object, ok := child.(map[string]any); ok {
					checkRequiredProperties(t, object, path+"/"+keyword)
				}
			}
		}
	}
}
