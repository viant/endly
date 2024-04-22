package process

import (
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/system/exec"
)

// StartRequest represents a start request
type StartRequest struct {
	Target  *location.Resource `required:"true" description:"host where process will be started"`
	Command string             `required:"true" description:"command to start process"`
	*exec.Options
	Arguments       []string
	AsSuperUser     bool
	ImmuneToHangups bool `description:"start process as nohup"`
	Watch           bool `description:"watch command output, work with nohup mode"`
}

// NewStartRequestFromURL creates a new request from URL
func NewStartRequestFromURL(URL string) (*StartRequest, error) {
	var request = &StartRequest{}
	resource := location.NewResource(URL)
	return request, resource.Decode(request)
}

// StartResponse represents a start response
type StartResponse struct {
	Command string
	Info    []*Info
	Pid     int
	Stdout  string
}

// StatusRequest represents a status check request
type StatusRequest struct {
	Target       *location.Resource
	Command      string `description:"command identifying a process, by default it is check that command is ps -ef suffix or is terminated by space / or dot "`
	ExactCommand bool   `description:"if this flag set do not try detect actual command but return all processes matched by command"`
}

// StatusResponse represents a status check response
type StatusResponse struct {
	Processes []*Info
	Pid       int
}

// Info represents process info
type Info struct {
	Name      string
	Pid       int
	Command   string
	Arguments []string
	Stdin     string
	Stdout    string
}

// StopRequest represents a stop request
type StopRequest struct {
	Target *location.Resource
	Pid    int
	Input  string `description:"if specified, matches all process Pid to stop"`
}

// StopResponse represents a stop response
type StopResponse struct {
	Stdout string
}

func (r *StartRequest) Init() error {
	r.Target = exec.GetServiceTarget(r.Target)
	return nil
}

// NewStopRequest creates a stop request
func NewStopRequest(pid int, target *location.Resource) *StopRequest {
	return &StopRequest{Target: target, Pid: pid}
}

func NewStatusRequest(command string, target *location.Resource) *StatusRequest {
	return &StatusRequest{Target: target, Command: command}
}
