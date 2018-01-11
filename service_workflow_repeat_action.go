package endly

//WorkflowRepeatActionRequest represent repeat action operation
type WorkflowRepeatActionRequest struct {
	*ActionRequest
	*Repeatable
}

//WorkflowRepeatActionResponse represents repeat action response
type WorkflowRepeatActionResponse struct {
	Response  interface{}
	Extracted map[string]string
	Repeated  int
}
