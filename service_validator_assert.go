package endly

//ValidatorAssertRequest represent assert request
type ValidatorAssertRequest struct {
	Name        string
	Description string
	Actual      interface{} //actual data structure
	Expected    interface{} //expecte data structure
}
