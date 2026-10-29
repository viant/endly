package exec

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/gosh/runner"
)

func TestCheckedRunPreservesCompletionFailureCause(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell")
	}
	for _, tc := range []struct {
		name, command string
		cause         error
	}{
		{"timeout", "sleep 1", runner.ErrTimeout},
		{"shell exit", "exit 7", runner.ErrIncompleteCommand},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := endly.New().NewContext(nil)
			t.Cleanup(ctx.Close)
			request := NewRunRequest(nil, false, tc.command)
			request.TimeoutMs = 100
			request.StrictCompletion = true
			request.CheckError = true
			result := New().Run(ctx, request)
			require.Error(t, result.Err)
			require.ErrorIs(t, result.Err, tc.cause, "the runner failure must remain distinguishable from a command exit status")
		})
	}
}
