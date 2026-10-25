package workflow

import (
	"fmt"
	"sync/atomic"

	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/model/msg"
)

// A listener belongs to this workflow's context, not its caller or siblings.
// State sharing and background task ownership retain the existing run policy.
type workflowAssertionScope struct {
	context *endly.Context
	passed  atomic.Int64
	failed  atomic.Int64
}

func newWorkflowAssertionScope(parent *endly.Context) *workflowAssertionScope {
	ctx := parent.Clone()
	ctx.SetState(parent.State())
	ctx.Wait = parent.Wait
	ctx.Source = parent.Source
	ctx.Debugger = parent.Debugger
	ctx.HasLogger = parent.HasLogger
	ctx.Logging = parent.Logging
	scope := &workflowAssertionScope{context: ctx}
	listener := parent.Listener
	ctx.SetListener(func(event msg.Event) {
		if response, ok := event.Value().(interface{ Assertion() []*assertly.Validation }); ok {
			for _, validation := range response.Assertion() {
				if validation != nil {
					scope.passed.Add(int64(validation.PassedCount))
					scope.failed.Add(int64(validation.FailedCount))
				}
			}
		}
		if listener != nil {
			listener(event)
		}
	})
	return scope
}

func (s *workflowAssertionScope) finish(response *RunResponse, err error) error {
	passed, failed := int(s.passed.Load()), int(s.failed.Load())
	if response != nil {
		response.PassedAssertions = passed
		response.FailedAssertions = failed
	}
	if failed == 0 {
		return err
	}
	if err != nil {
		return fmt.Errorf("%w (%d validation assertions failed)", err, failed)
	}
	return fmt.Errorf("%d validation assertions failed", failed)
}
