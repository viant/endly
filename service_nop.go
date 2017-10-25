package endly

//NopServiceID represents nop service id.
const NopServiceID = "nop"

//Nop represent no operation
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

//NewNopService creates a new NoOperation service.
func NewNopService() Service {
	var result = &nopService{
		AbstractService: NewAbstractService(NopServiceID),
	}
	result.AbstractService.Service = result
	return result
}
