package core

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"github.com/viant/endly/system/kubernetes/shared"
	"github.com/viant/toolbox"
	"io"
)

const (
	//ServiceID Kubernetes service ID.
	ServiceID = "kubernetes"
)

//no operation service
type service struct {
	*endly.AbstractService
}

//Create creates a specified resources
func (s *service) Create(context *endly.Context, request *CreateRequest) (*CreateResponse, error) {
	response := &CreateResponse{
		Items: make([]*ResourceInfo, 0),
	}
	ctxClient, err := shared.GetCtxClient(context)
	if err != nil {
		return nil, err
	}
	if request.ResourceMeta != nil && request.Kind != "" {
		operations, err := shared.Lookup(request.APIVersion, request.Kind)
		if err != nil {
			return nil, err
		}
		createRequest, err := operations.NewRequest("Create", ctxClient.RawRequest)
		if err != nil {
			return nil, err
		}
		getRequest, err := operations.NewRequest("Get", request.Metadata)
		if err != nil {
			return nil, err
		}
		watchRequest, _ := operations.NewRequest("Watch", request.Metadata)
		err = s.createResource(context, createRequest, request.ResourceMeta, response, getRequest, watchRequest)
		return response, err
	}
	err = s.createResources(context, request, response)
	return response, err
}

func (s *service) createResources(context *endly.Context, request *CreateRequest, response *CreateResponse) error {
	if request.Resource == nil {
		return nil
	}
	return ProcessResource(context, request.Expand, request.Resource, false, func(meta *ResourceMeta, requestData map[string]interface{}) error {
		operations, err := shared.Lookup(meta.APIVersion, meta.Kind)
		if err != nil {
			return err
		}
		createRequest, err := operations.NewRequest("Create", requestData)
		if err != nil {
			return err
		}
		watchRequest, _ := operations.NewRequest("Watch", requestData)
		getRequest, err := operations.NewRequest("Get", meta.Metadata)
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		return s.createResource(context, createRequest, meta, response, getRequest, watchRequest)
	})
}

func (s *service) createResource(context *endly.Context, createRequest interface{}, meta *ResourceMeta, response *CreateResponse, getRequest, watchRequest interface{}) error {
	createResponse := &ResourceInfo{}
	watcher, err := getWatcher(context, watchRequest)
	if err != nil {
		return err
	}
	if err = endly.RunWithoutLogging(context, createRequest, &createResponse); err != nil {
		return err
	}
	createResponse.TypeMeta = meta.TypeMeta
	createResponse.Name = meta.Metadata.Name
	createResponse.Labels = meta.Metadata.Labels
	response.Items = append(response.Items, createResponse)
	return waitUntilReady(watcher)
}

//Apply create or apply patch for specified resources
func (s *service) Apply(context *endly.Context, request *ApplyRequest) (*ApplyResponse, error) {
	response := &ApplyResponse{
		Items: make([]*ResourceInfo, 0),
	}
	err := ProcessResource(context, request.Expand, request.Resource, false, func(meta *ResourceMeta, requestData map[string]interface{}) error {
		operations, err := shared.Lookup(meta.APIVersion, meta.Kind)
		if err != nil {
			return err
		}
		applyResponse := &ResourceInfo{}
		createRequest, err := operations.NewRequest("Create", requestData)
		if err != nil {
			return err
		}
		getRequest, err := operations.NewRequest("Get", meta.Metadata)
		if err != nil {
			return err
		}
		var getResponse interface{}
		err = endly.RunWithoutLogging(context, getRequest, &getResponse)
		if err != nil {
			if !shared.IsNotFound(getResponse) {
				return err
			}
			if err = endly.RunWithoutLogging(context, createRequest, &applyResponse); err != nil {
				return err
			}
			response.Items = append(response.Items, applyResponse)
			return nil
		}
		pathData, err := NewResourcePatch(meta, getResponse, createRequest)
		if err != nil {
			return err
		}

		if !pathData.HasChanged {
			response.Items = append(response.Items, applyResponse)
			return nil
		}
		patchRequest, err := operations.NewRequest("Patch", pathData)
		if err != nil {
			return err
		}
		if err = endly.RunWithoutLogging(context, patchRequest, &applyResponse); err != nil {
			return err
		}
		response.Items = append(response.Items, applyResponse)
		return nil
	})
	return response, err
}

