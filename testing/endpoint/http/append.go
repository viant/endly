package http

import "github.com/viant/endly"

func (s *service) append(context *endly.Context, req *AppendRequest) (*AppendResponse, error) {
	resp := &AppendResponse{}
	server := s.servers[req.Port]
	trips := req.AsHTTPServerTrips(server.rotate, server.indexKeys)
	err := trips.Init(server.requestTemplate, server.responseTemplate)
	if err != nil {
		return nil, err
	}
	server.Append(trips)
	resp.Trips = trips.Trips
	return resp, nil
}
