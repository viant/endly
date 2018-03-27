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
	if process.Source != nil {
		context.Source = process.Source
	}
	processes.Push(process)
}

//Remove push process to context
func Pop(context *endly.Context) *model.Process {
	var processes = processes(context)
	var process = processes.Pop()
	if process.Source != nil {
		context.Source = process.Source
	}
	return process
}

//Last returns last process
func Last(context *endly.Context) *model.Process {
	var processes = processes(context)
	return processes.Last()
}

//LastWorkflow returns last workflow
func LastWorkflow(context *endly.Context) *model.Process {
	var processes = processes(context)
	return processes.LastWorkflow()
}



//FirstWorkflow returns last workflow
func FirstWorkflow(context *endly.Context) *model.Process {
	var processes = processes(context)
	return processes.FirstWorkflow()
}
