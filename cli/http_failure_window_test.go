package cli

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/assertly"
	"github.com/viant/endly/model/msg"
	runnerhttp "github.com/viant/endly/service/testing/runner/http"
	"github.com/viant/endly/service/workflow"
)

type diagnosticTimedEvent struct {
	msg.Event
	at time.Time
}

func (e diagnosticTimedEvent) Timestamp() time.Time { return e.at }

func TestHTTPFailureReportUsesActionWindow(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, failedIndex := range []int{0, 1} {
			t.Run(fmt.Sprintf("nested=%v/index=%d", nested, failedIndex), func(t *testing.T) {
				var output bytes.Buffer
				r := New()
				r.request = &workflow.RunRequest{}
				r.Renderer = NewRenderer(&output, 160)
				r.report = &ReportSummaryEvent{}
				tag := &Event{TagID: "case015"}
				r.AddTag(tag)
				child := &Event{}
				if nested {
					tag.subEvent = child
				}
				tick := 0
				event := func(value any) msg.Event {
					tick++
					return diagnosticTimedEvent{Event: msg.NewEvent(value), at: time.Unix(0, int64(tick))}
				}
				addTrip := func(name string, code int) {
					events := []msg.Event{event(&runnerhttp.Request{Method: "PATCH", URL: "http://example.invalid/" + name, Body: "request-" + name}), event(&runnerhttp.Response{Code: code, Body: "response-" + name})}
					if nested {
						child.Events = append(child.Events, events...)
					} else {
						tag.Events = append(tag.Events, events...)
					}
				}
				addValidation := func(failed bool, index, passed int) {
					v := &assertly.Validation{TagID: tag.TagID, PassedCount: passed}
					if failed {
						v.AddFailure(assertly.NewFailure("HTTP.responses", fmt.Sprintf("[HTTP.responses]:Responses[%d].Code", index), "value", 200, 400))
					}
					tag.PassedCount += v.PassedCount
					tag.FailedCount += v.FailedCount
					tag.Events = append(tag.Events, event(v))
				}
				addTrip("past-flight", 400)
				addValidation(false, 0, 1)
				addTrip("overlap-flight", 400)
				addValidation(false, 0, 1)
				if failedIndex == 1 {
					addTrip("same-action-passed", 200)
				}
				addTrip("locked-flight", 400)
				addValidation(true, failedIndex, failedIndex)
				// A later successful response must not overwrite the failed action's pair.
				addTrip("later-success", 200)
				addValidation(false, 0, 1)
				beforePassed, beforeFailed := tag.PassedCount, tag.FailedCount
				r.reportTagSummary()
				rendered := output.String()
				require.Contains(t, rendered, "request-locked-flight")
				require.Contains(t, rendered, "response-locked-flight")
				for _, name := range []string{"past-flight", "overlap-flight", "same-action-passed", "later-success"} {
					require.NotContains(t, rendered, "request-"+name)
					require.NotContains(t, rendered, "response-"+name)
				}
				require.Equal(t, beforePassed, tag.PassedCount)
				require.Equal(t, beforeFailed, tag.FailedCount)
				require.Equal(t, 1, tag.FailedCount)
				require.Len(t, r.xUnitSummary.TestCase, 1)
				require.Contains(t, r.xUnitSummary.TestCase[0].Sysout, "response-locked-flight")
				require.NotContains(t, r.xUnitSummary.TestCase[0].Sysout, "later-success")
			})
		}
	}
}

func TestHTTPFailureLogExtractionStopsAtActionEnd(t *testing.T) {
	r := New()
	input := msg.NewEvent(&runnerhttp.Request{Method: "GET", URL: "http://example.invalid/failed"})
	failed := msg.NewEvent(&runnerhttp.Response{Code: 400, Body: "failed-response"})
	future := msg.NewEvent(&runnerhttp.Response{Code: 200, Body: "future-response"})
	logs := r.extractRunnerLogs([]msg.Event{input, failed, future}, 0, 1)
	require.Len(t, logs[input.Package()], 1)
	require.Same(t, failed, logs[input.Package()][0].Out)
	require.Contains(t, logs[input.Package()][0].JSONOutput, "failed-response")
	require.NotContains(t, logs[input.Package()][0].JSONOutput, "future-response")
}
