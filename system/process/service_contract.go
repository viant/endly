package process

import (
	"github.com/viant/endly/system/exec"
	"github.com/viant/toolbox/url"
)

//StartRequest represents a start request
type StartRequest struct {
	Target          *url.Resource `required:"true" description:"host where process will be started"`
	Command         string        `required:"true" description:"command to start process"`
	Options         *exec.Options
	Directory       string
	Arguments       []string
	AsSuperUser     bool
	ImmuneToHangups bool `description:"start process as nohup"`
}

//StartResponse represents a start response
type StartResponse struct {
	Command string
	Info    []*Info
}

//StatusRequest represents a status check request
type StatusRequest struct {
	Target       *url.Resource
	Command      string `description:"command identifying a process, by default it is check that command is ps -ef suffix or is terminated by space / or dot "`
	ExactCommand bool   `description:"if this flag set do not try detect actual command but return all processes matched by command"`
}

//StatusResponse represents a status check response
type StatusResponse struct {
	Processes []*Info
	Pid       int
}

//Info represents process info
type Info struct {
	Name      string
	Pid       int
	Command   string
	Arguments []string
	Stdin     string
	Stdout    string
}

//StopRequest represents a stop request
type StopRequest struct {
	Target *url.Resource
	Pid    int
}

//StopResponse represents a stop response
type StopResponse struct {
	Stdout string
}

//StopAllRequest represents a stop all processes matching provided name request
type StopAllRequest struct {
	Target *url.Resource
	Input  string
}

//StopAllResponse represents a stop all response
type StopAllResponse struct {
	Stdout string
}
