package model

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
)

//Workflow repesents a workflow
type Workflow struct {
	Source      *url.Resource //source definition of the workflow
	Data        data.Map      //workflow data
	Name        string        //worfklow Id
	Description string        //description
	Init        Variables     //variables to initialise state before this workflow runs
	Post        Variables     //variables to initialise state before this workflow runs
	Tasks       []*Task       //workflow task
	OnErrorTask string        //task that will run if error occur, the final workflow will return this task response
	DeferTask   string        //task that will alway run if there has been previous  error or not
	SleepTimeMs int           //optional Sleep time
}

//Validate validates this workflow
func (w *Workflow) Validate() error {
	if len(w.Tasks) == 0 {
		return errors.New("tasks were empty")
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
	for _, task := range w.Tasks {
		if len(task.Actions) == 0 {
			continue
		}
		for _, action := range task.Actions {
			action.ServiceRequest = action.ServiceRequest.Init()
			action.Repeater = action.Repeater.Init()
			if err := action.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

//Task returns a task for supplied name
func (w *Workflow) Task(name string) (*Task, error) {
	name = strings.TrimSpace(name)
	for _, candidate := range w.Tasks {
		if candidate.Name == name {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("failed to lookup task: %v on %v", name, w.Name)
}

//Select returns tasked matching supplied selector.
func (w *Workflow) Select(selector TasksSelector) ([]*Task, error) {
	if selector.RunAll() {
		if w.DeferTask == "" && w.OnErrorTask == "" {
			return w.Tasks, nil
		}
		var result = make([]*Task, 0)
		for _, candidate := range w.Tasks {
			if w.DeferTask == candidate.Name || w.OnErrorTask == candidate.Name {
				continue
			}
			result = append(result, candidate)
		}
		return result, nil
	}
	var result = make([]*Task, 0)
	for _, taskName := range selector.Tasks() {
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
