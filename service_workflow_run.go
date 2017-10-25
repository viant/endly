package endly

//WorkflowRunRequest represents workflow run request
type WorkflowRunRequest struct {
	EnableLogging     bool                   //Enable logging
	LoggingDirectory  string                 //Logging directory
	WorkflowURL       string                 //Workflow URL if workflow is not found in the registry, it will be loaded
	Name              string                 //Id of the workflow to run
	Params            map[string]interface{} //workflow parameters
	Tasks             string                 //tasks to run with coma separated list or '*', or empty string for all tasks
	PublishParameters bool                   //publishes parameters Id into context state
	Async             bool                   //flag to run it asynchronously. Do not set it yourself runner only sets the first workflow asyn
}

//WorkflowRunResponse represents workflow run response
type WorkflowRunResponse struct {
	Data      map[string]interface{} //Workflow data populated by Workflow.Post variable section.
	SessionID string                 //session id
}
