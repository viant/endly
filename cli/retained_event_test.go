package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/assertly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
	runnerhttp "github.com/viant/endly/service/testing/runner/http"
	"github.com/viant/endly/service/workflow"
)

func TestRetainedWorkflowInitDoesNotOwnStateSnapshot(t *testing.T) {
	for _, logging := range []bool{false, true} {
		payload := &workflow.InitEvent{Tasks: "regression", State: map[string]interface{}{"fixture": make([]byte, 1024)}}
		original := msg.NewEvent(payload)
		original.SetLoggable(logging)
		r := New()
		r.Renderer = NewRenderer(io.Discard, 160)
		activity := &model.Activity{MetaTag: &model.MetaTag{TagID: "memory-regression"}}
		r.Push(activity)
		require.NoError(t, r.reportEvent(nil, original, nil))
		require.Len(t, r.EventTag().Events, 1, "retain the event boundary")
		retained := r.EventTag().Events[0]
		require.Equal(t, original.Timestamp(), retained.Timestamp())
		require.Equal(t, original.Type(), retained.Type())
		require.Equal(t, original.Package(), retained.Package())
		require.Equal(t, logging, retained.IsLoggable())
		require.Equal(t, "regression", retained.Value().(*workflow.InitEvent).Tasks)
		require.Empty(t, retained.Value().(*workflow.InitEvent).State)
		require.NotEmpty(t, payload.State, "other listeners must still receive the original snapshot")
	}
}

func TestRetainedFailureEvidenceIsUnchanged(t *testing.T) {
	for _, payload := range []interface{}{
		&runnerhttp.Request{Method: "GET", URL: "http://example.invalid/test"},
		&runnerhttp.Response{Code: 400, Body: "failure evidence"},
		&assertly.Validation{FailedCount: 1},
	} {
		event := msg.NewEvent(payload)
		require.Same(t, event, retainReportEvent(event))
	}
}
