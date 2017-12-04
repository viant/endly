package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

//RestServiceID represents rest service id.
const RestServiceID = "rest/runner"

type restService struct {
	*AbstractService
}

func (s *restService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualReuest := request.(type) {
	case *RestSendRequest:
		response.Response, err = s.sendRequest(actualReuest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to send %v %v", actualReuest.URL, err)
		}
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *restService) sendRequest(request *RestSendRequest) (*RestSendResponse, error) {
	var resetResponse = make(map[string]interface{})
	err := toolbox.RouteToService(request.Method, request.URL, request.Request, &resetResponse)
	if err != nil {
		return nil, err
	}
	return &RestSendResponse{
		Response: resetResponse,
	}, nil

}

func (s *restService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "send":
		return &RestSendRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewRestService creates a new reset service
func NewRestService() Service {
	var result = &restService{
		AbstractService: NewAbstractService(RestServiceID),
	}
	result.AbstractService.Service = result
	return result

}
