package endly

//WorkflowGotoRequest represents goto task action, this request will terminate current task execution to switch to specified task
type WorkflowGotoRequest struct {
	Task string
}

//WorkflowGotoResponse represents workflow task response
type WorkflowGotoResponse interface{}
