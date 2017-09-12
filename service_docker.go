package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"strings"
	"sync"
)

const DockerServiceId = "docker"
const containerInUse = "is already in use by container"

var dockerErrors = []string{"Error"}
var dockerIgnoreErrors = []string{}

type DockerPullRequest struct {
	Target     *Resource
	Repository string
	Tag        string
}

type DockerImagesRequest struct {
	Target     *Resource
	Repository string
	Tag        string
}

type DockerImageInfo struct {
	Repository string
	Tag        string
	ImageId    string
	Size       int
}

type DockerRunRequest struct {
	Target     *Resource
	Image      string
	Port       string
	Credential string
	Env        map[string]string
	Mount      map[string]string
	MappedPort map[int]int
	Workdir    string
}

type DockerContainerCheckRequest struct {
	Target *Resource
	Names  string
	Image  string
}

type DockerContainerStartRequest struct {
	Target *Resource
}

type DockerContainerRemoveRequest struct {
	Target *Resource
}

type DockerContainerStopRequest struct {
	Target *Resource
}

type DockerContainerCommandRequest struct {
	Target  *Resource
	Command string
}

type DockerContainerInfo struct {
	ContainerId string
	Image       string
	Command     string
	Status      string
	Port        string
	Names       string
}

type DockerService struct {
	*AbstractService
	SysPath []string
}

func (s *DockerService) NewRequest(name string) (interface{}, error) {
	switch name {
	case "images":
		return &DockerImagesRequest{}, nil
	case "pull":
		return &DockerPullRequest{}, nil
	case "process":
		return &DockerContainerCheckRequest{}, nil
	case "container-command":
		return &DockerContainerCommandRequest{}, nil
	case "container-start":
		return &DockerContainerStartRequest{}, nil
	case "container-stop":
		return &DockerContainerStopRequest{}, nil
	case "container-remove":
		return &DockerContainerStopRequest{}, nil

	}
	return nil, fmt.Errorf("Unsupported name: %v", name)
}

func (s *DockerService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
	var err error
	switch actualRequest := request.(type) {
	case *DockerImagesRequest:
		response.Response, err = s.checkImages(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to check images: %v, %v", actualRequest, err)
		}
	case *DockerPullRequest:
		response.Response, err = s.pullImage(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to pull images: %v, %v", actualRequest, err)
		}
	case *DockerContainerCheckRequest:
		response.Response, err = s.checkContainerProcesses(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to pull images: %v, %v", actualRequest, err)
		}

	case *DockerContainerCommandRequest:
		response.Response, err = s.runInContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to pull images: %v, %v", actualRequest, err)
		}
	case *DockerContainerStartRequest:
		response.Response, err = s.startContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to pull images: %v, %v", actualRequest, err)
		}
	case *DockerContainerStopRequest:
		response.Response, err = s.stopContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to pull images: %v, %v", actualRequest, err)
		}
	case *DockerContainerRemoveRequest:
		response.Response, err = s.remoteContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to pull images: %v, %v", actualRequest, err)
		}
	case *DockerRunRequest:
		response.Response, err = s.runContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run: %v(%v), %v", actualRequest.Target.Name, actualRequest.Image, err)
		}

	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

/**
	https://docs.docker.com/compose/reference/run/
Options:
    -d                    Detached mode: Run container in the background, print
                          new container name.
    --name NAME           Assign a name to the container
    --entrypoint CMD      Override the entrypoint of the image.
    -e KEY=VAL            Set an environment variable (can be used multiple times)
    -u, --user=""         Run as specified username or uid
    --no-deps             Don't start linked services.
    --rm                  Remove container after run. Ignored in detached mode.
    -p, --publish=[]      Publish a container's port(s) to the host
    --service-ports       Run command with the service's ports enabled and mapped
                          to the host.
    -v, --volume=[]       Bind mount a volume (default [])
    -T                    Disable pseudo-tty allocation. By default `docker-compose run`
                          allocates a TTY.
    -w, --workdir=""      Working directory inside the container

*/

