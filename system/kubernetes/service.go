package core

import (
	"bytes"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/system/cloud/gcp"
	"github.com/viant/endly/system/kubernetes/shared"
	"github.com/viant/toolbox"
	"io"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
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
	err := s.appleResources(request, context, response)
	return response, err
}

func (s *service) appleResources(request *ApplyRequest, context *endly.Context, response *ApplyResponse) error {
	if request.ResourceMeta != nil && request.Kind != "" {
		ctxClient, err := shared.GetCtxClient(context)
		if err != nil {
			return err
		}
		return s.appleResource(context, request, response, request.ResourceMeta, ctxClient.RawRequest)
	}
	if request.Resource == nil {
		return nil
	}
	err := ProcessResource(context, request.Expand, request.Resource, false, func(meta *ResourceMeta, requestData map[string]interface{}) error {
		return s.appleResource(context, request, response, meta, requestData)
	})
	return err
}

func (s *service) appleResource(context *endly.Context, request *ApplyRequest, response *ApplyResponse, meta *ResourceMeta, requestData map[string]interface{}) error {
	operations, err := shared.Lookup(meta.APIVersion, meta.Kind)
	if err != nil {
		return err
	}
	createResponse := &ResourceInfo{}
	createRequest, err := operations.NewRequest("Create", requestData)
	if err != nil {
		return err
	}
	getRequest, err := operations.NewRequest("Get", meta.Metadata)
	if err != nil {
		return err
	}
	var getResponse interface{}
	if err = endly.RunWithoutLogging(context, getRequest, &getResponse); err != nil {
		return err
	}

	if shared.IsNotFound(getResponse) {
		if err = endly.RunWithoutLogging(context, createRequest, &createResponse); err != nil {
			return err
		}
		response.Items = append(response.Items, createResponse)
		return nil
	}

	pathData, err := NewResourcePatch(meta, getResponse, createRequest)
	if err != nil {
		return err
	}

	if !pathData.HasChanged {
		_ = converter.AssignConverted(createResponse, getResponse)
		response.Items = append(response.Items, createResponse)
		return nil
	}
	patchRequest, err := operations.NewRequest("Patch", pathData)
	if err != nil {
		return err
	}
	if err = endly.RunWithoutLogging(context, patchRequest, &createResponse); err != nil {
		return err
	}
	response.Items = append(response.Items, createResponse)
	return nil
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
			return s.deleteResource(context, request, response, item)
		})
		return response, err
	}
	err := s.deleteResources(context, request, response)
	return response, err
}

func (s *service) deleteResource(context *endly.Context, request *DeleteRequest, response *DeleteResponse, item *ResourceInfo) error {
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

	if item.Name != "" && toolbox.IsMap(item.Raw) {
		rawRequest := toolbox.AsMap(item.Raw)
		rawRequest["Name"] = item.Name
	}

	getRequest, _ := operations.NewRequest("Get", item.Raw)

	deleteRequest, err := operations.NewRequest("Delete", item.Raw)
	if err != nil {
		return err
	}
	if err = endly.RunWithoutLogging(context, deleteRequest, nil); err != nil {
		return err
	}
	return s.waitForNotFound(context, getRequest, request.TimeoutMs)
}

