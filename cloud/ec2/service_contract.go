package ec2

//Call represents ec2 call.
type Call struct {
	Method     string        `required:"true" description:"ec2 client method name"`
	Parameters []interface{} `required:"true" description:"ec2 client method parameters"`
}

//CallRequest represents a aws EC2 run request to execute method on ec2 client with provided input.
type CallRequest struct {
	Credentials string      `required:"true" description:"ec2 credentials file see more at: github.com/viant/toolbox/cred/config.go"`
	Method      string      `required:"true" description:"ec2 client method name"`
	Input       interface{} `required:"true" description:"ec2 client method input/request"`
}

//CallResponse represents EC2 run response
type CallResponse interface{}

//AsCall converts request as Ec2 call
func (r *CallRequest) AsCall() *Call {
	var result = &Call{
		Method:     r.Method,
		Parameters: make([]interface{}, 0),
	}
	if r.Input != nil {
		result.Parameters = append(result.Parameters, r.Input)
	}
	return result
}
