package webplanner

import (
	"encoding/json"
	"github.com/viant/endly/internal/webplanner/node"
	"io"
	"log"
	"net/http"
	"strings"
)

type (
	Event struct {
		Type       string `json:"type"`
		TargetTag  string `json:"targetTag"`
		TargetHTML string `json:"targetHTML"`
		HolderHTML string `json:"holderHTML"`
		Key        string `json:"key"`
		MetaKey    bool   `json:"metaKey"`
	}

	Action struct {
		Method    string
		Tag       string
		Arguments string
		Target    string
		Selectors []string
	}
)

func (s *Service) handleEvent(writer http.ResponseWriter, request *http.Request) {
	enableCors(writer, request)
	if request.Method == http.MethodOptions {
		writer.WriteHeader(200)
		return
	}
	if request.Method != http.MethodPost {
		http.Error(writer, "invalid method", http.StatusInternalServerError)
		return
	}

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
	s.Target = event.TargetHTML
	action := &Action{
		Tag: event.TargetTag,
	}
	switch strings.ToLower(event.Type) {
	case "click":
		s.Keys = ""
		action.Method = "click"
	case "keyup":
		switch event.Key {
		case "Shift", "Meta":
		case "Backspace":
			if s.Keys != "" {
				s.Keys = s.Keys[:len(s.Keys)-1]
			}
		default:
			s.Keys += event.Key
		}

		action.Method = "sendKeys"
		action.Arguments = s.Keys
	}

	builder := node.NewBuilder(strings.Split(s.attributes, ",")...)
	attributes := builder.Attributes()
	aNode, err := builder.Build(event.HolderHTML, event.TargetHTML)
	if err == nil && s.ws != nil && aNode != nil {
		action.Selectors = aNode.Selectors(attributes, s.exclusion)
		err = s.ws.WriteJSON(action)
	}
	if err != nil {
		log.Printf("error: %v", err)
	}
	return nil
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
