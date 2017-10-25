package endly

import "github.com/viant/toolbox/url"

//ProcessStartRequest represents a start request
type ProcessStartRequest struct {
	Name          string
	Target        *url.Resource
	Options       *ExecutionOptions
	SystemService bool
	Directory     string
	Command       string
	Arguments     []string
}

//ProcessStartResponse represents a start response
type ProcessStartResponse struct {
	Command string
	Info    []*ProcessInfo
}
