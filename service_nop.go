package endly

import (
	"fmt"
)

const (
	//NopServiceID represents nop service id.
	NopServiceID = "nop"
)

//Nop represent no operation
type Nop struct{}

//NopService represents no operation service
type NopService struct {
	*AbstractService
}

func (s *NopService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "nop",
		RequestInfo: &ActionInfo{
			Description: "no operation action, helper for separating action.Init as self descriptive steps",
		},
		RequestProvider: func() interface{} {
			return &Nop{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*Nop); ok {
				return handlerRequest, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "parrot",
		RequestInfo: &ActionInfo{
			Description: "fail workflow",
		},
		RequestProvider: func() interface{} {
			return &NopParrotRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*NopParrotRequest); ok {
				return handlerRequest.In, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewNopService creates a new NoOperation service.
func NewNopService() Service {
	var result = &NopService{
		AbstractService: NewAbstractService(NopServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
