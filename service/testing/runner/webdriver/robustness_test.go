package webdriver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

type changingElement struct {
	calls int
	ready int
}

func (e *changingElement) Text() (string, error) {
	e.calls++
	if e.calls >= e.ready {
		return "Ready now", nil
	}
	return "Pending", nil
}

func TestCall_RetriesExpectation(t *testing.T) {
	manager := endly.New()
	ctx := manager.NewContext(toolbox.NewContext())
	defer ctx.Close()
	target := &changingElement{ready: 3}
	response := &ServiceCallResponse{Data: data.NewMap()}
	call := &MethodCall{
		Wait: Wait{
			WaitTimeMs:     500,
			PollIntervalMs: 1,
			Expectation:    &CallExpectation{Matcher: "contains", Value: "Ready"},
		},
		Method: "Text",
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	if err := service.call(ctx, target, call, response, "status"); err != nil {
		t.Fatal(err)
	}
	if target.calls != 3 {
		t.Fatalf("calls=%d, wanted 3", target.calls)
	}
	if actual := response.Data.Get("status"); actual != "Ready now" {
		t.Fatalf("status=%v", actual)
	}
}

func TestCall_ExpectationTimeoutIncludesActual(t *testing.T) {
	manager := endly.New()
	ctx := manager.NewContext(toolbox.NewContext())
	defer ctx.Close()
	response := &ServiceCallResponse{Data: data.NewMap()}
	call := &MethodCall{
		Wait: Wait{
			WaitTimeMs:     5,
			PollIntervalMs: 1,
			Expectation:    &CallExpectation{Matcher: "equal", Value: "Ready"},
		},
		Method: "Text",
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	err := service.call(ctx, &changingElement{ready: 1_000}, call, response, "status")
	if err == nil || !strings.Contains(err.Error(), "actual Pending") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCall_HonorsCancellation(t *testing.T) {
	manager := endly.New()
	ctx := manager.NewContext(toolbox.NewContext())
	background, cancel := context.WithCancel(context.Background())
	cancel()
	ctx.SetBackground(background)
	call := &MethodCall{
		Wait:   Wait{WaitTimeMs: int(time.Second / time.Millisecond), Expectation: &CallExpectation{Matcher: "equal", Value: "Ready"}},
		Method: "Text",
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	err := service.call(ctx, &changingElement{ready: 1_000}, call, &ServiceCallResponse{Data: data.NewMap()}, "status")
	if err != context.Canceled {
		t.Fatalf("error=%v, wanted context.Canceled", err)
	}
}
