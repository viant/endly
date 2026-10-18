package exec

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"runtime"
	"testing"
)

func TestLocalShellSurvivesCompletedOperation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	ctx := endly.New().NewContext(nil)
	t.Cleanup(ctx.Close)
	first, cancel := context.WithCancel(context.Background())
	ctx.SetBackground(first)
	service := New()
	result := service.Run(ctx, NewRunRequest(nil, false, "export ENDLY_TEST_MARKER=retained"))
	require.NoError(t, result.Err)
	cancel()
	ctx.SetBackground(context.Background())
	result = service.Run(ctx, NewRunRequest(nil, false, "printf \"$ENDLY_TEST_MARKER\""))
	require.NoError(t, result.Err)
	require.Equal(t, "retained", result.Response.(*RunResponse).Output)
}
func TestLocalShellQuietCommandCompletes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	ctx := endly.New().NewContext(nil)
	t.Cleanup(ctx.Close)
	result := New().Run(ctx, NewRunRequest(nil, false, "true"))
	require.NoError(t, result.Err)
}

func TestStrictShellCompletion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	for _, tc := range []struct {
		name, command string
		wantError     bool
	}{
		{"quiet", "true", false},
		{"no trailing newline", "printf complete", false},
		{"shell exits before status", "exit 0", true},
		{"continuous output cannot extend timeout", "while :; do printf output; sleep 0.02; done", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := endly.New().NewContext(nil)
			t.Cleanup(ctx.Close)
			request := NewRunRequest(nil, false, tc.command)
			request.StrictCompletion = true
			request.TimeoutMs = 200
			result := New().Run(ctx, request)
			if tc.wantError {
				require.Error(t, result.Err)
			} else {
				require.NoError(t, result.Err)
			}
		})
	}
}
