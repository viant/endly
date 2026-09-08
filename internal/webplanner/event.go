package webplanner

import (
	"encoding/json"
	"github.com/viant/endly/internal/webplanner/node"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type (
	Event struct {
		ID            string `json:"id"`
		Type          string `json:"type"`
		TargetTag     string `json:"targetTag"`
		TargetHTML    string `json:"targetHTML"`
		HolderHTML    string `json:"holderHTML"`
		Key           string `json:"key"`
		MetaKey       bool   `json:"metaKey"`
		Value         string `json:"value"`
		ValueRedacted bool   `json:"valueRedacted"`
		InputType     string `json:"inputType"`
		Checked       bool   `json:"checked"`
		URL           string `json:"url"`
		Title         string `json:"title"`
		Timestamp     int64  `json:"timestamp"`
	}

	Action struct {
		Method        string
		Tag           string
		Arguments     string
		Target        string
		Selectors     []string
		Expression    string
		URL           string
		Timestamp     int64
		ValueRedacted bool
	}
)

func (s *Service) handleEvent(writer http.ResponseWriter, request *http.Request) {
	authorized := s.authorize(request)
	if !enableCors(writer, request, authorized) {
		http.Error(writer, "origin not allowed", http.StatusForbidden)
		return
	}
	if request.Method == http.MethodOptions {
		writer.WriteHeader(200)
		return
	}
	if !authorized {
		http.Error(writer, "invalid planner token", http.StatusUnauthorized)
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost+", "+http.MethodOptions)
		http.Error(writer, "invalid method", http.StatusMethodNotAllowed)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 256<<10)

	err := s.processEvent(request)
	if err != nil {
		http.Error(writer, "failed to process event", http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(200)

}

func (s *Service) processEvent(request *http.Request) error {
	event, err := s.loadEvent(request)
	if err != nil {
		return err
	}
	return s.processActivity(event)
}

func (s *Service) processActivity(event *Event) error {
	if event == nil {
		return nil
	}
	s.mux.Lock()
	if !s.recording {
		s.mux.Unlock()
		return nil
	}
	if event.ID != "" {
		if _, ok := s.seenEvents[event.ID]; ok {
			s.mux.Unlock()
			return nil
		}
		s.seenEvents[event.ID] = struct{}{}
		s.seenEventOrder = append(s.seenEventOrder, event.ID)
		if len(s.seenEventOrder) > 2000 {
			oldest := s.seenEventOrder[0]
			s.seenEventOrder = s.seenEventOrder[1:]
			delete(s.seenEvents, oldest)
		}
	}
	s.Target = event.TargetHTML
	attributesSetting := s.attributes
	exclusion := s.exclusion
	s.mux.Unlock()
	action := &Action{
		Tag:           event.TargetTag,
		URL:           event.URL,
		Timestamp:     event.Timestamp,
		ValueRedacted: event.ValueRedacted,
	}
	switch strings.ToLower(event.Type) {
	case "click":
		action.Method = "click"
	case "input", "change":
		action.Method = "fill"
		action.Arguments = event.Value
		if event.ValueRedacted {
			action.Arguments = "$PASSWORD"
		}
		if strings.EqualFold(event.InputType, "checkbox") || strings.EqualFold(event.InputType, "radio") {
			if event.Checked {
				action.Method = "check"
			} else {
				action.Method = "uncheck"
			}
			action.Arguments = ""
		} else if strings.EqualFold(event.TargetTag, "select") {
			action.Method = "selectOption"
		}
	case "press":
		action.Method = "press"
		action.Arguments = event.Key
	case "submit":
		action.Method = "submit"
	case "navigation":
		action.Method = "goto"
		action.Arguments = event.URL
		action.Expression = "page.goto(" + strconv.Quote(event.URL) + ")"
		s.recordAction(action)
		return nil
	default:
		return nil
	}

	builder := node.NewBuilder(strings.Split(attributesSetting, ",")...)
	attributes := builder.Attributes()
	aNode, err := builder.Build(event.HolderHTML, event.TargetHTML)
	if err != nil {
		return err
	}
	if aNode == nil {
		return nil
	}
	action.Selectors = aNode.Selectors(attributes, exclusion)
	if len(action.Selectors) > 0 {
		action.Expression = recordedExpression(action.Selectors[0], action.Method, action.Arguments)
	}
	s.recordAction(action)
	if action.Method == "click" || action.Method == "submit" || (action.Method == "press" && strings.EqualFold(action.Arguments, "Enter")) {
		s.scheduleRecorderRefresh()
	}
	return nil
}

func (s *Service) scheduleRecorderRefresh() {
	if s.manager == nil || s.context == nil {
		return
	}
	s.mux.Lock()
	s.recorderGeneration++
	generation := s.recorderGeneration
	s.mux.Unlock()
	go func() {
		for _, delay := range []time.Duration{400 * time.Millisecond, 1200 * time.Millisecond, 3 * time.Second} {
			timer := time.NewTimer(delay)
			<-timer.C
			s.mux.Lock()
			active := s.recording && generation == s.recorderGeneration
			s.mux.Unlock()
			if !active {
				return
			}
			_ = s.injectTracker()
		}
	}()
}

func (s *Service) recordAction(action *Action) {
	if action == nil {
		return
	}
	s.mux.Lock()
	s.recorded = append(s.recorded, action)
	if len(s.recorded) > 1000 {
		s.recorded = append([]*Action(nil), s.recorded[len(s.recorded)-1000:]...)
	}
	s.mux.Unlock()
	s.writeLive(action)
}

func (s *Service) recordedActions() []*Action {
	s.mux.Lock()
	defer s.mux.Unlock()
	return append([]*Action(nil), s.recorded...)
}

func recordedExpression(selector, method, argument string) string {
	result := "page.locator(" + strconv.Quote(selector) + ")." + method + "("
	if argument != "" {
		result += strconv.Quote(argument)
	}
	return result + ")"
}

func (s *Service) loadEvent(request *http.Request) (*Event, error) {
	data, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	request.Body.Close()
	event := &Event{}
	err = json.Unmarshal(data, event)
	return event, err
}
