package apigateway

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws"
	"time"

	"github.com/viant/endly/service/system/cloud/aws/lambda"
	"github.com/viant/toolbox"
	"log"
)

const (
	//ServiceID aws api gateway service id.
	ServiceID = "aws/apigateway"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) getRestApi(context *endly.Context, request *GetRestAPIInput) (*GetRestAPIOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	keysResponse, err := client.GetRestApis(&apigateway.GetRestApisInput{})
	if err != nil {
		return nil, err
	}
	var restAPI *apigateway.RestApi
	for _, item := range keysResponse.Items {
		if *item.Name == *request.Name {
			restAPI = item
			break
		}
	}
	output := &GetRestAPIOutput{
		RestApi: restAPI,
	}
	if restAPI == nil {
		return output, nil
	}
	region := ""
	if client.Config.Region != nil {
		region = *client.Config.Region
	}
	output.EndpointURL = fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/%s/",
		*restAPI.Id,
		region,
		*request.StageName)
	return output, nil
}

func (s *service) getAuthorizers(context *endly.Context, restAPI string) ([]*apigateway.Authorizer, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output, err := client.GetAuthorizers(&apigateway.GetAuthorizersInput{
		RestApiId: &restAPI,
	})
	if err != nil {
		return nil, err
	}
	return output.Items, nil
}

func (s *service) deployAPI(context *endly.Context, request *SetupRestAPIInput) (*SetupRestAPIOutput, error) {
	restAPI, resources, err := s.getOrCreateRestAPI(context, &request.CreateRestApiInput)
	if err != nil {
		return nil, err
	}
	response := &SetupRestAPIOutput{
		Resources: make([]*SetupResourceOutput, 0),
	}
	response.RestApi = restAPI
	if len(request.Resources) == 0 {
		return response, nil
	}
	request.RestApiId = restAPI.Id
	response.Authorizers, _ = s.getAuthorizers(context, *restAPI.Id)
	authorizedByName := indexAuthorizers(response.Authorizers, func(auth *apigateway.Authorizer) string {
		return *auth.Name
	})

	if err := s.deployGatewayResponse(context, request); err != nil {
		return nil, err
	}

	if err := s.deployAuthorizers(context, request, authorizedByName); err != nil {
		return nil, err
	}
	var byPath = indexResource(resources.Items)
	for _, resource := range request.Resources {
		parent, ok := byPath[resource.ParentPath()]
		if !ok {
			available := toolbox.MapKeysToStringSlice(byPath)
			return nil, fmt.Errorf("unable locate parent resource: %v, for part %v,  available: %s", resource.ParentPath(), *resource.PathPart, available)
		}
		resource.ParentId = parent.Id
		resourceOutput, err := s.setupResource(context, resource, restAPI, byPath, authorizedByName)
		if err != nil {
			return nil, err
		}
		response.Resources = append(response.Resources, resourceOutput)
		if len(resourceOutput.ResourceMethods) > 0 {
			for _, v := range resourceOutput.ResourceMethods {
				if v.MethodIntegration != nil && v.MethodIntegration.Uri != nil {
					if ARN, err := arn.Parse(*v.MethodIntegration.Uri); err == nil {
						response.Region = ARN.Region
					}
				}
			}
		}
	}
	deploymentInput := &request.CreateDeploymentInput
	deploymentInput.RestApiId = restAPI.Id
	if response.Stage, err = s.setupDeployment(context, deploymentInput, request.Redeploy); err != nil {
		return nil, err
	}
	if response.Region != "" {
		response.EndpointURL = fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/%v/",
			*response.Id,
			response.Region,
			*response.Stage.StageName)
	}
	return response, nil
}

func (s *service) deployGatewayResponse(context *endly.Context, request *SetupRestAPIInput) error {
	if len(request.GatewayResponse) == 0 {
		return nil
	}
	client, err := GetClient(context)
	if err != nil {
		return err
	}
	list, err := client.GetGatewayResponses(&apigateway.GetGatewayResponsesInput{RestApiId: request.RestApiId})
	if err != nil {
		return err
	}
	responses := indexGatewayResponseOutput(list)
	for _, gwResponse := range request.GatewayResponse {
		if existing, ok := responses[*gwResponse.StatusCode]; ok {
			if _, err = client.DeleteGatewayResponse(&apigateway.DeleteGatewayResponseInput{
				ResponseType: existing.ResponseType,
				RestApiId:    request.RestApiId,
			}); err != nil {
				return err
			}
		}
		gwResponse.RestApiId = request.RestApiId
		_, err := client.PutGatewayResponse(gwResponse)
		if err != nil {
			return errors.Wrapf(err, "failed to putGatewayResponse with %v", gwResponse)
		}
	}
	return nil
}

