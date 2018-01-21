package endly

import (
	"github.com/viant/toolbox/url"
)

//DockerContainerStatusRequest represents a docker check container status request
type DockerContainerStatusRequest struct {
	Target *url.Resource
	Names  string
	Image  string
}

//DockerContainerStatusResponse represents a docker container check response
type DockerContainerStatusResponse struct {
	Containers []*DockerContainerInfo
}

//DockerContainerStartRequest represents a docker container start request.
type DockerContainerStartRequest struct {
	Target *url.Resource
}

//DockerContainerRemoveRequest represents a docker remove container request
type DockerContainerRemoveRequest struct {
	Target *url.Resource
}

//DockerContainerRemoveResponse represents a docker remove container response
type DockerContainerRemoveResponse struct {
	Stdout string
}

//DockerContainerStopRequest represents a docker stop container request.
type DockerContainerStopRequest struct {
	Target *url.Resource
}

//DockerContainerRunCommandRequest represents a docker run container command.
type DockerContainerRunCommandRequest struct {
	Target             *url.Resource
	Credentials        map[string]string
	Interactive        bool
	AllocateTerminal   bool
	RunInTheBackground bool
	Command            string
}

//DockerContainerRunCommandResponse represents a docker run command  response
type DockerContainerRunCommandResponse struct {
	Stdout string
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

//DockerContainerLogsRequest represents docker runner container logs to take stdout
type DockerContainerLogsRequest struct {
	Target *url.Resource
}

//DockerContainerLogsResponse represents docker container logs response
type DockerContainerLogsResponse struct {
	Stdout string
}
