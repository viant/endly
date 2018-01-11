package endly

//Ec2CallRequest represents a aws EC2 run request to execute method on ec2 client with provided input.
type Ec2CallRequest struct {
	Credential string
	Method     string
	Input      interface{}
}

//Ec2CallResponse represents EC2 run response
type Ec2CallResponse struct {
	Error    string
	Response interface{}
}

//AsEc2Call converts request as Ec2 call
func (r *Ec2CallRequest) AsEc2Call() *Ec2Call {
	var result = &Ec2Call{
		Method:     r.Method,
		Parameters: make([]interface{}, 0),
	}
	if r.Input != nil {
		result.Parameters = append(result.Parameters, r.Input)
	}
	return result
}
