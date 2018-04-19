package ec2

import (
	"fmt"
	"github.com/viant/endly/util"
)

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

//Init initialise request
func (r *CallRequest) Init() error {
	if r.Input == nil {
		return nil
	}

	if input, err := util.NormalizeMap(r.Input, true); err == nil {
		r.Input = input
	}
	return nil
}

//Init initialise request
func (r *CallRequest) Validate() error {
	if r.Method == "" {
		return fmt.Errorf("EC2 method was empty")
	}
	if r.Input == nil {
		return fmt.Errorf("EC2 %v Input was empty", r.Method)
	}
	if r.Credentials == "" {
		return fmt.Errorf("EC2 Credentials were empty")
	}
	return nil
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
