package endly

const NopService = "nop"

type Nop struct{}

//no operation service
type nopService struct {
	*AbstractService
}

func (s *nopService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	return response
}

func (s *nopService) NewRequest(action string) (interface{}, error) {

	return &Nop{}, nil
}

func NewNopService() Service {
	var result = &nopService{
		AbstractService: NewAbstractService(NopService),
	}
	result.AbstractService.Service = result
	return result
}
