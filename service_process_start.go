package endly

import "github.com/viant/toolbox/url"

//ProcessStartRequest represents a start request
type ProcessStartRequest struct {
	Command         string
	Target          *url.Resource
	Options         *ExecutionOptions
	Directory       string
	Arguments       []string
	ImmuneToHangups bool //start process as nohup
}

//ProcessStartResponse represents a start response
type ProcessStartResponse struct {
	Command string
	Info    []*ProcessInfo
}