/*
	Expose a resource as a new Kubernetes service.
	If kind is not specified it lookup the first resource of "Deployment", "Service", "ReplicaSet", "ReplicationController", "Pod" kind.
*/
func (s *service) Expose(context *endly.Context, request *ExposeRequest) (*ExposeResponse, error) {
	ctxClient, err := shared.GetCtxClient(context)
	if err != nil {
		return nil, err
	}
	var resource *ResourceInfo
	ctxClient.RawRequest = nil //reset raw request
	getRequest := &GetRequest{Name: request.Resource, kinds: request.kinds}
	if err = s.get(context, getRequest, func(item *ResourceInfo) error {
		resource = item
		return io.EOF
	}); err != nil {
		if err != io.EOF {
			return nil, err
		}
	}
	if resource == nil {
		return nil, fmt.Errorf("resource '%v' not found", request.Resource)
	}
	requestData, err := NewExposeTemplateParams(resource, request)
	if err != nil {
		return nil, err
	}
	var createResponse *CreateResponse
	if err := GenerateRequest(ServiceV1GeneratorName, exposeTemplates, requestData, func(meta *ResourceMeta, rawRequest map[string]interface{}) error {
		ctxClient.RawRequest = rawRequest
		createResponse, err = s.Create(context, &CreateRequest{ResourceMeta: meta})
		return err
	}); err != nil {
		return nil, err
	}
	response := ExposeResponse(*createResponse)
	return &response, err
}

//Delete deletes supplied resources
func (s *service) Delete(context *endly.Context, request *DeleteRequest) (*DeleteResponse, error) {
	response := &DeleteResponse{
		Items: make([]*ResourceInfo, 0),
	}
	if request.Kind != "" {
		err := s.get(context, request.AsGetRequest(), func(item *ResourceInfo) error {
			if item.Name == "" {
				return nil
			}
			operations, err := shared.Lookup(item.APIVersion, item.Kind)
			if err != nil {
				return err
			}
			response.Items = append(response.Items, &ResourceInfo{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
			})
			deleteRequest, err := operations.NewRequest("Delete", item.Raw)
			if err != nil {
				return err
			}
			return endly.RunWithoutLogging(context, deleteRequest, nil)
		})
		return response, err
	}
	err := s.deleteResources(context, request, response)
	return response, err
}

func (s *service) deleteResources(context *endly.Context, request *DeleteRequest, response *DeleteResponse) error {
	if request.Resource != nil {
		return ProcessResource(context, false, request.Resource, true, func(meta *ResourceMeta, requestData map[string]interface{}) error {
			response.Items = append(response.Items, &ResourceInfo{
				TypeMeta:   meta.TypeMeta,
				ObjectMeta: meta.Metadata,
			})
			operations, err := shared.Lookup(meta.APIVersion, meta.Kind)
			if err != nil {
				return err
			}
			request, err := operations.NewRequest("Delete", requestData)
			if err != nil {
				return err
			}
			return endly.Run(context, request, nil)
		})
	}
	return nil
}

//Get returns information about resource(s) for supplied resources kind or name
func (s *service) Get(context *endly.Context, request *GetRequest) (*GetResponse, error) {
	response := &GetResponse{
		Items: make([]*ResourceInfo, 0),
	}
	err := s.get(context, request, func(item *ResourceInfo) error {
		response.Items = append(response.Items, item)
		return nil
	})
	return response, err
}

