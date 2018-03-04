package docker

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/endly"
	"github.com/viant/endly/system/daemon"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
	"strings"
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

func (s *service) stopImages(context *endly.Context, request *DockerStopImagesRequest) (*DockerStopImagesResponse, error) {
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
				var name = strings.Split(container.Names, ",")[0]
				_, err = s.stopContainer(context, &DockerContainerStopRequest{
					DockerContainerBaseRequest: &DockerContainerBaseRequest{
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

func (s *service) resetContainerIfNeeded(context *endly.Context, target *url.Resource, statusResponse *DockerContainerStatusResponse) error {
	if len(statusResponse.Containers) > 0 {
		_, err := s.stopContainer(context, &DockerContainerStopRequest{
			DockerContainerBaseRequest: &DockerContainerBaseRequest{
				Target: target,
			},
		})
		if err != nil {
			return err
		}
		_, err = s.removeContainer(context, &DockerContainerRemoveRequest{
			DockerContainerBaseRequest: &DockerContainerBaseRequest{
				Target: target,
			}})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) runContainer(context *endly.Context, request *DockerRunRequest) (*DockerRunResponse, error) {
	var err error

	var credentials = s.applyCredentialIfNeeded(request.Credentials)

	checkResponse, err := s.checkContainerProcesses(context, &DockerContainerStatusRequest{
		Target: request.Target,
		Names:  request.Name,
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
	commandInfo, err := s.executeSecureDockerCommand(true, credentials, context, request.Target, dockerIgnoreErrors, fmt.Sprintf("docker run --name %v %v -d %v %v", request.Name, args, request.Image, params))
	if err != nil {
		return nil, err
	}

	if strings.Contains(commandInfo.Stdout(), containerInUse) {
		_, _ = s.stopContainer(context, &DockerContainerStopRequest{DockerContainerBaseRequest: &DockerContainerBaseRequest{
			Target: request.Target,
			Name:   request.Name,
		}})
		_, _ = s.removeContainer(context, &DockerContainerRemoveRequest{DockerContainerBaseRequest: &DockerContainerBaseRequest{
			Target: request.Target,
			Name:   request.Name,
		}})
		commandInfo, err = s.executeSecureDockerCommand(true, credentials, context, request.Target, dockerErrors, fmt.Sprintf("docker run --name %v %v -d %v", request.Name, args, request.Image))
		if err != nil {
			return nil, err
		}
	}

	info, err := s.checkContainerProcess(context, &DockerContainerStatusRequest{
		Target: request.Target,
		Names:  request.Name,
	})
	if info == nil {
		return nil, err
	}
	return &DockerRunResponse{info}, err
}

func (s *service) checkContainerProcess(context *endly.Context, request *DockerContainerStatusRequest) (*DockerContainerInfo, error) {
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
	if len(commandResult.Commands) > 1 {
		//Truncate password auth, to process vanila container output
		var stdout = commandResult.Commands[0].Stdout
		if strings.Contains(stdout, "Password:") {
			commandResult.Commands = commandResult.Commands[1:]
		}
	}

	return commandResult.Stdout(), nil
}

func (s *service) startContainer(context *endly.Context, request *DockerContainerStartRequest) (*DockerContainerStartResponse, error) {
	_, err := s.runContainerCommand(context, nil, request.Name, request.Target, "start", "")
	if err != nil {
		return nil, err
	}
	info, err := s.checkContainerProcess(context, &DockerContainerStatusRequest{
		Target: request.Target,
		Names:  request.Name,
	})
	if info == nil {
		return nil, err
	}
	return &DockerContainerStartResponse{info}, err
}

func (s *service) stopContainer(context *endly.Context, request *DockerContainerStopRequest) (*DockerContainerStopResponse, error) {
	info, err := s.checkContainerProcess(context, &DockerContainerStatusRequest{
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

	return &DockerContainerStopResponse{info}, nil
}

func (s *service) removeContainer(context *endly.Context, request *DockerContainerRemoveRequest) (response *DockerContainerRemoveResponse, err error) {
	response = &DockerContainerRemoveResponse{}
	response.Stdout, err = s.runContainerCommand(context, nil, request.Name, request.Target, "rm", "")
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *service) inspect(context *endly.Context, request *DockerInspectRequest) (response *DockerInspectResponse, err error) {
	response = &DockerInspectResponse{}
	response.Stdout, err = s.runContainerCommand(context, nil, request.Name, request.Target, "inspect", "")
	if err != nil {
		return nil, err
	}
	_, structured := endly.AsExtractable(response.Stdout)
	response.Info = structured[endly.SliceKey]
	return response, nil
}

func (s *service) containerLogs(context *endly.Context, request *DockerContainerLogsRequest) (response *DockerContainerLogsResponse, err error) {
	response = &DockerContainerLogsResponse{}
	response.Stdout, err = s.runContainerCommand(context, nil, request.Name, request.Target, "logs", "")
	return response, err
}

func (s *service) runInContainer(context *endly.Context, request *DockerContainerRunRequest) (response *DockerContainerRunResponse, err error) {
	response = &DockerContainerRunResponse{}
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
	response.Stdout, err = s.runContainerCommand(context, request.Credentials, request.Name, request.Target, "exec", executionOptions, execArguments)
	return response, err
}

func (s *service) checkContainerProcesses(context *endly.Context, request *DockerContainerStatusRequest) (*DockerContainerStatusResponse, error) {

	info, err := s.executeSecureDockerCommand(true, nil, context, request.Target, dockerErrors, "docker ps")
	if err != nil {
		return nil, err
	}
	stdout := info.Stdout()

	var containers = make([]*DockerContainerInfo, 0)
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

func (s *service) pullImage(context *endly.Context, request *DockerPullRequest) (*DockerPullResponse, error) {
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
	imageResponse, err := s.checkImages(context, &DockerImagesRequest{Target: request.Target, Repository: request.Repository, Tag: request.Tag})
	if err != nil {
		return nil, err
	}
	if len(imageResponse.Images) > 0 {
		return &DockerPullResponse{imageResponse.Images[0]}, nil
	}
	return nil, fmt.Errorf("not found:  %v %v", request.Repository, request.Tag)
}

func (s *service) checkImages(context *endly.Context, request *DockerImagesRequest) (*DockerImagesResponse, error) {
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

func (s *service) executeDockerCommand(secure map[string]string, context *endly.Context, target *url.Resource, errors []string, template string, arguments ...interface{}) (*exec.CommandResponse, error) {
	return s.executeSecureDockerCommand(false, secure, context, target, errors, fmt.Sprintf(template, arguments...))
}

func (s *service) startDockerIfNeeded(context *endly.Context, target *url.Resource) {
	daemonService, _ := context.Service(daemon.ServiceID)
	daemonService.Run(context, &daemon.StartRequest{
		Target:  target,
		Service: "docker",
	})

}

func (s *service) executeSecureDockerCommand(asRoot bool, secure map[string]string, context *endly.Context, target *url.Resource, errors []string, command string) (*exec.CommandResponse, error) {
	s.applySysPathIfNeeded([]string{})
	if len(secure) == 0 {
		secure = make(map[string]string)
	}
	secure[exec.SudoCredentialKey] = target.Credential
	command = strings.Replace(command, "\n", " ", len(command))
	var extractableCommand = &exec.ExtractableCommand{
		Options: &exec.ExecutionOptions{
			SystemPaths: s.SysPath,
			TimeoutMs:   120000,
		},
		Executions: []*exec.Execution{
			{
				Credentials: secure,
				Command:     command,
				Errors:      append(errors, []string{util.CommandNotFound}...),
			},
		},
	}
	var runRequest interface{} = extractableCommand
	if asRoot {
		runRequest = &exec.SuperUserCommandRequest{
			MangedCommand: extractableCommand,
		}
	}
	response, err := exec.Execute(context, target, runRequest)

	if err != nil {
		if util.CheckCommandNotFound(err.Error()) {
			return nil, err
		}
		if response != nil && !util.EscapedContains(response.Stdout(), dockerNotRunning) {
			return nil, err
		}
		s.startDockerIfNeeded(context, target)
		response, err = exec.Execute(context, target, runRequest)
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

func (s *service) build(context *endly.Context, request *DockerBuildRequest) (*DockerBuildResponse, error) {
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
	if !util.EscapedContains(response.Stdout, "Successfully built") {
		return nil, fmt.Errorf("failed to build: %v, stdout:%v", request.Tag, response.Stdout)
	}
	return response, nil
}
func (s *service) tag(context *endly.Context, request *DockerTagRequest) (*DockerTagResponse, error) {
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

func (s *service) getGoogleCloudCredential(context *endly.Context, credential string, config *cred.Config) *cred.Config {
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
func (s *service) login(context *endly.Context, request *DockerLoginRequest) (*DockerLoginResponse, error) {
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
	if !util.EscapedContains(stdout, "Login Succeeded") {
		return nil, fmt.Errorf("failed to authenticate: %v, stdout: %v", response.Username, stdout)
	}
	response.Username = credConfig.Username
	response.Stdout = stdout
	return response, nil
}

func (s *service) logout(context *endly.Context, request *DockerLogoutRequest) (*DockerLogoutResponse, error) {
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

func (s *service) push(context *endly.Context, request *DockerPushRequest) (*DockerPushResponse, error) {
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
	if !(util.EscapedContains(stdout, "Pushed") || util.EscapedContains(stdout, "Layer already exists")) {
		return nil, fmt.Errorf("failed to push tag: %v, stdout: %v", request.Tag, stdout)
	}
	return response, nil
}

const (
	dockerServiceRunExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  },
  "Name": "udb_aerospike",
  "Image": "aerospike/aerospike-server:latest",
  "Mount": {
    "/tmp/aerospikeudb_aerospike.conf": "/etc/aerospike/aerospike.conf"
  },
  "MappedPort": {
    "3000": "3000",
    "3001": "3001",
    "3002": "3002",
    "3004": "3004",
    "8081": "8081"
  },
}`

	dockerServiceStopImagesExample = `{
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  },
  "Images": [
    "aerospike",
    "mysql"
  ]
}`

	dockerServiceImagesExample = `{
    "Target": {
		"URL": "ssh://127.0.0.1/",
		"Credential": "${env.HOME}/.secret/localhost.json"
    },
	"Repository": "mysql",
	"Tag"":        "5.6"
}`

	dockerServicePullExample = `{
    "Target": {
		"URL": "ssh://127.0.0.1/",
		"Credential": "${env.HOME}/.secret/localhost.json"
    },
	"Repository": "aerospike",
	"Tag"":        "latest"
}`

	dockerServiceBuildExample = `{
  "Target": {
    "URL": "ssh://127.0.0.1/Projects/store_backup/app",
    "Credential": "${env.HOME}/.secret/localhost.json"
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
    "Credential": "${env.HOME}/.secret/localhost.json"
   
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
    "Credential": "${env.HOME}/.secret/localhost.json"
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
    "Credential": "${env.HOME}/.secret/aws-west.json"
  },
  "Credential": "${env.HOME}/.secret/docker.json",
  "Repository": "us.gcr.io/xxxxx"
}`

	dockerServiceLogoutExample = `{
  "Target": {
    "URL": "ssh://10.10.1.1/",
    "Credential": "${env.HOME}/.secret/aws-west.json"
  },
  "Credential": "${env.HOME}/.secret/docker.json",
  "Repository": "us.gcr.io/xxxxx"
}`

	dockerServiceContainerRunMsqlDumpExample = `{
  "Target": {
    "URL": "ssh://10.10.1.1/",
    "Credential": "${env.HOME}/.secret/aws-west.json"
  },
  "Name": "mydb1",
  "Interactive": true,
  "AllocateTerminal": true,
  "Command": "mysqldump  -uroot -p***mysql*** --all-databases --routines | grep -v 'Warning' > /tmp/dump.sql",
  "Credentials": {
    "***mysql***": "${env.HOME}/.secret/aws-west-mysql.json"
  }
}`

	dockerServiceContainerRunMysqlImportExample = `{
  "Target": {
    "URL": "ssh://10.10.1.1/",
    "Credential": "${env.HOME}/.secret/aws-west.json"
  },
  "Name": "mydb1",
  "Interactive": true,
  "Command": "mysql  -uroot -p**mysql** < /tmp/dump.sql",
  "Credentials": {
    "***mysql***": "${env.HOME}/.secret/aws-west-mysql.json"
  }
}`

	dockerServiceContainerExample = `{
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  },
  "Name": "udb_aerospike"
}`
)

func (s *service) registerRoutes() {

	s.Register(&endly.ServiceActionRoute{
		Action: "run",
		RequestInfo: &endly.ActionInfo{
			Description: "run docker image",

			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "run docker image on the target host",
					Data:    dockerServiceRunExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerRunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerRunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerRunRequest); ok {
				return s.runContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "stop-images",
		RequestInfo: &endly.ActionInfo{
			Description: "stops docker container matching supplied images",

			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "stop images",
					Data:    dockerServiceStopImagesExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerStopImagesRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerStopImagesResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerStopImagesRequest); ok {
				return s.stopImages(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "images",
		RequestInfo: &endly.ActionInfo{
			Description: "return images info for supplied matching images",

			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "check image",
					Data:    dockerServiceImagesExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerImagesRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerImagesResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerImagesRequest); ok {
				return s.checkImages(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "pull",
		RequestInfo: &endly.ActionInfo{
			Description: "pull docker image",

			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "pull example",
					Data:    dockerServicePullExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerPullRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerPullResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerPullRequest); ok {
				return s.pullImage(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "build",
		RequestInfo: &endly.ActionInfo{
			Description: "build docker image",

			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "build image",
					Data:    dockerServiceBuildExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerBuildRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerBuildResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerBuildRequest); ok {
				return s.build(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "tag",
		RequestInfo: &endly.ActionInfo{
			Description: "tag docker image",

			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "tag image",
					Data:    dockerServiceTagExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerTagRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerTagResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerTagRequest); ok {
				return s.tag(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "login",
		RequestInfo: &endly.ActionInfo{
			Description: "add credential for supplied docker repository, required docker 17+ for secure credential handling",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "login ",
					Data:    dockerServiceLoginExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerLoginRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerLoginResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerLoginRequest); ok {
				return s.login(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "logout",
		RequestInfo: &endly.ActionInfo{
			Description: "remove credential for supplied docker repository",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "logout ",
					Data:    dockerServiceLogoutExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerLogoutRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerLogoutResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerLogoutRequest); ok {
				return s.logout(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "push",
		RequestInfo: &endly.ActionInfo{
			Description: "push docker image into docker repository",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "push ",
					Data:    dockerServicePushExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerPushRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerPushResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerPushRequest); ok {
				return s.push(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "container-run",
		RequestInfo: &endly.ActionInfo{
			Description: "run command inside container",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "mysqldump from docker container",
					Data:    dockerServiceContainerRunMsqlDumpExample,
				},
				{
					UseCase: "mysql import into docker container",
					Data:    dockerServiceContainerRunMysqlImportExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerContainerRunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerContainerRunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerContainerRunRequest); ok {
				return s.runInContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "inspect",
		RequestInfo: &endly.ActionInfo{
			Description: "inspect docker container",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "inspect",
					Data:    dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerInspectRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerInspectResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerInspectRequest); ok {
				return s.inspect(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "container-start",
		RequestInfo: &endly.ActionInfo{
			Description: "start container",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "container start",
					Data:    dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerContainerStartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerContainerStartResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerContainerStartRequest); ok {
				return s.startContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "container-stop",
		RequestInfo: &endly.ActionInfo{
			Description: "stop container",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "container stop",
					Data:    dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerContainerStopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerContainerStopResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerContainerStopRequest); ok {
				return s.stopContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "container-status",
		RequestInfo: &endly.ActionInfo{
			Description: "check containers status",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "container check status",
					Data:    dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerContainerStatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerContainerStatusResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerContainerStatusRequest); ok {
				return s.checkContainerProcesses(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "container-remove",
		RequestInfo: &endly.ActionInfo{
			Description: "remove docker container",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "remove container",
					Data:    dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerContainerRemoveRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerContainerRemoveResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerContainerRemoveRequest); ok {
				return s.removeContainer(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "container-logs",
		RequestInfo: &endly.ActionInfo{
			Description: "remove docker container",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "read  container stdout/stderr",
					Data:    dockerServiceContainerExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DockerContainerLogsRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DockerContainerLogsResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DockerContainerLogsRequest); ok {
				return s.containerLogs(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewCriteria creates a new docker service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
