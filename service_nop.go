package endly

import "fmt"

const (
	//NopServiceID represents nop service id.
	NopServiceID = "nop"

	//NopServiceNopAction represents nop action
	NopServiceNopAction = "nop"
)

//Nop represent no operation
type Nop struct{}

//no operation service
type nopService struct {
	*AbstractService
}

func (s *nopService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok", Response: request}
	switch request.(type) {
	case *Nop:
	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}

	defer s.End(context)(startEvent, Pairs("response", response))
	return response
}

func (s *nopService) NewRequest(action string) (interface{}, error) {
	if action == NopServiceNopAction {
		return &Nop{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewNopService creates a new NoOperation service.
func NewNopService() Service {
	var result = &nopService{
		AbstractService: NewAbstractService(NopServiceID,
			NopServiceNopAction),
	}
	result.AbstractService.Service = result
	return result
}
