package workflow

import (
	"github.com/viant/endly"
	"github.com/viant/endly/model"
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

//Push push process to context
func Push(context *endly.Context, process *model.Process) {
	var processes = processes(context)
	if process.Workflow != nil {
		context.Source = process.Workflow.Source
	}
	processes.Push(process)
}

//Remove push process to context
func Pop(context *endly.Context) *model.Process {
	var processes = processes(context)
	var process = processes.Pop()
	if process.Workflow != nil {
		context.Source = process.Workflow.Source
	}
	return process
}

//Returns last process
func Last(context *endly.Context) *model.Process {
	var processes = processes(context)
	return processes.Last()
}
