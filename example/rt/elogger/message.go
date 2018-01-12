package elogger

import (
	"github.com/satori/go.uuid"
	"github.com/viant/toolbox/bridge"
	"net/http"
	"strings"
	"time"
)

//Message represent event log message
type Message struct {
	Timestamp time.Time
	EventType string
	EventID   string
	ClientIP  string
	ServerIP  string
	Request   *bridge.HttpRequest
	Error     string
}

//NewMessage creates a new message for provided request.
func NewMessage(request *http.Request) *Message {
	var scheme = "http"
	var result = &Message{
		Timestamp: time.Now(),
		ClientIP:  request.RemoteAddr,
		ServerIP:  request.Host,
		Request: &bridge.HttpRequest{
			URL:    scheme + "://" + request.Host + request.URL.String(),
			Method: request.Method,
			Header: request.Header,
		},
	}
	result.EventType = strings.Trim(request.URL.Path, "/")
	if UUID, err := uuid.NewV1(); err == nil {
		result.EventID = UUID.String()
	}
	return result
}
