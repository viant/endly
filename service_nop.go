package endly

import "fmt"

const (
	//NopServiceID represents nop service id.
	NopServiceID = "nop"

	//NopServiceNopAction represents nop action
	NopServiceNopAction = "nop"

	//NopServiceFailAction represents fail action
	NopServiceFailAction = "fail"

	//NopServiceParrotAction represents parrot action
	NopServiceParrotAction = "parrot"
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
	switch actualRequest := request.(type) {
	case *Nop:
	case *NopFailRequest:
		response.Error = context.Expand(actualRequest.Message)
	case *NopParrotRequest:
		response.Response = actualRequest.In
	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}

	if response.Error != "" {
		response.Status = "error"
	}

	defer s.End(context)(startEvent, Pairs("response", response))
	return response
}

func (s *nopService) NewRequest(action string) (interface{}, error) {

	switch action {
	case NopServiceNopAction:
		return &Nop{}, nil
	case NopServiceFailAction:
		return &NopFailRequest{}, nil
	case NopServiceParrotAction:
		return &NopParrotRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewNopService creates a new NoOperation service.
func NewNopService() Service {
	var result = &nopService{
		AbstractService: NewAbstractService(NopServiceID,
			NopServiceNopAction,
			NopServiceFailAction,
			NopServiceParrotAction),
	}
	result.AbstractService.Service = result
	return result
}
