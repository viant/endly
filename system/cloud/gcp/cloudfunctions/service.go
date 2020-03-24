package cloudfunctions

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"google.golang.org/api/cloudfunctions/v1"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	//ServiceID Google Cloud Function Service Id
	ServiceID = "gcp/cloudfunctions"

	functionInvokerRole = "roles/cloudfunctions.invoker"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) expandWithContext(context *endly.Context, credConfig *cred.Config, region, text string) string {
	state := context.State()
	gcpNode := data.NewMap()
	gcpNode.Put("projectID", credConfig.ProjectID)
	gcpNode.Put("region", region)
	state.Put("gcp", gcpNode)
	return state.ExpandAsText(text)
}

func (s *service) Deploy(context *endly.Context, request *DeployRequest) (*DeployResponse, error) {
	output, err := s.deploy(context, request)
	if err != nil {
		return nil, err
	}
	if !output.Done {
		if err = s.AbstractService.RunInBackground(context, func() error {
			output, err = s.waitForOperationCompletion(context, output.Name)
			return err
		}); err != nil {
			return nil, err
		}
	}
	var meta = make(map[string]interface{})
	err = json.Unmarshal(output.Metadata, &meta)
	response := &DeployResponse{
		Operation: output.Name,
		Meta:      meta,
	}
	response.Function, err = s.get(context, &GetRequest{
		Name: request.Name,
	})

	if response.Function.Status == "OFFLINE" {
		context.Publish(gcp.NewOutputEvent(request.Name, "deploy", response))
		return nil, fmt.Errorf("failed to deploy funciton")
	}
	if len(request.Members) > 0 {
		err = s.updateInvokers(context, response.Function.Name, request.Members...)
	}
	return response, err
}

func (s *service) waitForOperationCompletion(context *endly.Context, name string) (*cloudfunctions.Operation, error) {
	ctxClient, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	operationService := cloudfunctions.NewOperationsService(ctxClient.service)
	for {
		getCall := operationService.Get(name)
		getCall.Context(ctxClient.Context())
		operation, err := getCall.Do()
		if err != nil {
			return nil, err
		}
		if operation.Done {
			return operation, err
		}
		time.Sleep(time.Second)
	}
}

func (s *service) get(context *endly.Context, request *GetRequest) (*cloudfunctions.CloudFunction, error) {
	ctxClient, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	request.Name = s.expandWithContext(context, ctxClient.CredConfig, request.Region, request.Name)
	service := cloudfunctions.NewProjectsLocationsFunctionsService(ctxClient.service)
	getCall := service.Get(request.Name)
	getCall.Context(ctxClient.Context())
	return getCall.Do()

}

func (s *service) getFunctionPackageReader(resource *url.Resource) (io.ReadCloser, error) {
	storageService, err := storage.NewServiceForURL(resource.URL, resource.Credentials)
	if err != nil {
		return nil, err
	}
	object, err := storageService.StorageObject(resource.URL)
	if err != nil {
		return nil, err
	}
	if object.IsContent() {
		return storageService.DownloadWithURL(resource.URL)
	}

	gitIgnorePath := toolbox.URLPathJoin(resource.URL, ".gcloudignore")
	ignoreList := util.GetIgnoreList(storageService, gitIgnorePath)
	writer := new(bytes.Buffer)
	archive := zip.NewWriter(writer)
	err = storage.ArchiveWithFilter(storageService, resource.URL, archive, func(candidate storage.Object) bool {
		candidateName := candidate.FileInfo().Name()
		if strings.HasSuffix(candidateName, ".zip") {
			return false
		}
		if len(ignoreList) == 0 {
			return true
		}
		relativePath := candidate.URL()
		if relativePathIndex := strings.Index(relativePath, resource.URL); relativePathIndex != -1 {
			relativePath = string(relativePath[relativePathIndex+len(resource.URL):])
			if strings.HasPrefix(relativePath, "/") {
				relativePath = string(relativePath[1:])
			}
		}
		return !util.ShouldIgnoreLocation(relativePath, ignoreList)

	})
	err = archive.Close()
	payload := writer.Bytes()
	return ioutil.NopCloser(bytes.NewReader(payload)), err
}

