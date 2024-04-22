package apigateway

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/lambda"
	"strings"
)

// GetRestAPIInput endpoint URL inpit
type GetRestAPIInput struct {
	Name      *string `json:",inline"`
	StageName *string ` json:",inline"`
}

func (i *GetRestAPIInput) Init() error {
	if i.StageName == nil {
		i.StageName = aws.String("e2e")
	}
	return nil
}

func (i *GetRestAPIInput) Validate() error {
	if i.Name == nil {
		return errors.New("name was empty")
	}
	return nil
}

// GetRestAPIInput endpoint URL output
type GetRestAPIOutput struct {
	*apigateway.RestApi
	EndpointURL string ` json:",inline"`
}

// SetupRestAPIInput represent a request to setup API with specified resources
type SetupRestAPIInput struct {
	apigateway.CreateRestApiInput    ` json:",inline"`
	Resources                        []*SetupResourceInput
	apigateway.CreateDeploymentInput ` json:",inline"`

	Authorizers     []*CreateAuthorizerInput
	GatewayResponse []*apigateway.PutGatewayResponseInput
	Redeploy        bool
}

type CreateAuthorizerInput struct {
	apigateway.CreateAuthorizerInput
	FunctionName string
}

// Diff computes patches
func (i CreateAuthorizerInput) Diff(source *apigateway.Authorizer) []*apigateway.PatchOperation {
	var result = make([]*apigateway.PatchOperation, 0)
	if patch, ok := patchString(source.AuthorizerUri, i.AuthorizerUri, "/authorizerUri"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.AuthType, i.AuthType, "/authType"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.Type, i.Type, "/authorizerType"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.Name, i.Name, "/name"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.AuthorizerCredentials, i.AuthorizerCredentials, "/authorizerCredentials"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.IdentitySource, i.IdentitySource, "/identitySource"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.IdentityValidationExpression, i.IdentityValidationExpression, "/identityValidationExpression"); ok {
		result = append(result, patch)
	}
	if patch, ok := pathInt64(source.AuthorizerResultTtlInSeconds, i.AuthorizerResultTtlInSeconds, "/authorizerResultTtlInSeconds"); ok {
		result = append(result, patch)
	}

	return result
}

// SetupRestAPIInput represent setup API response
type SetupRestAPIOutput struct {
	*apigateway.RestApi
	Resources   []*SetupResourceOutput
	Authorizers []*apigateway.Authorizer
	Stage       *apigateway.Stage
	EndpointURL string
	Region      string
}

// SetupResourceInput represents resource input
type SetupResourceInput struct {
	Path                           string
	apigateway.CreateResourceInput `json:",inline"`
	Methods                        []*ResourceMethod
}

// ResourceMethod represents resource method
type ResourceMethod struct {
	FunctionName string
	Authorizer   string
	HttpMethod   string
	*apigateway.PutMethodInput
	*apigateway.PutIntegrationInput
	*lambda.AddPermissionInput
}

// SetupResourceOutput represents setup resource output
type SetupResourceOutput struct {
	*apigateway.Resource `json:",inline"`
	ResourceMethods      map[string]*apigateway.Method
}

// RemoveRestAPI removes API for supplied name
type RemoveRestAPIInput struct {
	Name *string
}

func (i *RemoveRestAPIInput) Validate() error {
	if i.Name == nil {
		return errors.New("name was empty")
	}
	return nil
}

func (i *SetupRestAPIInput) Init() error {
	if len(i.Resources) == 0 {
		return nil
	}

	var URIs = make(map[string]bool)
	var resources = make([]*SetupResourceInput, 0)

	if len(i.Authorizers) > 0 {
		for _, auth := range i.Authorizers {
			_ = auth.Init()
		}
	}

	for _, resource := range i.Resources {
		if err := resource.Init(); err != nil {
			return err
		}
		URIs[resource.Path] = true

		parts := strings.Split(resource.Path, "/")

		for i := 1; i < len(parts)-1; i++ {
			ancestor := strings.Join(parts[0:i+1], "/")
			if URIs[ancestor] {
				continue
			}
			ancestorResource := &SetupResourceInput{Path: ancestor}
			_ = ancestorResource.Init()
			resources = append(resources, ancestorResource)
		}
		resources = append(resources, resource)
	}
	i.Resources = resources
	return nil
}

