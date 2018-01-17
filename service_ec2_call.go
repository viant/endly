package endly

//EC2CallRequest represents a aws EC2 run request to execute method on ec2 client with provided input.
type EC2CallRequest struct {
	Credential string
	Method     string
	Input      interface{}
}

//EC2CallResponse represents EC2 run response
type EC2CallResponse struct {
	Error    string
	Response interface{}
}

//AsEc2Call converts request as Ec2 call
func (r *EC2CallRequest) AsEc2Call() *EC2Call {
	var result = &EC2Call{
		Method:     r.Method,
		Parameters: make([]interface{}, 0),
	}
	if r.Input != nil {
		result.Parameters = append(result.Parameters, r.Input)
	}
	return result
}