func (s *service) deleteResources(context *endly.Context, request *DeleteRequest, response *DeleteResponse) error {
	if request.Resource != nil {
		return ProcessResource(context, false, request.Resource, true, func(meta *ResourceMeta, requestData map[string]interface{}) error {
			item := &ResourceInfo{}
			item.TypeMeta = meta.TypeMeta
			item.ObjectMeta = meta.Metadata
			item.Raw = requestData
			return s.deleteResource(context, request, response, item)
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
		operations, err := shared.Lookup(request.APIVersion, kind)
		if err != nil {
			return err
		}
		response, err := s.getResource(context, request, kind, ctxClient.RawRequest)
		if err != nil {
			return err
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

func (s *service) getResource(context *endly.Context, request *GetRequest, kind string, requestData interface{}) (interface{}, error) {
	ctxClient, err := shared.GetCtxClient(context)
	if err != nil {
		return nil, err
	}
	ctxClient.RawRequest["kind"] = kind
	apiVersion := request.APIVersion
	operations, err := shared.Lookup(apiVersion, kind)
	if err != nil {
		return nil, err
	}
	getRequest, err := operations.NewRequest(request.apiKindMethod_, requestData)
	if err != nil {
		return nil, err
	}
	var response interface{}
	err = endly.RunWithoutLogging(context, getRequest, &response)
	return response, nil
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

	s.Register(&endly.Route{
		Action: "forwardPorts",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "expose", &ForwardPortsRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &ForwardPortsResponse{}),
		},
		RequestProvider: func() interface{} {
			return &ForwardPortsRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ForwardPortsResponse{}
		},
		OnRawRequest: shared.Init,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ForwardPortsRequest); ok {
				output, err := s.Forward(context, req)
				if err != nil {
					return nil, err
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) getPod(context *endly.Context, request *GetRequest, timeoutMs int) (*v1.Pod, error) {
	ctxClient, err := shared.GetCtxClient(context)
	if err != nil {
		return nil, err
	}
	if err = converter.AssignConverted(&ctxClient.RawRequest, request); err != nil {
		return nil, err
	}

	startTime := time.Now()
	timeout := time.Duration(timeoutMs) * time.Millisecond


	for ;time.Now().Sub(startTime) < timeout; {
		resource, err := s.getResource(context, request, request.Kind, ctxClient.RawRequest)
		if err != nil {
			return nil, err
		}
		getPodRequest := &GetRequest{}
		getPodRequest.Kind = "Pod"
		switch val := resource.(type) {
		case *v1.Pod:
			return val, nil
		case *v1.PodList:
			if len(val.Items) == 0 {
				time.Sleep(time.Second)
				continue
			}
			for _, candidate := range val.Items {
				if candidate.Status.Phase == v1.PodRunning {
					return &candidate, nil
				}
			}
			return &val.Items[0], nil
		default:
			resourceInfo := &ResourceInfo{}
			if err = converter.AssignConverted(resourceInfo, resource); err != nil {
				return nil, err
			}
			var specMap= make(map[string]interface{})
			if err = converter.AssignConverted(&specMap, resourceInfo.Spec); err != nil {
				return nil, err
			}
			rawSelector, _ := specMap["Selector"]
			switch selector := rawSelector.(type) {
			case string:
				getPodRequest.LabelSelector = selector
			case map[string]string:
				getPodRequest.LabelSelector = shared.ToSelector(selector)
			}
			_ = getPodRequest.Init()
			if getPodRequest.LabelSelector == "" {
				return nil, fmt.Errorf("service selector was empty")
			}

			return s.getPod(context, getPodRequest, timeoutMs)
		}
	}
	return nil, fmt.Errorf("getPod timeout exceeded")
}

func (s *service) Forward(context *endly.Context, request *ForwardPortsRequest) (*ForwardPortsResponse, error) {
	getRequest, err := request.AsGetRequest()
	if err != nil {
		return nil, err
	}
	pod, err := s.getPod(context, getRequest, request.TimeoutMs)
	if err != nil {
		return nil, err
	}
	if err = s.waitForPodReadyIfNeeded(context, pod, request.TimeoutMs); err != nil {
		return nil, err
	}

	response := &ForwardPortsResponse{
		Name: pod.Name,
	}
	ctxClient, err := shared.GetCtxClient(context)
	if err != nil {
		return nil, err
	}
	roundTripper, upgrader, err := spdy.RoundTripperFor(ctxClient.ResetConfig)
	if err != nil {
		return nil, err
	}
	forwardURL := &url.URL{Scheme: "https", Path: fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", ctxClient.Namespace, pod.Name), Host: ctxClient.EndpointIP()}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, forwardURL)
	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	context.Deffer(func() {
		stop := struct{}{}
		select {
		case stopChan <- stop:
		case <-time.After(time.Millisecond):
		}
	})
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	forwarder, err := portforward.New(dialer, request.Ports, stopChan, readyChan, out, errOut)
	if err != nil {
		return nil, err
	}


	go func() {
		select {
		case <-readyChan:
		}
		if len(errOut.String()) != 0 {
			context.Publish(msg.NewErrorEvent(errOut.String()))
		} else if len(out.String()) != 0 {
			context.Publish(shared.NewOutputEvent(forwardURL.String(), "forwardPort", out.String()))
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		context.Publish(shared.NewOutputEvent(forwardURL.String(), "forwardPort", nil))
		wg.Done()
		if err = forwarder.ForwardPorts(); err != nil {
			log.Print(err)
		}
	}()
	wg.Wait()
	return response, nil
}

func (s *service) waitForPodReadyIfNeeded(context *endly.Context, pod *v1.Pod, timeoutMs int) error {
	getPodRequest := &GetRequest{}
	getPodRequest.Kind = "pod"
	getPodRequest.Name = pod.Name
	_ = getPodRequest.Init()
	timeout := time.Duration(timeoutMs) * time.Millisecond
	startTime := time.Now()
	var ok bool
	phase := ""
	for ; time.Now().Sub(startTime) <= timeout; {
		response, err := s.getResource(context, getPodRequest, "pod", getPodRequest)
		if err != nil {
			return err
		}
		if shared.IsNotFound(response) {
			time.Sleep(time.Second)
			continue
		}
		pod, ok = response.(*v1.Pod)
		if ! ok {
			return fmt.Errorf("unable determine pod type expected %T but had: %T", pod, response)
		}
		if phase != string(pod.Status.Phase) {
			context.Publish(shared.NewOutputEvent(fmt.Sprintf("pod/%v - %v", pod.Name, pod.Status.Phase), "waitForPod", nil))
		}
		phase = string(pod.Status.Phase)
		if pod.Status.Phase == v1.PodRunning {
			return nil
		}
		s.Sleep(context, 1000)
	}
	return fmt.Errorf("podRunning timeout exceeded")
}

func (s *service) waitForNotFound(context *endly.Context, getRequest interface{}, timeoutMs int) error {
	var response interface{}

	timeout := time.Duration(timeoutMs) * time.Millisecond
	startTime := time.Now()
	for ; time.Now().Sub(startTime) <= timeout; {
		if err := endly.RunWithoutLogging(context, getRequest, &response); err != nil {
			return err
		}
		if shared.IsNotFound(response) {
			return nil
		}
		time.Sleep(time.Second)
	}

	return fmt.Errorf("timeout exceeded")
}

//New creates a new service
func New() *service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
