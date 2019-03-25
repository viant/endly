package docker

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/go-errors/errors"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/secret"
	"github.com/viant/toolbox/url"
	"strings"
)

//RunRequest represents a docker runAdapter request
type RunRequest struct {
	Credentials                 string `description:"credentials"`
	Name                        string
	Image                       string            `required:"true" description:"container image to runAdapter" example:"mysql:5.6"`
	Port                        string            `description:"publish a container’s port(s) to the host, docker -p option"`
	Env                         map[string]string `description:"set docker container an environment variable, docker -e KEY=VAL  option"`
	Mount                       map[string]string `description:"bind mount a volume, docker -v option"`
	Ports                       map[string]string `description:"publish a container’s port(s) to the host, docker -p option"`
	Workdir                     string            `description:"working directory inside the container, docker -w option"`
	Reuse                       bool              `description:"reuse existing container if exists, otherwise always removes"`
	Cmd                         []string
	Entrypoint                  []string
	types.ContainerCreateConfig `json:",inline" yaml:",inline"`
	Secrets                     map[secret.SecretKey]secret.Secret `description:"map of secrets used within env"`
}

type RunResponse struct {
	ContainerID string
	Status      string
	Stdout      string
}

//BuildRequest represents docker build request
type BuildRequest struct {
	Tag                     *Tag   `required:"true" description:"build docker tag"`
	Path                    string `description:"location of dockerfile"`
	types.ImageBuildOptions `json:",inline" yaml:",inline"`
}

func (r *BuildRequest) Init() error {
	if r.Path == "" {
		r.Path = url.NewResource(".").ParsedURL.Path
	}
	if len(r.Tags) == 0 {
		r.Tags = make([]string, 0)
		if r.Tag != nil {
			r.Tags = append(r.Tags, r.Tag.String())
		}
	}
	return nil
}

//BuildResponse represents image ID
type BuildResponse struct {
	ImageID string
	Stdout  []string
}

//LoginRequest represents a docker pull request
type LoginRequest struct {
	Credentials string `required:"true" description:"credentials path"`
	Repository  string `required:"true" description:"repository url"`
}

//LoginResponse represents login response
type LoginResponse struct {
}

//TagRequest represents docker tag request
type TagRequest struct {
	SourceTag *Tag `required:"true"`
	TargetTag *Tag `required:"true"`
}

//TagResponse represents docker tag response
type TagResponse struct {
	Stdout string
}

//PushRequest represents a docker push request
type PushRequest struct {
	Credentials string
	Tag         *Tag `required:"true"`
}

//PushResponse represents push response
type PushResponse struct {
	Stdout []string
}

//StatusRequest represents a docker check container status request
type StatusRequest struct {
	Name   string
	Names  []string
	Images []string
	IDs    []string
}

//StatusResponse represents status response
type StatusResponse struct {
	Containers []types.Container
}

//StartRequest start request
type StartRequest StatusRequest

//StartResponse represents docker start response
type StartResponse StopResponse

//StopRequest represents docker stop running images/containers request
type StopRequest StatusRequest

//StopImagesResponse represents docker stop images response
type StopResponse StatusResponse

//RemoveRequest represents docker remove request
type RemoveRequest StatusRequest

//RemoveResponse represents remove response
type RemoveResponse StatusResponse

//LogsRequest represents docker runner container logs to take stdout
type LogsRequest struct {
	StatusRequest
	*types.ContainerLogsOptions
}

//LogsResponse represents docker container logs response
type LogsResponse struct {
	Stdout string
}

//PullRequest represents pull request
type PullRequest struct {
	Credentials            string
	Image                  string
	types.ImagePullOptions `json:",inline" yaml:",inline"`
}

//PullResponse represents pull response
type PullResponse struct {
	types.ImageSummary
	Stdout []string
}

//LogoutRequest represents a docker logout request
type LogoutRequest struct {
	Repository string `required:"true" description:"repository URL"`
}

//LogoutResponse represents a docker logout response
type LogoutResponse struct{}

//PushResponse represents a docker push request
type CopyRequest struct {
	Assets map[string]string
}

//CopyResponse represents a copy response
type CopyResponse struct{}

//InspectRequest represents a docker inspect request, target name refers to container name
type InspectRequest StatusRequest

//InspectResponse represents a docker inspect request
type InspectResponse struct {
	Info []types.ContainerJSON //you can extract any instance default, for instance to get Ip you can use Info[0].NetworkSettings.IPAddress in the variable action post from key
}

