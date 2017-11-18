package endly

import (
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
)

//ServiceAction represents a workflow service action
type ServiceAction struct {
	RunCriteria    string      //criteria to run this action
	SkipCriteria   string      //criteria to skip current action to continue to next tag id action
	Service        string      //service Id
	Action         string      //Id of the action used to create service request
	Tag            string      //neatly tag
	TagIndex       string      //neatly tag index
	TagId          string      //neatly tag id
	TagDescription string      //tag description
	Name           string      //Id of the service action
	Description    string      //description
	Init           Variables   //variables to initialise state before action runs
	Post           Variables   //variable to update state after action completes
	Request        interface{} //service request
	SleepInMs      int         //optional Sleep time
	Async          bool
}

//WorkflowTask represents a group of action
type WorkflowTask struct {
	RunCriteria string           //criteria to run this task
	Seq         int              //sequence of the task
	Name        string           //Id of the task
	Description string           //description
	Actions     []*ServiceAction //actions
	Init        Variables        //variables to initialise state before this taks runs
	Post        Variables        //variable to update state after this task completes
	TimeSpentMs int              //optional min required time spent in this task, reamining will force Sleep
}

//Workflow repesents a workflow
type Workflow struct {
	Source      *url.Resource   //source definition of the workflow
	Data        data.Map        //workflow data
	Name        string          //worfklow Id
	Description string          //description
	Init        Variables       //variables to initialise state before this workflow runs
	Post        Variables       //variables to initialise state before this workflow runs
	Tasks       []*WorkflowTask //workflow task
	SleepInMs   int             //optional Sleep time
}

//Validate validates this workflow TODO add implementation.
func (w *Workflow) Validate() error {
	return nil
}

//Workflows  represents workflows
type Workflows []*Workflow

//Push adds a workflow to the workflow stack.
func (w *Workflows) Push(workflow *Workflow) {
	*w = append(*w, workflow)
}

//Pop removes the first workflow from the workflow stack.
func (w *Workflows) Pop() *Workflow {
	if len(*w) == 0 {
		return nil
	}
	var result = (*w)[len(*w)-1]
	(*w) = (*w)[0: len(*w)-1]
	return result
}

//Last returns the last workflow from the workflow stack.
func (w *Workflows) Last() *Workflow {
	if w == nil {
		return nil
	}
	var workflowCount = len(*w)
	if workflowCount == 0 {
		return nil
	}
	return (*w)[workflowCount-1]
}
