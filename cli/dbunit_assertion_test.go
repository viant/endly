package cli

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	core "github.com/viant/dbunit"
	"github.com/viant/endly/model/msg"
	dbunit "github.com/viant/endly/service/testing/dbunit"
	"github.com/viant/endly/service/workflow"
)

func TestDBUnitMixedAssertionCounts(t *testing.T) {
	runner := New()
	runner.Renderer = NewRenderer(io.Discard, 120)
	runner.context = runner.manager.NewContext(nil)
	defer runner.context.Close()
	runner.report = &ReportSummaryEvent{}
	runner.filter = WildcardFilter()
	runner.request = &workflow.RunRequest{}
	response := &dbunit.ExpectResponse{Report: &core.Report{Passed: 2, Failed: 1, Mismatches: []core.Mismatch{{Path: "users[0].name", Category: "value", Expected: "alice", Actual: "wrong", Message: "name differs"}}}}
	require.True(t, runner.processAssertable(msg.NewEvent(response)))
	require.Equal(t, 2, runner.indexedTag[""].PassedCount)
	require.Equal(t, 1, runner.indexedTag[""].FailedCount)
	require.True(t, runner.hasValidationFailures)
}
