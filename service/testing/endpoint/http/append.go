package http

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
)

func (s *service) append(context *endly.Context, req *AppendRequest) (*AppendResponse, error) {
	resp := &AppendResponse{}
	server, ok := s.servers[req.Port]
	if !ok {
		return nil, fmt.Errorf("server not started on port: %v", req.Port)
	}
	state := context.State()
	if req.BaseDirectory != "" {
		req.BaseDirectory = location.NewResource(state.ExpandAsText(req.BaseDirectory)).Path()
	}

	trips := req.AsHTTPServerTrips(server.rotate, server.indexKeys)
	err := trips.Init(server.requestTemplate, server.responseTemplate)
	if err != nil {
		return nil, err
	}

	server.Append(trips)
	resp.Trips = trips.Trips
	return resp, nil
}
