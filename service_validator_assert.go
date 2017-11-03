package endly

//ValidatorAssertRequest represent assert request
type ValidatorAssertRequest struct {
	TagId       string
	Description string
	Actual      interface{} //actual data structure
	Expected    interface{} //expecte data structure
}
