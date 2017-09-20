package endly

type ServiceAction struct {
	RunCriteria string
	Service     string
	Action      string
	Description string
	Variables   Variables
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
	Variables   Variables
	SleepInMs   int
}

type Workflow struct {
	source      *Resource
	Name        string
	Description string
	Variables   Variables
	Tasks       []*WorkflowTask
}

func (w *Workflow) Validate() error {
	return nil
}
