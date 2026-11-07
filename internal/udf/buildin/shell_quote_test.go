package buildin

import (
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
)

func TestShellQuote(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a POSIX shell")
	}
	for _, input := range []string{"", "a path's name", "$HOME ${PATH} `printf wrong`; $(printf wrong)", "[]", `{"k":"v"}`} {
		state := data.Map{"HOME": "expanded home", "PATH": "expanded path", "args": map[string]interface{}{"value": input}}
		quoted, err := ShellQuote("args.value", state)
		require.NoError(t, err)
		command := state.ExpandAsText("printf '%s' " + quoted.(string))
		output, err := exec.Command("/bin/sh", "-c", command).Output()
		require.NoError(t, err)
		require.Equal(t, input, string(output))
	}
	_, err := ShellQuote(1, nil)
	require.Error(t, err)
	_, err = ShellQuote("missing", nil)
	require.Error(t, err)
	_, err = ShellQuote("invalid", data.Map{"invalid": "NUL\x00"})
	require.Error(t, err)
}
