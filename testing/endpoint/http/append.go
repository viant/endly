package http

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
)

func (s *service) append(context *endly.Context, req *AppendRequest) (*AppendResponse, error) {
	resp := &AppendResponse{}
	server := s.servers[req.Port]
	state := context.State()
	if req.BaseDirectory != "" {
		req.BaseDirectory = url.NewResource(state.ExpandAsText(req.BaseDirectory)).ParsedURL.Path
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