func (s *service) deploy(context *endly.Context, request *DeployRequest) (*cloudfunctions.Operation, error) {
	ctxClient, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	parent := s.expandWithContext(context, ctxClient.CredConfig, request.Region, parentLocationTemplate)
	request.Name = s.expandWithContext(context, ctxClient.CredConfig, request.Region, request.Name)
	//TODO add support for simple name based on trigger type
	if request.EventTrigger != nil {
		request.EventTrigger.Resource = s.expandWithContext(context, ctxClient.CredConfig, request.Region, request.EventTrigger.Resource)
	}

	projectService := cloudfunctions.NewProjectsLocationsFunctionsService(ctxClient.service)
	cloudFunction, err := projectService.Get(request.Name).Do()
	if err != nil {
		cloudFunction = nil
	}
	generateRequest := &cloudfunctions.GenerateUploadUrlRequest{}
	uploadCall := cloudfunctions.NewProjectsLocationsFunctionsService(ctxClient.service).GenerateUploadUrl(parent, generateRequest)
	uploadCall.Context(ctxClient.Context())
	uploadResponse, err := uploadCall.Do()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate source url")
	}
	reader, err := s.getFunctionPackageReader(request.Source)
	if err != nil {
		return nil, err
	}



	defer reader.Close()
	if err = gcp.Upload(http.DefaultClient, uploadResponse.UploadUrl, reader); err != nil {
		return nil, err
	}
	request.SourceUploadUrl = uploadResponse.UploadUrl
	if cloudFunction == nil {
		createCall := projectService.Create(parent, &request.CloudFunction)
		createCall.Context(ctxClient.Context())
		output, err := createCall.Do()
		if err != nil {
			JSON, _ := json.Marshal(request.CloudFunction)
			return nil, errors.Wrapf(err, "failed to create function: %v; %s", request.Name, JSON)
		}
		return output, err
	} else {
		cloudFunction = &request.CloudFunction
	}
	updateCall := projectService.Patch(request.Name, cloudFunction)
	updateCall.Context(ctxClient.Context())
	operation, err := updateCall.Do()
	if err != nil {
		JSON, _ := json.Marshal(request.CloudFunction)
		return nil, errors.Wrapf(err, "failed to update function: %v; %s", request.Name, JSON)

	}
	return operation, nil
}

func (s *service) updateInvokers(context *endly.Context, resource string, members ...string) error {
	ctxClient, err := GetClient(context)
	if err != nil {
		return err
	}
	projectService := cloudfunctions.NewProjectsLocationsFunctionsService(ctxClient.service)
	call := projectService.GetIamPolicy(resource)
	call.Context(ctxClient.Context())
	policy, err := call.Do()
	if err != nil || policy == nil {
		return nil
	}

	if len(policy.Bindings) == 0 {
		policy.Bindings = make([]*cloudfunctions.Binding, 0)
	}
	updated := false
	for _, binding := range policy.Bindings {
		if binding.Role == functionInvokerRole {
			binding.Members = members
			updated = true
			break
		}
	}
	if !updated {
		policy.Bindings = append(policy.Bindings, &cloudfunctions.Binding{
			Members: members,
			Role:    functionInvokerRole,
		})
	}
	policyRequest := &cloudfunctions.SetIamPolicyRequest{
		Policy: policy,
	}
	updatePolicyCall := projectService.SetIamPolicy(resource, policyRequest)
	updatePolicyCall.Context(ctxClient.Context())
	_, err = updatePolicyCall.Do()
	return err
}

func (s *service) list(context *endly.Context, request *ListRequest) (*ListResponse, error) {
	ctxClient, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	projectService := cloudfunctions.NewProjectsLocationsFunctionsService(ctxClient.service)
	parent := s.expandWithContext(context, ctxClient.CredConfig, request.Region, parentLocationTemplate)
	listCall := projectService.List(parent)
	listCall.Context(ctxClient.Context())
	list, err := listCall.Do()
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Function: list.Functions,
	}, nil
}

