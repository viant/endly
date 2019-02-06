package core

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"github.com/viant/endly/system/kubernetes/shared"
	"github.com/viant/toolbox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	//ServiceID Kubernetes service ID.
	ServiceID = "kubernetes"
)

//no operation service
type service struct {
	*endly.AbstractService
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
					context.Publish(gcp.NewOutputEvent("...", "get", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "create",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "create", &GetRequest{}),
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
					context.Publish(gcp.NewOutputEvent("...", "create", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

func (s *service) Create(context *endly.Context, request *CreateRequest) (*CreateResponse, error) {
	response := &CreateResponse{
		Items: make([]interface{}, 0),
	}
	err := ProcessResource(context, request.Resource, func(meta *metav1.TypeMeta, requestData map[string]interface{}) error {
		kindOperations, err := shared.Lookup(meta.APIVersion, meta.Kind)
		if err != nil {
			return err
		}
		item, err := s.createOrUpdate(context, kindOperations, requestData)
		if err != nil {
			return err
		}
		response.Items = append(response.Items, item)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *service) createOrUpdate(context *endly.Context, operations *shared.KindOperations, requestData interface{}) (interface{}, error) {
	//TODO check if resource exist to update
	adapterRequest, err := operations.Lookup("Create")
	if err != nil {
		return nil, err
	}

	var response interface{}
	if err = buildRequest(context, adapterRequest, requestData); err != nil {
		return nil, err
	}
	logging := context.Logging
	context.SetLogging(false)
	defer func() {
		context.Logging = logging
	}()
	if err = endly.Run(context, adapterRequest, &response); err != nil {
		return nil, err
	}
	return response, err
}

//Get returns information about resource(s) for supplied resources kind or name
func (s *service) Get(context *endly.Context, request *GetRequest) (GetResponse, error) {
	kindOperations, err := shared.Lookup(request.APIVersion, request.Kind)
	if err != nil {
		return nil, err
	}

	adapterRequest, err := kindOperations.Lookup(request.apiKindMethod_)
	if err != nil {
		return nil, err
	}
	var response interface{}
	if err = buildRequest(context, adapterRequest, request); err != nil {
		return nil, err
	}

	logging := context.Logging
	context.SetLogging(false)
	err = endly.Run(context, adapterRequest, &response)
	context.Logging = logging
	if err != nil {
		return nil, err
	}

	var resultMap = make(map[string]interface{})
	if err := converter.AssignConverted(resultMap, response); err != nil {
		return nil, err
	}
	outputRules := getOutputRules(request.OutputTemplate, request.Kind, request.OutputPaths)
	if len(outputRules) == 0 || request.OutputTemplate == "*" {
		return resultMap, nil
	}
	resultMap = toolbox.DeleteEmptyKeys(resultMap)
	var result = make(map[string]interface{})
	if !request.multiItem {
		transformOutput(outputRules, resultMap, result)
	} else {
		transformedItems := make([]interface{}, 0)
		itemValue, ok := resultMap["items"]
		if ! ok || itemValue == nil {
			return resultMap, nil
		}
		items := toolbox.AsSlice(itemValue)
		for _, item := range items {
			itemMap := toolbox.AsMap(item)
			var transformed = make(map[string]interface{})
			transformOutput(outputRules, itemMap, transformed)
			transformedItems = append(transformedItems, transformed)
		}
		result["items"] = transformedItems
	}
	return result, nil
}

func buildRequest(context *endly.Context, target shared.ContractAdapter, request interface{}) error {
	ctxClient, err := shared.GetCtxClient(context)
	if err != nil {
		return err
	}
	return converter.AssignConverted(target, ctxClient.RawRequest)
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
