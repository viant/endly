package endly

import "github.com/pkg/errors"

//WorkflowSwitchCase represent matching candidate case
type WorkflowSwitchCase struct {
	*ActionRequest `description:"action to run if matched"`
	Task           string      `description:"task to run if matched"`
	Value          interface{} `required:"true" description:"matching sourceKey value"`
}

//WorkflowSwitchRequest represent switch action request
type WorkflowSwitchRequest struct {
	SourceKey string                `required:"true" description:"sourceKey for matching value"`
	Cases     []*WorkflowSwitchCase `required:"true" description:"matching value cases"`
	Default   *WorkflowSwitchCase   `description:"in case no value was match case"`
}

//Match matches source with supplied action request.
func (r *WorkflowSwitchRequest) Match(source interface{}) *WorkflowSwitchCase {
	for _, switchCase := range r.Cases {
		if switchCase.Value == source {
			return switchCase
		}
	}
	return r.Default
}

//WorkflowSwitchResponse represents actual action or task response
type WorkflowSwitchResponse interface{}

//Validate checks if workflow is valid
func (r *WorkflowSwitchRequest) Validate() error {
	if r.SourceKey == "" {
		return errors.New("sourceKey was empty")
	}
	if len(r.Cases) == 0 {
		return errors.New("cases were empty")
	}
	for _, matchingCase := range r.Cases {
		if matchingCase.Value == nil {
			return errors.New("cases.value was empty")
		}
	}
	return nil
}
