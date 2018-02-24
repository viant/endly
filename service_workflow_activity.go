package endly

import (
	"github.com/viant/toolbox/data"
	"time"
)

//WorkflowServiceActivity represents workflow Activity
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
	ServiceResponse *ServiceResponse
}

//FormatTag return a formatted tag
func (a *WorkflowServiceActivity) FormatTag() string {
	if a.TagIndex != "" {
		return "[" + a.Tag + a.TagIndex + "]"
	}
	return "[" + a.Tag + "]"
}

//NewWorkflowServiceActivity returns a new workflow Activity.
func NewWorkflowServiceActivity(context *Context, action *ServiceAction, state data.Map) *WorkflowServiceActivity {
	return &WorkflowServiceActivity{
		Workflow:    context.Workflows.Last().Name,
		Action:      state.ExpandAsText(action.Action),
		Service:     state.ExpandAsText(action.Service),
		NeatlyTag:   action.NeatlyTag,
		Description: context.Expand(action.Description),
		Request:     state.Expand(action.Request),
		Response:    make(map[string]interface{}),
		StartTime:   time.Now()}
}
