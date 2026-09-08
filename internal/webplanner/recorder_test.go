package webplanner

import (
	"encoding/base64"
	"testing"

	"github.com/viant/endly/service/testing/runner/webdriver"
)

func TestActivityFromConsole(t *testing.T) {
	payload := `{"id":"one","type":"click","targetTag":"BUTTON"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))
	event := activityFromConsole(&webdriver.ConsoleEntry{Message: `console.debug "` + activityConsolePrefix + encoded + `"`})
	if event == nil || event.ID != "one" || event.Type != "click" {
		t.Fatalf("event=%#v", event)
	}
}

func TestProcessActivityDeduplicatesFetchAndConsoleFallback(t *testing.T) {
	service := NewService(&Config{Token: "secret"})
	event := &Event{ID: "same", Type: "navigation", URL: "https://example.test"}
	if err := service.processActivity(event); err != nil {
		t.Fatal(err)
	}
	if err := service.processActivity(event); err != nil {
		t.Fatal(err)
	}
	if recorded := service.recordedActions(); len(recorded) != 1 {
		t.Fatalf("recorded=%d, wanted 1", len(recorded))
	}
}
