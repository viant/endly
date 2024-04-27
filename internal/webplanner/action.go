package webplanner

import (
	"log"
	"net/http"
)

func (s *Service) handleActions(writer http.ResponseWriter, request *http.Request) {
	ws, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	s.ws = ws
	// Infinite loop to read messages from the client
	for {
		var msg *ActionRequest
		// Read in a new message as JSON and map it to a Message object
		if err := ws.ReadJSON(&msg); err != nil {
			log.Printf("error: %v", err)
			break
		}
		s.exclusion = msg.Exclusion
		s.attributes = msg.Attributes
	}
	return
}

type ActionRequest struct {
	Exclusion  string `json:"exclusion"`
	Attributes string `json:"attributes"`
}
