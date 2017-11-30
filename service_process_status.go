package endly

import "github.com/viant/toolbox/url"

//ProcessStatusRequest represents a status check request
type ProcessStatusRequest struct {
	Target  *url.Resource
	Command string
}

//ProcessStatusResponse represents a status check response
type ProcessStatusResponse struct {
	Processes []*ProcessInfo
	Pid       int
}

//ProcessInfo represents process info
type ProcessInfo struct {
	Name      string
	Pid       int
	Command   string
	Arguments []string
	Stdin     string
	Stdout    string
}
