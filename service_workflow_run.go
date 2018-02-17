package endly

//WorkflowRunRequest represents workflow run request
type WorkflowRunRequest struct {
	EnableLogging     bool                   `description:"flag to enable logging"`
	LoggingDirectory  string                 `description:"log directory"`
	WorkflowURL       string                 `description:"workflow URL if workflow is not found in the registry, it is loaded"`
	Name              string                 `required:"true" description:"name defined in workflow document"`
	Params            map[string]interface{} `description:"workflow parameters, accessibly by paras.[Key], if PublishParameters is set, all parameters are place in context.state"`
	Tasks             string                 `required:"true" description:"coma separated task list or '*'to run all tasks sequencialy"` //tasks to run with coma separated list or '*', or empty string for all tasks
	TagIDs            string                 `description:"coma separated TagID list, if present in a task, only matched runs, other task run as normal"`
	PublishParameters bool                   `description:"flag to publish parameters directly into context state"`
	Async             bool                   `description:"flag to run it asynchronously. Do not set it your self runner sets the flag for the first workflow"`
}

//WorkflowRunResponse represents workflow run response
type WorkflowRunResponse struct {
	Data      map[string]interface{} //Workflow data populated by Workflow.Post variable section.
	SessionID string                 //session id
}
