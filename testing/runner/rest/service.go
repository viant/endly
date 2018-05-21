package rest

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox"
)

//ServiceID represents rest service id.
const ServiceID = "rest/runner"

type restService struct {
	*endly.AbstractService
}

func (s *restService) sendRequest(context *endly.Context, request *Request) (*Response, error) {
	var resetResponse = make(map[string]interface{})
	var response = &Response{
		Response: resetResponse,
	}

	repeater := request.Repeater.Init()

	var extracted = make(map[string]interface{})
	handler := func() (interface{}, error) {
		err := toolbox.RouteToService(request.Method, request.URL, request.Request, &resetResponse)
		if err != nil {
			return nil, err
		}
		return resetResponse, nil
	}
	err := repeater.Run(s.AbstractService, "RESTRunner", context, handler, extracted)
	if err != nil {
		return response, err
	}
	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, resetResponse, "REST.response", "assert REST response")
	}
	return response, err
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
	s.Register(&endly.Route{
		Action: "send",
		RequestInfo: &endly.ActionInfo{
			Description: "send REST request",
			Examples: []*endly.UseCase{
				{
					Description: "send request",
					Data:        restSendExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &Request{}
		},
		ResponseProvider: func() interface{} {
			return &Response{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*Request); ok {
				return s.sendRequest(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewRestService creates a new reset service
func New() endly.Service {
	var result = &restService{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result

}
