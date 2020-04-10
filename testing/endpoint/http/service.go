package http

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
)

const (
	//ServiceID represents hTTPEndpoint service id.
	ServiceID = "http/endpoint"
)

//service represents http endpoint service, that has ability to replay HTTP trips
type service struct {
	*endly.AbstractService
}


func (s *service) listen(context *endly.Context, request *ListenRequest) (*ListenResponse, error) {
	state := context.State()
	request.BaseDirectory = url.NewResource(state.ExpandAsText(request.BaseDirectory)).ParsedURL.Path
	key := ServiceID + ":" + request.BaseDirectory
	s.Mutex().Lock()
	var response *ListenResponse
	defer s.Mutex().Unlock()
	var serviceState = s.State()
	value := serviceState.Get(key)
	if value != nil {
		if response = value.(*ListenResponse); response != nil {
			return response, nil
		}
	}
	trips := request.AsHTTPServerTrips()
	err := StartServer(request.Port, trips)
	if err != nil {
		return nil, err
	}
	response = &ListenResponse{
		Trips: trips.Trips,
	}
	serviceState.Put(key, response)
	return response, nil
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
				return s.listen(context, req)
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
