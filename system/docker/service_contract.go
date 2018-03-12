package docker

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/url"
)

//BuildRequest represents docker build request
type BuildRequest struct {
	Target    *url.Resource     `required:"true" description:"host with docker service"` //target host
	Tag       *Tag              `required:"true" description:"build docker tag"`
	Path      string            `required:"true" description:"docker build source path"`
	Arguments map[string]string `description:"docker build command line arguments, see more: https://docs.docker.com/engine/reference/commandline/build/#description "` //https://docs.docker.com/engine/reference/commandline/build/#description
}

//BuildResponse represents docker build response
type BuildResponse struct {
	Stdout string
}

//Init initialises default values
func (r *BuildRequest) Init() {
	if len(r.Arguments) == 0 && r.Tag != nil {
		r.Arguments = make(map[string]string)
	}
	if r.Tag != nil {
		r.Arguments["-t"] = r.Tag.String()
	}
}

//Validate check if request is valid
func (r *BuildRequest) Validate() error {
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

//ContainerStatusRequest represents a docker check container status request
type ContainerStatusRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"` //target host
	Names  string
	Image  string
}

//ContainerStatusResponse represents a docker container check response
type ContainerStatusResponse struct {
	Containers []*ContainerInfo
}

//ContainerBaseRequest represents container base request
type ContainerBaseRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"`                //target host
	Name   string        `description:"container name to inspect, if empty it uses target.Name"` //docker container name
}

//Init initializes request
func (r *ContainerBaseRequest) Init() error {
	if r == nil || r.Target == nil {
		return nil
	}
	if r.Name != "" {
		return nil
	}
	return nil
}

//Validate checks if request is valid
func (r *ContainerBaseRequest) Validate() error {
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

//ContainerStartRequest represents a docker container start request.
type ContainerStartRequest struct {
	*ContainerBaseRequest
}

//ContainerStartResponse represents a docker container start response
type ContainerStartResponse struct {
	*ContainerInfo
}

//ContainerRemoveRequest represents a docker remove container request
type ContainerRemoveRequest struct {
	*ContainerBaseRequest
}

//ContainerRemoveResponse represents a docker remove container response
type ContainerRemoveResponse struct {
	Stdout string
}

//ContainerStopRequest represents a docker stop container request.
type ContainerStopRequest struct {
	*ContainerBaseRequest
}

//ContainerStopResponse represents a docker stop container response.
type ContainerStopResponse struct {
	*ContainerInfo
}

//ContainerRunRequest represents a docker run container command.
type ContainerRunRequest struct {
	*ContainerBaseRequest
	Secrets            map[string]string
	Interactive        bool
	AllocateTerminal   bool
	RunInTheBackground bool
	Command            string
}

//ContainerInspectRequest represents a docker inspect request, target name refers to container name
type ContainerInspectRequest struct {
	*ContainerBaseRequest
}

//ContainerInspectResponse represents a docker inspect request
type ContainerInspectResponse struct {
	Stdout string
	Info   interface{} //you can extract any instance default, for instance to get Ip you can use Info[0].NetworkSettings.IPAddress in the variable action post from key
}

//ContainerRunResponse represents a docker run command  response
type ContainerRunResponse struct {
	Stdout string
}

//ContainerInfo represents a docker container info
type ContainerInfo struct {
	ContainerID string
	Image       string
	Command     string
	Status      string
	Port        string
	Names       string
}

//ContainerLogsRequest represents docker runner container logs to take stdout
type ContainerLogsRequest struct {
	*ContainerBaseRequest
}

//ContainerLogsResponse represents docker container logs response
type ContainerLogsResponse struct {
	Stdout string
}

//ImagesRequest represents docker check image request
type ImagesRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Repository string        `required:"true"`
	Tag        string        `required:"true"`
}

//ImagesResponse represents a docker check image response
type ImagesResponse struct {
	Images []*ImageInfo
}

//ImageInfo represents docker image info
type ImageInfo struct {
	Repository string
	Tag        string
	ImageID    string
	Size       int
}

//LoginRequest represents a docker pull request
type LoginRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Credential string        `required:"true" description:"credential path"`
	Repository string        `required:"true" description:"repository url"`
}

//LoginResponse represents a docker pull request
type LoginResponse struct {
	Stdout   string
	Username string
}

//Validate checks if request is valid
func (r *LoginRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was nil")
	}
	if r.Repository == "" {
		return errors.New("repository was empty")
	}
	return nil
}

//LogoutRequest represents a docker pull request
type LogoutRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Repository string        `required:"true" description:"repository URL"`
}

//LogoutResponse represents a docker pull request
type LogoutResponse struct {
	Stdout string
}

//PullRequest represents a docker pull request
type PullRequest struct {
	Target     *url.Resource `required:"true" description:"host with docker service"` //target host
	Repository string        `required:"true"`
	Tag        string        `required:"true"`
}

//PullResponse represents a docker pull response
type PullResponse struct {
	*ImageInfo
}

//PushRequest represents a docker push request
type PushRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"` //target host
	Tag    *Tag          `required:"true"`
}

//PushResponse represents a docker push request
type PushResponse struct {
}

//RunRequest represents a docker run request
type RunRequest struct {
	Target  *url.Resource     `required:"true" description:"host with docker service"`                //target host
	Name    string            `description:"container name to inspect, if empty it uses target.Name"` //docker container name
	Secrets map[string]string `description:"map of secret key to obfuscate terminal output with corresponding filename storing credential compatible with github.com/viant/toolbox/cred/config.go"`
	Image   string            `required:"true" description:"container image to run" example:"mysql:5.6"`
	Port    string            `description:"publish a container’s port(s) to the host, docker -p option"`
	Env     map[string]string `description:"set docker container an environment variable, docker -e KEY=VAL  option"`
	Mount   map[string]string `description:"bind mount a volume, docker -v option"`
	Ports   map[string]string `description:"publish a container’s port(s) to the host, docker -p option"`
	Params  map[string]string `description:"other free form docker parameters"`
	Workdir string            `description:"working directory inside the container, docker -w option"`
}

//RunResponse represents a docker run response
type RunResponse struct {
	*ContainerInfo
}

//Validate checks if request is valid
func (r *RunRequest) Validate() error {
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

//StopImagesRequest represents docker stop running images request
type StopImagesRequest struct {
	Target *url.Resource `required:"true" description:"host with docker service"` //target host
	Images []string      `required:"true"`
}

//StopImagesResponse represents docker stop images response
type StopImagesResponse struct {
	StoppedImages []string
}

//TagRequest represents docker tag request
type TagRequest struct {
	Target    *url.Resource `required:"true" description:"host with docker service"` //target host
	SourceTag *Tag          `required:"true"`
	TargetTag *Tag          `required:"true"`
}

//Tag represent a docker tag
type Tag struct {
	Username string
	Registry string
	Image    string
	Version  string
}

//TagResponse represents docker tag response
type TagResponse struct {
	Stdout string
}

//Validate checks if request valid
func (r *TagRequest) Validate() error {
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
func (t *Tag) Validate() error {
	if t.Image == "" {
		return errors.New("image was empty")
	}
	return nil
}

//String stringify docker tag
func (t *Tag) String() string {
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
