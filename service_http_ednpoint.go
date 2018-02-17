package endly

import "fmt"

const (
	//HTTPEndpointServiceID represents hTTPEndpoint service id.
	HTTPEndpointServiceID = "http/endpoint"
)

//HTTPEndpoint represent no operation
type HTTPEndpoint struct{}

//hTTPEndpointService represents http endpoint service, that has ability to replay HTTP trips
type hTTPEndpointService struct {
	*AbstractService
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

func (s *hTTPEndpointService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "listen",
		RequestInfo: &ActionInfo{
			Description: "start HTTP endpoint",
		},
		RequestProvider: func() interface{} {
			return &HTTPEndpointListenRequest{}
		},
		ResponseProvider: func() interface{} {
			return &HTTPEndpointListenResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*HTTPEndpointListenRequest); ok {
				return s.listen(handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewHTTPEndpointService creates a new HTTP endpoint service, to replay previously recorded HTTP trips.
func NewHTTPEndpointService() Service {
	var result = &hTTPEndpointService{
		AbstractService: NewAbstractService(HTTPEndpointServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
