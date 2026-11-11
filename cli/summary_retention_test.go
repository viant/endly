package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/stretchr/testify/require"
	"github.com/viant/assertly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
	httprunner "github.com/viant/endly/service/testing/runner/http"
	"github.com/viant/endly/service/testing/validator"
	"github.com/viant/endly/service/workflow"
)

type customSummaryEvent struct {
	msg.Event
}

func (e *customSummaryEvent) Type() string    { return "custom_event" }
func (e *customSummaryEvent) Package() string { return "custom" }

func TestEventAddEventKeepsPublicContract(t *testing.T) {
	tag := &Event{}
	original := msg.NewEventWithInit(&workflow.InitEvent{}, msg.NewEvent("init"))
	tag.AddEvent(original)
	require.Same(t, original, tag.Events[0])
	require.Same(t, original.Init(), tag.Events[0].Init())
}

func TestSummaryRecorderOnlyDropsNativeGraphEvents(t *testing.T) {
	runner := New()
	for _, payload := range []interface{}{
		&workflow.LoadedEvent{}, &workflow.InitEvent{}, &model.Activity{}, &model.ActivityEndEvent{},
	} {
		native := msg.NewEventWithInit(payload, msg.NewEvent("init"))
		custom := &customSummaryEvent{Event: native}
		tag := &Event{}
		runner.recordSummaryEvent(tag, native)
		runner.recordSummaryEvent(tag, custom)
		require.Len(t, tag.Events, 2)
		require.Nil(t, tag.Events[0])
		require.Same(t, custom, tag.Events[1])
		require.Same(t, native.Init(), tag.Events[1].Init())
		require.Equal(t, native.Timestamp(), tag.firstTimestamp)
	}
}

func TestSummaryRecorderDetachesOnlyNativeDiagnostics(t *testing.T) {
	runner := New()
	payload := &assertly.Validation{PassedCount: 1}
	init := msg.NewEvent(make([]byte, 1024))
	native := msg.NewEventWithInit(payload, init)
	native.SetLoggable(true)
	custom := &customSummaryEvent{Event: native}
	tag := &Event{}
	runner.recordSummaryEvent(tag, native)
	runner.recordSummaryEvent(tag, custom)
	detached := tag.Events[0]
	require.Nil(t, detached.Init())
	require.Same(t, payload, detached.Value())
	require.Equal(t, native.Type(), detached.Type())
	require.Equal(t, native.Package(), detached.Package())
	require.Equal(t, native.Timestamp(), detached.Timestamp())
	require.True(t, detached.IsLoggable())
	detached.SetLoggable(false)
	require.True(t, native.IsLoggable())
	require.Same(t, init, native.Init())
	require.Same(t, custom, tag.Events[1])
	require.Same(t, init, tag.Events[1].Init())
}

func TestSummaryRecorderKeepsUnknownValuesAndOrdinals(t *testing.T) {
	runner := New()
	original := msg.NewEvent(42)
	tag := &Event{}
	runner.recordSummaryEvent(tag, msg.NewEvent(&workflow.InitEvent{}))
	runner.recordSummaryEvent(tag, original)
	require.Len(t, tag.Events, 2)
	require.Nil(t, tag.Events[0])
	require.Same(t, original, tag.Events[1])
}

