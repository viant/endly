package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//DockerRunRequest represents a docker run request
type DockerRunRequest struct {
	Target      *url.Resource     `required:"true" description:"host with docker service"`                //target host
	Name        string            `description:"container name to inspect, if empty it uses target.Name"` //docker container name
	Credentials map[string]string `description:"map of secret key to obfuscate terminal output with corresponding filename storing credential compatible with github.com/viant/toolbox/cred/config.go"`
	Image       string            `required:"true" description:"container image to run" example:"mysql:5.6"`
	Port        string            `description:"publish a container’s port(s) to the host, docker -p option"`
	Env         map[string]string `description:"set docker container an environment variable, docker -e KEY=VAL  option"`
	Mount       map[string]string `description:"bind mount a volume, docker -v option"`
	MappedPort  map[string]string `description:"publish a container’s port(s) to the host, docker -p option"`
	Params      map[string]string `description:"other free form docker parameters"`
	Workdir     string            `description:"working directory inside the container, docker -w option"`
}

//DockerRunResponse represents a docker run response
type DockerRunResponse struct {
	*DockerContainerInfo
}

//Validate checks if request is valid
func (r *DockerRunRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target was nil")
	}
	if r.Name == "" {
		return fmt.Errorf("container  name was empty for %v", r.Target.URL)
	}
	if r.Image == "" {
		return fmt.Errorf("image was empty for %v", r.Target.URL)
	}
	return nil
}
