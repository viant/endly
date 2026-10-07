package exec

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/gosh/runner"
)

func TestRunPropagatesIncompleteShellCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell and sleep")
	}
	for _, checkExitCode := range []bool{false, true} {
		name := "runner-error"
		if checkExitCode {
			name = "checked-exit-code"
		}
		t.Run(name, func(t *testing.T) {
			ctx := endly.New().NewContext(nil)
			t.Cleanup(ctx.Close)
			request := NewRunRequest(nil, false, "printf partial; sleep 0.2")
			request.TimeoutMs = 20
			request.CheckError = checkExitCode
			result := New().Run(ctx, request)
			require.Equal(t, "error", result.Status, "an incomplete command must never report success")
			require.Error(t, result.Err)
			if checkExitCode {
				require.Contains(t, result.Error, "exit code: -1")
			} else {
				require.Contains(t, result.Error, runner.ErrTimeout.Error())
			}
		})
	}
}
