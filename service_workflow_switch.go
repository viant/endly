package endly

//WorkflowSwitchCase represent matching candidate case
type WorkflowSwitchCase struct {
	*ActionRequest
	Task  string
	Value interface{} //when value is match run task
}

//WorkflowSwitchActionRequest represent switch action request
type WorkflowSwitchActionRequest struct {
	SourceKey string
	Cases     []*WorkflowSwitchCase
	Default   *WorkflowSwitchCase
}

//Match matches source with supplied action request.
func (r *WorkflowSwitchActionRequest) Match(source interface{}) *WorkflowSwitchCase {
	for _, switchCase := range r.Cases {
		if switchCase.Value == source {
			return switchCase
		}
	}
	return r.Default
}

//WorkflowSwitchResponse represents actual action or task response
type WorkflowSwitchResponse interface{}