func (s *service) delete(context *endly.Context, request *DeleteRequest) (*DeleteResponse, error) {
	ctxClient, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	projectService := cloudfunctions.NewProjectsLocationsFunctionsService(ctxClient.service)
	request.Name = s.expandWithContext(context, ctxClient.CredConfig, request.Region, request.Name)
	deleteCall := projectService.Delete(request.Name)
	deleteCall.Context(ctxClient.Context())
	output, err := deleteCall.Do()
	if err != nil {

	}
	if err != nil {
		return nil, err
	}
	if output != nil && !output.Done {
		if err = s.AbstractService.RunInBackground(context, func() error {
			output, err = s.waitForOperationCompletion(context, output.Name)
			return err
		}); err != nil {
			return nil, err
		}
	}
	var meta = make(map[string]interface{})
	err = json.Unmarshal(output.Metadata, &meta)
	response := &DeleteResponse{
		Operation: output.Name,
		Meta:      meta,
	}
	return response, nil
}

func (s *service) call(context *endly.Context, request *CallRequest) (*cloudfunctions.CallFunctionResponse, error) {
	ctxClient, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	projectService := cloudfunctions.NewProjectsLocationsFunctionsService(ctxClient.service)
	request.Name = s.expandWithContext(context, ctxClient.CredConfig, request.Region, request.Name)
	callFunctionRequest := &cloudfunctions.CallFunctionRequest{}
	if request.Data != nil { //TODO check for binary to encode with base64
		dataText := ""
		if toolbox.IsMap(request.Data) || toolbox.IsSlice(request.Data) {
			JSON, err := toolbox.AsJSONText(request.Data)
			if err != nil {
				return nil, fmt.Errorf("unable convert data to JSON")
			}
			dataText = JSON
		} else {
			dataText = toolbox.AsString(request.Data)
		}
		callFunctionRequest.Data = dataText
	}
	call := projectService.Call(request.Name, callFunctionRequest)
	call.Context(ctxClient.Context())
	return call.Do()
}

func (s *service) registerRoutes() {
	client := &cloudfunctions.Service{}
	routes, err := gcp.BuildRoutes(client, func(name string) string {
		return strings.Replace(name, "Functions", "", 1)
	}, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}

	s.Register(&endly.Route{
		Action: "deploy",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "deploy", &DeployRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &DeployResponse{}),
		},
		RequestProvider: func() interface{} {
			return &DeployRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DeployResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DeployRequest); ok {
				output, err := s.Deploy(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(gcp.NewOutputEvent(req.Name, "deploy", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "get",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "get", &GetRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &cloudfunctions.CloudFunction{}),
		},
		RequestProvider: func() interface{} {
			return &GetRequest{}
		},
		ResponseProvider: func() interface{} {
			return &cloudfunctions.CloudFunction{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetRequest); ok {
				output, err := s.get(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(gcp.NewOutputEvent(req.Name, "get", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "delete",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "delete", &DeleteRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &DeleteResponse{}),
		},
		RequestProvider: func() interface{} {
			return &DeleteRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DeleteResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DeleteRequest); ok {
				output, err := s.delete(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(gcp.NewOutputEvent(req.Name, "delete", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "call",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "call", &CallRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &cloudfunctions.CallFunctionResponse{}),
		},
		RequestProvider: func() interface{} {
			return &CallRequest{}
		},
		ResponseProvider: func() interface{} {
			return &cloudfunctions.CallFunctionResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CallRequest); ok {
				output, err := s.call(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(gcp.NewOutputEvent(req.Name, "call", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "list",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "list", &ListRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &ListResponse{}),
		},
		RequestProvider: func() interface{} {
			return &ListRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ListResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ListRequest); ok {
				output, err := s.list(context, req)
				if err != nil {
					return nil, err
				}
				context.Publish(gcp.NewOutputEvent("...", "list", output))
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	for _, route := range routes {
		if _, err := s.Route(route.Action); err == nil {
			continue
		}
		route.OnRawRequest = InitRequest
		s.Register(route)
	}
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
