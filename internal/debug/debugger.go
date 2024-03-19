package debug

import (
	"fmt"
	"sync"
)

// Debugger is responsible for debugging Endly workflows.
type Debugger struct {
	mux          sync.Mutex
	Breakpoints  map[Step]struct{} // Task names where the debugger will pause execution
	StepMode     bool              // Flag to step through the workflow one task at a time
	continueExec chan bool         // Channel used to control execution flow
}

// NewDebugger creates a new debugger instance with initialized channels.
func NewDebugger() *Debugger {
	return &Debugger{
		Breakpoints:  make(map[Step]struct{}),
		continueExec: make(chan bool),
	}
}

// SetBreakpoint sets a breakpoint on a task by its name.
func (d *Debugger) SetBreakpoint(step Step) {
	d.mux.Lock()
	defer d.mux.Unlock()
	d.Breakpoints[step] = struct{}{}
}

// RemoveBreakpoint removes a breakpoint from a task by its name.
func (d *Debugger) RemoveBreakpoint(breakpoint Step) {
	d.mux.Lock()
	defer d.mux.Unlock()
	delete(d.Breakpoints, breakpoint)
}

// BeforeTaskExecution is modified to pause at breakpoints or in step mode, waiting for channel input to continue.
func (d *Debugger) BeforeTaskExecution(step Step, request interface{}) {
	fmt.Printf("Before executing task %s: request = %+v\n", step, request)
	hasBreakpoint := false
	d.mux.Lock()
	_, hasBreakpoint = d.Breakpoints[step]
	d.mux.Unlock()

	if hasBreakpoint || d.StepMode {
		fmt.Println("Execution paused. Press enter to continue...")
		go func() {
			fmt.Scanln()           // Wait for user input
			d.continueExec <- true // Signal to continue execution
		}()
		<-d.continueExec // Wait for signal to continue
	}
}

// AfterTaskExecution logs task results, similar to the previous implementation.
func (d *Debugger) AfterTaskExecution(step Step, result interface{}) {
	fmt.Printf("After executing task %s: result = %+v\n", step, result)
	if d.StepMode {
		// In step mode, pause after each task execution.
		fmt.Println("Step execution paused. Press enter to continue to the next task...")
		go func() {
			fmt.Scanln()           // Wait for user input
			d.continueExec <- true // Signal to continue execution
		}()
		<-d.continueExec // Wait for signal to continue
	}
}

// EnableStepMode enables or disables step mode.
func (d *Debugger) EnableStepMode(enable bool) {
	d.StepMode = enable
}
