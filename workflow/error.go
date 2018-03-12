package workflow

import "github.com/viant/endly"

//WorkflowError represent workflow error
type WorkflowError struct {
	Error    string
	Workflow string
	TaskName string
	Activity *endly.Activity
}
