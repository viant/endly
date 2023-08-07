package udf

import (
	"fmt"
	"github.com/viant/endly"
)

const (
	//ServiceID represents UDF service id.
	ServiceID = "udf"
)

// service represents no operation service (deprecated, use workflow, nop instead)
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "register",
		RequestInfo: &endly.ActionInfo{
			Description: "no operation action, helper for separating action.Init as self descriptive steps",
		},
		RequestProvider: func() interface{} {
			return &RegisterRequest{}
		},
		ResponseProvider: func() interface{} {
			return RegisterResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RegisterRequest); ok {
				return s.register(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) register(context *endly.Context, request *RegisterRequest) (interface{}, error) {
	state := context.State()
	for _, udf := range request.UDFs {
		for i, item := range udf.Params {
			udf.Params[i] = state.Expand(item)
		}
	}
	if err := RegisterProviders(request.UDFs); err != nil {
		return nil, err
	}
	return &RegisterResponse{}, nil
}

// New creates a new udf service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
