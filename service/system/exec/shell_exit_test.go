package exec

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
)

func TestRunDoesNotSucceedAfterShellExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell")
	}
	for _, checked := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchecked", true: "checked"}[checked], func(t *testing.T) {
			ctx := endly.New().NewContext(nil)
			t.Cleanup(ctx.Close)
			service := New()
			exit := NewRunRequest(nil, false, "exit 1")
			exit.TimeoutMs = 200
			exit.StrictCompletion = true
			exit.CheckError = checked
			result := service.Run(ctx, exit)
			require.Equal(t, "error", result.Status)
			require.Error(t, result.Err)
			next := NewRunRequest(nil, false, "printf should-execute")
			next.TimeoutMs = 200
			next.CheckError = checked
			result = service.Run(ctx, next)
			require.Equal(t, "error", result.Status, "a dead shell must not silently accept a later command")
			require.Error(t, result.Err)
		})
	}
}

func TestRunPreservesSilentCompletionAndUncheckedExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell")
	}
	ctx := endly.New().NewContext(nil)
	t.Cleanup(ctx.Close)
	service := New()
	for _, tc := range []struct {
		command string
		check   bool
		output  string
	}{
		{"true", true, ""},
		{"export ENDLY_SHELL_EXIT_TEST=preserved", true, ""},
		{"false", false, ""},
		{"printf \"$ENDLY_SHELL_EXIT_TEST\"", true, "preserved"},
	} {
		request := NewRunRequest(nil, false, tc.command)
		request.TimeoutMs = 500
		request.CheckError = tc.check
		result := service.Run(ctx, request)
		require.Equal(t, "ok", result.Status, tc.command)
		require.NoError(t, result.Err, tc.command)
		require.Equal(t, tc.output, result.Response.(*RunResponse).Output)
	}
}
