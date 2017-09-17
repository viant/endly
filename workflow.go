package endly

type ServiceAction struct {
	Enabled   bool
	Service   string
	Action    string
	Variables Variables
	Request   interface{}
	SleepInMs int
}

type WorkflowTask struct {
	Seq         int
	Name        string
	UseCase     string
	Description string
	Actions     []*ServiceAction
	Variables   Variables
	SleepInMs int
}

type Workflow struct {
	Workflows   map[string]*Workflow
	Name        string
	Description string
	Variables   Variables
	Tasks       []*WorkflowTask
}

func (w *Workflow) Validate() error {
	return nil
}