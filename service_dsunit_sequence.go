package endly

//DsUnitTableSequenceRequest represents a sequence request for specified tables.
type DsUnitTableSequenceRequest struct {
	Datastore string
	Tables    []string
}

//DsUnitTableSequenceResponse represents a current database sequences.
type DsUnitTableSequenceResponse struct {
	Sequences map[string]int
}
