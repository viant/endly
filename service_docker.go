package endly

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
	"strings"
)

const (
	//DockerServiceID represents docker service id
	DockerServiceID = "docker"

	//DockerRunAction represents docker run action
	DockerRunAction = "run"

	//DockerServiceSysPathAction represents docker syspath action
	DockerServiceSysPathAction = "syspath"

	//DockerServiceStopImagesAction represents docker stop-images" action
	DockerServiceStopImagesAction = "stop-images"

	//DockerServiceImagesAction represents docker images action
	DockerServiceImagesAction = "images"

	//DockerServicePullAction represents docker pull action
	DockerServicePullAction = "pull"

	//DockerServiceBuildAction represents docker build action
	DockerServiceBuildAction = "build"

	//DockerServiceTagAction represents docker tag action
	DockerServiceTagAction = "tag"

	//DockerServiceLoginAction represents docker login action
	DockerServiceLoginAction = "login"

	//DockerServiceLogoutAction represents docker logout action
	DockerServiceLogoutAction = "logout"

	//DockerServicePushAction represents docker push action
	DockerServicePushAction = "push"

	//DockerServiceInspectAction represents docker inspect action
	DockerServiceInspectAction = "inspect"

	//DockerServiceContainerCommandAction represents docker container-command action
	DockerServiceContainerCommandAction = "container-command"

	//DockerServiceContainerStartAction represents docker container-start action
	DockerServiceContainerStartAction = "container-start"

	//DockerServiceContainerStopAction represents docker container-stop action
	DockerServiceContainerStopAction = "container-stop"

	//DockerServiceContainerStatusAction represents docker container-status action
	DockerServiceContainerStatusAction = "container-status"

	//DockerServiceContainerRemoveAction represents docker container-remove action
	DockerServiceContainerRemoveAction = "container-remove"

	//DockerServiceContainerLogsAction represents docker container-log action
	DockerServiceContainerLogsAction = "container-logs"

	containerInUse    = "is already in use by container"
	unableToFindImage = "unable to find image"
	dockerError       = "Error response"
	dockerSyntaxError = "syntax error near"
	dockerNotRunning  = "Is the docker daemon running?"
)

var dockerErrors = []string{"failed", unableToFindImage, dockerSyntaxError}
var dockerIgnoreErrors = []string{}

type dockerService struct {
	*AbstractService
	SysPath []string
}

func (s *dockerService) NewRequest(action string) (interface{}, error) {
	switch action {
	case DockerRunAction:
		return &DockerRunRequest{}, nil
	case DockerServiceSysPathAction:
		return struct{}{}, nil
	case DockerServiceStopImagesAction:
		return &DockerStopImagesRequest{}, nil
	case DockerServiceImagesAction:
		return &DockerImagesRequest{}, nil
	case DockerServicePullAction:
		return &DockerPullRequest{}, nil
	case DockerServiceContainerCommandAction:
		return &DockerContainerRunCommandRequest{}, nil
	case DockerServiceContainerStartAction:
		return &DockerContainerStartRequest{}, nil
	case DockerServiceContainerStopAction:
		return &DockerContainerStopRequest{}, nil
	case DockerServiceContainerStatusAction:
		return &DockerContainerStatusRequest{}, nil
	case DockerServiceContainerRemoveAction:
		return &DockerContainerRemoveRequest{}, nil
	case DockerServiceBuildAction:
		return &DockerBuildRequest{}, nil
	case DockerServiceTagAction:
		return &DockerTagRequest{}, nil
	case DockerServiceLoginAction:
		return &DockerLoginRequest{}, nil
	case DockerServiceLogoutAction:
		return &DockerLogoutRequest{}, nil
	case DockerServicePushAction:
		return &DockerPushRequest{}, nil
	case DockerServiceInspectAction:
		return &DockerInspectRequest{}, nil
	case DockerServiceContainerLogsAction:
		return &DockerContainerLogsRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)
}