func (s *DockerService) runContainer(context *Context, request *DockerRunRequest) (*DockerContainerInfo, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v", request.Target.URL)
	}
	if request.Image == "" {
		return nil, fmt.Errorf("Image was empty for %v", request.Target.URL)
	}

	var secure = ""
	if request.Credential != "" {
		credential := &storage.PasswordCredential{}
		err := LoadCredential(context.CredentialFile(request.Credential), credential)
		if err != nil {
			return nil, err
		}
		secure = credential.Password
	}
	var args = ""
	for k, v := range request.Env {
		args += fmt.Sprintf("-e %v=%v ", k, context.Expand(v))
	}
	for k, v := range request.Mount {
		args += fmt.Sprintf("-v %v:%v ", k, context.Expand(v))
	}
	for k, v := range request.MappedPort {
		args += fmt.Sprintf("-p %v:%v ", k, v)
	}
	if request.Workdir != "" {
		args += fmt.Sprintf("-w %v ", request.Workdir)
	}
	commandInfo, err := s.executeSecureDockerCommand(secure, context, request.Target, dockerIgnoreErrors, "docker run --name %v %v -d %v", request.Target.Name, args, request.Image)
	if err != nil {
		return nil, err
	}
	if strings.Contains(commandInfo.Stdout(), containerInUse) {
		s.stopContainer(context, &DockerContainerStopRequest{Target: request.Target})
		s.remoteContainer(context, &DockerContainerRemoveRequest{Target: request.Target})
		commandInfo, err = s.executeSecureDockerCommand(secure, context, request.Target, dockerErrors, "docker run --name %v %v -d %v", request.Target.Name, args, request.Image)
		if err != nil {
			return nil, err
		}
	}
	return s.checkContainerProcess(context, &DockerContainerCheckRequest{
		Target: request.Target,
		Names:  request.Target.Name,
	})
}

func (s *DockerService) checkContainerProcess(context *Context, request *DockerContainerCheckRequest) (*DockerContainerInfo, error) {
	info, err := s.checkContainerProcesses(context, request)
	if err != nil {
		return nil, err
	}
	if len(info) == 1 {
		return info[0], nil
	}
	return nil, nil
}

func (s *DockerService) startContainer(context *Context, request *DockerContainerStartRequest) (*DockerContainerInfo, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v and command %v", request.Target.URL)
	}
	_, err := s.executeDockerCommand(context, request.Target, dockerErrors, "docker start %v", request.Target.Name)
	if err != nil {
		return nil, err
	}
	return s.checkContainerProcess(context, &DockerContainerCheckRequest{
		Target: request.Target,
		Names:  request.Target.Name,
	})

}

func (s *DockerService) stopContainer(context *Context, request *DockerContainerStopRequest) (*DockerContainerInfo, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v and command %v", request.Target.URL)
	}
	info, err := s.checkContainerProcess(context, &DockerContainerCheckRequest{
		Target: request.Target,
		Names:  request.Target.Name,
	})
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, err
	}

	_, err = s.executeDockerCommand(context, request.Target, dockerErrors, "docker stop %v", request.Target.Name)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (s *DockerService) remoteContainer(context *Context, request *DockerContainerRemoveRequest) (*CommandInfo, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v and command %v", request.Target.URL)
	}

	commandInfo, err := s.executeDockerCommand(context, request.Target, dockerErrors, "docker rm %v", request.Target.Name)
	if err != nil {
		return nil, err
	}
	return commandInfo, nil
}

func (s *DockerService) runInContainer(context *Context, request *DockerContainerCommandRequest) (*CommandInfo, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v and command %v", request.Target.URL, request.Command)
	}
	return s.executeDockerCommand(context, request.Target, dockerErrors, "docker exec %v /bin/sh -c \"%v\"", request.Target.Name, request.Command)
}