func indexGatewayResponseOutput(list *apigateway.GetGatewayResponsesOutput) map[string]*apigateway.UpdateGatewayResponseOutput {
	index := map[string]*apigateway.UpdateGatewayResponseOutput{}
	for i, item := range list.Items {
		index[*item.StatusCode] = list.Items[i]
	}
	return index
}

func (s *service) deployAuthorizers(context *endly.Context, request *SetupRestAPIInput, authorizers map[string]*apigateway.Authorizer) error {
	if len(request.Authorizers) == 0 {
		return nil
	}
	state := context.State()
	client, err := GetClient(context)
	if err != nil {
		return err
	}

	for _, authorizer := range request.Authorizers {
		if authorizer.FunctionName != "" {
			function, err := aws.GetFunctionConfiguration(context, authorizer.FunctionName)
			if err != nil {
				return err
			}
			aws.SetFunctionInfo("authorizer", function, state)
		}
		*authorizer.AuthorizerUri = state.ExpandAsText(*authorizer.AuthorizerUri)
		authorizer.RestApiId = request.RestApiId
		existing, ok := authorizers[*authorizer.Name]
		if ok {
			patchOperations := authorizer.Diff(existing)
			updateRequest := &apigateway.UpdateAuthorizerInput{
				AuthorizerId:    existing.Id,
				PatchOperations: patchOperations,
				RestApiId:       request.RestApiId,
			}
			if _, err = client.UpdateAuthorizer(updateRequest); err != nil {
				return errors.Wrapf(err, "failed to patch authorizer with %v", updateRequest)
			}
			return nil
		}
		output, err := client.CreateAuthorizer(&authorizer.CreateAuthorizerInput)
		if err != nil {
			return errors.Wrapf(err, "failed to create authorizer with %v", authorizer.CreateAuthorizerInput)
		}
		//let add extra time to  propagate changes
		time.Sleep(time.Second)
		authorizers[*output.Id] = output
	}
	return nil
}

func indexAuthorizers(authorizers []*apigateway.Authorizer, fn func(auth *apigateway.Authorizer) string) map[string]*apigateway.Authorizer {
	var index = make(map[string]*apigateway.Authorizer)
	if len(authorizers) == 0 {
		return index
	}
	for i, auth := range authorizers {
		index[fn(auth)] = authorizers[i]
	}
	return index
}

func (s *service) getOrCreateRestAPI(context *endly.Context, request *apigateway.CreateRestApiInput) (*apigateway.RestApi, *apigateway.GetResourcesOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, nil, err
	}
	keysResponse, err := client.GetRestApis(&apigateway.GetRestApisInput{})
	if err != nil {
		return nil, nil, err
	}
	var restAPI *apigateway.RestApi

	for _, item := range keysResponse.Items {
		if *item.Name == *request.Name {
			restAPI = item
			break
		}
	}
	if restAPI == nil {
		if restAPI, err = client.CreateRestApi(request); err != nil {
			return nil, nil, err
		}
	}
	resources, err := client.GetResources(&apigateway.GetResourcesInput{
		RestApiId: restAPI.Id,
	})
	if err != nil || len(resources.Items) == 0 {
		_, err := client.CreateResource(&apigateway.CreateResourceInput{
			RestApiId: restAPI.Id,
		})
		if err != nil {
			return nil, nil, err
		}
		resources, err = client.GetResources(&apigateway.GetResourcesInput{
			RestApiId: restAPI.Id,
		})
	}
	return restAPI, resources, err
}

