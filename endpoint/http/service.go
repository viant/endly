package http

import (
	"fmt"
	"github.com/viant/endly"
)

const (
	//ServiceID represents hTTPEndpoint service id.
	ServiceID = "http/endpoint"
)

//service represents http endpoint service, that has ability to replay HTTP trips
type service struct {
	*endly.AbstractService
}

func (s *service) listen(request *ListenRequest) (*ListenResponse, error) {
	trips := request.AsHTTPServerTrips()
	err := StartServer(request.Port, trips)
	if err != nil {
		return nil, err
	}
	return &ListenResponse{
		Trips: trips.Trips,
	}, nil

}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "listen",
		RequestInfo: &endly.ActionInfo{
			Description: "start HTTP endpoint",
		},
		RequestProvider: func() interface{} {
			return &ListenRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ListenResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ListenRequest); ok {
				return s.listen(req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new HTTP endpoint service, to replay previously recorded HTTP trips.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