func TestSummaryRecorderPreservesFailureSummary(t *testing.T) {
	for _, withSubEvent := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "subEvent"}[withSubEvent], func(t *testing.T) {
			validation := &assertly.Validation{FailedCount: 1, Failures: []*assertly.Failure{
				{Path: "result:Responses[0].id", Expected: 2, Actual: 1, Message: "id differs"},
			}}
			candidates := []msg.Event{
				msg.NewEvent(&workflow.InitEvent{}),
				msg.NewEvent(&httprunner.Request{Method: "GET", URL: "http://example.invalid/check"}),
				msg.NewEvent(&model.ActivityEndEvent{}),
				msg.NewEvent(&httprunner.Response{Code: 200, Body: `{"id":1}`}),
				msg.NewEvent(validation),
			}
			var expectedOutput string
			var expectedSummary []byte
			for _, optimized := range []bool{false, true} {
				var output bytes.Buffer
				runner := New()
				runner.Renderer = NewRenderer(&output, 120)
				runner.report = &ReportSummaryEvent{}
				runner.request = &workflow.RunRequest{}
				tag := &Event{TagID: "case", FailedCount: 1}
				runner.tags = []*Event{tag}
				record := func(target *Event) {
					for _, event := range candidates {
						if optimized {
							runner.recordSummaryEvent(target, event)
						} else {
							target.AddEvent(event)
						}
					}
				}
				record(tag)
				if withSubEvent {
					tag.subEvent = &Event{}
					record(tag.subEvent)
				}
				runner.reportTagSummary()
				summary, err := json.Marshal(runner.xUnitSummary)
				require.NoError(t, err)
				if !optimized {
					expectedOutput, expectedSummary = output.String(), summary
				} else {
					require.Equal(t, expectedOutput, output.String())
					require.Equal(t, string(expectedSummary), string(summary))
				}
			}
		})
	}
}

type summaryGCPayload struct{ bytes []byte }

// The helper frame must return before collection so local construction variables
// cannot keep the payload alive. Each fixture allocates only eight MiB.
func summaryGCRoot(optimized, assertion bool) (*Runner, weak.Pointer[summaryGCPayload]) {
	payload := &summaryGCPayload{bytes: make([]byte, 8<<20)}
	pointer := weak.Make(payload)
	var event msg.Event
	if assertion {
		request := msg.NewEvent(&validator.AssertRequest{Actual: payload, Expect: payload})
		event = msg.NewEventWithInit(&validator.AssertResponse{
			Validation: &assertly.Validation{PassedCount: 1},
		}, request)
	} else {
		event = msg.NewEvent(&workflow.InitEvent{State: map[string]interface{}{"payload": payload}})
	}
	runner := New()
	tag := &Event{}
	runner.tags = []*Event{tag}
	if optimized {
		runner.recordSummaryEvent(tag, event)
	} else {
		tag.AddEvent(event)
	}
	return runner, pointer
}

func TestSummaryRecorderReleasesOnlyUnneededPayloads(t *testing.T) {
	for _, assertion := range []bool{false, true} {
		t.Run(map[bool]string{false: "workflow-state", true: "passed-assertion-init"}[assertion], func(t *testing.T) {
			baseline, retained := summaryGCRoot(false, assertion)
			optimized, released := summaryGCRoot(true, assertion)
			runtime.GC()
			runtime.GC()
			require.NotNil(t, retained.Value(), "old retention must keep the same fixture alive")
			require.Nil(t, released.Value(), "summary-only retention must release the exclusive payload graph")
			runtime.KeepAlive(baseline)
			runtime.KeepAlive(optimized)
		})
	}
}

func TestSummaryRecorderPreservesDebugLogArtifacts(t *testing.T) {
	events := []msg.Event{
		msg.NewEvent(&workflow.InitEvent{State: map[string]interface{}{"value": "fixture"}}),
		msg.NewEvent(&httprunner.Request{Method: "GET", URL: "http://example.invalid/check"}),
		msg.NewEvent(&httprunner.Response{Code: 200, Body: "fixture"}),
		msg.NewEvent(&assertly.Validation{PassedCount: 1}),
	}
	var expected map[string]string
	for _, optimized := range []bool{false, true} {
		directory := t.TempDir()
		runner := New()
		tag := &Event{}
		logger := workflow.NewLogger(directory, func(event msg.Event) {
			if optimized {
				runner.recordSummaryEvent(tag, event)
			} else {
				tag.AddEvent(event)
			}
		})
		listener := logger.AsEventListener()
		for _, event := range events {
			listener(event)
		}
		actual := map[string]string{}
		err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			key, err := filepath.Rel(directory, path)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			actual[key] = string(content)
			return nil
		})
		require.NoError(t, err)
		require.NotEmpty(t, actual)
		if !optimized {
			expected = actual
		} else {
			require.Equal(t, expected, actual)
		}
	}
}