func (s *service) setupResource(context *endly.Context, setup *SetupResourceInput, api *apigateway.RestApi, resources map[string]*apigateway.Resource, authorizers map[string]*apigateway.Authorizer) (*SetupResourceOutput, error) {
	response := &SetupResourceOutput{
		ResourceMethods: make(map[string]*apigateway.Method),
	}

	resourceInput := &setup.CreateResourceInput

	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	resource, ok := resources[setup.Path]
	if !ok {
		resourceInput.RestApiId = api.Id
		resourceInput.PathPart = setup.PathPart
		if resource, err = client.CreateResource(resourceInput); err != nil {
			return nil, err
		}
		resource.PathPart = resourceInput.PathPart
		resources[*resource.Path] = resource
	}

	response.Resource = resource
	if err = s.removeUnlistedMethods(client, api, resource, setup); err != nil {
		return nil, err
	}
	for _, resourceMethod := range setup.Methods {
		if resourceMethod.Authorizer != "" && resourceMethod.AuthorizationType != nil && "CUSTOM" == *resourceMethod.AuthorizationType {
			authorizer, ok := authorizers[resourceMethod.Authorizer]
			if !ok {
				return nil, errors.Errorf("failed to loolup %v authorizer", resourceMethod.Authorizer)
			}
			resourceMethod.AuthorizerId = authorizer.Id
		}

		method, err := s.setupResourceMethod(context, api, resource, resourceMethod)
		if err != nil {
			return nil, err
		}
		response.ResourceMethods[*method.HttpMethod] = method
	}
	return response, err
}

func (s *service) setupResourceMethod(context *endly.Context, api *apigateway.RestApi, resource *apigateway.Resource, resourceMethod *ResourceMethod) (*apigateway.Method, error) {
	var state = context.State()
	state = state.Clone()
	if resourceMethod.FunctionName != "" {
		function, err := aws.GetFunctionConfiguration(context, resourceMethod.FunctionName)
		if err != nil {
			return nil, err
		}
		aws.SetFunctionInfo("function", function, state)
		SetAPIInfo(api, state)

	}
	if resourceMethod.Uri != nil {
		*resourceMethod.Uri = state.ExpandAsText(*resourceMethod.Uri)
	}

	setupMethod := SetupMethodInput(*resourceMethod.PutMethodInput)
	setupMethod.RestApiId = api.Id
	setupMethod.ResourceId = resource.Id
	method, err := s.setupMethod(context, &setupMethod)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to setup method integration: %v, %v", method, resourceMethod.PutIntegrationInput)
	}
	integrationInput := resourceMethod.PutIntegrationInput
	integrationInput.RestApiId = api.Id
	integrationInput.ResourceId = resource.Id
	integrationInput.HttpMethod = setupMethod.HttpMethod
	method.MethodIntegration, err = s.setupMethodIntegration(context, method, resourceMethod.PutIntegrationInput)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create method integration: %v, %v", method, resourceMethod.PutIntegrationInput)
	}
	permissionInput := resourceMethod.AddPermissionInput

	if resourceMethod.FunctionName != "" && permissionInput != nil {
		*permissionInput.SourceArn = state.ExpandAsText(*permissionInput.SourceArn)
		*permissionInput.StatementId = state.ExpandAsText(*permissionInput.StatementId)

		request := lambda.SetupPermissionInput(*permissionInput)
		if err = endly.Run(context, &request, nil); err != nil {
			return nil, err
		}
	}
	return method, nil
}

func (s *service) setupMethodIntegration(context *endly.Context, method *apigateway.Method, request *apigateway.PutIntegrationInput) (*apigateway.Integration, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}

	if method.MethodIntegration == nil || method.MethodIntegration.Uri == nil {
		return client.PutIntegration(request)
	}
	patchOperations := PutIntegrationInput(*request).Diff(method.MethodIntegration)
	if len(patchOperations) == 0 {
		return method.MethodIntegration, nil
	}
	return client.UpdateIntegration(&apigateway.UpdateIntegrationInput{
		HttpMethod:      request.HttpMethod,
		RestApiId:       request.RestApiId,
		ResourceId:      request.ResourceId,
		PatchOperations: patchOperations,
	})
}

func (s *service) setupMethod(context *endly.Context, request *SetupMethodInput) (*apigateway.Method, error) {
	if request.HttpMethod == nil {
		return nil, nil
	}
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	existingMethod, err := client.GetMethod(&apigateway.GetMethodInput{
		RestApiId:  request.RestApiId,
		ResourceId: request.ResourceId,
		HttpMethod: request.HttpMethod,
	})

	if err != nil || existingMethod == nil {
		putMethod := apigateway.PutMethodInput(*request)
		method, err := client.PutMethod(&putMethod)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create method input: %v", putMethod)
		}
		return method, nil
	}

	patchOperations := request.Diff(existingMethod)
	if len(patchOperations) == 0 {
		return existingMethod, nil
	}
	updateRequest := &apigateway.UpdateMethodInput{
		HttpMethod:      existingMethod.HttpMethod,
		ResourceId:      request.ResourceId,
		RestApiId:       request.RestApiId,
		PatchOperations: patchOperations,
	}

	method, err := client.UpdateMethod(updateRequest)
	if err != nil {
		client.DeleteMethod(&apigateway.DeleteMethodInput{
			RestApiId:  request.RestApiId,
			ResourceId: request.ResourceId,
			HttpMethod: request.HttpMethod,
		})
		putMethod := apigateway.PutMethodInput(*request)
		method, err = client.PutMethod(&putMethod)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to recreate method input: %v", putMethod)
		}
	}
	return method, nil
}

