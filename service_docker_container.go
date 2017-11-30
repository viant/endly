package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerContainerStatusRequest represents a docker check container status request
type DockerContainerStatusRequest struct {
	Target  *url.Resource
	SysPath []string
	Names   string
	Image   string
}

//DockerContainerStatusResponse represents a docker container check response
type DockerContainerStatusResponse struct {
	SysPath    []string
	Containers []*DockerContainerInfo
}

//DockerContainerStartRequest represents a docker container start request.
type DockerContainerStartRequest struct {
	SysPath []string
	Target  *url.Resource
}

//DockerContainerRemoveRequest represents a docker remove container request
type DockerContainerRemoveRequest struct {
	SysPath []string
	Target  *url.Resource
}

//DockerContainerStopRequest represents a docker stop container request.
type DockerContainerStopRequest struct {
	SysPath []string
	Target  *url.Resource
}

//DockerContainerCommandRequest represents a docker run command in the container.
type DockerContainerCommandRequest struct {
	Target             *url.Resource
	SysPath            []string
	Credentials        map[string]string
	Interactive        bool
	AllocateTerminal   bool
	RunInTheBackground bool
	Command            string
}

//DockerContainerInfo represents a docker container info
type DockerContainerInfo struct {
	ContainerID string
	Image       string
	Command     string
	Status      string
	Port        string
	Names       string
}
