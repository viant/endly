package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
)

//ActionRequest represent an action request
type ActionRequest struct {
	Service string      //service Id
	Action  string      //Id of the action used to create service request
	Request interface{} //service request
}

//ServiceAction represents a workflow service action
type ServiceAction struct {
	*ActionRequest
	*NeatlyTag
	RunCriteria  string    //criteria to run this action
	SkipCriteria string    //criteria to skip current action to continue to next tag id action
	Name         string    //Id of the service action
	Description  string    //description
	Init         Variables //variables to initialise state before action runs
	Post         Variables //variable to update state after action completes
	SleepInMs    int       //optional Sleep time
	Async        bool
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
	TimeSpentMs int              //optional min required time spent in this task, remaining will force Sleep
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
	OnErrorTask string          //task that will run if error occur, the final workflow will return this task response
	SleepInMs   int             //optional Sleep time
}

//NeatlyTag represent a neatly tag
type NeatlyTag struct {
	Tag            string //neatly tag
	TagIndex       string //neatly tag index
	TagID          string //neatly tag id
	TagDescription string //tag description
}

//Validate check is action request is valid
func (r *ActionRequest) Validate() error {
	if r == nil {
		return errors.New("actionRequest was nil")
	}
	if r.Service == "" {
		return errors.New("actionRequest.Service was empty")
	}
	if r.Action == "" {
		return errors.New("actionRequest.Action was empty")
	}
	if r.Request == nil {
		return fmt.Errorf("request was nil for %v.%v", r.Service, r.Action)
	}
	return nil
}

//Validate validates this workflow TODO add implementation.
func (w *Workflow) Validate() error {
	return nil
}

func (w *Workflow) Task(name string) (*WorkflowTask, error) {
	name = strings.TrimSpace(name)
	for _, candidate := range w.Tasks {
		if candidate.Name == name {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("failed to lookup task: %v on %v", name, w.Name)
}


func (w *Workflow) FilterTasks(filter string) ([]*WorkflowTask, error) {
	if filter == "" || filter == "*" {
		return w.Tasks, nil
	}
	var taskNames = strings.Split(filter, ",")
	var result = make([]*WorkflowTask, 0)
	for _, taskName := range taskNames {
		 task, err := w.Task(taskName)
		 if err != nil {
		 	return nil, err
		 }
		result = append(result, task)
	}
	return result, nil
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
