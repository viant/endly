package endly

import (
	"github.com/viant/toolbox/url"
	"github.com/pkg/errors"
)

//DockerContainerStatusRequest represents a docker check container status request
type DockerContainerStatusRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"` //target host
	Names  string
	Image  string
}

//DockerContainerStatusResponse represents a docker container check response
type DockerContainerStatusResponse struct {
	Containers []*DockerContainerInfo
}


//DockerContainerBaseRequest represents container base request
type DockerContainerBaseRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"`                //target host
	Name   string        `description:"container name to inspect, if empty it uses target.Name"` //docker container name
}




func (r *DockerContainerBaseRequest) Init() error {
	if r == nil || r.Target == nil {
		return nil
	}
	if r.Name != "" {
		return nil
	}
	r.Name = r.Target.Name
	return nil
}


func (r *DockerContainerBaseRequest) Validate() error {
	if r == nil {
		return errors.New("base container request was nil")
	}
	if r.Target == nil {
		return errors.New("target docker resource was empty")
	}
	if r.Name == "" {
		return errors.New("docker instance name was empty")
	}
	return nil
}


//DockerContainerStartRequest represents a docker container start request.
type DockerContainerStartRequest struct {
	*DockerContainerBaseRequest
}





//DockerContainerStartResponse represents a docker container start response
type DockerContainerStartResponse struct {
	*DockerContainerInfo
}


//DockerContainerRemoveRequest represents a docker remove container request
type DockerContainerRemoveRequest struct {
	*DockerContainerBaseRequest
}

//DockerContainerRemoveResponse represents a docker remove container response
type DockerContainerRemoveResponse struct {
	Stdout string
}

//DockerContainerStopRequest represents a docker stop container request.
type DockerContainerStopRequest struct {
	*DockerContainerBaseRequest
}


//DockerContainerStopResponse represents a docker stop container response.
type DockerContainerStopResponse struct {
	*DockerContainerInfo
}


//DockerContainerRunRequest represents a docker run container command.
type DockerContainerRunRequest struct {
	*DockerContainerBaseRequest
	Credentials        map[string]string
	Interactive        bool
	AllocateTerminal   bool
	RunInTheBackground bool
	Command            string
}



//DockerInspectRequest represents a docker inspect request, target name refers to container name
type DockerInspectRequest struct {
	*DockerContainerBaseRequest
}

//DockerInspectResponse represents a docker inspect request
type DockerInspectResponse struct {
	Stdout string
	Info   interface{} //you can extract any instance default, for instance to get Ip you can use Info[0].NetworkSettings.IPAddress in the variable action post from key
}

//DockerContainerRunResponse represents a docker run command  response
type DockerContainerRunResponse struct {
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
	*DockerContainerBaseRequest
}

//DockerContainerLogsResponse represents docker container logs response
type DockerContainerLogsResponse struct {
	Stdout string
}
