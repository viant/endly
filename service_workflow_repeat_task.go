package endly

//WorkflowRepeatTaskRequest represent repeat task operation
type WorkflowRepeatTaskRequest struct {
	Task string
	*Repeatable
}

//WorkflowRepeatTaskResponse represent repeat response
type WorkflowRepeatTaskResponse struct {
	Response  map[string]interface{} //Post data
	Extracted map[string]string
	Repeated  int
}
