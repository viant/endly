package docker

import (
	"fmt"
	"strings"

	"github.com/lunixbochs/vtclean"
	"github.com/viant/endly"
	"github.com/viant/endly/system/daemon"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/secret"
	"github.com/viant/toolbox/url"
)

const (
	//ServiceID represents docker service id
	ServiceID = "docker"

	containerInUse    = "is already in use by container"
	unableToFindImage = "unable to find image"
	dockerError       = "Error response"
	dockerSyntaxError = "syntax error near"
	dockerNotRunning  = "Is the docker daemon running?"
)

var dockerErrors = []string{"failed", unableToFindImage, dockerSyntaxError}
var dockerIgnoreErrors = []string{}

type service struct {
	*endly.AbstractService
	SysPath []string
}

func (s *service) stopImages(context *endly.Context, request *StopImagesRequest) (*StopImagesResponse, error) {
	var response = &StopImagesResponse{
		StoppedImages: make([]string, 0),
	}
	processResponse, err := s.checkContainerProcesses(context, &ContainerStatusRequest{
		Target: request.Target,
	})
	if err != nil {
		return nil, err
	}

	for _, image := range request.Images {
		for _, container := range processResponse.Containers {
			if strings.Contains(container.Image, image) {
				var name = strings.Split(container.Names, ",")[0]
				_, err = s.stopContainer(context, &StopRequest{
					BaseRequest: &BaseRequest{
						Target: request.Target,
						Name:   name,
					},
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

func (s *service) applySysPathIfNeeded(sysPath []string) {
	if len(sysPath) > 0 {
		s.SysPath = sysPath
	}
	if len(s.SysPath) == 0 {
		s.SysPath = []string{"/usr/local/bin"}
	}
}

func (s *service) applyCredentialIfNeeded(credentials map[string]string) map[string]string {
	var result = make(map[string]string)
	if len(credentials) > 0 {
		for k, v := range credentials {
			result[k] = v
		}
	}
	return result
}

func (s *service) resetContainerIfNeeded(context *endly.Context, request *RunRequest, statusResponse *ContainerStatusResponse) error {
	if len(statusResponse.Containers) > 0 {
		_, err := s.stopContainer(context, &StopRequest{
			BaseRequest: &BaseRequest{
				Target: request.Target,
				Name:   request.Name,
			},
		})
		if err != nil {
			return err
		}
		_, err = s.removeContainer(context, &RemoveRequest{
			BaseRequest: &BaseRequest{
				Target: request.Target,
				Name:   request.Name,
			}})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) runContainer(context *endly.Context, request *RunRequest) (*RunResponse, error) {
	var err error
	var credentials = s.applyCredentialIfNeeded(request.Secrets)
	checkResponse, err := s.checkContainerProcesses(context, &ContainerStatusRequest{
		Target: request.Target,
		Names:  request.Name,
	})
	if err == nil && !request.Reuse {
		err = s.resetContainerIfNeeded(context, request, checkResponse)
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
	for k, v := range request.Ports {
		args += fmt.Sprintf("-p %v:%v ", context.Expand(toolbox.AsString(k)), context.Expand(toolbox.AsString(v)))
	}
	if request.Workdir != "" {
		args += fmt.Sprintf("-w %v ", context.Expand(request.Workdir))
	}

	var params = ""
	for k, v := range request.Params {
		params += fmt.Sprintf("%v %v", k, v)
	}
	commandInfo, err := s.executeSecureDockerCommand(true, credentials, context, request.Target, dockerIgnoreErrors, fmt.Sprintf("docker run --name %v %v -d %v %v", request.Name, args, request.Image, params))
	if err != nil {
		return nil, err
	}

	if strings.Contains(commandInfo.Stdout(), containerInUse) {
		if request.Reuse {
			startResponse, err := s.startContainer(context, &StartRequest{BaseRequest: &BaseRequest{
				Target: request.Target,
				Name:   request.Name,
			}})
			if err != nil {
				return nil, err
			}
			return &RunResponse{startResponse.ContainerInfo}, err
		}
		_, _ = s.stopContainer(context, &StopRequest{BaseRequest: &BaseRequest{
			Target: request.Target,
			Name:   request.Name,
		}})
		_, _ = s.removeContainer(context, &RemoveRequest{BaseRequest: &BaseRequest{
			Target: request.Target,
			Name:   request.Name,
		}})
		commandInfo, err = s.executeSecureDockerCommand(true, credentials, context, request.Target, dockerErrors, fmt.Sprintf("docker run --name %v %v -d %v %v", request.Name, args, request.Image, params))
		if err != nil {
			return nil, err
		}
	}

	info, err := s.checkContainerProcess(context, &ContainerStatusRequest{
		Target: request.Target,
		Names:  request.Name,
	})
	if info == nil {
		return nil, err
	}
	return &RunResponse{info}, err
}

func (s *service) checkContainerProcess(context *endly.Context, request *ContainerStatusRequest) (*ContainerInfo, error) {
	checkResponse, err := s.checkContainerProcesses(context, request)
	if err != nil {
		return nil, err
	}
	if len(checkResponse.Containers) > 0 {
		return checkResponse.Containers[0], nil
	}
	return nil, nil
}

func (s *service) runContainerCommand(context *endly.Context, securet map[string]string, instance string, target *url.Resource, containerCommand, containerCommandOption string, containerCommandArguments ...string) (string, error) {
	target, err := context.ExpandResource(target)
	if err != nil {
		return "", err
	}
	var command = "docker " + containerCommand

	if containerCommandOption != "" {
		command += " " + containerCommandOption
	}
	command += " " + instance
	if len(containerCommandArguments) > 0 {
		command += " " + strings.Join(containerCommandArguments, " ")
	}
	commandResult, err := s.executeSecureDockerCommand(true, securet, context, target, dockerErrors, command)
	if err != nil {
		return "", err
	}
	if len(commandResult.Cmd) > 1 {
		//Truncate password auth, to process vanila container output
		var stdout = commandResult.Cmd[0].Stdout
		if strings.Contains(stdout, "Password:") {
			commandResult.Cmd = commandResult.Cmd[1:]
		}
	}

	return commandResult.Stdout(), nil
}

func (s *service) startContainer(context *endly.Context, request *StartRequest) (*StartResponse, error) {
	_, err := s.runContainerCommand(context, nil, request.Name, request.Target, "start", "")
	if err != nil {
		return nil, err
	}
	info, err := s.checkContainerProcess(context, &ContainerStatusRequest{
		Target: request.Target,
		Names:  request.Name,
	})
	if info == nil {
		return nil, err
	}
	return &StartResponse{info}, err
}

func (s *service) stopContainer(context *endly.Context, request *StopRequest) (*StopResponse, error) {
	info, err := s.checkContainerProcess(context, &ContainerStatusRequest{
		Target: request.Target,
		Names:  request.Name,
	})
	if err != nil || info == nil {
		return nil, err
	}
	_, err = s.runContainerCommand(context, nil, request.Name, request.Target, "stop", "")
	if err != nil {
		return nil, err
	}
	if info != nil {
		info.Status = "down"
	}

	return &StopResponse{info}, nil
}

func (s *service) removeContainer(context *endly.Context, request *RemoveRequest) (response *RemoveResponse, err error) {
	response = &RemoveResponse{}
	response.Stdout, err = s.runContainerCommand(context, nil, request.Name, request.Target, "rm", "")
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *service) inspect(context *endly.Context, request *InspectRequest) (response *InspectResponse, err error) {
	response = &InspectResponse{}
	response.Stdout, err = s.runContainerCommand(context, nil, request.Name, request.Target, "inspect", "")
	if err != nil {
		return nil, err
	}
	_, structured := util.AsExtractable(response.Stdout)
	response.Info = structured["value"]
	return response, nil
}

func (s *service) containerLogs(context *endly.Context, request *LogsRequest) (response *LogsResponse, err error) {
	response = &LogsResponse{}
	response.Stdout, err = s.runContainerCommand(context, nil, request.Name, request.Target, "logs", "")
	return response, err
}

func (s *service) runInContainer(context *endly.Context, request *ExecRequest) (response *ExecResponse, err error) {
	response = &ExecResponse{}
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
	response.Stdout, err = s.runContainerCommand(context, request.Secrets, request.Name, request.Target, "exec", executionOptions, execArguments)
	return response, err
}

func (s *service) checkContainerProcesses(context *endly.Context, request *ContainerStatusRequest) (*ContainerStatusResponse, error) {

	info, err := s.executeSecureDockerCommand(true, nil, context, request.Target, dockerErrors, "docker ps")
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()

	var containers = make([]*ContainerInfo, 0)
	var lines = strings.Split(stdout, "\r\n")
	for i := 1; i < len(lines); i++ {
		columns, ok := util.ExtractColumns(lines[i])
		if !ok || len(columns) < 7 {
			continue
		}
		var status = "down"
		if strings.Contains(lines[i], "Up") {
			status = "up"
		}
		info := &ContainerInfo{
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
	return &ContainerStatusResponse{Containers: containers}, nil
}

func (s *service) pullImage(context *endly.Context, request *PullRequest) (*PullResponse, error) {
	if request.Tag == "" {
		request.Tag = "latest"
	}
	info, err := s.executeSecureDockerCommand(true, nil, context, request.Target, dockerErrors, fmt.Sprintf("docker pull %v:%v", request.Repository, request.Tag))
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()
	if strings.Contains(stdout, "not found") {
		return nil, fmt.Errorf("%v", stdout)
	}
	imageResponse, err := s.checkImages(context, &ImagesRequest{Target: request.Target, Repository: request.Repository, Tag: request.Tag})
	if err != nil {
		return nil, err
	}
	if len(imageResponse.Images) > 0 {
		return &PullResponse{imageResponse.Images[0]}, nil
	}
	return nil, fmt.Errorf("not found:  %v %v", request.Repository, request.Tag)
}

func (s *service) checkImages(context *endly.Context, request *ImagesRequest) (*ImagesResponse, error) {
	info, err := s.executeSecureDockerCommand(true, nil, context, request.Target, dockerErrors, "docker images")
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()
	var images = make([]*ImageInfo, 0)

	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "REPOSITORY") {
			continue
		}
		columns, ok := util.ExtractColumns(line)
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
		info := &ImageInfo{
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
	return &ImagesResponse{Images: images}, nil

}

func (s *service) executeDockerCommand(secure map[string]string, context *endly.Context, target *url.Resource, errors []string, template string, arguments ...interface{}) (*exec.RunResponse, error) {
	return s.executeSecureDockerCommand(false, secure, context, target, errors, fmt.Sprintf(template, arguments...))
}

func (s *service) startDockerIfNeeded(context *endly.Context, target *url.Resource) {
	daemonService, _ := context.Service(daemon.ServiceID)
	daemonService.Run(context, &daemon.StartRequest{
		Target:  target,
		Service: "docker",
	})

}

func (s *service) executeSecureDockerCommand(asRoot bool, secrets map[string]string, context *endly.Context, target *url.Resource, errors []string, command string) (*exec.RunResponse, error) {
	s.applySysPathIfNeeded([]string{})
	if len(secrets) == 0 {
		secrets = make(map[string]string)
	}
	secrets[exec.SudoCredentialKey] = target.Credentials
	command = strings.Replace(command, "\n", " ", len(command))

	var extractRequest = exec.NewExtractRequest(target, exec.DefaultOptions(),
		exec.NewExtractCommand(command, "", nil, []string{util.CommandNotFound}))

	extractRequest.TimeoutMs = 120000
	extractRequest.SystemPaths = s.SysPath
	extractRequest.Secrets = secret.NewSecrets(secrets)
	extractRequest.SuperUser = asRoot

	var runResponse = &exec.RunResponse{}

	err := endly.Run(context, extractRequest, runResponse)
	if err != nil {
		if util.CheckCommandNotFound(err.Error()) {
			return nil, err
		}
		if runResponse != nil && !util.EscapedContains(runResponse.Stdout(), dockerNotRunning) {
			return nil, err
		}
		s.startDockerIfNeeded(context, target)

		if err := endly.Run(context, extractRequest, runResponse); err != nil {
			return nil, err
		}
	}

	var stdout = runResponse.Stdout()
	if strings.Contains(stdout, containerInUse) {
		return runResponse, nil
	}
	if strings.Contains(stdout, dockerError) {
		return runResponse, fmt.Errorf("error executing %v, %v", command, vtclean.Clean(stdout, false))
	}
	return runResponse, nil
}

func (s *service) build(context *endly.Context, request *BuildRequest) (*BuildResponse, error) {
	request.Init()
	var response = &BuildResponse{}
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
	if !util.EscapedContains(response.Stdout, "Successfully built") {
		return nil, fmt.Errorf("failed to build: %v, stdout:%v", request.Tag, response.Stdout)
	}
	return response, nil
}
func (s *service) tag(context *endly.Context, request *TagRequest) (*TagResponse, error) {
	var response = &TagResponse{}

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

func (s *service) getGoogleCloudCredential(context *endly.Context, credentials string, config *cred.Config) *cred.Config {
	var result = &cred.Config{
		Username: "oauth2accesstoken",
		Password: "$(gcloud auth application-default print-access-token)",
	}
	if config.PrivateKeyID != ""  {
		result.Username = "_json_key"
		result.Password = strings.Replace(config.Data, "\n", " ", len(config.Data))
	}
	return result
}



func (s *service) runDockerProcessChecklist(context *endly.Context, target *url.Resource) (string, error) {
	var extractRequest = exec.NewExtractRequest(target, exec.DefaultOptions(),
		exec.NewExtractCommand("docker ps", "", nil, nil))
	extractRequest.SuperUser = true
	extractRequest.SystemPaths = []string{"/usr/local/bin"}
	var runResponse = &exec.RunResponse{}
	err := endly.Run(context, extractRequest, &runResponse)
	if err != nil {
		return "", err
	}
	return runResponse.Output, nil
}

/**
on osx when hitting Errors saving credentials: error storing credentials - err: exit status 1, out: `User interaction is not allowed.`
on docker service -> preferences -> and I untick "Securely store docker logins in macOS keychain" this problem goes away.
*/
func (s *service) login(context *endly.Context, request *LoginRequest) (*LoginResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	if _, err = s.runDockerProcessChecklist(context, target); err != nil {
		return nil, err
	}

	var response = &LoginResponse{}
	credentials := context.Expand(request.Credentials)
	credConfig, err := context.Secrets.GetCredentials(credentials)
	if err != nil {
		return nil, err
	}
	repository := context.Expand(request.Repository)
	if IsGoogleCloudRegistry(repository) {
		credConfig = s.getGoogleCloudCredential(context, credentials, credConfig)
		credentials = credConfig.Password
	}
	if credConfig.Username == "" {
		return nil, fmt.Errorf("username was empty: %v", credentials)
	}
	if credConfig.Password == "" {
		return nil, fmt.Errorf("password was empty: %v", credentials)
	}

	secrets := map[string]string{
		"**docker-secret**": credentials,
	}
	if strings.Contains(repository, "hub.docker.com") {
		repository = ""
	}
	commandResponse, err := s.executeSecureDockerCommand(true, secrets, context, target, dockerErrors, fmt.Sprintf(`echo '**docker-secret**' | sudo docker login -u %v  %v --password-stdin`, credConfig.Username, repository))
	if err != nil {
		return nil, err
	}

	stdout := commandResponse.Stdout()
	if !util.EscapedContains(stdout, "Login Succeeded") {
		return nil, fmt.Errorf("failed to authenticate: %v, stdout: %v", response.Username, stdout)
	}
	response.Username = credConfig.Username
	response.Stdout = stdout
	return response, nil
}

func (s *service) logout(context *endly.Context, request *LogoutRequest) (*LogoutResponse, error) {
	var response = &LogoutResponse{}
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

func (s *service) push(context *endly.Context, request *PushRequest) (*PushResponse, error) {
	var response = &PushResponse{}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	commandResponse, err := s.executeSecureDockerCommand(true, nil, context, target, dockerErrors, fmt.Sprintf(`docker push %v`, request.Tag))
	if err != nil {
		return nil, err
	}
	stdout := commandResponse.Stdout()
	if !(util.EscapedContains(stdout, "Pushed") || util.EscapedContains(stdout, "Layer already exists")) {
		return nil, fmt.Errorf("failed to push tag: %v, stdout: %v", request.Tag, stdout)
	}
	return response, nil
}

/*
	Build, re-create and start docker services and their linked/dependent services.
	This will force stop any previous containers and recreates all containers.
	ComposeError is returned on any failure
*/
func (s *service) composeUp(context *endly.Context, request *ComposeRequestUp) (*ComposeResponse, error) {
	//Expand variables
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, NewComposeError(err.Error(), request.Source)
	}
	composePath, err := context.ExpandResource(request.Source)
	if err != nil {
		return nil, NewComposeError(err.Error(), request.Source)
	}

	//Build & execute command
	command := "docker-compose -f " + composePath.ParsedURL.Path + " up"
	if request.RunInBackground {
		command = command + " -d"
	}
	_, e := s.executeSecureDockerCommand(true, request.Credentials, context, target, dockerErrors, command)
	if e != nil {
		return nil, NewComposeError(err.Error(), request.Source)
	}

	//Check if the services are running
	response := &ComposeResponse{Containers: make([]*ContainerInfo, 0)}
	compose, err := mapToComposeStructureFromURL(request.Source.URL)
	if compose.Services != nil {
		for k := range compose.Services {
			if statusResponse, err := s.checkContainerProcess(context, &ContainerStatusRequest{Target: request.Target, Names: k}); err == nil && statusResponse != nil {
				response.Containers = append(response.Containers, statusResponse)
			}
		}
	}

	return response, nil
}

/*
	Stop all the services that were brought up by compose up
*/
func (s *service) composeDown(context *endly.Context, request *ComposeRequestDown) (*ComposeResponse, error) {
	//Expand variables
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, NewComposeError(err.Error(), request.Source)
	}
	composePath, err := context.ExpandResource(request.Source)
	if err != nil {
		return nil, NewComposeError(err.Error(), request.Source)
	}

	//Build & execute command
	command := "docker-compose -f " + composePath.ParsedURL.Path + " down"
	_, e := s.executeSecureDockerCommand(true, request.Credentials, context, target, dockerErrors, command)
	if e != nil {
		return nil, NewComposeError(err.Error(), request.Source)
	}

	response := &ComposeResponse{Containers: make([]*ContainerInfo, 0)}
	compose, err := mapToComposeStructureFromURL(request.Source.URL)
	if compose.Services != nil {
		for k := range compose.Services {
			if statusResponse, err := s.checkContainerProcess(context, &ContainerStatusRequest{Target: request.Target, Names: k}); err == nil && statusResponse != nil {
				response.Containers = append(response.Containers, statusResponse)
			}
		}
	}

	return response, nil
}

const (
	dockerServiceRunExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "Name": "udb_aerospike",
  "Image": "aerospike/aerospike-server:latest",
  "Mount": {
    "/tmp/aerospikeudb_aerospike.conf": "/etc/aerospike/aerospike.conf"
  },
  "Ports": {
    "3000": "3000",
    "3001": "3001",
    "3002": "3002",
    "3004": "3004",
    "8081": "8081"
  }
}`

	dockerServiceStopImagesExample = `{
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "Images": [
    "aerospike",
    "mysql"
  ]
}`

	dockerServiceImagesExample = `{
    "Target": {
		"URL": "ssh://127.0.0.1/",
		"Credentials": "${env.HOME}/.secret/localhost.json"
    },
	"Repository": "mysql",
	"Tag"":        "5.6"
}`

	dockerServicePullExample = `{
    "Target": {
		"URL": "ssh://127.0.0.1/",
		"Credentials": "${env.HOME}/.secret/localhost.json"
    },
	"Repository": "aerospike",
	"Tag"":        "latest"
}`

	dockerServiceBuildExample = `{
  "Target": {
    "URL": "ssh://127.0.0.1/Projects/store_backup/app",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "Tag": {
    "Username": "viant",
    "Image": "store_backup",
    "Version": "0.1.2"
  },
  "Path": "/Projects/store_backup/app"
}`

	dockerServiceTagExample = `{
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
   
  },
  "SourceTag": {
    "Username": "viant",
    "Registry": "",
    "Image": "store_backup",
    "Version": "0.1.2"
  },
  "TargetTag": {
    "Username": "",
    "Registry": "us.gcr.io/xxxx",
    "Image": "store_backup",
    "Version": "0.1.2"
  }
}`

	dockerServicePushExample = `{
  "Target": {
    "URL": "scp://127.0.0.1//Projects//store_backup/app",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "Tag": {
    "Username": "",
    "Registry": "us.gcr.io/tech-ops-poc",
    "Image": "site_profile_backup",
    "Version": "0.1.2\n"
  }
}`

	dockerServiceLoginExample = `{
  "Target": {
    "URL": "ssh://10.10.1.1/",
    "Credentials": "${env.HOME}/.secret/aws-west.json"
  },
  "Credentials": "${env.HOME}/.secret/docker.json",
  "Repository": "us.gcr.io/xxxxx"
}`

	dockerServiceLogoutExample = `{
  "Target": {
    "URL": "ssh://10.10.1.1/",
    "Credentials": "${env.HOME}/.secret/aws-west.json"
  },
  "Credentials": "${env.HOME}/.secret/docker.json",
  "Repository": "us.gcr.io/xxxxx"
}`

	dockerServiceContainerRunMsqlDumpExample = `{
  "Target": {
    "URL": "ssh://10.10.1.1/",
    "Credentials": "${env.HOME}/.secret/aws-west.json"
  },
  "Name": "mydb1",
  "Interactive": true,
  "AllocateTerminal": true,
  "Command": "mysqldump  -uroot -p***mysql*** --all-databases --routines | grep -v 'Warning' > /tmp/dump.sql",
  "Secrets": {
    "***mysql***": "${env.HOME}/.secret/aws-west-mysql.json"
  }
}`

	dockerServiceContainerRunMysqlImportExample = `{
  "Target": {
    "URL": "ssh://10.10.1.1/",
    "Credentials": "${env.HOME}/.secret/aws-west.json"
  },
  "Name": "mydb1",
  "Interactive": true,
  "Command": "mysql  -uroot -p**mysql** < /tmp/dump.sql",
  "Secrets": {
    "***mysql***": "${env.HOME}/.secret/aws-west-mysql.json"
  }
}`

	dockerServiceContainerExample = `{
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "Name": "udb_aerospike"
}`

	dockerServiceComposeUpExample = `{
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "Source": "test/compose/up/docker-compose.yaml"
}`

	dockerServiceComposeDownExample = `{
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "Source": "test/compose/down/docker-compose.yaml"
}`
)

func (s *service) registerRoutes() {

	s.Register(&endly.Route{
		Action: "run",
		RequestInfo: &endly.ActionInfo{
			Description: "run docker image",

			Examples: []*endly.UseCase{
				{
					Description: "run docker image on the target host",
					Data:        dockerServiceRunExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &RunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RunRequest); ok {
				return s.runContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "stop-images",
		RequestInfo: &endly.ActionInfo{
			Description: "stops docker container matching supplied images",

			Examples: []*endly.UseCase{
				{
					Description: "stop images",
					Data:        dockerServiceStopImagesExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &StopImagesRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StopImagesResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StopImagesRequest); ok {
				return s.stopImages(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "images",
		RequestInfo: &endly.ActionInfo{
			Description: "return images info for supplied matching images",

			Examples: []*endly.UseCase{
				{
					Description: "check image",
					Data:        dockerServiceImagesExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &ImagesRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ImagesResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ImagesRequest); ok {
				return s.checkImages(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "pull",
		RequestInfo: &endly.ActionInfo{
			Description: "pull docker image",

			Examples: []*endly.UseCase{
				{
					Description: "pull example",
					Data:        dockerServicePullExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &PullRequest{}
		},
		ResponseProvider: func() interface{} {
			return &PullResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PullRequest); ok {
				return s.pullImage(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "build",
		RequestInfo: &endly.ActionInfo{
			Description: "build docker image",

			Examples: []*endly.UseCase{
				{
					Description: "build image",
					Data:        dockerServiceBuildExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &BuildRequest{}
		},
		ResponseProvider: func() interface{} {
			return &BuildResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*BuildRequest); ok {
				return s.build(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "tag",
		RequestInfo: &endly.ActionInfo{
			Description: "tag docker image",

			Examples: []*endly.UseCase{
				{
					Description: "tag image",
					Data:        dockerServiceTagExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &TagRequest{}
		},
		ResponseProvider: func() interface{} {
			return &TagResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*TagRequest); ok {
				return s.tag(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "login",
		RequestInfo: &endly.ActionInfo{
			Description: "add credentials for supplied docker repository, required docker 17+ for secure credentials handling",
			Examples: []*endly.UseCase{
				{
					Description: "login ",
					Data:        dockerServiceLoginExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &LoginRequest{}
		},
		ResponseProvider: func() interface{} {
			return &LoginResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*LoginRequest); ok {
				return s.login(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "logout",
		RequestInfo: &endly.ActionInfo{
			Description: "remove credentials for supplied docker repository",
			Examples: []*endly.UseCase{
				{
					Description: "logout ",
					Data:        dockerServiceLogoutExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &LogoutRequest{}
		},
		ResponseProvider: func() interface{} {
			return &LogoutResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*LogoutRequest); ok {
				return s.logout(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "push",
		RequestInfo: &endly.ActionInfo{
			Description: "push docker image into docker repository",
			Examples: []*endly.UseCase{
				{
					Description: "push ",
					Data:        dockerServicePushExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &PushRequest{}
		},
		ResponseProvider: func() interface{} {
			return &PushResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PushRequest); ok {
				return s.push(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "exec",
		RequestInfo: &endly.ActionInfo{
			Description: "run command inside container",
			Examples: []*endly.UseCase{
				{
					Description: "mysqldump from docker container",
					Data:        dockerServiceContainerRunMsqlDumpExample,
				},
				{
					Description: "mysql import into docker container",
					Data:        dockerServiceContainerRunMysqlImportExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &ExecRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ExecResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ExecRequest); ok {
				return s.runInContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "inspect",
		RequestInfo: &endly.ActionInfo{
			Description: "inspect docker container",
			Examples: []*endly.UseCase{
				{
					Description: "inspect",
					Data:        dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &InspectRequest{}
		},
		ResponseProvider: func() interface{} {
			return &InspectResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*InspectRequest); ok {
				return s.inspect(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "start",
		RequestInfo: &endly.ActionInfo{
			Description: "start container",
			Examples: []*endly.UseCase{
				{
					Description: "container start",
					Data:        dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &StartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StartResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StartRequest); ok {
				return s.startContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "stop",
		RequestInfo: &endly.ActionInfo{
			Description: "stop container",
			Examples: []*endly.UseCase{
				{
					Description: "container stop",
					Data:        dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &StopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StopResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StopRequest); ok {
				return s.stopContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "status",
		RequestInfo: &endly.ActionInfo{
			Description: "check containers status",
			Examples: []*endly.UseCase{
				{
					Description: "container check status",
					Data:        dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &ContainerStatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ContainerStatusResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ContainerStatusRequest); ok {
				return s.checkContainerProcesses(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "remove",
		RequestInfo: &endly.ActionInfo{
			Description: "remove docker container",
			Examples: []*endly.UseCase{
				{
					Description: "remove container",
					Data:        dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &RemoveRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RemoveResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RemoveRequest); ok {
				return s.removeContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "logs",
		RequestInfo: &endly.ActionInfo{
			Description: "remove docker container",
			Examples: []*endly.UseCase{
				{
					Description: "read  container stdout/stderr",
					Data:        dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &LogsRequest{}
		},
		ResponseProvider: func() interface{} {
			return &LogsResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*LogsRequest); ok {
				return s.containerLogs(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "composeUp",
		RequestInfo: &endly.ActionInfo{
			Description: "docker compose up",
			Examples: []*endly.UseCase{
				{
					Description: "Build, re-create and start docker services and their linked/dependent services",
					Data:        dockerServiceComposeUpExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &ComposeRequestUp{}
		},
		ResponseProvider: func() interface{} {
			return &ComposeResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ComposeRequestUp); ok {
				return s.composeUp(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "composeDown",
		RequestInfo: &endly.ActionInfo{
			Description: "docker compose down",
			Examples: []*endly.UseCase{
				{
					Description: "Stop all the services that were brought up by compose up",
					Data:        dockerServiceComposeDownExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &ComposeRequestDown{}
		},
		ResponseProvider: func() interface{} {
			return &ComposeResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ComposeRequestDown); ok {
				return s.composeDown(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new docker service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
