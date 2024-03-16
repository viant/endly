package model

import (
	"github.com/pkg/errors"
	"github.com/viant/endly/model/location"
	"github.com/viant/toolbox/data"
)

// Workflow represents a workflow
type Workflow struct {
	Source *location.Resource //source definition of the workflow
	Data   data.Map           //workflow data
	*AbstractNode
	*TasksNode //workflow tasks
}

// Init validates this workflow
func (w *Workflow) Init() error {
	for _, task := range w.Tasks {
		if w.Logging != nil && task.Logging == nil {
			task.Logging = w.Logging
		}
		if err := task.init(); err != nil {
			return err
		}
	}
	return nil
}

// Validate validates this workflow
func (w *Workflow) Validate() error {
	if len(w.Tasks) == 0 {
		return errors.New("tasks were empty")
	}
	if w.DeferredTask != "" {
		if _, err := w.Task(w.DeferredTask); err != nil {
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
