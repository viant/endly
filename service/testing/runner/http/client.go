package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

// clientFor reuses equivalent transports across send actions in this service's
// session. Cookies remain explicit per request; clients have no shared cookie jar.
func (s *service) clientFor(context *endly.Context, options []*toolbox.HttpOptions) (*http.Client, func(), error) {
	values := map[string]interface{}{}
	for _, option := range options {
		values[option.Key] = option.Value
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP options: %w", err)
	}
	key := string(raw)
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()
	if client := s.clients[key]; client != nil {
		return client, func() {}, nil
	}
	client, err := toolbox.NewHttpClient(options...)
	if err != nil {
		return nil, nil, err
	}
	if len(s.clients) >= 64 || context == nil {
		return client, client.CloseIdleConnections, nil
	}
	if s.clients == nil {
		s.clients = map[string]*http.Client{}
	}
	s.clients[key] = client
	context.Deffer(client.CloseIdleConnections)
	return client, func() {}, nil
}
