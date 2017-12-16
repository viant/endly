package endly

//ValidatorAssertRequest represent assert request
type ValidatorAssertRequest struct {
	TagID       string
	Description string
	Actual      interface{} //actual data structure
	Expected    interface{} //expecte data structure
}
