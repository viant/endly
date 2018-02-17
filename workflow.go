package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
	"sync/atomic"
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
	*Repeatable
	RunCriteria  string    //criteria to run this action
	SkipCriteria string    //criteria to skip current action to continue to next tag id action
	Name         string    //Id of the service action
	Description  string    //description
	Init         Variables //variables to initialise state before action runs
	Post         Variables //variable to update state after action completes
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
	DeferTask   string          //task that will alway run if there has been previous  error or not
	SleepTimeMs int             //optional Sleep time
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

//Validate validates this workflow
func (w *Workflow) Validate() error {
	if len(w.Tasks) == 0 {
		return errors.New("Tasks were empty")
	}
	if w.DeferTask != "" {
		if _, err := w.Task(w.DeferTask); err != nil {
			return err
		}
	}
	if w.OnErrorTask != "" {
		if _, err := w.Task(w.OnErrorTask); err != nil {
			return err
		}
	}
	return nil
}

//Task returns a task for supplied name
func (w *Workflow) Task(name string) (*WorkflowTask, error) {
	name = strings.TrimSpace(name)
	for _, candidate := range w.Tasks {
		if candidate.Name == name {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("failed to lookup task: %v on %v", name, w.Name)
}

//FilterTasks returns filter tasked for provided filter.
func (w *Workflow) FilterTasks(filter string) ([]*WorkflowTask, error) {
	if filter == "" || filter == "*" {
		if w.DeferTask == "" && w.OnErrorTask == "" {
			return w.Tasks, nil
		}
		var result = make([]*WorkflowTask, 0)
		for _, candidate := range w.Tasks {
			if w.DeferTask == candidate.Name || w.OnErrorTask == candidate.Name {
				continue
			}
			result = append(result, candidate)
		}
		return result, nil
	}
	var taskNames = strings.Split(filter, ",")
	var result = make([]*WorkflowTask, 0)
	for _, taskName := range taskNames {
		if w.DeferTask == taskName || w.OnErrorTask == taskName {
			continue
		}
		task, err := w.Task(taskName)
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	return result, nil
}

func (t *WorkflowTask) HasTagID(tagIDs map[string]bool) bool {
	if tagIDs == nil {
		return false
	}
	for _, action := range t.Actions {
		if tagIDs[action.TagID] {
			return true
		}
	}
	return false
}

//WorkflowError represent workflow error
type WorkflowError struct {
	Error        string
	WorkflowName string
	TaskName     string
	*ActionRequest
	Request  interface{}
	Response interface{}
}

//WorkflowControl control workflow execution
type WorkflowControl struct {
	*Workflow
	Terminated    int32
	ScheduledTask *WorkflowTask
	*WorkflowError
}

//Terminate flags current workflow as terminated
func (c *WorkflowControl) Terminate() {
	atomic.StoreInt32(&c.Terminated, 1)
}

//CanRun returns true if current workflow can run
func (c *WorkflowControl) CanRun() bool {
	return !(c.IsTerminated() || c.ScheduledTask != nil)
}

//IsTerminated returns true if current workflow has been terminated
func (c *WorkflowControl) IsTerminated() bool {
	return atomic.LoadInt32(&c.Terminated) == 1
}

//Workflows  represents workflows
type Workflows []*WorkflowControl

//Push adds a workflow to the workflow stack.
func (w *Workflows) Push(workflow *Workflow) *WorkflowControl {
	var result = &WorkflowControl{Workflow: workflow, WorkflowError: &WorkflowError{WorkflowName: workflow.Name}}
	*w = append(*w, result)
	return result
}

//Pop removes the first workflow from the workflow stack.
func (w *Workflows) Pop() *Workflow {
	if len(*w) == 0 {
		return nil
	}
	var result = (*w)[len(*w)-1]
	(*w) = (*w)[0 : len(*w)-1]
	return result.Workflow
}

//Last returns the last workflow from the workflow stack.
func (w *Workflows) Last() *Workflow {
	control := w.LastControl()
	if control == nil {
		return nil
	}
	return control.Workflow
}

//LastControl returns the last workflow from the workflow stack.
func (w *Workflows) LastControl() *WorkflowControl {
	if w == nil {
		return nil
	}
	var workflowCount = len(*w)
	if workflowCount == 0 {
		return nil
	}
	return (*w)[workflowCount-1]
}
