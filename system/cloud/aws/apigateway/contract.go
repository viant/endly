package apigateway

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/lambda"
	"strings"
)

//SetupRestAPIInput represent a request to setup API with specified resources
type SetupRestAPIInput struct {
	apigateway.CreateRestApiInput
	Resources []*SetupResourceInput
	apigateway.CreateDeploymentInput
}

//SetupRestAPIInput represent setup API response
type SetupRestAPIOutput struct {
	*apigateway.RestApi
	Resources   []*SetupResourceOutput
	Stage       *apigateway.Stage
	EndpointURL string
	Region      string
}

//SetupResourceInput represents resource input
type SetupResourceInput struct {
	Path string
	apigateway.CreateResourceInput
	Methods []*ResourceMethod
}

//ResourceMethod represents resource method
type ResourceMethod struct {
	FunctionName string
	HttpMethod   string
	*apigateway.PutMethodInput
	*apigateway.PutIntegrationInput
	*lambda.AddPermissionInput
}

//SetupResourceOutput represents setup resource output
type SetupResourceOutput struct {
	*apigateway.Resource
	ResourceMethods map[string]*apigateway.Method
}

//RemoveRestAPI removes API for supplied name
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
	for _, resource := range i.Resources {
		if err := resource.Init(); err != nil {
			return err
		}
	}
	return nil
}

//Validate checks is input is valud
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

//Init initialize input
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
	if len(i.Methods) == 0 {
		return fmt.Errorf("methods were empty")
	}
	if len(i.Methods) == 0 {
		for _, method := range i.Methods {
			if err := method.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

//ParentPath returns parent path
func (i *SetupResourceInput) ParentPath() string {
	if i.PathPart == nil {
		return i.Path
	}
	index := strings.LastIndex(i.Path, *i.PathPart)
	if index == -1 {
		return i.Path
	}
	return string(i.Path[:index])
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
	-	$restAPI.ID
	-	$uuid.next

*/
func (i *ResourceMethod) Init() error {

	methodInput := i.PutMethodInput
	if methodInput == nil {
		i.PutMethodInput = &apigateway.PutMethodInput{}
		methodInput = i.PutMethodInput
	}
	if methodInput != nil {
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
	if integrationInput.Type == nil {
		integrationInput.Type = aws.String("AWS_PROXY")
	}

	if i.FunctionName != "" {
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
			permissionInput.SourceArn = aws.String("arn:aws:execute-api:${function.region}:${function.accountID}:${restAPI.ID}/*/*/*")
		}
	}
	return nil
}

func (i *ResourceMethod) Validate() error {
	if i.HttpMethod == "" {
		return fmt.Errorf("httpMethod was empty")
	}
	return nil
}

//SetupMethodInput setups method input
type SetupMethodInput apigateway.PutMethodInput

//Diff computes patches
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

//Diff computes patches
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