func (s *service) removeUnlistedMethods(client *apigateway.APIGateway, api *apigateway.RestApi, resource *apigateway.Resource, input *SetupResourceInput) error {
	var listedMethods = make(map[string]bool)
	for _, method := range input.Methods {
		listedMethods[method.HttpMethod] = true
	}
	for k := range resource.ResourceMethods {
		if _, ok := listedMethods[k]; !ok {
			if _, err := client.DeleteMethod(&apigateway.DeleteMethodInput{
				HttpMethod: &k,
				ResourceId: resource.Id,
				RestApiId:  api.Id,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) setupDeployment(context *endly.Context, request *apigateway.CreateDeploymentInput, redeploy bool) (*apigateway.Stage, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}

	if deployment, _ := client.GetDeployment(&apigateway.GetDeploymentInput{
		RestApiId: request.RestApiId,
	}); deployment != nil {
		if stage, err := s.getStage(context, deployment, *request.RestApiId, *request.StageName); err == nil {
			if redeploy {
				_, _ = client.DeleteStage(&apigateway.DeleteStageInput{
					RestApiId: request.RestApiId,
					StageName: stage.StageName,
				})
				_, _ = client.DeleteDeployment(&apigateway.DeleteDeploymentInput{
					DeploymentId: deployment.Id,
					RestApiId:    request.RestApiId,
				})
			} else {
				return stage, err
			}
		}
	}
	deployment, err := client.CreateDeployment(request)
	if err != nil {
		return nil, err
	}
	return s.getStage(context, deployment, *request.RestApiId, *request.StageName)
}

func (s *service) getStage(context *endly.Context, deployment *apigateway.Deployment, restApiId, stageName string) (*apigateway.Stage, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	stages, err := client.GetStages(&apigateway.GetStagesInput{
		DeploymentId: deployment.Id,
		RestApiId:    &restApiId,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range stages.Item {
		if *item.StageName == stageName {
			return item, nil
		}
	}
	return nil, fmt.Errorf("failed to lookup stage for name: %v, api %v", stageName, restApiId)
}

func (s *service) removeRestAPI(context *endly.Context, request *RemoveRestAPIInput) (*apigateway.DeleteRestApiOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	keysResponse, err := client.GetRestApis(&apigateway.GetRestApisInput{})
	if err != nil {
		return nil, err
	}
	var restAPI *apigateway.RestApi

	for _, item := range keysResponse.Items {
		if *item.Name == *request.Name {
			restAPI = item
			break
		}
	}
	if restAPI == nil {
		return nil, nil
	}
	return client.DeleteRestApi(&apigateway.DeleteRestApiInput{
		RestApiId: restAPI.Id,
	})
}

func (s *service) registerRoutes() {
	client := &apigateway.APIGateway{}
	routes, err := aws.BuildRoutes(client, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}

	for _, route := range routes {
		route.OnRawRequest = setClient
		s.Register(route)
	}

	s.Register(&endly.Route{
		Action: "deployAPI",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "deployAPI", &SetupRestAPIInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &SetupRestAPIOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetupRestAPIInput{}
		},
		ResponseProvider: func() interface{} {
			return &SetupRestAPIOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupRestAPIInput); ok {
				return s.deployAPI(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "removeRestAPI",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "removeRestAPI", &RemoveRestAPIInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &apigateway.DeleteRestApiOutput{}),
		},
		RequestProvider: func() interface{} {
			return &RemoveRestAPIInput{}
		},
		ResponseProvider: func() interface{} {
			return &apigateway.DeleteRestApiOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RemoveRestAPIInput); ok {
				return s.removeRestAPI(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "getRestAPI",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getRestAPI", &GetRestAPIInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetRestAPIOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetRestAPIInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetRestAPIOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetRestAPIInput); ok {
				return s.getRestApi(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

// New creates a new AWS API Gateway service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
