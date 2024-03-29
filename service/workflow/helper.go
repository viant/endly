package workflow

import (
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/toolbox/data"
)

var processesKey = (*model.Processes)(nil)

func processes(context *endly.Context) *model.Processes {
	var result *model.Processes
	if !context.Contains(processesKey) {
		result = model.NewProcesses()
		_ = context.Put(processesKey, result)
	} else {
		context.GetInto(processesKey, &result)
	}
	return result
}

// Push push process to context
func Push(context *endly.Context, process *model.Process) {
	var processes = processes(context)
	if process.Source != nil {
		context.Source = process.Source
	}
	processes.Push(process)
}

// Remove push process to context
func Pop(context *endly.Context) *model.Process {
	var processes = processes(context)
	var process = processes.Pop()
	if process != nil && process.Source != nil {
		context.Source = process.Source
	}
	return process
}

// Last returns last process
func Last(context *endly.Context) *model.Process {
	var processes = processes(context)
	return processes.Last()
}

// LastWorkflow returns last workflow
func LastWorkflow(context *endly.Context) *model.Process {
	var processes = processes(context)
	return processes.LastWorkflow()
}

func isWorkflowRunAction(action *model.Action) bool {
	return action.Action == "run" && action.Service == ServiceID
}

func runWithoutSelfIfNeeded(process *model.Process, action *model.Action, state data.Map, handler func() error) error {
	if !isWorkflowRunAction(action) {
		return handler()
	}
	state.Delete(selfStateKey)
	defer state.Put(selfStateKey, process.State)
	return handler()
}
