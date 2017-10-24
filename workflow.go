package endly

import (
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
)

type ServiceAction struct {
	RunCriteria string
	Service     string
	Action      string
	Tag         string
	TagIndex    string
	Name        string
	Description string
	Subpath     string
	Init        Variables
	Post        Variables
	Request     interface{}
	SleepInMs   int
	IgnoreError bool
}

type WorkflowTask struct {
	RunCriteria string
	Seq         int
	Name        string
	Description string
	Actions     []*ServiceAction
	Init        Variables
	Post        Variables
	SleepInMs   int
}

type Workflow struct {
	Source      *url.Resource
	Data        data.Map
	Name        string
	Description string
	Init        Variables
	Post        Variables
	Tasks       []*WorkflowTask
}

func (w *Workflow) Validate() error {
	return nil
}
