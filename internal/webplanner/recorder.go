package webplanner

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/viant/endly/service/testing/runner/webdriver"
)

const activityConsolePrefix = "__ENDLY_EVENT_B64__"

func (s *Service) startActivityCapture() error {
	enableConsole := true
	enableNetwork := false
	includeBodies := false
	if _, err := s.manager.Run(s.context, &webdriver.CaptureStartRequest{
		EnableConsole: &enableConsole,
		EnableNetwork: &enableNetwork,
		IncludeBodies: &includeBodies,
	}); err != nil {
		return err
	}
	s.mux.Lock()
	if s.pollerStarted {
		s.mux.Unlock()
		return nil
	}
	s.pollerStarted = true
	s.mux.Unlock()
	go s.pollActivityConsole()
	return nil
}

func (s *Service) pollActivityConsole() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	includeConsole := true
	includeNetwork := false
	for range ticker.C {
		if s.context == nil || s.context.IsClosed() {
			return
		}
		result, err := s.manager.Run(s.context, &webdriver.CaptureExportRequest{
			MaxEntries:     1_000,
			IncludeConsole: &includeConsole,
			IncludeNetwork: &includeNetwork,
		})
		if err != nil {
			continue
		}
		response, ok := result.(*webdriver.CaptureExportResponse)
		if !ok || response == nil {
			continue
		}
		for _, entry := range response.Console {
			if event := activityFromConsole(entry); event != nil {
				_ = s.processActivity(event)
			}
		}
	}
}

func activityFromConsole(entry *webdriver.ConsoleEntry) *Event {
	if entry == nil {
		return nil
	}
	index := strings.Index(entry.Message, activityConsolePrefix)
	if index == -1 {
		return nil
	}
	encoded := entry.Message[index+len(activityConsolePrefix):]
	end := 0
	for end < len(encoded) && isBase64Byte(encoded[end]) {
		end++
	}
	if end == 0 {
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(encoded[:end])
	if err != nil {
		return nil
	}
	event := &Event{}
	if err := json.Unmarshal(data, event); err != nil {
		return nil
	}
	return event
}

func isBase64Byte(value byte) bool {
	return value >= 'a' && value <= 'z' ||
		value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' ||
		value == '+' || value == '/' || value == '='
}
