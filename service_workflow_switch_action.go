package endly

//WorkflowSwitchActionCase represent matching candidate case
type WorkflowSwitchActionCase struct {
	*ActionRequest
	Value interface{} //when value is match run task
}

//WorkflowSwitchActionRequest represent switch action request
type WorkflowSwitchActionRequest struct {
	SourceKey string
	Cases     []*WorkflowSwitchActionCase
	Default   *ActionRequest
}

//Match matches source with supplied action request.
func (r *WorkflowSwitchActionRequest) Match(source interface{}) *ActionRequest {
	for _, switchCase := range r.Cases {
		if switchCase.Value == source {
			return switchCase.ActionRequest
		}
	}
	return r.Default
}

//WorkflowSwitchActionResponse represents switch action response
type WorkflowSwitchActionResponse struct {
	Service  string
	Action   string
	Response interface{}
}
