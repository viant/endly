package endly

import "fmt"

const (
	//HTTPEndpointServiceID represents hTTPEndpoint service id.
	HTTPEndpointServiceID = "http/endpoint"

	//HTTPEndpointServiceListenAction represents listen HTTP endpoint action
	HTTPEndpointServiceListenAction = "listen"
)

//HTTPEndpoint represent no operation
type HTTPEndpoint struct{}

//hTTPEndpointService represents http endpoint service, that has ability to replay HTTP trips
type hTTPEndpointService struct {
	*AbstractService
}

//Run runs service action
func (s *hTTPEndpointService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok", Response: request}
	var err = s.Validate(request, response)
	if err != nil {
		return response
	}
	switch actualRequest := request.(type) {
	case *HTTPEndpointListenRequest:
		response.Response, err = s.listen(actualRequest)
	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}
	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	}

	defer s.End(context)(startEvent, Pairs("response", response))
	return response
}

//NewRequest returns new request for supplied action
func (s *hTTPEndpointService) NewRequest(action string) (interface{}, error) {

	switch action {
	case HTTPEndpointServiceListenAction:
		return &HTTPEndpointListenRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewRequest returns new request for supplied action
func (s *hTTPEndpointService) NewResponse(action string) (interface{}, error) {
	switch action {
	case HTTPEndpointServiceListenAction:
		return &HTTPEndpointListenResponse{}, nil
	}
	return s.AbstractService.NewResponse(action)
}

func (s *hTTPEndpointService) listen(request *HTTPEndpointListenRequest) (*HTTPEndpointListenResponse, error) {
	trips := request.AsHTTPServerTrips()
	err := StartHTTPServer(request.Port, trips)
	if err != nil {
		return nil, err
	}
	return &HTTPEndpointListenResponse{
		Trips: trips.Trips,
	}, nil

}

//NewHTTPEndpointService creates a new HTTP endpoint service, to replay previously recorded HTTP trips.
func NewHTTPEndpointService() Service {
	var result = &hTTPEndpointService{
		AbstractService: NewAbstractService(HTTPEndpointServiceID,
			HTTPEndpointServiceListenAction),
	}
	result.AbstractService.Service = result
	return result
}
