package docker

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/url"
)

//DockerBuildRequest represents docker build request
type DockerBuildRequest struct {
	Target    *url.Resource     `required:"true" description:"host with docker service"` //target host
	Tag       *DockerTag        `required:"true" description:"build docker tag"`
	Path      string            `required:"true" description:"docker build source path"`
	Arguments map[string]string `description:"docker build command line arguments, see more: https://docs.docker.com/engine/reference/commandline/build/#description "` //https://docs.docker.com/engine/reference/commandline/build/#description
}

//DockerBuildResponse represents docker build response
type DockerBuildResponse struct {
	Stdout string
}

//Init initialises default values
func (r *DockerBuildRequest) Init() {
	if len(r.Arguments) == 0 && r.Tag != nil {
		r.Arguments = make(map[string]string)
	}
	if r.Tag != nil {
		r.Arguments["-t"] = r.Tag.String()
	}
}

//Validate check if request is valid
func (r *DockerBuildRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was nil")
	}
	if r.Path == "" {
		return errors.New("path was empty was nil")
	}
	if r.Tag != nil {
		err := r.Tag.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

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

//Init initializes request
func (r *DockerContainerBaseRequest) Init() error {
	if r == nil || r.Target == nil {
		return nil
	}
	if r.Name != "" {
		return nil
	}
	return nil
}

//Validate checks if request is valid
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

//DockerImagesRequest represents docker check image request
type DockerImagesRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Repository string        `required:"true"`
	Tag        string        `required:"true"`
}

//DockerImagesResponse represents a docker check image response
type DockerImagesResponse struct {
	Images []*DockerImageInfo
}

//DockerImageInfo represents docker image info
type DockerImageInfo struct {
	Repository string
	Tag        string
	ImageID    string
	Size       int
}

//DockerLoginRequest represents a docker pull request
type DockerLoginRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Credential string        `required:"true" description:"credential path"`
	Repository string        `required:"true" description:"repository url"`
}

//DockerLoginResponse represents a docker pull request
type DockerLoginResponse struct {
	Stdout   string
	Username string
}

//Validate checks if request is valid
func (r *DockerLoginRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was nil")
	}
	if r.Repository == "" {
		return errors.New("repository was empty")
	}
	return nil
}

//DockerLogoutRequest represents a docker pull request
type DockerLogoutRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Repository string        `required:"true" description:"repository URL"`
}

//DockerLogoutResponse represents a docker pull request
type DockerLogoutResponse struct {
	Stdout string
}

//DockerPullRequest represents a docker pull request
type DockerPullRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Repository string        `required:"true"`
	Tag        string        `required:"true"`
}

//DockerPullResponse represents a docker pull response
type DockerPullResponse struct {
	*DockerImageInfo
}

//DockerPushRequest represents a docker push request
type DockerPushRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"` //target host
	Tag    *DockerTag    `required:"true"`
}

//DockerPushResponse represents a docker push request
type DockerPushResponse struct {
}

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

//DockerStopImagesRequest represents docker stop running images request
type DockerStopImagesRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"` //target host
	Images []string      `required:"true"`
}

//DockerStopImagesResponse represents docker stop images response
type DockerStopImagesResponse struct {
	StoppedImages []string
}

//DockerTagRequest represents docker tag request
type DockerTagRequest struct {
	Target    *url.Resource `required:"true" description:"host with docker service"` //target host
	SourceTag *DockerTag    `required:"true"`
	TargetTag *DockerTag    `required:"true"`
}

//DockerTag represent a docker tag
type DockerTag struct {
	Username string
	Registry string
	Image    string
	Version  string
}

//DockerTagResponse represents docker tag response
type DockerTagResponse struct {
	Stdout string
}

//Validate checks if request valid
func (r *DockerTagRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was empty")
	}
	if r.SourceTag == nil {
		return errors.New("sourceImage was empty")
	}
	if r.TargetTag == nil {
		return errors.New("sourceImage was empty")
	}
	if err := r.SourceTag.Validate(); err != nil {
		return err
	}
	return r.TargetTag.Validate()
}

//Validate checks if tag is valid
func (t *DockerTag) Validate() error {
	if t.Image == "" {
		return errors.New("image was empty")
	}
	return nil
}

//String stringify docker tag
func (t *DockerTag) String() string {
	var result = t.Username
	if result == "" {
		result = t.Registry
	}
	if result != "" {
		result += "/"
	}
	result += t.Image
	if t.Version != "" {
		result += ":" + t.Version
	}
	return result
}
