package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

//RestServiceID represents rest service id.
const RestServiceID = "rest/runner"

//RestServiceSendAction represents send action
const RestServiceSendAction = "send"

type restService struct {
	*AbstractService
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

const restSendExample = `
{
		"URL": "http://127.0.0.1:8085/v1/reporter/register/",
		"Method": "POST",
		"Request": {
			"Report": {
				"Columns": [
					{
						"Alias": "",
						"Name": "category"
					}
				],
				"From": "expenditure",
				"Groups": [
					"year"
				],
				"Name": "report1",
				"Values": [
					{
						"Column": "expenditure",
						"Function": "SUM"
					}
				]
			},
			"ReportType": "pivot"
		}
	}`

func (s *restService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "send",
		RequestInfo: &ActionInfo{
			Description: "send REST request",
		},
		RequestProvider: func() interface{} {
			return &RestSendRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RestSendResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*RestSendRequest); ok {
				return s.sendRequest(handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewRestService creates a new reset service
func NewRestService() Service {
	var result = &restService{
		AbstractService: NewAbstractService(RestServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result

}
