package process

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStopRejectsProcessGroupAndMissingSelectors(t *testing.T) {
	for _, input := range []StopRequest{{}, {Pid: -1}, {Pid: 0, Input: " "}, {Pid: 42, Signal: "TERM; true"}} {
		require.Error(t, input.Validate())
	}
	for _, input := range []StopRequest{{Pid: 42}, {Input: "/tmp/fixture"}, {Pid: 42, Signal: "TERM"}, {Pid: 42, Signal: "9"}} {
		require.NoError(t, input.Validate())
	}
}
