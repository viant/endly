package buildin

import (
	"fmt"
	"strings"

	"github.com/viant/endly/internal/util"
	"github.com/viant/toolbox/data"
)

// ShellQuote renders a stored string as one literal POSIX shell argument.
func ShellQuote(source interface{}, state data.Map) (interface{}, error) {
	key, ok := source.(string)
	if !ok || strings.TrimSpace(key) == "" || strings.Contains(key, "$") {
		return nil, fmt.Errorf("ShellQuote requires a nonempty literal state key/path")
	}
	stored, found := state.LookupValue(key)
	if !found {
		return nil, fmt.Errorf("ShellQuote source was not found")
	}
	value, ok := stored.(string)
	if !ok {
		return nil, fmt.Errorf("ShellQuote source must be a string, got %T", stored)
	}
	if strings.ContainsRune(value, 0) {
		return nil, fmt.Errorf("ShellQuote does not accept NUL bytes")
	}
	// Exec expands the authored command again before passing it to the shell.
	// Place each dollar in a separate quoted fragment to keep it literal there.
	return strings.ReplaceAll(util.QuoteShellArgument(value), "$", "''$''"), nil
}
