package docker

import (
	"archive/tar"
	"bytes"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-errors/errors"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"io"
	"io/ioutil"
	"log"
	"path"
	"strings"
)

const (
	//ServiceID aws Simple Queue Service ID.
	ServiceID = "docker"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) build(context *endly.Context, request *BuildRequest) (*BuildResponse, error) {
	var buildResponse = &BuildResponse{
		Stdout: make([]string, 0),
	}
	location := request.Path

	if !toolbox.IsDirectory(location) {
		location, _ = path.Split(request.Path)
	}
	tarReader, err := AsTarReader(url.NewResource(location), false)
	if err != nil {
		return nil, err
	}
	buildRequest := &ImageBuildRequest{
		ImageBuildOptions: request.ImageBuildOptions,
		BuildContext:      tarReader,
	}
	imgBuildResponse := &types.ImageBuildResponse{}
	if err = runAdapter(context, buildRequest, imgBuildResponse); err != nil {
		return nil, err
	}
	defer imgBuildResponse.Body.Close()
	errorMessage := ""
	if err = readStream(context, "build", imgBuildResponse.Body, &buildResponse.Stdout, func(stream *DataStream) {
		if stream.ErrorDetail != nil && stream.ErrorDetail.Message != "" {
			errorMessage = stream.ErrorDetail.Message
		}
		if stream.Error != "" {
			errorMessage = stream.Error
		}
		indexOf := strings.LastIndex(stream.Stream, "fully built")
		if indexOf != -1 {
			buildResponse.ImageID = strings.TrimSpace(string(stream.Stream[indexOf+12:]))
		}
	}); err != nil {
		return nil, err
	}
	if errorMessage != "" {
		err = errors.New(errorMessage)
	}
	return buildResponse, err
}

