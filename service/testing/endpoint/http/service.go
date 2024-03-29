package http

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"strconv"
)

const (
	//ServiceID represents hTTPEndpoint service id.
	ServiceID = "http/endpoint"
)

// service represents http endpoint service, that has ability to replay HTTP trips
type service struct {
	*endly.AbstractService
	servers map[int]*Server
}

func (s *service) shutdown(context *endly.Context, req *ShutdownRequest) (interface{}, error) {
	s.Mutex().Lock()
	defer s.Mutex().Unlock()
	server, ok := s.servers[req.Port]
	if !ok {
		return nil, fmt.Errorf("ednpoint at %v, not found", req.Port)
	}
	err := server.Shutdown(context.Background())
	return &struct{}{}, err
}

func (s *service) listen(context *endly.Context, request *ListenRequest) (*ListenResponse, error) {
	state := context.State()
	if request.BaseDirectory != "" {
		request.BaseDirectory = location.NewResource(state.ExpandAsText(request.BaseDirectory)).Path()
	}
	key := ServiceID + ":" + strconv.Itoa(request.Port)
	s.Mutex().Lock()
	defer s.Mutex().Unlock()
	var response *ListenResponse
	var serviceState = s.State()
	value := serviceState.Get(key)
	if value != nil {
		if response = value.(*ListenResponse); response != nil {
			return response, nil
		}
	}
	trips := request.AsHTTPServerTrips()

	server, err := StartServer(request.Port, trips, request.RequestTemplate, request.ResponseTemplate)
	if err != nil {
		return nil, err
	}

	s.servers[request.Port] = server
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
	},
		&endly.Route{
			Action: "append",
			RequestInfo: &endly.ActionInfo{
				Description: "append http trips",
			},
			RequestProvider: func() interface{} {
				return &AppendRequest{}
			},
			ResponseProvider: func() interface{} {
				return &AppendResponse{}
			},
			Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
				if req, ok := request.(*AppendRequest); ok {
					return s.append(context, req)
				}
				return nil, fmt.Errorf("unsupported request type: %T", request)
			},
		},
		&endly.Route{
			Action: "shutdown",
			RequestInfo: &endly.ActionInfo{
				Description: "stop HTTP endpoint",
			},
			RequestProvider: func() interface{} {
				return &ShutdownRequest{}
			},
			ResponseProvider: func() interface{} {
				return &struct{}{}
			},
			Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
				if req, ok := request.(*ShutdownRequest); ok {
					return s.shutdown(context, req)
				}
				return nil, fmt.Errorf("unsupported request type: %T", request)
			},
		})
}

// New creates
//
// a new HTTP endpoint service, to replay previously recorded HTTP trips.
func New() endly.Service {
	var result = &service{
		servers:         make(map[int]*Server),
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