func TestSummaryRecorderPreservesCancellationErrorAndReset(t *testing.T) {
	var expectedOutput string
	for _, optimized := range []bool{false, true} {
		var output bytes.Buffer
		runner := New()
		runner.Renderer = NewRenderer(&output, 120)
		runner.report = &ReportSummaryEvent{}
		runner.request = &workflow.RunRequest{}
		tag := &Event{}
		for _, value := range []interface{}{msg.NewErrorEvent(context.Canceled.Error()), &msg.ResetError{}} {
			event := msg.NewEvent(value)
			runner.processEvent(event, WildcardFilter())
			if optimized {
				runner.recordSummaryEvent(tag, event)
			} else {
				tag.AddEvent(event)
			}
			if _, reset := value.(*msg.ResetError); !reset {
				require.EqualError(t, runner.err, context.Canceled.Error())
				require.True(t, runner.report.Error)
			} else {
				require.NoError(t, runner.err)
				require.False(t, runner.report.Error)
			}
			require.Same(t, event, tag.Events[len(tag.Events)-1])
		}
		if !optimized {
			expectedOutput = output.String()
		} else {
			require.Equal(t, expectedOutput, output.String())
		}
	}
}

func TestSummaryRecorderPreservesNonzeroFailureOffsets(t *testing.T) {
	runner := New()
	first := &httprunner.Request{Method: "GET", URL: "http://example.invalid/first"}
	second := &httprunner.Request{Method: "GET", URL: "http://example.invalid/second"}
	response := &httprunner.Response{Code: 200, Body: "second"}
	candidates := []msg.Event{
		msg.NewEvent(&workflow.InitEvent{}),
		msg.NewEvent(first),
		msg.NewEvent(&httprunner.Response{Code: 200, Body: "first"}),
		msg.NewEvent(&assertly.Validation{PassedCount: 1}),
		msg.NewEvent(&model.ActivityEndEvent{}),
		msg.NewEvent(second),
		msg.NewEvent(response),
		msg.NewEvent(&assertly.Validation{FailedCount: 1}),
	}
	tag := &Event{}
	for _, event := range candidates {
		runner.recordSummaryEvent(tag, event)
	}
	expected := runner.extractRunnerLogs(candidates, 4, 6)
	actual := runner.extractRunnerLogs(tag.Events, 4, 6)
	key := candidates[5].Package()
	require.Len(t, expected[key], 1)
	require.Len(t, actual[key], 1)
	require.Same(t, second, actual[key][0].In.Value())
	require.Same(t, response, actual[key][0].Out.Value())
	require.Same(t, expected[key][0].In.Value(), actual[key][0].In.Value())
	require.Same(t, expected[key][0].Out.Value(), actual[key][0].Out.Value())
	require.Equal(t, expected[key][0].JSONOutput, actual[key][0].JSONOutput)
}

type zeroTimestampSummaryEvent struct{ msg.Event }

func (e *zeroTimestampSummaryEvent) Timestamp() time.Time { return time.Time{} }

func TestCLIListenerPreservesCustomZeroFirstTimestamp(t *testing.T) {
	runner := New()
	runner.Renderer = NewRenderer(io.Discard, 120)
	runner.context = runner.manager.NewContext(nil)
	defer runner.context.Close()
	runner.report = &ReportSummaryEvent{}
	listener := runner.AsListener()
	first := &zeroTimestampSummaryEvent{Event: msg.NewEvent(1)}
	second := msg.NewEvent(2)
	listener(first)
	listener(second)
	expected := int(second.Timestamp().UnixNano()-first.Timestamp().UnixNano()) / int(time.Millisecond)
	require.Equal(t, expected, runner.report.ElapsedMs)
	require.Same(t, first, runner.tags[0].Events[0])
}