// Validate checks is input is valud
func (i *SetupRestAPIInput) Validate() error {
	if len(i.Resources) == 0 {
		return fmt.Errorf("resources was empty")
	}
	for _, resource := range i.Resources {
		if err := resource.Validate(); err != nil {
			return err
		}
	}
	createDeployment := &i.CreateDeploymentInput
	if createDeployment.StageName == nil {
		createDeployment.StageName = aws.String("e2e")
	}
	return nil
}

// Init initialize input
func (i *SetupResourceInput) Init() error {
	if i.Path == "" {
		if i.PathPart != nil {
			i.Path = "/" + *i.PathPart
		} else {
			i.Path = "/"
		}
	}
	if i.PathPart == nil {
		parts := strings.Split(i.Path, "/")
		if len(parts) > 1 {
			i.PathPart = &parts[len(parts)-1]
		}
	}
	if len(i.Methods) == 0 {
		return nil
	}
	for _, method := range i.Methods {
		if err := method.Init(); err != nil {
			return err
		}
	}
	return nil
}

func (i *SetupResourceInput) Validate() error {
	resourceInput := i.CreateResourceInput
	if resourceInput.PathPart == nil {
		return fmt.Errorf("pathPart was empty")
	}
	if len(i.Methods) > 0 {
		for _, method := range i.Methods {
			if err := method.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

// ParentPath returns parent path
func (i *SetupResourceInput) ParentPath() string {
	if i.PathPart == nil {
		return i.Path
	}
	index := strings.LastIndex(i.Path, *i.PathPart)
	if index == -1 {
		return i.Path
	}
	result := string(i.Path[:index])
	if result == "/" {
		return result
	}
	if strings.HasSuffix(result, "/") {
		return string(result[:len(result)-1])
	}
	return result
}

/*
Initialise resource with the following defaults

<ul>

	<li>AuthorizationType: NONE</li>
	<li>HttpMethod: ANY</li>
	<li>ApiKeyRequired: false</li>
	<li>IntegrationHttpMethod: POST</li>
	<li>Type: AWS_PROXY</li>
	<li>Uri: arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/${function.arn}/invocations</li>

</ul>

It supports data substitution on uri attribute i.e.
- 	$function.arn
-	$function.region
-	$function.accountID
-	$restAPI.SessionID
-	$uuid.next
*/
func (i *ResourceMethod) Init() error {

	methodInput := i.PutMethodInput
	if methodInput == nil {
		i.PutMethodInput = &apigateway.PutMethodInput{}
		methodInput = i.PutMethodInput
	}

	if methodInput.AuthorizationType == nil {
		methodInput.AuthorizationType = aws.String("NONE")
	}
	if methodInput.ApiKeyRequired == nil {
		methodInput.ApiKeyRequired = aws.Bool(false)
	}
	integrationInput := i.PutIntegrationInput
	if integrationInput == nil {
		i.PutIntegrationInput = &apigateway.PutIntegrationInput{}
		integrationInput = i.PutIntegrationInput
	}

	if i.HttpMethod != "" {
		methodInput.HttpMethod = &i.HttpMethod
		integrationInput.HttpMethod = &i.HttpMethod
	}

	if integrationInput.HttpMethod == nil {
		integrationInput.HttpMethod = methodInput.HttpMethod
	}
	if integrationInput.IntegrationHttpMethod == nil {
		integrationInput.IntegrationHttpMethod = aws.String("POST")
	}

	if i.FunctionName != "" {
		if integrationInput.Type == nil {
			integrationInput.Type = aws.String("AWS_PROXY")
		}
		if integrationInput.Uri == nil {
			integrationInput.Uri = aws.String("arn:aws:apigateway:${function.region}:lambda:path/2015-03-31/functions/${function.arn}/invocations")
		}
		permissionInput := i.AddPermissionInput
		if i.AddPermissionInput == nil {
			i.AddPermissionInput = &lambda.AddPermissionInput{}
			permissionInput = i.AddPermissionInput
		}
		if permissionInput.FunctionName == nil {
			permissionInput.FunctionName = &i.FunctionName
		}
		if permissionInput.Action == nil {
			permissionInput.Action = aws.String("lambda:InvokeFunction")
		}
		if permissionInput.Principal == nil {
			permissionInput.Principal = aws.String("apigateway.amazonaws.com")
		}
		if permissionInput.StatementId == nil {
			permissionInput.StatementId = aws.String("${uuid.next}")
		}
		if permissionInput.SourceArn == nil {
			permissionInput.SourceArn = aws.String("arn:aws:execute-api:${function.region}:${function.accountID}:${restAPI.SessionID}/*/*/*")
		}
	}
	return nil
}

func (i *CreateAuthorizerInput) Init() error {
	if i.FunctionName != "" {
		i.AuthorizerUri = aws.String("arn:aws:apigateway:${authorizer.region}:lambda:path/2015-03-31/functions/${authorizer.arn}/invocations")
	}
	if i.AuthType == nil {
		i.AuthType = aws.String("custom")
	}
	if i.AuthorizerResultTtlInSeconds == nil {
		i.AuthorizerResultTtlInSeconds = aws.Int64(0)
	}

	return nil
}

func (i *ResourceMethod) Validate() error {
	if i.HttpMethod == "" {
		return fmt.Errorf("httpMethod was empty")
	}
	if i.Type == nil {
		return fmt.Errorf("type was empty")
	}
	return nil
}

// SetupMethodInput setups method input
type SetupMethodInput apigateway.PutMethodInput

// Diff computes patches
func (i SetupMethodInput) Diff(source *apigateway.Method) []*apigateway.PatchOperation {
	var result = make([]*apigateway.PatchOperation, 0)
	if patch, ok := patchBool(source.ApiKeyRequired, i.ApiKeyRequired, "/apiKeyRequired"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.AuthorizationType, i.AuthorizationType, "/authorizationType"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.AuthorizerId, i.AuthorizerId, "/authorizerId"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.OperationName, i.OperationName, "/operationName"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.RequestValidatorId, i.RequestValidatorId, "/requestValidatorId"); ok {
		result = append(result, patch)
	}
	return result
}

type PutIntegrationInput apigateway.PutIntegrationInput

// Diff computes patches
func (i PutIntegrationInput) Diff(source *apigateway.Integration) []*apigateway.PatchOperation {
	var result = make([]*apigateway.PatchOperation, 0)
	if patch, ok := patchString(source.Uri, i.Uri, "/uri"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.Credentials, i.Credentials, "/credentials"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.Type, i.Type, "/integrationType"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.ConnectionId, i.ConnectionId, "/connectionId"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.ConnectionType, i.ConnectionType, "/connectionType"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.ContentHandling, i.ContentHandling, "/contentHandling"); ok {
		result = append(result, patch)
	}
	if i.PassthroughBehavior != nil {
		if patch, ok := patchString(source.PassthroughBehavior, i.PassthroughBehavior, "/passthroughBehavior"); ok {
			result = append(result, patch)
		}
	}
	return result
}

type SetupDeploymentInput apigateway.CreateDeploymentInput

func (i *SetupDeploymentInput) Diff(source *apigateway.CreateDeploymentInput) []*apigateway.PatchOperation {
	var result = make([]*apigateway.PatchOperation, 0)
	if patch, ok := patchString(source.StageName, i.StageName, "/stageName"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.Description, i.Description, "/description"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchString(source.StageDescription, i.StageDescription, "/stageDescription"); ok {
		result = append(result, patch)
	}
	if patch, ok := patchBool(source.CacheClusterEnabled, i.CacheClusterEnabled, "/cacheClusterEnabled"); ok {
		result = append(result, patch)
	}

	return result
}
