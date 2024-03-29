package model

import (
	"github.com/viant/endly/model/location"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"sync"
	"sync/atomic"
)

// Process represents a running instance of workflow/pipeline process.
type Process struct {
	Source   *location.Resource
	Owner    string
	TagIDs   map[string]bool
	HasTagID bool
	Workflow *Workflow
	Task     *Task
	TaskNode *TasksNode
	*Activities
	State      data.Map
	Terminated int32
	Scheduled  *Task
	*ExecutionError
}

// Terminate flags current workflow as terminated
func (p *Process) Terminate() {
	atomic.StoreInt32(&p.Terminated, 1)
}

// SetTask sets process task
func (p *Process) SetTask(task *Task) {
	p.Task = task
	p.HasTagID = false
	if len(p.TagIDs) > 0 {
		p.HasTagID = task.HasTagID(p.TagIDs)
	}
}

// CanRun returns true if current workflow can run
func (p *Process) CanRun() bool {
	return !(p.IsTerminated() || p.Scheduled != nil)
}

// IsTerminated returns true if current workflow has been terminated
func (p *Process) IsTerminated() bool {
	return atomic.LoadInt32(&p.Terminated) == 1
}

// Push adds supplied activity
func (p *Process) Push(activity *Activity) {
	if p.Workflow != nil {
		activity.Caller = p.Workflow.Name
	}
	p.Activities.Push(activity)
}

// Push adds a workflow to the workflow stack.
func (p *Process) AddTagIDs(tagIDs ...string) {
	for _, tagID := range tagIDs {
		p.TagIDs[tagID] = true
	}
}

// NewProcess creates a new workflow, pipeline process
func NewProcess(source *location.Resource, workflow *Workflow, upstream *Process) *Process {
	var process = &Process{
		Source:         source,
		ExecutionError: &ExecutionError{},
		Workflow:       workflow,
		Activities:     NewActivities(),
	}
	if source != nil {
		_, process.Owner = toolbox.URLSplit(source.URL)
	}
	process.TagIDs = map[string]bool{}
	if upstream != nil && len(upstream.TagIDs) > 0 {
		for k := range upstream.TagIDs {
			process.TagIDs[k] = true
		}
	}
	return process
}

// processes  represents running workflow/pipe process stack.
type Processes struct {
	mux       *sync.RWMutex
	processes []*Process
}

// Push adds a workflow to the workflow stack.
func (p *Processes) Push(process *Process) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.processes = append(p.processes, process)
}

// Pop removes the first workflow from the workflow stack.
func (p *Processes) Pop() *Process {
	p.mux.Lock()
	defer p.mux.Unlock()
	if len(p.processes) == 0 {
		return nil
	}
	var result = (p.processes)[len(p.processes)-1]
	p.processes = p.processes[0 : len(p.processes)-1]
	return result
}

// Last returns the last process.
func (p *Processes) Last() *Process {
	p.mux.RLock()
	defer p.mux.RUnlock()
	for i := len(p.processes) - 1; i >= 0; i-- {
		return p.processes[i]
	}
	return nil
}

// Recent returns the most reset process.
func (p *Processes) Recent(count int) []*Process {
	p.mux.RLock()
	defer p.mux.RUnlock()
	var result = make([]*Process, 0)
	for i := len(p.processes) - 1; i >= 0; i-- {
		result = append(result, p.processes[i])
		if len(result) >= count {
			return result
		}
	}
	return result
}

// LastWorkflow returns the last workflow.
func (p *Processes) LastWorkflow() *Process {
	p.mux.RLock()
	defer p.mux.RUnlock()
	for i := len(p.processes) - 1; i >= 0; i-- {
		if p.processes[i].Workflow != nil {
			return p.processes[i]
		}
	}
	return nil
}

// FirstWorkflow returns the first workflow.
func (p *Processes) FirstWorkflow() *Process {
	p.mux.RLock()
	defer p.mux.RUnlock()
	for i := 0; i < len(p.processes); i++ {
		if p.processes[i].Workflow != nil {
			return p.processes[i]
		}
	}
	return nil
}

// First returns the first process.
func (p *Processes) First() *Process {
	p.mux.RLock()
	defer p.mux.RUnlock()
	for i := 0; i < len(p.processes); i++ {
		return p.processes[i]
	}
	return nil
}

// NewProcesses creates a new processes
func NewProcesses() *Processes {
	return &Processes{
		processes: make([]*Process, 0),
		mux:       &sync.RWMutex{},
	}
}

// Error represent workflow error
type ExecutionError struct {
	Error    string
	Caller   string
	TaskName string
	Request  interface{}
	Response interface{}
}

// AsMap returns error map
func (e *ExecutionError) AsMap() map[string]interface{} {
	var result = map[string]interface{}{}
	toolbox.DefaultConverter.AssignConverted(&result, e)
	return result
}
