package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
	"strings"
)

//DockerServiceID represents docker service id
const DockerServiceID = "docker"
const containerInUse = "is already in use by container"

var dockerErrors = []string{"failed"}
var dockerIgnoreErrors = []string{}

type dockerService struct {
	*AbstractService
	SysPath []string
}

func (s *dockerService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "run":
		return &DockerRunRequest{}, nil
	case "syspath":
		return &DockerSystemPathRequest{}, nil
	case "stop-images":
		return &DockerStopImagesRequest{}, nil
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
		return &DockerContainerRemoveRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)
}

func (s *dockerService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {

	case *DockerSystemPathRequest:
		s.SysPath = actualRequest.SysPath

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
			response.Error = fmt.Sprintf("Failed to check process: %v, %v", actualRequest, err)
		}

	case *DockerContainerCommandRequest:
		response.Response, err = s.runInContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run docker command: %v, %v", actualRequest, err)
		}
	case *DockerContainerStartRequest:
		response.Response, err = s.startContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed strart container: %v, %v", actualRequest, err)
		}
	case *DockerContainerStopRequest:
		response.Response, err = s.stopContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to start container: %v, %v", actualRequest, err)
		}
	case *DockerContainerRemoveRequest:
		response.Response, err = s.removeContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to remove container: %v, %v", actualRequest, err)
		}
	case *DockerRunRequest:
		response.Response, err = s.runContainer(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run: %v(%v), %v", actualRequest.Target.Name, actualRequest.Image, err)
		}
	case *DockerStopImagesRequest:
		response.Response, err = s.stopImages(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run: %v(%v), %v", actualRequest.Target.Name, actualRequest.Images, err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)

	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *dockerService) stopImages(context *Context, request *DockerStopImagesRequest) (*DockerStopImagesResponse, error) {
	if len(request.SysPath) > 0 {
		s.SysPath = request.SysPath
	}
	var response = &DockerStopImagesResponse{
		StoppedImages: make([]string, 0),
	}
	processResponse, err := s.checkContainerProcesses(context, &DockerContainerCheckRequest{
		Target:  request.Target,
		SysPath: request.SysPath,
	})
	if err != nil {
		return nil, err
	}

	for _, image := range request.Images {
		for _, container := range processResponse.Containers {
			if strings.Contains(container.Image, image) {

				var containerTarget = request.Target.Clone()
				containerTarget.Name = strings.Split(container.Names, ",")[0]
				_, err = s.stopContainer(context, &DockerContainerStopRequest{
					Target: containerTarget,
				})
				if err != nil {
					return nil, err
				}
				response.StoppedImages = append(response.StoppedImages, container.Image)
			}
		}
	}
	return response, nil
}