func (s *service) expandSecrets(context *endly.Context, request *RunRequest) error {
	if len(request.Secrets) == 0 || len(request.Config.Env) == 0 {
		return nil
	}
	var err error
	for i, env := range request.Config.Env {
		request.Config.Env[i], err = context.Secrets.Expand(env, request.Secrets)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *service) run(context *endly.Context, request *RunRequest) (*RunResponse, error) {
	response := &RunResponse{}
	err := s.expandSecrets(context, request)
	if err != nil {
		return nil, err
	}
	var containerInfo *types.Container
	if request.Name != "" {
		status, err := s.status(context, &StatusRequest{
			Name: request.Name,
		})
		if err != nil {
			return nil, err
		}
		if len(status.Containers) == 1 {
			containerInfo = &status.Containers[0]
		}
	}

	if containerInfo != nil {
		if request.Reuse {
			response.ContainerID = containerInfo.ID
			if IsContainerUp(containerInfo) {
				return response, nil
			}
			_, err := s.start(context, &StartRequest{
				IDs: []string{containerInfo.ID},
			})
			return response, err
		}

		if _, err := s.remove(context, &RemoveRequest{
			IDs: []string{containerInfo.ID},
		}); err != nil {
			return nil, err
		}
	}

	pullRequest := &PullRequest{
		Image:            request.Image,
		Credentials:      request.Credentials,
		ImagePullOptions: request.ImagePullOptions,
	}

	if _, err := s.pull(context, pullRequest); err != nil {
		return nil, fmt.Errorf("unable to pull %v, %v", request.Image, err)
	}

	if request.Name != "" {
		if _, err := s.remove(context, &RemoveRequest{
			Names: []string{request.Name},
		}); err != nil {
			return nil, err
		}
	}
	createRequest := request.CreateContainerRequest()
	createResponse := &container.ContainerCreateCreatedBody{}
	if err := runAdapter(context, createRequest, createResponse); err != nil {
		return nil, err
	}
	response.ContainerID = createResponse.ID
	startRequest := &StartRequest{IDs: []string{createResponse.ID}}

	if _, err := s.start(context, startRequest); err != nil {
		return nil, err
	}
	status, err := s.status(context, startRequest.AsStatusRequest())
	if err != nil {
		return nil, err
	}

	if len(status.Containers) == 0 {
		return nil, fmt.Errorf("unable to locate container %v", status.Containers[0].ID)
	}
	response.Status = status.Containers[0].Status
	logRequest := &ContainerLogsRequest{Container: response.ContainerID}
	logRequest.ShowStdout = true
	var logReader io.ReadCloser
	if err := runAdapter(context, logRequest, &logReader); err == nil {
		if data, err := ioutil.ReadAll(logReader); err == nil {
			response.Stdout = string(data)
		}
	}
	return response, nil
}

func (s *service) pull(context *endly.Context, request *PullRequest) (*PullResponse, error) {
	err := request.Init()
	if err != nil {
		return nil, err
	}
	response := &PullResponse{
		Stdout: make([]string, 0),
	}
	listRequest := &ImageListRequest{}
	listRequest.All = true
	listSummary := make([]types.ImageSummary, 0)
	if err = runAdapter(context, listRequest, &listSummary); err != nil {
		return nil, err
	}
	if len(listSummary) > 0 {
		for _, image := range listSummary {
			for _, tag := range image.RepoTags {
				if tag == request.Image {
					response.ImageSummary = image
					return response, nil
				}
			}
		}
	}
	var reader io.ReadCloser
	pullRequest := &ImagePullRequest{}
	pullRequest.ImagePullOptions = request.ImagePullOptions
	pullRequest.RefStr = request.Image
	tag := NewTag(request.Image)

	pullRequest.RegistryAuth, err = getAuthToken(context, tag.Registry, request.Credentials)
	if err != nil {
		return nil, err
	}
	if err := runAdapter(context, pullRequest, &reader); err != nil {
		return nil, err
	}
	defer reader.Close()
	errorMessage := ""
	err = readStream(context, "pull", reader, &response.Stdout, func(stream *DataStream) {
		if stream.ErrorDetail != nil {
			errorMessage = stream.ErrorDetail.Message
		}
	})
	if err != nil {
		return nil, err
	}
	if errorMessage != "" {
		return nil, errors.New(errorMessage)
	}
	return response, nil
}

func (s *service) status(context *endly.Context, request *StatusRequest) (*StatusResponse, error) {
	response := &StatusResponse{
		Containers: make([]types.Container, 0),
	}
	_ = request.Init()
	getContainers := &ContainerListRequest{}
	getContainers.All = true
	var containersResp = make([]types.Container, 0)
	if err := runAdapter(context, getContainers, &containersResp); err != nil {
		return nil, err
	}
	if len(containersResp) == 0 {
		return response, nil
	}
	byImage := len(request.Images) > 0
	byName := len(request.Names) > 0
	byIDs := len(request.IDs) > 0
outer:
	for _, candidate := range containersResp {
		if byIDs {
			for _, id := range request.IDs {
				if candidate.ID == id {
					response.Containers = append(response.Containers, candidate)
					continue outer
				}
			}
		}
		if byImage {
			for _, image := range request.Images {
				if strings.Contains(candidate.Image, image) {
					response.Containers = append(response.Containers, candidate)
					continue outer
				}
			}
		}
		if byName {
			for _, name := range request.Names {
				for _, containerName := range candidate.Names {
					if !strings.HasPrefix(name, "/") {
						name = "/" + name
					}
					if containerName == name {
						response.Containers = append(response.Containers, candidate)
						break outer
					}
				}
			}
		}
	}
	return response, nil
}

func (s *service) stop(context *endly.Context, request *StopRequest) (*StopResponse, error) {
	response := &StopResponse{}
	status, err := s.status(context, request.AsStatusRequest())
	if err != nil {
		return nil, err
	}
	response.Containers = status.Containers
	for _, containerInfo := range status.Containers {
		if !IsContainerUp(&containerInfo) {
			continue
		}
		stopRequest := &ContainerStopRequest{}
		response.Containers = append(response.Containers, containerInfo)
		stopRequest.ContainerID = containerInfo.ID
		err := runAdapter(context, stopRequest, nil)
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}

func (s *service) remove(context *endly.Context, request *RemoveRequest) (*RemoveResponse, error) {
	response := &RemoveResponse{
		Containers: make([]types.Container, 0),
	}
	status, err := s.status(context, request.AsStatusRequest())
	if err != nil {
		return nil, err
	}
	response.Containers = status.Containers
	if len(response.Containers) == 0 {
		return response, nil
	}
	for _, containerInfo := range status.Containers {
		containerRemoveRequest := &ContainerRemoveRequest{}
		if IsContainerUp(&containerInfo) {
			if _, err = s.stop(context, &StopRequest{
				IDs: []string{containerInfo.ID},
			}); err != nil {
				return nil, err
			}
		}
		containerRemoveRequest.Force = true
		containerRemoveRequest.ContainerID = containerInfo.ID
		err := runAdapter(context, containerRemoveRequest, nil)
		if err != nil {
			return nil, err
		}
		response.Containers = append(response.Containers, containerInfo)
	}
	return response, nil
}

func (s *service) tag(context *endly.Context, request *TagRequest) (*TagResponse, error) {
	response := &TagResponse{}
	tagRequest := &ImageTagRequest{
		Source: context.Expand(request.SourceTag.String()),
		Target: context.Expand(request.TargetTag.String()),
	}
	err := runAdapter(context, tagRequest, nil)
	if err == nil {
		publishEvent(context, "tag", tagRequest)
	}
	return response, err
}

func (s *service) push(context *endly.Context, request *PushRequest) (*PushResponse, error) {
	response := &PushResponse{
		Stdout: make([]string, 0),
	}
	var err error
	pushRequest := &ImagePushRequest{}
	repository := request.Tag.Repository()
	pushRequest.Image = request.Tag.String()
	pushRequest.RegistryAuth, err = getAuthToken(context, repository, request.Credentials)
	if err != nil {
		return nil, err
	}
	if pushRequest.RegistryAuth == "" {
		return nil, fmt.Errorf("failed to lookup auth for '%v' repository", repository)
	}
	var reader io.ReadCloser
	if err = runAdapter(context, pushRequest, &reader); err != nil {
		return nil, err
	}
	errorMessage := ""
	err = readStream(context, "push", reader, &response.Stdout, func(stream *DataStream) {
		if stream.ErrorDetail != nil {
			errorMessage = stream.ErrorDetail.Message
		}
	})
	if errorMessage != "" {
		return nil, errors.New(errorMessage)
	}
	return response, err
}

func (s *service) inspect(context *endly.Context, request *InspectRequest) (*InspectResponse, error) {
	response := &InspectResponse{
		Info: make([]types.ContainerJSON, 0),
	}
	status, err := s.status(context, request.AsStatusRequest())
	if err != nil {
		return nil, err
	}
	if len(status.Containers) == 0 {
		return response, nil
	}
	for _, containerInfo := range status.Containers {
		inspectResponse := types.ContainerJSON{}
		inspectRequest := &ContainerInspectRequest{ContainerID: containerInfo.ID}
		if err = runAdapter(context, inspectRequest, &inspectResponse); err != nil {
			return nil, err
		}
		response.Info = append(response.Info, inspectResponse)
	}
	return response, nil
}

func (s *service) start(context *endly.Context, request *StartRequest) (*StartResponse, error) {
	response := &StartResponse{}
	containers, err := s.status(context, request.AsStatusRequest())
	if err != nil {
		return nil, err
	}

	if len(containers.Containers) == 0 {
		return nil, fmt.Errorf("unable to locate container by names: %v, or ids: %v", request.Names, request.IDs)
	}
	for _, candidate := range containers.Containers {
		if IsContainerUp(&candidate) {
			continue
		}
		startRequest := &ContainerStartRequest{ContainerID: candidate.ID}
		if err = runAdapter(context, startRequest, nil); err != nil {
			return nil, err
		}
	}
	return response, nil
}

func (s *service) login(context *endly.Context, request *LoginRequest) (*LoginResponse, error) {
	response := &LoginResponse{}
	token, err := authCredentialsToken(context, request.Credentials)
	ctxClient, err := GetCtxClient(context)
	if err != nil {
		return nil, err
	}
	ctxClient.AuthToken[request.Repository] = token
	return response, nil
}

func (s *service) logout(context *endly.Context, request *LogoutRequest) (*LogoutResponse, error) {
	response := &LogoutResponse{}
	ctxClient, err := GetCtxClient(context)
	if err != nil {
		return nil, err
	}
	delete(ctxClient.AuthToken, request.Repository)
	return response, nil
}

func (s *service) copyFromContainer(context *endly.Context, name, source, dest string) error {
	ctxClient, err := GetCtxClient(context)
	if err != nil {
		return err
	}
	status, err := s.status(context, &StatusRequest{Name: name})
	if err != nil {
		return err
	}
	if len(status.Containers) == 0 {
		return fmt.Errorf("unknown container: %v", name)
	}
	reader, stat, err := ctxClient.Client.CopyFromContainer(ctxClient.Context, status.Containers[0].ID, source)
	if err != nil {
		return err
	}
	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if stat.LinkTarget != "" {
		return s.copyToContainer(context, name, stat.LinkTarget, dest)
	}
	tarReader := tar.NewReader(bytes.NewReader(data))
	return UnTar(tarReader, dest)
}

func (s *service) copyToContainer(context *endly.Context, name, source, dest string) error {
	reader, err := AsTarReader(url.NewResource(source), true)
	if err != nil {
		return err
	}
	status, err := s.status(context, &StatusRequest{Name: name})
	if err != nil {
		return err
	}
	if len(status.Containers) == 0 {
		return fmt.Errorf("unknown container: %v", name)
	}
	copyRequest := &CopyToContainerRequest{
		ContainerID: status.Containers[0].ID,
		DstPath:     dest,
		Content:     reader,
	}
	return runAdapter(context, copyRequest, nil)
}

func (s *service) copy(context *endly.Context, request *CopyRequest) (*CopyResponse, error) {
	var response = &CopyResponse{}
	for source, dest := range request.Assets {

		if strings.Count(source, ":") == 1 {
			parts := strings.SplitN(source, ":", 2)
			dest = expandHomeDirectory(dest)
			dest = url.NewResource(dest).ParsedURL.Path
			if err := s.copyFromContainer(context, parts[0], parts[1], dest); err != nil {
				return nil, err
			}
			continue
		}

		if strings.Count(dest, ":") == 1 {
			parts := strings.SplitN(dest, ":", 2)
			source = expandHomeDirectory(source)
			source = url.NewResource(source).ParsedURL.Path
			if err := s.copyToContainer(context, parts[0], source, parts[1]); err != nil {
				return nil, err
			}
			continue
		}
		return nil, fmt.Errorf("continer is missing: %v -> %v", source, dest)
	}
	return response, nil
}

func (s *service) logs(context *endly.Context, request *LogsRequest) (*LogsResponse, error) {
	response := &LogsResponse{}
	status, err := s.status(context, request.AsStatusRequest())
	if err != nil {
		return nil, err
	}
	if len(status.Containers) == 0 {
		return response, nil
	}
	var reader io.ReadCloser
	logRequest := &ContainerLogsRequest{
		ContainerLogsOptions: *request.ContainerLogsOptions,
		Container:            status.Containers[0].ID,
	}
	err = runAdapter(context, logRequest, &reader)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	response.Stdout = string(data)
	return response, nil
}

func (s *service) registerRoutes() {
	dockerClient := &client.Client{}
	routes, err := BuildRoutes(dockerClient, GetCtxClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = initClient
		s.Register(route)
	}

	s.Register(&endly.Route{
		Action:       "build",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "build", &BuildRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &BuildResponse{}),
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
		Action:       "run",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: "run docker image",
		},
		RequestProvider: func() interface{} {
			return &RunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RunRequest); ok {
				response, err := s.run(context, req)
				if err == nil {
					publishEvent(context, "run", response)
				}
				return response, err

			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "stop",
		RequestInfo: &endly.ActionInfo{
			Description: "stops docker containers by name or image names",
		},
		RequestProvider: func() interface{} {
			return &StopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StopResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StopRequest); ok {
				return s.stop(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action:       "start",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: "start docker containers by names or ids",
		},
		RequestProvider: func() interface{} {
			return &StartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StartResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StartRequest); ok {
				return s.start(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action:       "status",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: "returns docker containers by images or container names",
		},
		RequestProvider: func() interface{} {
			return &StatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StatusResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StatusRequest); ok {
				response, err := s.status(context, req)
				if err == nil {
					publishEvent(context, "status", response)
				}
				return response, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action:       "remove",
		OnRawRequest: initClient,

		RequestInfo: &endly.ActionInfo{
			Description: "remove docker containers by ids or container names",
		},
		RequestProvider: func() interface{} {
			return &RemoveRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RemoveResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RemoveRequest); ok {
				return s.remove(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action:       "pull",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: "pull docker image",
		},
		RequestProvider: func() interface{} {
			return &PullRequest{}
		},
		ResponseProvider: func() interface{} {
			return &PullResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PullRequest); ok {
				return s.pull(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action:       "tag",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: "tag docker image",
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
		Action:       "push",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: "push docker image into docker repository",
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
		Action:       "login",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: "add credentials for supplied docker repository to endly context",
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
		Action:       "logout",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: "remove credentials for supplied docker repository from endly context",
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
		Action:       "copy",
		OnRawRequest: initClient,
		RequestInfo: &endly.ActionInfo{
			Description: "copy asset from container",
		},
		RequestProvider: func() interface{} {
			return &CopyRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CopyResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CopyRequest); ok {
				return s.copy(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "inspect",
		RequestInfo: &endly.ActionInfo{
			Description: "inspect docker container",
		},
		RequestProvider: func() interface{} {
			return &InspectRequest{}
		},
		ResponseProvider: func() interface{} {
			return &InspectResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*InspectRequest); ok {
				response, err := s.inspect(context, req)
				if err == nil {
					publishEvent(context, "status", response)
				}
				return response, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "logs",
		RequestInfo: &endly.ActionInfo{
			Description: "remove docker container",
		},
		RequestProvider: func() interface{} {
			return &LogsRequest{}
		},
		ResponseProvider: func() interface{} {
			return &LogsResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*LogsRequest); ok {
				response, err := s.logs(context, req)
				if err == nil {
					publishEvent(context, "status", response)
				}
				return response, err

			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new Docker service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
