package cloudfunctions

import (
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gc"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"google.golang.org/api/cloudfunctions/v1beta2"
	"log"
)

const (
	//ServiceID Google Cloud Function Service Id
	ServiceID = "gc/cloudfunctions"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &cloudfunctions.Service{}
	routes, err := gc.BuildRoutes(client, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = InitRequest
		s.Register(route)
	}
}

func (s *service) Deploy(context *endly.Context, httpRequest *DeployRequest) (interface{}, error) {
	ctxClient, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	projectService := cloudfunctions.NewProjectsLocationsFunctionsService(ctxClient.service)
	cloudFunction, err := projectService.Get(httpRequest.Name).Do()
	if err != nil {
		cloudFunction = nil
	}
	state := context.State()
	credConfig := ctxClient.CredConfig
	gcNode := data.NewMap()

	gcNode.Put("projectID", credConfig.ProjectID)
	gcNode.Put("location", httpRequest.Location)
	state.Put("gc", gcNode)

	storageService, err := storage.NewServiceForURL(httpRequest.Source.URL, httpRequest.Source.Credentials)
	if err != nil {
		return nil, err
	}
	reader, err := storageService.DownloadWithURL(httpRequest.Source.URL)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	parent := state.ExpandAsText("projects/${gc.projectID}/locations/${gc.location}")
	generateRequest := &cloudfunctions.GenerateUploadUrlRequest{}
	uploadCall := cloudfunctions.NewProjectsLocationsFunctionsService(ctxClient.service).GenerateUploadUrl(parent, generateRequest)
	uploadCall.Context(ctxClient.Context())
	uploadResponse, err := uploadCall.Do()
	if err != nil {
		return nil, err
	}
	if err = gc.Upload(ctxClient.HttpClinet, uploadResponse.UploadUrl, reader); err != nil {
		return nil, err
	}
	httpRequest.SourceArchiveUrl = uploadResponse.UploadUrl
	if cloudFunction == nil {
		createCall := projectService.Create(httpRequest.Location, httpRequest.CloudFunction)
		createCall.Context(ctxClient.Context())
		return createCall.Do()
	}
	updateCall := projectService.Update(httpRequest.Name, cloudFunction)
	updateCall.Context(ctxClient.Context())
	return updateCall.Do()
}

//New creates a new Dataflow service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