func (s *DockerService) checkContainerProcesses(context *Context, request *DockerContainerCheckRequest) ([]*DockerContainerInfo, error) {
	info, err := s.executeDockerCommand(context, request.Target, dockerErrors, "docker ps")
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()
	var result = make([]*DockerContainerInfo, 0)
	var lines = strings.Split(stdout, "\r\n")
	for i := 1; i < len(lines); i++ {
		columns, ok := ExtractColumns(lines[i])
		if !ok || len(columns) < 7 {
			continue
		}
		var status = "down"
		if strings.Contains(lines[i], "Up") {
			status = "up"
		}
		info := &DockerContainerInfo{
			ContainerId: columns[0],
			Image:       columns[1],
			Command:     columns[2],
			Status:      status,
			Port:        columns[len(columns)-2],
			Names:       columns[len(columns)-1],
		}
		if request.Image != "" && request.Image != info.Image {
			continue
		}
		if request.Names != "" && request.Names != info.Names {
			continue
		}
		result = append(result, info)
	}
	return result, nil
}

func (s *DockerService) pullImage(context *Context, request *DockerPullRequest) (*DockerImageInfo, error) {
	if request.Tag == "" {
		request.Tag = "latest"
	}
	info, err := s.executeDockerCommand(context, request.Target, dockerErrors, "docker pull %v:%v", request.Repository, request.Tag)
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()
	if strings.Contains(stdout, "not found") {
		return nil, fmt.Errorf("Failed to pull docker image,  %v", stdout)
	}
	images, err := s.checkImages(context, &DockerImagesRequest{Target: request.Target, Repository: request.Repository, Tag: request.Tag})
	if err != nil {
		return nil, err
	}
	if len(images) == 1 {
		return images[0], nil
	}
	return nil, fmt.Errorf("Failed to check image status: %v:%v found: %v", request.Repository, request.Tag, len(images))
}

func (s *DockerService) checkImages(context *Context, request *DockerImagesRequest) ([]*DockerImageInfo, error) {
	info, err := s.executeDockerCommand(context, request.Target, dockerErrors, "docker images")
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()
	var result = make([]*DockerImageInfo, 0)
	for _, line := range strings.Split(stdout, "\r\n") {
		columns, ok := ExtractColumns(line)
		if !ok || len(columns) < 4 {
			continue
		}
		var sizeUnit = columns[len(columns)-1]
		var sizeFactor = 1
		switch strings.ToUpper(sizeUnit) {
		case "MB":
			sizeFactor = 1024 * 1024
		case "KB":
			sizeFactor = 1024
		}

		info := &DockerImageInfo{
			Repository: columns[0],
			Tag:        columns[1],
			ImageId:    columns[2],
			Size:       toolbox.AsInt(columns[len(columns)-2]) * sizeFactor,
		}
		if request.Repository != "" {
			if info.Repository != request.Repository {
				continue
			}
		}
		if request.Tag != "" {
			if info.Tag != request.Tag {
				continue
			}
		}
		result = append(result, info)
	}
	return result, nil

}

func (s *DockerService) executeDockerCommand(context *Context, target *Resource, errors []string, template string, arguments ...interface{}) (*CommandInfo, error) {
	return s.executeSecureDockerCommand("", context, target, errors, template, arguments...)
}

func (s *DockerService) executeSecureDockerCommand(secure string, context *Context, target *Resource, errors []string, template string, arguments ...interface{}) (*CommandInfo, error) {
	command := fmt.Sprintf(template, arguments...)
	return context.Execute(target, &ManagedCommand{
		Options: &ExecutionOptions{
			SystemPaths: s.SysPath,
		},
		Executions: []*Execution{
			{
				Secure:  secure,
				Command: command,
				Error:   errors,
			},
		},
	})

}

var _dockerService *DockerService
var _dockerServiceMutext = &sync.Mutex{}

func GetDockerService() *DockerService {
	if _dockerService != nil {
		return _dockerService
	}
	_dockerServiceMutext.Lock()
	defer _dockerServiceMutext.Unlock()
	if _dockerService != nil {
		return _dockerService
	}
	var _dockerService = &DockerService{
		AbstractService: NewAbstractService(DockerServiceId),
	}
	_dockerService.AbstractService.Service = _dockerService
	return _dockerService
}
