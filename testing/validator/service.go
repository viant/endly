package validator

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model/criteria"
	"github.com/viant/toolbox"
)

//ServiceID represents validator service id
const ServiceID = "validator"

//AssertAction represents assert action
const AssertAction = "assert"

type service struct {
	*endly.AbstractService
}

func (s *service) Assert(context *endly.Context, request *AssertRequest) (response *AssertResponse, err error) {
	var state = context.State()
	var actual = request.Actual
	var expect = request.Expect
	response = &AssertResponse{}

	if request.Ignore != nil {
		actual, expect = s.applyIgnore(request, actual, expect)
	}

	if toolbox.IsString(request.Actual) {
		if actualValue, ok := state.GetValue(toolbox.AsString(request.Actual)); ok {
			actual = actualValue
		}
	}
	name := request.Name
	if name == "" {
		name = "/"
	}

	response.Validation, err = criteria.Assert(context, name, expect, actual)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *service) applyIgnore(request *AssertRequest, actual interface{}, expect interface{}) (interface{}, interface{}) {

	if request.Ignore == nil {
		return actual, expect
	}
	ignoreKey := request.IgnoreKeys()
	if len(ignoreKey) == 0 {
		return actual, expect
	}
	actualMap, _ := request.Actual.(map[string]interface{})
	exoectedMap, _ := request.Actual.(map[string]interface{})

	if len(actualMap) > 0 {
		for _, key := range ignoreKey {
			delete(actualMap, toolbox.AsString(key))
		}
		request.Actual = actualMap
		actual = request.Actual
	}
	if len(exoectedMap) > 0 {
		for _, key := range ignoreKey {
			delete(exoectedMap, toolbox.AsString(key))
		}
	}
	request.Expect = exoectedMap
	expect = request.Expect
	return actual, expect
}

const validationExample = `{
  "Actual": [
    {
      "k": 10,
      "seq": "22",
      "y": 2
    },
    {
      "b": 2,
      "k": "Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.",
      "seq": "11"
    }
  ],
  "Expected": {
    "11": {
      "b": 2,
      "k": "/Lorem Ipsum/",
      "seq": "11"
    },
    "22": {
      "k": 10,
      "seq": "22",
      "y": 2
    },
    "@indexBy@": "seq"
  }
}`

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "assert",
		RequestInfo: &endly.ActionInfo{
			Description: "validate provided data (it uses https://github.com/viant/assertly)",
			Examples: []*endly.UseCase{
				{
					Description: "validation",
					Data:        validationExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &AssertRequest{}
		},
		ResponseProvider: func() interface{} {
			return &AssertResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*AssertRequest); ok {
				return s.Assert(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new validation service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result

}
