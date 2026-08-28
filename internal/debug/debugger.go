package debug

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	CommandPause    = "pause"
	CommandStep     = "step"
	CommandNext     = "next"
	CommandContinue = "continue"
	CommandStop     = "stop"
	StateRunning    = "running"
	StatePaused     = "paused"
	StateStopped    = "stopped"
)

type State struct {
	Status      string      `json:"status"`
	Mode        string      `json:"mode"`
	Point       Step        `json:"point"`
	Breakpoints []Step      `json:"breakpoints,omitempty"`
	PausedAt    *time.Time  `json:"pausedAt,omitempty"`
	Snapshot    interface{} `json:"snapshot,omitempty"`
}

type Debugger struct {
	mux            sync.RWMutex
	Breakpoints    map[Step]struct{}
	StepMode       bool
	pauseRequested bool
	paused         bool
	stopped        bool
	nextTask       string
	point          Step
	pausedAt       *time.Time
	snapshot       interface{}
	commands       chan string
}

func NewDebugger() *Debugger {
	return &Debugger{Breakpoints: make(map[Step]struct{}), commands: make(chan string, 1)}
}

func (d *Debugger) SetBreakpoint(step Step) {
	d.mux.Lock()
	d.Breakpoints[step] = struct{}{}
	d.mux.Unlock()
}

func (d *Debugger) RemoveBreakpoint(step Step) {
	d.mux.Lock()
	delete(d.Breakpoints, step)
	d.mux.Unlock()
}

func (d *Debugger) EnableStepMode(enable bool) {
	d.mux.Lock()
	d.StepMode = enable
	d.mux.Unlock()
}

func (d *Debugger) Before(ctx context.Context, point Step, snapshot interface{}) error {
	d.mux.Lock()
	if d.stopped {
		d.mux.Unlock()
		return context.Canceled
	}
	breakpoint := false
	for candidate := range d.Breakpoints {
		if matches(candidate, point) {
			breakpoint = true
			break
		}
	}
	nextBoundary := d.nextTask != "" && point.Kind == "task" && point.TaskName != d.nextTask
	shouldPause := d.StepMode || d.pauseRequested || breakpoint || nextBoundary
	if !shouldPause {
		d.point = point
		d.mux.Unlock()
		return nil
	}
	now := time.Now().UTC()
	d.pauseRequested = false
	if nextBoundary {
		d.nextTask = ""
	}
	d.paused = true
	d.point = point
	d.pausedAt = &now
	d.snapshot = snapshot
	d.mux.Unlock()

	select {
	case <-ctx.Done():
		d.markStopped()
		return ctx.Err()
	case command := <-d.commands:
		d.mux.Lock()
		d.paused = false
		d.pausedAt = nil
		d.snapshot = nil
		switch command {
		case CommandContinue:
			d.StepMode = false
			d.nextTask = ""
		case CommandStep:
			d.StepMode = true
			d.nextTask = ""
		case CommandNext:
			d.StepMode = false
			d.nextTask = d.point.TaskName
		case CommandStop:
			d.stopped = true
			d.mux.Unlock()
			return context.Canceled
		}
		d.mux.Unlock()
		return nil
	}
}

func matches(candidate, point Step) bool {
	return (candidate.Workflow == "" || candidate.Workflow == point.Workflow) &&
		(candidate.TaskName == "" || candidate.TaskName == point.TaskName) &&
		(candidate.Action == "" || candidate.Action == point.Action) &&
		(candidate.TagID == "" || candidate.TagID == point.TagID) &&
		(candidate.Kind == "" || candidate.Kind == point.Kind)
}

func (d *Debugger) Control(command string) error {
	d.mux.Lock()
	switch command {
	case CommandPause:
		if !d.paused {
			d.pauseRequested = true
		}
		d.mux.Unlock()
		return nil
	case CommandContinue, CommandStep, CommandNext, CommandStop:
		if !d.paused {
			d.mux.Unlock()
			return errors.New("debug operation was not paused")
		}
	default:
		d.mux.Unlock()
		return fmt.Errorf("unsupported debug command %q", command)
	}
	d.mux.Unlock()
	select {
	case d.commands <- command:
		return nil
	default:
		return errors.New("a debug command was already pending")
	}
}

func (d *Debugger) State() *State {
	d.mux.RLock()
	defer d.mux.RUnlock()
	status := StateRunning
	if d.paused {
		status = StatePaused
	} else if d.stopped {
		status = StateStopped
	}
	mode := CommandContinue
	if d.StepMode {
		mode = CommandStep
	}
	result := &State{Status: status, Mode: mode, Point: d.point, PausedAt: d.pausedAt, Snapshot: d.snapshot}
	for breakpoint := range d.Breakpoints {
		result.Breakpoints = append(result.Breakpoints, breakpoint)
	}
	return result
}

// Legacy methods remain for compatibility; served workflows use Before directly.
func (d *Debugger) BeforeTaskExecution(step Step, request interface{}) {
	_ = d.Before(context.Background(), step, request)
}

func (d *Debugger) AfterTaskExecution(_ Step, _ interface{}) {}

func (d *Debugger) markStopped() {
	d.mux.Lock()
	d.paused = false
	d.stopped = true
	d.pausedAt = nil
	d.snapshot = nil
	d.mux.Unlock()
}
