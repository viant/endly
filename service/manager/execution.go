package manager

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/viant/endly"
	"github.com/viant/endly/internal/debug"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/service/workflow"
)

type queuedEvent struct {
	Type        string
	EndlyType   string
	Timestamp   time.Time
	Value       interface{}
	Messages    []*msg.Message
	Activity    *model.Activity
	ActivityEnd bool
}

func (s *Service) executeWorkflow(session *Session, operationID string, input *RunWorkflowRequest) {
	session.execMu.Lock()
	defer session.execMu.Unlock()

	session.mu.Lock()
	operation := session.operations[operationID]
	if operation == nil || operation.Status == OperationCancelled || session.closed {
		session.mu.Unlock()
		return
	}
	parent := context.Background()
	timeout := input.Timeout
	if timeout <= 0 && input.TimeoutMillis > 0 {
		timeout = time.Duration(input.TimeoutMillis) * time.Millisecond
	}
	var operationContext context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		operationContext, cancel = context.WithTimeout(parent, timeout)
	} else {
		operationContext, cancel = context.WithCancel(parent)
	}
	now := time.Now().UTC()
	operation.Status = OperationRunning
	operation.StartedAt = &now
	operation.cancel = cancel
	logging := session.logging
	if input.Debug {
		operation.debugger = debug.NewDebugger()
		operation.debugger.EnableStepMode(true)
		for _, breakpoint := range input.Breakpoints {
			operation.debugger.SetBreakpoint(breakpoint)
		}
	}
	operationDebugger := operation.debugger
	session.mu.Unlock()

	originalListener := session.context.Listener
	originalDebugger := session.context.Debugger
	session.context.SetBackground(operationContext)
	session.context.SetLogging(logging)
	session.context.Debugger = operationDebugger
	stopEvents := s.startEventPump(session, operation, originalListener, input.Callbacks)
	defer stopEvents()

	defer func() {
		cancel()
		session.context.SetBackground(context.Background())
		session.context.Debugger = originalDebugger
	}()

	publishParameters := true
	if input.PublishParameters != nil {
		publishParameters = *input.PublishParameters
	}
	sharedState := true
	if input.SharedState != nil {
		sharedState = *input.SharedState
	}
	runRequest := &workflow.RunRequest{
		Name:              input.Workflow,
		URL:               input.Workflow,
		Tasks:             input.Tasks,
		TagIDs:            input.TagIDs,
		Params:            input.Params,
		PublishParameters: publishParameters,
		SharedState:       sharedState,
	}
	response := &workflow.RunResponse{}
	err := endly.Run(session.context, runRequest, response)
	stopEvents()

	finished := time.Now().UTC()
	session.mu.Lock()
	defer session.mu.Unlock()
	operation = session.operations[operationID]
	if operation == nil {
		return
	}
	operation.cancel = nil
	operation.FinishedAt = &finished
	if operationContext.Err() != nil || operation.Status == OperationCancelling {
		operation.Status = OperationCancelled
		operation.Error = operationContext.Err().Error()
		return
	}
	if err != nil {
		operation.Status = OperationFailed
		operation.Error = err.Error()
		return
	}
	operation.Status = OperationSucceeded
	operation.Result = map[string]interface{}{
		"data":      response.Data,
		"sessionId": response.SessionID,
	}
}

func (s *Service) executeAction(session *Session, operationID string, input *RunActionRequest) {
	session.execMu.Lock()
	defer session.execMu.Unlock()
	session.mu.Lock()
	operation := session.operations[operationID]
	if operation == nil || operation.Status == OperationCancelled || session.closed {
		session.mu.Unlock()
		return
	}
	parent := context.Background()
	var operationContext context.Context
	var cancel context.CancelFunc
	if input.TimeoutMillis > 0 {
		operationContext, cancel = context.WithTimeout(parent, time.Duration(input.TimeoutMillis)*time.Millisecond)
	} else {
		operationContext, cancel = context.WithCancel(parent)
	}
	now := time.Now().UTC()
	operation.Status = OperationRunning
	operation.StartedAt = &now
	operation.cancel = cancel
	logging := session.logging
	session.mu.Unlock()

	originalListener := session.context.Listener
	session.context.SetBackground(operationContext)
	session.context.SetLogging(logging)
	stopEvents := s.startEventPump(session, operation, originalListener, input.Callbacks)
	defer stopEvents()
	defer func() {
		cancel()
		session.context.SetBackground(context.Background())
	}()

	typedRequest, err := session.context.NewRequest(input.Service, input.Action, input.Request)
	var response interface{}
	if err == nil {
		err = endly.Run(session.context, typedRequest, &response)
	}
	stopEvents()
	finished := time.Now().UTC()
	session.mu.Lock()
	defer session.mu.Unlock()
	operation = session.operations[operationID]
	if operation == nil {
		return
	}
	operation.cancel = nil
	operation.FinishedAt = &finished
	if operationContext.Err() != nil || operation.Status == OperationCancelling {
		operation.Status = OperationCancelled
		if operationContext.Err() != nil {
			operation.Error = operationContext.Err().Error()
		}
		return
	}
	if err != nil {
		operation.Status = OperationFailed
		operation.Error = err.Error()
		return
	}
	operation.Status = OperationSucceeded
	operation.Result = map[string]interface{}{"response": encodable(response), "sessionId": session.info.SessionID}
}

