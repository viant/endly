package endly

import (
	"fmt"
	"github.com/robertkrimen/otto"
	"github.com/viant/toolbox/storage"
	"io/ioutil"
)

const jsStdCode = "function getOrFail(s,n){if(n||(n=0),s.length>1&&s[s.length-1])throw s[s.length-1];return s[n]}\n"

const ScriptServiceId = "script"

type ScriptCommand struct {
	Libraries []*Resource
	Code      string
}

type scriptService struct {
	*AbstractService
}

func (t *scriptService) loadLibraries(context *Context, request *ScriptCommand) (string, error) {
	if len(request.Libraries) == 0 {
		return "", nil
	}
	result := ""
	for _, resource := range request.Libraries {
		resource, err := context.ExpandResource(resource)
		if err != nil {
			return "", err
		}
		service, err := storage.NewServiceForURL(resource.URL, resource.CredentialFile)
		if err != nil {
			return "", err
		}
		objects, err := service.List(resource.URL)
		if err != nil {
			return "", err
		}
		if len(objects) == 0 {
			return "", fmt.Errorf("Failed to locate: %v", resource.URL)
		}
		reader, err := service.Download(objects[0])
		if err != nil {
			return "", err
		}
		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return "", err
		}
		result += string(content) + "\n"
	}
	return result, nil
}

func (s *scriptService) runScriptCommand(context *Context, request *ScriptCommand) (interface{}, error) {
	vm := otto.New()
	vm.Set("context", &JsContext{context})
	vm.Set("DeploymentServiceId", DeploymentServiceId)
	vm.Set("TransferServiceId", TransferServiceId)
	vm.Set("ExecServiceId", ExecServiceId)
	vm.Set("ExtractColumns", ExtractColumns)
	libraries, err := s.loadLibraries(context, request)
	if err != nil {
		return nil, err
	}
	var code = jsStdCode + libraries + request.Code
	result, err := vm.Run(code)
	if err != nil {
		return nil, err
	}
	return result.String(), nil
}

func (s *scriptService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
	var err error
	switch actualRequest := request.(type) {
	case *ScriptCommand:
		response.Response, err = s.runScriptCommand(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run script: %v, %v", actualRequest.Code, err)
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *scriptService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "run":
		return &ScriptCommand{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewScriptService() Service {
	var result = &scriptService{
		AbstractService: NewAbstractService(ScriptServiceId),
	}
	result.AbstractService.Service = result
	return result
}

type JsContext struct {
	*Context
}

func (c *JsContext) Execute(targetMap map[string]interface{}, commandMap map[string]interface{}) (*CommandInfo, error) {

	var target = &Resource{}
	err := converter.AssignConverted(target, targetMap)
	if err != nil {
		return nil, err
	}
	var command = &ManagedCommand{}
	err = converter.AssignConverted(command, commandMap)
	if err != nil {
		return nil, err
	}
	return c.Context.Execute(target, command)
}