func (s *dockerService) NewResponse(action string) (interface{}, error) {
	switch action {
	case DockerRunAction:
		return &DockerContainerInfo{}, nil
	case DockerServiceSysPathAction:
		return struct{}{}, nil
	case DockerServiceStopImagesAction:
		return &DockerStopImagesResponse{}, nil
	case DockerServiceImagesAction:
		return &DockerImagesResponse{}, nil
	case DockerServicePullAction:
		return &DockerImageInfo{}, nil
	case DockerServiceContainerCommandAction:
		return &DockerContainerRunCommandResponse{}, nil
	case DockerServiceContainerStartAction:
		return &DockerContainerInfo{}, nil
	case DockerServiceContainerStopAction:
		return &DockerContainerInfo{}, nil
	case DockerServiceContainerStatusAction:
		return &DockerContainerStatusResponse{}, nil
	case DockerServiceContainerRemoveAction:
		return &DockerContainerRemoveResponse{}, nil
	case DockerServiceBuildAction:
		return &DockerBuildResponse{}, nil
	case DockerServiceTagAction:
		return &DockerTagResponse{}, nil
	case DockerServiceLoginAction:
		return &DockerLoginResponse{}, nil
	case DockerServiceLogoutAction:
		return &DockerLogoutResponse{}, nil
	case DockerServicePushAction:
		return &DockerPushResponse{}, nil
	case DockerServiceInspectAction:
		return &DockerInspectResponse{}, nil
	case DockerServiceContainerLogsAction:
		return &DockerContainerLogsResponse{}, nil
	}
	return s.AbstractService.NewResponse(action)
}

func (s *dockerService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err = s.Validate(request, response)
	if err != nil {
		return response
	}
	var errorMessage string
	switch actualRequest := request.(type) {

	case *DockerSystemPathRequest:
		s.SysPath = actualRequest.SysPath
	case *DockerImagesRequest:
		response.Response, err = s.checkImages(context, actualRequest)
		errorMessage = "failed to check images"
	case *DockerPullRequest:
		response.Response, err = s.pullImage(context, actualRequest)
		errorMessage = "failed to pull image"
	case *DockerContainerStatusRequest:
		response.Response, err = s.checkContainerProcesses(context, actualRequest)
		errorMessage = "failed to check process"
	case *DockerContainerRunCommandRequest:
		response.Response, err = s.runInContainer(context, actualRequest)
		errorMessage = "failed to run docker command"
	case *DockerContainerStartRequest:
		response.Response, err = s.startContainer(context, actualRequest)
		errorMessage = "failed start container"
	case *DockerContainerStopRequest:
		response.Response, err = s.stopContainer(context, actualRequest)
		errorMessage = "failed to stop container"
	case *DockerContainerRemoveRequest:
		response.Response, err = s.removeContainer(context, actualRequest)
		errorMessage = "failed to remove container"
	case *DockerRunRequest:
		response.Response, err = s.runContainer(context, actualRequest)
		errorMessage = "failed to run container"
	case *DockerStopImagesRequest:
		response.Response, err = s.stopImages(context, actualRequest)
		errorMessage = "failed to stop images"
	case *DockerBuildRequest:
		response.Response, err = s.build(context, actualRequest)
		errorMessage = "failed to build image"
	case *DockerTagRequest:
		response.Response, err = s.tag(context, actualRequest)
		errorMessage = "failed to tag "
	case *DockerLoginRequest:
		response.Response, err = s.login(context, actualRequest)
		errorMessage = "failed to login"
	case *DockerLogoutRequest:
		response.Response, err = s.logout(context, actualRequest)
		errorMessage = "failed to logout"
	case *DockerPushRequest:
		response.Response, err = s.push(context, actualRequest)
		errorMessage = "failed to push"

	case *DockerInspectRequest:
		response.Response, err = s.inspect(context, actualRequest)
		errorMessage = "failed to inspect"

	case *DockerContainerLogsRequest:
		response.Response, err = s.containerLogs(context, actualRequest)
		errorMessage = "failed to get logs:"

	default:
		err = fmt.Errorf("unsupported request type: %T", request)
	}
	if err != nil {
		jsonRequest, _ := toolbox.AsJSONText(request)
		response.Status = "error"
		response.Error = errorMessage + " " + jsonRequest + ", " + err.Error()
	}
	return response
}

