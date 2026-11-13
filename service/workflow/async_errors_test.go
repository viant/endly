package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
)

type asyncProbeRequest struct{ Lane string }

func TestAsyncTaskWaitsForAllLanesAndPropagatesFailures(t *testing.T) {
	_, ctx := runtimeURLFixture(t)
	cause := errors.New("synthetic lane failure")
	var started atomic.Int32
	ready := make(chan struct{})
	completed := make(chan string, 2)
	probe := endly.NewAbstractService("async-probe")
	probe.Register(&endly.Route{Action: "run", RequestProvider: func() interface{} { return &asyncProbeRequest{} }, ResponseProvider: func() interface{} { return &map[string]interface{}{} }, Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
		lane := request.(*asyncProbeRequest).Lane
		state := ctx.State()
		state.Put("lane", lane)
		if started.Add(1) == 2 {
			close(ready)
		}
		select {
		case <-ready:
		case <-time.After(2 * time.Second):
			return nil, errors.New("sibling did not start")
		}
		if state.Get("lane") != lane {
			return nil, errors.New("lane state contaminated")
		}
		completed <- lane
		if lane == "failure" {
			return nil, cause
		}
		return map[string]interface{}{"lane": lane}, nil
	}})
	manager, err := ctx.Manager()
	require.NoError(t, err)
	manager.Register(probe)
	filename := filepath.Join(t.TempDir(), "run.yaml")
	require.NoError(t, os.WriteFile(filename, []byte(`pipeline:
  lanes:
    multiAction: true
    success:
      action: async-probe:run
      lane: success
      async: true
    failure:
      action: async-probe:run
      lane: failure
      async: true
`), 0600))
	request := &RunRequest{URL: filename}

	var response RunResponse
	err = endly.Run(ctx, request, &response)
	require.Len(t, completed, 2, "the parent must join every sibling before returning")
	require.ErrorContains(t, err, cause.Error())
	require.ErrorIs(t, err, cause)
}

func registerAsyncProbe(t *testing.T, ctx *endly.Context, handler func(*endly.Context, string) error) {
	t.Helper()
	probe := endly.NewAbstractService("async-probe")
	probe.Register(&endly.Route{
		Action: "run", RequestProvider: func() interface{} { return &asyncProbeRequest{} },
		ResponseProvider: func() interface{} { return &map[string]interface{}{} },
		Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
			return nil, handler(ctx, request.(*asyncProbeRequest).Lane)
		},
	})
	manager, err := ctx.Manager()
	require.NoError(t, err)
	manager.Register(probe)
}

func asyncProbeWorkflow(t *testing.T, extra string) *RunRequest {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "run.yaml")
	require.NoError(t, os.WriteFile(filename, []byte(`pipeline:
  lanes:
    multiAction: true
    first:
      action: async-probe:run
      lane: first
      async: true
    second:
      action: async-probe:run
      lane: second
      async: true
`+extra), 0600))
	return &RunRequest{URL: filename, SharedState: true}
}

func TestAsyncTaskRetainsEveryFailureAndSynchronousCause(t *testing.T) {
	_, ctx := runtimeURLFixture(t)
	firstCause := errors.New("first concurrent failure")
	secondCause := errors.New("second concurrent failure")
	primaryCause := errors.New("synchronous primary failure")
	ready, release := make(chan struct{}), make(chan struct{})
	var started, completed atomic.Int32
	registerAsyncProbe(t, ctx, func(ctx *endly.Context, lane string) error {
		if lane == "primary" {
			select {
			case <-ready:
			case <-time.After(2 * time.Second):
				return errors.New("concurrent probes did not start")
			}
			close(release)
			return primaryCause
		}
		if started.Add(1) == 2 {
			close(ready)
		}
		select {
		case <-release:
		case <-time.After(2 * time.Second):
			return errors.New("synchronous probe did not release workers")
		}
		completed.Add(1)
		if lane == "first" {
			return firstCause
		}
		return secondCause
	})
	request := asyncProbeWorkflow(t, `    primary:
      action: async-probe:run
      lane: primary
`)
	var response RunResponse
	err := endly.Run(ctx, request, &response)
	require.EqualValues(t, 2, completed.Load())
	require.ErrorIs(t, err, primaryCause)
	require.ErrorIs(t, err, firstCause)
	require.ErrorIs(t, err, secondCause)
}

func TestAsyncTaskCancellationJoinsEveryStartedLane(t *testing.T) {
	_, ctx := runtimeURLFixture(t)
	background, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx.SetBackground(background)
	started := make(chan struct{}, 2)
	var completed atomic.Int32
	registerAsyncProbe(t, ctx, func(ctx *endly.Context, lane string) error {
		started <- struct{}{}
		<-ctx.Background().Done()
		completed.Add(1)
		return ctx.Background().Err()
	})
	request := asyncProbeWorkflow(t, "")
	result := make(chan error, 1)
	go func() {
		var response RunResponse
		result <- endly.Run(ctx, request, &response)
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("both concurrent probes must start before cancellation")
		}
	}
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
		require.EqualValues(t, 2, completed.Load(), "cancellation must join both workers")
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled workflow did not join its workers")
	}
}

func TestAsyncTaskPostRunsAfterEveryLaneCompletes(t *testing.T) {
	_, ctx := runtimeURLFixture(t)
	var completed atomic.Int32
	ready := make(chan struct{})
	var started atomic.Int32
	registerAsyncProbe(t, ctx, func(ctx *endly.Context, lane string) error {
		if started.Add(1) == 2 {
			close(ready)
		}
		select {
		case <-ready:
		case <-time.After(2 * time.Second):
			return errors.New("sibling probe did not start")
		}
		completed.Add(1)
		return nil
	})
	state := ctx.State()
	state.Put("CompletedProbeCount", func(interface{}, data.Map) (interface{}, error) {
		return completed.Load(), nil
	})
	request := asyncProbeWorkflow(t, "    post: {joined: '$CompletedProbeCount()'}\n")
	var response RunResponse
	require.NoError(t, endly.Run(ctx, request, &response))
	require.EqualValues(t, 2, state.Get("joined"), "task post must observe joined workers")
}
