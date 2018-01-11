package endly

//WorkflowSwitchTaskCase switch task case
type WorkflowSwitchTaskCase struct {
	Value interface{} //when value if match run task
	Task  string
}

//WorkflowSwitchTaskRequest represent switch task request
type WorkflowSwitchTaskRequest struct {
	SourceKey   string
	Cases       []*WorkflowSwitchTaskCase
	DefaultTask string
}

//Match matches source with task
func (r *WorkflowSwitchTaskRequest) Match(source interface{}) string {
	for _, switchCase := range r.Cases {
		if switchCase.Value == source {
			return switchCase.Task
		}
	}
	return r.DefaultTask
}

//WorkflowSwitchTaskResponse represents switch task response
type WorkflowSwitchTaskResponse struct {
	Task string
}
