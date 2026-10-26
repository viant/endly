package webdriver

import (
	"encoding/json"
	"errors"
)

// CDP's public result envelope is JSON, rather than an opaque byte slice.
// Decode at this boundary so native workflow paths see its object fields.
// Other reflected method results retain their original Go types.
func normalizeCDPResult(results []interface{}) error {
	if len(results) == 0 {
		return errors.New("CDP command returned no result envelope")
	}
	raw, ok := results[0].(json.RawMessage)
	if !ok {
		return nil
	}
	var object map[string]interface{}
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		// Never expose response bytes, including cookies or driver diagnostics.
		return errors.New("CDP command returned an invalid object envelope")
	}
	results[0] = object
	return nil
}