func (s *Service) startEventPump(session *Session, operation *Operation, original msg.Listener, callbacks []Callback) func() {
	operation.eventQueue = make(chan *queuedEvent, 256)
	operation.eventDone = make(chan struct{})
	var sequence int64
	go func() {
		defer close(operation.eventDone)
		for event := range operation.eventQueue {
			operationEvent := &OperationEvent{
				Sequence:    atomic.AddInt64(&sequence, 1),
				Type:        event.Type,
				EndlyType:   event.EndlyType,
				Timestamp:   event.Timestamp,
				Value:       event.Value,
				Messages:    event.Messages,
				Activity:    event.Activity,
				ActivityEnd: event.ActivityEnd,
			}
			session.mu.Lock()
			current := session.operations[operation.ID]
			if current != nil && len(current.Events) < s.maxEvents {
				current.Events = append(current.Events, operationEvent)
			}
			session.mu.Unlock()
			for _, callback := range callbacks {
				if callbackMatches(callback, event.Type) {
					callbackCopy := callback
					go s.deliverCallback(callbackCopy, operation.ID, session.info.SessionID, operationEvent)
				}
			}
		}
	}()
	session.context.SetListener(func(event msg.Event) {
		immutable := snapshotEvent(event)
		select {
		case operation.eventQueue <- immutable:
		default:
			atomic.AddInt64(&operation.droppedEvents, 1)
		}
	})
	var stopped atomic.Bool
	return func() {
		if !stopped.CompareAndSwap(false, true) {
			return
		}
		session.context.SetListener(original)
		close(operation.eventQueue)
		<-operation.eventDone
	}
}

func snapshotEvent(event msg.Event) *queuedEvent {
	result := &queuedEvent{Type: callbackEventName(event), EndlyType: event.Type(), Timestamp: event.Timestamp().UTC()}
	if reporter, ok := event.Value().(msg.Reporter); ok {
		result.Messages = cloneMessages(reporter.Messages())
	}
	switch value := event.Value().(type) {
	case *workflow.WorkflowStartEvent, *workflow.WorkflowEndEvent,
		*model.TaskStartEvent, *model.TaskEndEvent,
		*workflow.PrintRequest, *workflow.RunResponse,
		*msg.ErrorEvent:
		result.Value = encodable(value)
	case *model.Activity:
		activity := *value
		if value.MetaTag != nil {
			meta := *value.MetaTag
			activity.MetaTag = &meta
		}
		activity.Request = nil
		activity.Response = nil
		activity.ServiceResponse = nil
		result.Activity = &activity
		result.Value = map[string]interface{}{
			"Service": value.Service,
			"Action":  value.Action,
			"Request": encodable(value.Request),
			"TagID":   value.TagID,
		}
	case *model.ActivityEndEvent:
		result.ActivityEnd = true
		result.Value = map[string]interface{}{"Response": encodable(value.Response)}
	default:
		result.Value = nil
	}
	return result
}

func cloneMessages(source []*msg.Message) []*msg.Message {
	result := make([]*msg.Message, 0, len(source))
	for _, message := range source {
		if message == nil {
			continue
		}
		copy := *message
		if message.Header != nil {
			header := *message.Header
			copy.Header = &header
		}
		if message.Tag != nil {
			tag := *message.Tag
			copy.Tag = &tag
		}
		copy.Items = make([]*msg.Styled, 0, len(message.Items))
		for _, item := range message.Items {
			if item == nil {
				continue
			}
			itemCopy := *item
			copy.Items = append(copy.Items, &itemCopy)
		}
		result = append(result, &copy)
	}
	return result
}

func (s *Service) WaitOperation(ctx context.Context, sessionID, operationID string) (*Operation, error) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		operation, err := s.GetOperation(sessionID, operationID)
		if err != nil {
			return nil, err
		}
		switch operation.Status {
		case OperationSucceeded, OperationFailed, OperationCancelled:
			return operation, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for operation %s: %w", operationID, ctx.Err())
		case <-ticker.C:
		}
	}
}