func (s *service) get(context *endly.Context, request *GetRequest, handler func(item *ResourceInfo) error) error {
	if len(request.kinds) == 0 {
		return nil
	}
	ctxClient, err := shared.GetCtxClient(context)
	if err != nil {
		return err
	}
	if request.apiKindMethod_ == "" {
		if err = request.Init(); err != nil {
			return err
		}
	}
	if len(ctxClient.RawRequest) == 0 {
		ctxClient.RawRequest = make(map[string]interface{})
		if err = converter.AssignConverted(&ctxClient.RawRequest, request); err != nil {
			return err
		}
		ctxClient.RawRequest = toolbox.DeleteEmptyKeys(ctxClient.RawRequest)
	}

	for _, kind := range request.kinds {
		ctxClient.RawRequest["kind"] = kind
		apiVersion := request.APIVersion
		operations, err := shared.Lookup(apiVersion, kind)
		if err != nil {
			return err
		}
		getRequest, err := operations.NewRequest(request.apiKindMethod_, ctxClient.RawRequest)
		if err != nil {
			return err
		}
		var response interface{}
		if err = endly.RunWithoutLogging(context, getRequest, &response); err != nil {
			if err != nil {
				return err
			}
		}
		if shared.IsNotFound(response) {
			if len(request.kinds) == 1 {
				return fmt.Errorf("%v", response)
			}
			continue
		}

		var responseMap = make(map[string]interface{})
		if err = converter.AssignConverted(&responseMap, response); err != nil {
			return err
		}
		responseMap = toolbox.DeleteEmptyKeys(responseMap)
		itemsValue, ok := responseMap["items"]
		if !ok || itemsValue == nil || !toolbox.IsSlice(itemsValue) {
			info := &ResourceInfo{}
			if err := converter.AssignConverted(info, responseMap); err != nil {
				return err
			}
			info.Raw = responseMap
			info.TypeMeta = operations.TypeMeta
			if err = handler(info); err != nil {
				return err
			}
			continue
		}
		for _, item := range toolbox.AsSlice(itemsValue) {
			info := &ResourceInfo{}
			if err := converter.AssignConverted(info, item); err != nil {
				return err
			}
			info.Raw = item
			info.TypeMeta = operations.TypeMeta
			if err = handler(info); err != nil {
				return err
			}
		}
	}
	return nil
}

//RunTemplate applies k8 resource template based on supplied parameters and create a resource
func (s *service) RunTemplate(context *endly.Context, request *RunRequest) (*RunResponse, error) {
	ctxClient, err := shared.GetCtxClient(context)
	if err != nil {
		return nil, err
	}
	var createResponse *CreateResponse

	requestData, err := NewRunTemplateParams(request)
	if err != nil {
		return nil, err
	}
	if err := GenerateRequest(request.Template, runTemplates, requestData, func(meta *ResourceMeta, rawRequest map[string]interface{}) error {
		ctxClient.RawRequest = rawRequest

		createResponse, err = s.Create(context, &CreateRequest{ResourceMeta: meta})
		return err
	}); err != nil {
		return nil, err
	}
	response := RunResponse(*createResponse)
	return &response, nil
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "get",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "get", &GetRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &struct{}{}),
		},
		RequestProvider: func() interface{} {
			return &GetRequest{}
		},
		ResponseProvider: func() interface{} {
			return &struct{}{}
		},
		OnRawRequest: shared.Init,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetRequest); ok {
				output, err := s.Get(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					var outputEvent interface{}
					if req.Describe {
						outputEvent = output
					} else {
						outputEvent = NewShortResourceResponse(ResourceInfoResponse(*output))
					}
					context.Publish(gcp.NewOutputEvent("...", "get", outputEvent))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "create",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "create", &CreateRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &struct{}{}),
		},
		RequestProvider: func() interface{} {
			return &CreateRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CreateResponse{}
		},
		OnRawRequest: shared.Init,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CreateRequest); ok {
				output, err := s.Create(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "create", normalizeOutput(ToResourceMetas(output.Items))))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "apply",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "apply", &ApplyRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &ApplyResponse{}),
		},
		RequestProvider: func() interface{} {
			return &ApplyRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ApplyResponse{}
		},
		OnRawRequest: shared.Init,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ApplyRequest); ok {
				output, err := s.Apply(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "apply", normalizeOutput(ToResourceMetas(output.Items))))
				}
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
		OnRawRequest: shared.Init,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DeleteRequest); ok {
				output, err := s.Delete(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "delete", normalizeOutput(ToResourceMetas(output.Items))))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "run",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "apply", &RunRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &RunResponse{}),
		},
		RequestProvider: func() interface{} {
			return &RunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RunResponse{}
		},
		OnRawRequest: shared.Init,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RunRequest); ok {
				output, err := s.RunTemplate(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "run", normalizeOutput(ToResourceMetas(output.Items))))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "expose",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "expose", &ExposeRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &ExposeResponse{}),
		},
		RequestProvider: func() interface{} {
			return &ExposeRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ExposeResponse{}
		},
		OnRawRequest: shared.Init,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ExposeRequest); ok {
				output, err := s.Expose(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "run", normalizeOutput(ToResourceMetas(output.Items))))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new Storage service
func New() *service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
