package endly

import (
	"github.com/viant/toolbox"
	"fmt"
)

const RestServiceId = "transfer"

type restService struct {
	*AbstractService
}


type RestSendRequest struct {
	URL string
	Method string
	Request interface{}
}


type RestSendResponse struct {
	Response interface{}
}



func (s *restService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}

	var err error
	switch actualReuest := request.(type) {
	case *RestSendRequest:
		response.Response, err  =s.sendRequest(actualReuest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to send %v %v", actualReuest.URL, err)
		}
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}


func (s *restService) sendRequest(request *RestSendRequest) (*RestSendResponse, error) {
	var resetResponse  = make(map[string]interface{})
	err := toolbox.RouteToService(request.Method, request.URL, request.Request, &resetResponse)
	if err != nil {
		return nil, err
	}
	return &RestSendResponse{
		Response:resetResponse,
	}, nil

}

func (s *restService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "semd":
		return &RestSendRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewRestService() Service {
	var result = &restService{
		AbstractService: NewAbstractService(TransferServiceId),
	}
	result.AbstractService.Service = result
	return result

}