func (r *CopyRequest) Validate() error {
	if len(r.Assets) == 0 {
		return fmt.Errorf("asset was empty")
	}

	return nil
}

func (r *RunRequest) Init() error {
	if r.Config == nil {
		r.Config = &container.Config{}
	}
	if r.HostConfig == nil {
		r.HostConfig = &container.HostConfig{}
	}

	if r.Image != "" {
		r.Config.Image = r.Image
	}
	if len(r.Mount) > 0 {
		r.HostConfig.Mounts = make([]mount.Mount, 0)
		r.Config.Volumes = make(map[string]struct{})
		for source, dest := range r.Mount {
			if parts := strings.SplitN(source, ":", 2); len(parts) == 2 {
				source = parts[0]
				dest = parts[1]
			}
			source = expandHomeDirectory(source)
			source = url.NewResource(source).ParsedURL.Path
			r.HostConfig.Mounts = append(r.HostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: source,
				Target: dest,
			})
		}
	}
	if r.Port != "" {
		portSet := nat.PortSet{nat.Port(r.Port): struct{}{}}
		if err := toolbox.DefaultConverter.AssignConverted(&r.Config.ExposedPorts, portSet);err != nil {
			return err
		}
	}
	if len(r.Ports) > 0 {

		r.HostConfig.PortBindings = make(map[nat.Port][]nat.PortBinding)
		for source, dest := range r.Ports {
			if !strings.Contains(dest, "/") {
				dest += "/tcp"
			}
			ports := []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: source}}
			var portsBindings = make(map[nat.Port][]nat.PortBinding)
			portsBindings[nat.Port(dest)] = ports
			if err := toolbox.DefaultConverter.AssignConverted(&r.HostConfig.PortBindings, portsBindings);err != nil {
				return err
			}
		}
	}
	if len(r.Env) > 0 {
		r.Config.Env = make([]string, 0)
		for k, v := range r.Env {
			r.Config.Env = append(r.Config.Env, fmt.Sprintf("%v=%v", k, v))
		}
	}
	if r.Workdir != "" {
		r.Config.WorkingDir = r.Workdir
	}
	if len(r.Cmd) > 0 {
		r.Config.Cmd = r.Cmd
	}
	if len(r.Entrypoint) > 0 {
		r.Config.Entrypoint = r.Entrypoint
	}
	if r.Name != "" {
		r.ContainerCreateConfig.Name = r.Name
	}
	return nil
}

func (r *RunRequest) Validate() error {
	if r.Config.Image == "" {
		return errors.New("image was empty")
	}
	return nil
}

func (r *PullRequest) Init() error {
	return nil
}

func (r *RunRequest) CreateContainerRequest() *ContainerCreateRequest {
	createRequest := &ContainerCreateRequest{}
	createRequest.Config = r.ContainerCreateConfig.Config
	createRequest.NetworkingConfig = r.ContainerCreateConfig.NetworkingConfig
	createRequest.HostConfig = r.ContainerCreateConfig.HostConfig
	createRequest.ContainerName = r.Name
	return createRequest
}

func (r *LoginRequest) Validate() error {
	if r.Credentials == "" {
		return errors.New("credentials were empty")
	}
	if r.Repository == "" {
		return errors.New("repository was empty")
	}
	return nil
}

func (r *StatusRequest) Init() error {
	if r.Name != "" && len(r.Names) == 0 {
		r.Names = strings.Split(r.Name, ",")
	}
	return nil
}

//StatusRequest returns status request
func (r *StopRequest) AsStatusRequest() *StatusRequest {
	result := StatusRequest(*r)
	return &result
}

//StatusRequest returns status request
func (r *RemoveRequest) AsStatusRequest() *StatusRequest {
	result := StatusRequest(*r)
	return &result
}

//StatusRequest returns status request
func (r *StartRequest) AsStatusRequest() *StatusRequest {
	result := StatusRequest(*r)
	return &result
}

//StatusRequest returns status request
func (r *InspectRequest) AsStatusRequest() *StatusRequest {
	result := StatusRequest(*r)
	return &result
}

//StatusRequest returns status request
func (r *LogsRequest) AsStatusRequest() *StatusRequest {
	return &r.StatusRequest
}

//StatusRequest returns status request
func (r *LogsRequest) Init() error {
	if r.ContainerLogsOptions == nil {
		r.ContainerLogsOptions = &types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		}
	}
	return nil
}
