package endly

import (
	"errors"
	"fmt"
)

//DsUnitTableSequenceRequest represents a sequence request for specified tables.
type DsUnitTableSequenceRequest struct {
	Datastore string
	Tables    []string
}

//DsUnitTableSequenceResponse represents a current database sequences.
type DsUnitTableSequenceResponse struct {
	Sequences map[string]int
}


//Validate validate sequence request
func (r *DsUnitTableSequenceRequest) Validate() error {
	if r.Datastore == "" {
		return errors.New("Datastore was empty")
	}
	if len(r.Tables) == 0 {
		return fmt.Errorf("Tables was empty on %v", r.Datastore)
	}
	return nil
}