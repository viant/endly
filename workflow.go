package endly

import (
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
)

//ServiceAction represents a workflow service action
type ServiceAction struct {
	RunCriteria    string      //criteria to run this action
	Service        string      //service Id
	Action         string      //Id of the action used to create service request
	Tag            string      //neatly tag
	TagIndex       string      //neatly tag index
	TagDescription string      //tag description
	Name           string      //Id of the service action
	Description    string      //description
	Subpath        string      //subpath
	Init           Variables   //variables to initialise state before action runs
	Post           Variables   //variable to update state after action completes
	Request        interface{} //service request
	SleepInMs      int         //optional sleep time
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
	SleepInMs   int              //optional sleep time
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
}

//Validate validates this workflow TODO add implementation.
func (w *Workflow) Validate() error {
	return nil
}
