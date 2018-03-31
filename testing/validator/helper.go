package validator

import (
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/endly/workflow"
	"github.com/viant/toolbox"
)

//NewAssertRequestFromContext creates a new assert rquest from context for current activity
func NewAssertRequestFromContext(context *endly.Context, source, expected, actual interface{}, name, description string) (*AssertRequest, error) {
	process := workflow.Last(context)
	if process == nil {
		return nil, errors.New("process was empty")
	}
	activity := process.Last()
	if process == nil {
		return nil, errors.New("activity was empty")
	}

	if description == "" {
		description = activity.Description
	}

	if expected != nil && toolbox.IsSlice(expected) {
		if normalized, err := util.NormalizeMap(expected, true); err == nil {
			expected = normalized
		}
	}
	return NewAssertRequest(activity.TagID, name, description, source, expected, actual), nil

}

//Assert compares provided expected amd actual
func Assert(context *endly.Context, source, expected, actual interface{}, name, description string) (*AssertResponse, error) {
	var request, err = NewAssertRequestFromContext(context, source, expected, actual, name, description)
	if err != nil {
		return nil, err
	}
	var response = &AssertResponse{}
	err = endly.Run(context, request, response)
	return response, err
}