/**
	https://docs.docker.com/compose/reference/run/
Options:
    -d                    Detached mode: Run container in the background, print
                          new container Id.
    --Id NAME           Assign a Id to the container
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

func (s *dockerService) secure(credential string) (string, error) {
	var secure = ""
	if credential != "" {
		credential, err := cred.NewConfig(credential)
		if err != nil {
			return "", err
		}
		secure = credential.Password
	}
	return secure, nil
}

func (s *dockerService) runContainer(context *Context, request *DockerRunRequest) (*DockerContainerInfo, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v", request.Target.URL)
	}
	if request.Image == "" {
		return nil, fmt.Errorf("Image was empty for %v", request.Target.URL)
	}

	if len(request.SysPath) > 0 {
		s.SysPath = request.SysPath
	}

	var secure, err = s.secure(request.Credential)
	var args = ""
	for k, v := range request.Env {
		args += fmt.Sprintf("-e %v=%v ", k, context.Expand(v))
	}
	for k, v := range request.Mount {
		args += fmt.Sprintf("-v %v:%v ", context.Expand(k), context.Expand(v))
	}
	for k, v := range request.MappedPort {
		args += fmt.Sprintf("-p %v:%v ", context.Expand(toolbox.AsString(k)), context.Expand(toolbox.AsString(v)))
	}
	if request.Workdir != "" {
		args += fmt.Sprintf("-w %v ", context.Expand(request.Workdir))
	}
	var params = ""
	for k, v := range request.Params {
		params += fmt.Sprintf("%v %v", k, v)
	}
	commandInfo, err := s.executeSecureDockerCommand(secure, context, request.Target, dockerIgnoreErrors, "docker run --name %v %v -d %v %v", request.Target.Name, args, request.Image, params)
	if err != nil {
		return nil, err
	}
	if strings.Contains(commandInfo.Stdout(), containerInUse) {
		s.stopContainer(context, &DockerContainerStopRequest{Target: request.Target})
		s.removeContainer(context, &DockerContainerRemoveRequest{Target: request.Target})
		commandInfo, err = s.executeSecureDockerCommand(secure, context, request.Target, dockerErrors, "docker run --name %v %v -d %v", request.Target.Name, args, request.Image)
		if err != nil {
			return nil, err
		}
	}
	return s.checkContainerProcess(context, &DockerContainerCheckRequest{
		Target:  request.Target,
		Names:   request.Target.Name,
		SysPath: request.SysPath,
	})
}

func (s *dockerService) checkContainerProcess(context *Context, request *DockerContainerCheckRequest) (*DockerContainerInfo, error) {
	if len(request.SysPath) > 0 {
		s.SysPath = request.SysPath
	}

	checkResponse, err := s.checkContainerProcesses(context, request)
	if err != nil {
		return nil, err
	}

	if len(checkResponse.Containers) == 1 {
		return checkResponse.Containers[0], nil
	}
	return nil, nil
}

func (s *dockerService) startContainer(context *Context, request *DockerContainerStartRequest) (*DockerContainerInfo, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v and command %v", request.Target.URL)
	}
	if len(request.SysPath) > 0 {
		s.SysPath = request.SysPath
	}

	_, err := s.executeDockerCommand(context, request.Target, dockerErrors, "docker start %v", request.Target.Name)
	if err != nil {
		return nil, err
	}
	return s.checkContainerProcess(context, &DockerContainerCheckRequest{
		Target:  request.Target,
		Names:   request.Target.Name,
		SysPath: request.SysPath,
	})

}

func (s *dockerService) stopContainer(context *Context, request *DockerContainerStopRequest) (*DockerContainerInfo, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v and command %v", request.Target.URL)
	}
	if len(request.SysPath) > 0 {
		s.SysPath = request.SysPath
	}

	info, err := s.checkContainerProcess(context, &DockerContainerCheckRequest{
		Target:  request.Target,
		Names:   request.Target.Name,
		SysPath: request.SysPath,
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

func (s *dockerService) removeContainer(context *Context, request *DockerContainerRemoveRequest) (*CommandResponse, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v and command %v", request.Target.URL)
	}
	if len(request.SysPath) > 0 {
		s.SysPath = request.SysPath
	}

	commandInfo, err := s.executeDockerCommand(context, request.Target, dockerErrors, "docker rm %v", request.Target.Name)
	if err != nil {
		return nil, err
	}
	return commandInfo, nil
}

func (s *dockerService) runInContainer(context *Context, request *DockerContainerCommandRequest) (*CommandResponse, error) {
	if request.Target.Name == "" {
		return nil, fmt.Errorf("Target name was empty for %v and command %v", request.Target.URL, request.Command)
	}
	if len(request.SysPath) > 0 {
		s.SysPath = request.SysPath
	}

	var command = context.Expand(request.Command)
	if request.Credential != "" {
		targetSecure, err := s.secure(request.Credential)
		if err != nil {
			return nil, err
		}
		command = strings.Replace(command, "****", targetSecure, len(command))
	}

	var executionOptions = ""
	if request.Interactive {
		executionOptions += "i"
	}
	if request.AllocateTerminal {
		executionOptions += "t"
	}
	if request.RunInTheBackground {
		executionOptions += "d"
	}
	if executionOptions != "" {
		executionOptions = "-" + executionOptions
	}
	return s.executeDockerCommand(context, request.Target, dockerErrors, "docker exec %v %v %v", executionOptions, request.Target.Name, command)
}

func (s *dockerService) checkContainerProcesses(context *Context, request *DockerContainerCheckRequest) (*DockerContainerCheckResponse, error) {

	info, err := s.executeSecureDockerCommand("", context, request.Target, dockerErrors, "docker ps")
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()
	var containers = make([]*DockerContainerInfo, 0)
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
			ContainerID: columns[0],
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
		containers = append(containers, info)
	}
	return &DockerContainerCheckResponse{Containers: containers}, nil
}

func (s *dockerService) pullImage(context *Context, request *DockerPullRequest) (*DockerImageInfo, error) {
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
	imageResponse, err := s.checkImages(context, &DockerImagesRequest{Target: request.Target, Repository: request.Repository, Tag: request.Tag})
	if err != nil {
		return nil, err
	}
	if len(imageResponse.Images) == 1 {
		return imageResponse.Images[0], nil
	}
	return nil, fmt.Errorf("Failed to check image status: %v:%v found: %v", request.Repository, request.Tag, len(imageResponse.Images))
}

func (s *dockerService) checkImages(context *Context, request *DockerImagesRequest) (*DockerImagesResponse, error) {
	info, err := s.executeDockerCommand(context, request.Target, dockerErrors, "docker images")
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()
	var images = make([]*DockerImageInfo, 0)
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
			ImageID:    columns[2],
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
		images = append(images, info)
	}
	return &DockerImagesResponse{Images: images}, nil

}

func (s *dockerService) executeDockerCommand(context *Context, target *url.Resource, errors []string, template string, arguments ...interface{}) (*CommandResponse, error) {
	return s.executeSecureDockerCommand("", context, target, errors, template, arguments...)
}

func (s *dockerService) executeSecureDockerCommand(secure string, context *Context, target *url.Resource, errors []string, template string, arguments ...interface{}) (*CommandResponse, error) {
	command := fmt.Sprintf(template, arguments...)
	if secure != "" {
		command = strings.Replace(command, "****", secure, len(command))
	}
	targetSecure, err := s.secure(target.Credential)
	if err != nil {
		return nil, err
	}

	return context.Execute(target, &superUserCommandRequest{
		MangedCommand: &ManagedCommand{
			Options: &ExecutionOptions{
				SystemPaths: s.SysPath,
			},
			Executions: []*Execution{
				{
					Secure:  targetSecure,
					Command: command,
					Error:   append(errors, []string{commandNotFound}...),
				},
			},
		},
	})
}

//NewDockerService returns a new docker service.
func NewDockerService() Service {
	var result = &dockerService{
		AbstractService: NewAbstractService(DockerServiceID),
	}
	result.AbstractService.Service = result
	return result
}

//DockerSystemPathRequest represents system path request to set docker in the path
type DockerSystemPathRequest struct {
	SysPath []string
}
