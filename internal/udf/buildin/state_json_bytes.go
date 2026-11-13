package buildin

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/viant/toolbox/data"
)

// StateJSONBytes serializes one stored value without interpreting its contents
// as workflow expressions. The argument is a literal state key/path, not data.
func StateJSONBytes(source interface{}, state data.Map) (interface{}, error) {
	key, ok := source.(string)
	if !ok || strings.TrimSpace(key) == "" || strings.Contains(key, "$") || strings.HasPrefix(key, "<-") || strings.HasPrefix(key, "++") || strings.HasSuffix(key, "++") {
		return nil, errors.New("StateJSONBytes requires a nonempty literal state key/path")
	}
	value, found := state.GetValue(key)
	if !found {
		return nil, errors.New("StateJSONBytes source was not found")
	}
	// Match AsJSON's JSON formatting while retaining bytes for literal storage.
	// Standard JSON scalars, including present nil, false and zero, stay distinct.
	encoded, err := json.MarshalIndent(value, "", "\t")
	if err != nil {
		// Custom marshalers can embed private source data in their errors. The
		// expression engine logs UDF errors, so never wrap or print that cause.
		return nil, errors.New("StateJSONBytes serialization failed")
	}
	return encoded, nil
}