func (s *dockerService) stopImages(context *Context, request *DockerStopImagesRequest) (*DockerStopImagesResponse, error) {
	var response = &DockerStopImagesResponse{
		StoppedImages: make([]string, 0),
	}
	processResponse, err := s.checkContainerProcesses(context, &DockerContainerStatusRequest{
		Target: request.Target,
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
                          new container ID.
    --ID NAME           Assign a ID to the container
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

func (s *dockerService) applySysPathIfNeeded(sysPath []string) {
	if len(sysPath) > 0 {
		s.SysPath = sysPath
	}
	if len(s.SysPath) == 0 {
		s.SysPath = []string{"/usr/local/bin"}
	}
}

func (s *dockerService) applyCredentialIfNeeded(credentials map[string]string) map[string]string {
	var result = make(map[string]string)
	if len(credentials) > 0 {
		for k, v := range credentials {
			result[k] = v
		}
	}
	return result
}

func (s *dockerService) resetContainerIfNeeded(context *Context, target *url.Resource, statusResponse *DockerContainerStatusResponse) error {
	if len(statusResponse.Containers) > 0 {
		_, err := s.stopContainer(context, &DockerContainerStopRequest{
			Target: target,
		})
		if err != nil {
			return err
		}
		_, err = s.removeContainer(context, &DockerContainerRemoveRequest{Target: target})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *dockerService) runContainer(context *Context, request *DockerRunRequest) (*DockerContainerInfo, error) {
	var err error

	var credentials = s.applyCredentialIfNeeded(request.Credentials)

	checkResponse, err := s.checkContainerProcesses(context, &DockerContainerStatusRequest{
		Target: request.Target,
		Names:  request.Target.Name,
	})
	if err == nil {
		err = s.resetContainerIfNeeded(context, request.Target, checkResponse)
	}
	if err != nil {
		return nil, err
	}
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
	commandInfo, err := s.executeSecureDockerCommand(true, credentials, context, request.Target, dockerIgnoreErrors, fmt.Sprintf("docker run --name %v %v -d %v %v", request.Target.Name, args, request.Image, params))
	if err != nil {
		return nil, err
	}

	if strings.Contains(commandInfo.Stdout(), containerInUse) {
		_, _ = s.stopContainer(context, &DockerContainerStopRequest{Target: request.Target})
		_, _ = s.removeContainer(context, &DockerContainerRemoveRequest{Target: request.Target})
		commandInfo, err = s.executeSecureDockerCommand(true, credentials, context, request.Target, dockerErrors, fmt.Sprintf("docker run --name %v %v -d %v", request.Target.Name, args, request.Image))
		if err != nil {
			return nil, err
		}
	}

	return s.checkContainerProcess(context, &DockerContainerStatusRequest{
		Target: request.Target,
		Names:  request.Target.Name,
	})
}

func (s *dockerService) checkContainerProcess(context *Context, request *DockerContainerStatusRequest) (*DockerContainerInfo, error) {

	checkResponse, err := s.checkContainerProcesses(context, request)
	if err != nil {
		return nil, err
	}

	if len(checkResponse.Containers) == 1 {
		return checkResponse.Containers[0], nil
	}
	return nil, nil
}

func (s *dockerService) runContainerCommand(context *Context, securet map[string]string, target *url.Resource, containerCommand, containerCommandOption string, containerCommandArguments ...string) (string, error) {
	target, err := context.ExpandResource(target)
	if err != nil {
		return "", err
	}
	if target.Name == "" {
		return "", fmt.Errorf("target name was empty url: %v", target.URL)
	}
	var command = "docker " + containerCommand

	if containerCommandOption != "" {
		command += " " + containerCommandOption
	}
	command += " " + target.Name
	if len(containerCommandArguments) > 0 {
		command += " " + strings.Join(containerCommandArguments, " ")
	}
	commandResult, err := s.executeSecureDockerCommand(true, securet, context, target, dockerErrors, command)
	if err != nil {
		return "", err
	}
	if len(commandResult.Commands) > 1 {
		//Truncate password auth, to process vanila container output
		var stdout = commandResult.Commands[0].Stdout
		if strings.Contains(stdout, "Password:") {
			commandResult.Commands = commandResult.Commands[1:]
		}
	}

	return commandResult.Stdout(), nil
}

func (s *dockerService) startContainer(context *Context, request *DockerContainerStartRequest) (*DockerContainerInfo, error) {
	_, err := s.runContainerCommand(context, nil, request.Target, "start", "")
	if err != nil {
		return nil, err
	}
	return s.checkContainerProcess(context, &DockerContainerStatusRequest{
		Target: request.Target,
		Names:  request.Target.Name,
	})

}

func (s *dockerService) stopContainer(context *Context, request *DockerContainerStopRequest) (*DockerContainerInfo, error) {
	info, err := s.checkContainerProcess(context, &DockerContainerStatusRequest{
		Target: request.Target,
		Names:  request.Target.Name,
	})
	if err != nil || info == nil {
		return nil, err
	}
	_, err = s.runContainerCommand(context, nil, request.Target, "stop", "")
	if err != nil {
		return nil, err
	}
	if info != nil {
		info.Status = "down"
	}
	return info, nil
}

func (s *dockerService) removeContainer(context *Context, request *DockerContainerRemoveRequest) (response *DockerContainerRemoveResponse, err error) {
	response = &DockerContainerRemoveResponse{}
	response.Stdout, err = s.runContainerCommand(context, nil, request.Target, "rm", "")
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *dockerService) inspect(context *Context, request *DockerInspectRequest) (response *DockerInspectResponse, err error) {
	response = &DockerInspectResponse{}
	response.Stdout, err = s.runContainerCommand(context, nil, request.Target, "inspect", "")
	if err != nil {
		return nil, err
	}
	_, structured := AsExtractable(response.Stdout)
	response.Info = structured[SliceKey]
	return response, nil
}

func (s *dockerService) containerLogs(context *Context, request *DockerContainerLogsRequest) (response *DockerContainerLogsResponse, err error) {
	response = &DockerContainerLogsResponse{}
	response.Stdout, err = s.runContainerCommand(context, nil, request.Target, "logs", "")
	return response, err
}

func (s *dockerService) runInContainer(context *Context, request *DockerContainerRunCommandRequest) (response *DockerContainerRunCommandResponse, err error) {
	response = &DockerContainerRunCommandResponse{}
	var executionOptions = ""
	var execArguments = context.Expand(request.Command)

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
	response.Stdout, err = s.runContainerCommand(context, request.Credentials, request.Target, "exec", executionOptions, execArguments)
	return response, err
}

func (s *dockerService) checkContainerProcesses(context *Context, request *DockerContainerStatusRequest) (*DockerContainerStatusResponse, error) {

	info, err := s.executeSecureDockerCommand(true, nil, context, request.Target, dockerErrors, "docker ps")
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
			Command:     strings.Trim(columns[2], "\""),
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
	return &DockerContainerStatusResponse{Containers: containers}, nil
}

func (s *dockerService) pullImage(context *Context, request *DockerPullRequest) (*DockerImageInfo, error) {
	if request.Tag == "" {
		request.Tag = "latest"
	}
	info, err := s.executeSecureDockerCommand(true, nil, context, request.Target, dockerErrors, fmt.Sprintf("docker pull %v:%v", request.Repository, request.Tag))
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()
	if strings.Contains(stdout, "not found") {
		return nil, fmt.Errorf("failed to pull docker image,  %v", stdout)
	}
	imageResponse, err := s.checkImages(context, &DockerImagesRequest{Target: request.Target, Repository: request.Repository, Tag: request.Tag})
	if err != nil {
		return nil, err
	}
	if len(imageResponse.Images) == 1 {
		return imageResponse.Images[0], nil
	}
	return nil, fmt.Errorf("failed to check image status: %v:%v found: %v", request.Repository, request.Tag, len(imageResponse.Images))
}

func (s *dockerService) checkImages(context *Context, request *DockerImagesRequest) (*DockerImagesResponse, error) {
	info, err := s.executeSecureDockerCommand(true, nil, context, request.Target, dockerErrors, "docker images")
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()
	var images = make([]*DockerImageInfo, 0)

	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "REPOSITORY") {
			continue
		}
		columns, ok := ExtractColumns(line)
		if !ok || len(columns) < 4 {
			continue
		}

		var size = ""
		for j := len(columns) - 2; j < len(columns); j++ {
			if strings.Contains(columns[j], "ago") {
				continue
			}
			size += columns[j]
		}
		var sizeFactor = 1024
		var sizeUnit = "KB"
		if strings.Contains(line, "MB") {
			sizeUnit = "MB"
			sizeFactor *= 1024
		}

		size = strings.Replace(size, sizeUnit, "", 1)
		info := &DockerImageInfo{
			Repository: columns[0],
			Tag:        columns[1],
			ImageID:    columns[2],
			Size:       toolbox.AsInt(size) * sizeFactor,
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

func (s *dockerService) executeDockerCommand(secure map[string]string, context *Context, target *url.Resource, errors []string, template string, arguments ...interface{}) (*CommandResponse, error) {
	return s.executeSecureDockerCommand(false, secure, context, target, errors, fmt.Sprintf(template, arguments...))
}

func (s *dockerService) startDockerIfNeeded(context *Context, target *url.Resource) {
	daemonService, _ := context.Service(DaemonServiceID)
	daemonService.Run(context, &DaemonStartRequest{
		Target:  target,
		Service: "docker",
	})

}

func (s *dockerService) executeSecureDockerCommand(asRoot bool, secure map[string]string, context *Context, target *url.Resource, errors []string, command string) (*CommandResponse, error) {
	s.applySysPathIfNeeded([]string{})
	if len(secure) == 0 {
		secure = make(map[string]string)
	}
	secure[sudoCredentialKey] = target.Credential
	command = strings.Replace(command, "\n", " ", len(command))
	var extractableCommand = &ExtractableCommand{
		Options: &ExecutionOptions{
			SystemPaths: s.SysPath,
			TimeoutMs:   120000,
		},
		Executions: []*Execution{
			{
				Credentials: secure,
				Command:     command,
				Errors:      append(errors, []string{commandNotFound}...),
			},
		},
	}
	var runRequest interface{} = extractableCommand
	if asRoot {
		runRequest = &superUserCommandRequest{
			MangedCommand: extractableCommand,
		}
	}
	response, err := context.Execute(target, runRequest)

	if err != nil {
		if escapedContains(err.Error(), commandNotFound) {
			return nil, err
		}
		if response != nil && ! escapedContains(response.Stdout(), dockerNotRunning) {
			return nil, err
		}
		s.startDockerIfNeeded(context, target)
		response, err = context.Execute(target, runRequest)
		if err != nil {
			return nil, err
		}

	}
	var stdout = response.Stdout()

	if strings.Contains(stdout, containerInUse) {
		return response, nil
	}
	if strings.Contains(stdout, dockerError) {
		return response, fmt.Errorf("error executing %v, %v", command, vtclean.Clean(stdout, false))
	}
	return response, nil
}

func (s *dockerService) build(context *Context, request *DockerBuildRequest) (*DockerBuildResponse, error) {
	request.Init()
	var response = &DockerBuildResponse{}
	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var args = ""
	for k, v := range request.Arguments {
		args += fmt.Sprintf("%v %v ", k, context.Expand(v))
	}
	if request.Path == "" {
		request.Path = "."
	}

	commandInfo, err := s.executeDockerCommand(nil, context, target, dockerIgnoreErrors, fmt.Sprintf("docker build %v %v", args, request.Path))
	if err != nil {
		return nil, err
	}
	response.Stdout = commandInfo.Stdout()
	if !escapedContains(response.Stdout, "Successfully built") {
		return nil, fmt.Errorf("failed to build: %v, stdout:%v", request.Tag, response.Stdout)
	}
	return response, nil
}
func (s *dockerService) tag(context *Context, request *DockerTagRequest) (*DockerTagResponse, error) {
	var response = &DockerTagResponse{}

	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	commandInfo, err := s.executeDockerCommand(nil, context, target, dockerIgnoreErrors, fmt.Sprintf("docker tag %v %v", request.SourceTag, request.TargetTag))
	if err != nil {
		return nil, err
	}
	response.Stdout = commandInfo.Stdout()
	return response, nil
}

//IsGoogleCloudRegistry returns true if url is google docker cloud registry
func IsGoogleCloudRegistry(URL string) bool {
	return strings.Contains(URL, "gcr.io")
}

func (s *dockerService) getGoogleCloudCredential(context *Context, credential string, config *cred.Config) *cred.Config {
	var result = &cred.Config{
		Username: "oauth2accesstoken",
		Password: "$(gcloud auth application-default print-access-token)",
	}
	if config.PrivateKeyID != "" && config.PrivateKey != "" {
		content, _ := url.NewResource(credential).DownloadText()
		result.Username = "_json_key"
		result.Password = strings.Replace(content, "\n", " ", len(content))
	}
	return result
}

/**
on osx when hitting Errors saving credentials: error storing credentials - err: exit status 1, out: `User interaction is not allowed.`
on docker service -> preferences -> and I untick "Securely store docker logins in macOS keychain" this problem goes away.
*/
func (s *dockerService) login(context *Context, request *DockerLoginRequest) (*DockerLoginResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	var response = &DockerLoginResponse{}
	credential := context.Expand(request.Credential)
	credConfig, err := cred.NewConfig(credential)
	repository := context.Expand(request.Repository)
	if IsGoogleCloudRegistry(repository) {
		credConfig = s.getGoogleCloudCredential(context, credential, credConfig)
		credential = credConfig.Password
	}
	if credConfig.Username == "" {
		return nil, fmt.Errorf("username was empty: %v", credential)
	}
	if credConfig.Password == "" {
		return nil, fmt.Errorf("password was empty: %v", credential)
	}
	credentials := map[string]string{
		"**docker-secret**": credential,
	}
	commandResponse, err := s.executeSecureDockerCommand(true, credentials, context, target, dockerErrors, fmt.Sprintf(`echo '**docker-secret**' | sudo docker login -u %v  %v --password-stdin`, credConfig.Username, repository))
	if err != nil {
		return nil, err
	}

	stdout := commandResponse.Stdout()
	if !escapedContains(stdout, "Login Succeeded") {
		return nil, fmt.Errorf("failed to authenticate: %v, stdout: %v", response.Username, stdout)
	}
	response.Username = credConfig.Username
	response.Stdout = stdout
	return response, nil
}

func (s *dockerService) logout(context *Context, request *DockerLogoutRequest) (*DockerLogoutResponse, error) {
	var response = &DockerLogoutResponse{}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	repository := context.Expand(request.Repository)
	_, err = s.executeSecureDockerCommand(true, nil, context, target, dockerErrors, fmt.Sprintf(`docker logout %v`, repository))
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *dockerService) push(context *Context, request *DockerPushRequest) (*DockerPushResponse, error) {
	var response = &DockerPushResponse{}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	commandResponse, err := s.executeSecureDockerCommand(true, nil, context, target, dockerErrors, fmt.Sprintf(`docker push %v`, request.Tag))
	if err != nil {
		return nil, err
	}
	stdout := commandResponse.Stdout()
	if !(escapedContains(stdout, "Pushed") || escapedContains(stdout, "Layer already exists")) {
		return nil, fmt.Errorf("failed to push tag: %v, stdout: %v", request.Tag, stdout)
	}
	return response, nil
}

//NewDockerService returns a new docker service.
func NewDockerService() Service {
	var result = &dockerService{
		AbstractService: NewAbstractService(DockerServiceID,
			DockerRunAction,
			DockerServiceSysPathAction,
			DockerServiceStopImagesAction,
			DockerServiceImagesAction,
			DockerServicePullAction,
			DockerServiceContainerCommandAction,
			DockerServiceContainerStartAction,
			DockerServiceContainerStopAction,
			DockerServiceContainerStatusAction,
			DockerServiceContainerRemoveAction,
			DockerServiceContainerLogsAction,
			DockerServiceBuildAction,
			DockerServiceTagAction,
			DockerServiceLoginAction,
			DockerServiceLogoutAction,
			DockerServicePushAction,
			DockerServiceInspectAction,
		),
	}
	result.AbstractService.Service = result
	return result
}

//DockerSystemPathRequest represents system path request to set docker in the path
type DockerSystemPathRequest struct {
	SysPath []string
}
