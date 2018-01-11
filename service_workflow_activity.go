package endly

import "time"

//WorkflowServiceActivity represents workflow activity
type WorkflowServiceActivity struct {
	*NeatlyTag
	Workflow        string
	Service         string
	Action          string
	Description     string
	Error           string
	StartTime       time.Time
	Ineligible      bool
	Request         interface{}
	Response        map[string]interface{}
	ServiceResponse interface{}
}

//FormatTag return a formatted tag
func (a *WorkflowServiceActivity) FormatTag() string {
	if a.TagIndex != "" {
		return "[" + a.Tag + a.TagIndex + "]"
	}
	return "[" + a.Tag + "]"
}


func NewWorkflowServiceActivity (context *Context, action *ServiceAction) *WorkflowServiceActivity {
	return &WorkflowServiceActivity{
		Workflow:    context.Workflows.Last().Name,
		Action:      action.Action,
		Service:     action.Service,
		NeatlyTag:   action.NeatlyTag,
		Description: context.Expand(action.Description),
		Request:     action.Request,
		Response:    make(map[string]interface{}),
		StartTime:   time.Now(),
	}
}
