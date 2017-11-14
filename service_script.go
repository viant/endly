package endly

import (
	"fmt"
	"github.com/robertkrimen/otto"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"io/ioutil"
)

const jsStdCode = "function getOrFail(s,n){if(n||(n=0),s.length>1&&s[s.length-1])throw s[s.length-1];return s[n]}\n"

//ScriptServiceID represents java script service id
const ScriptServiceID = "script"

//ScriptCommand represnets a script command
type ScriptCommand struct {
	Libraries []*url.Resource
	Code      string
}

type scriptService struct {
	*AbstractService
}

func (s *scriptService) loadLibraries(context *Context, request *ScriptCommand) (string, error) {
	if len(request.Libraries) == 0 {
		return "", nil
	}
	result := ""
	for _, resource := range request.Libraries {
		resource, err := context.ExpandResource(resource)
		if err != nil {
			return "", err
		}
		service, err := storage.NewServiceForURL(resource.URL, resource.Credential)
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
	vm.Set("context", &JsContextBridge{context})
	vm.Set("DeploymentServiceID", DeploymentServiceID)
	vm.Set("TransferServiceID", TransferServiceID)
	vm.Set("ExecServiceID", ExecServiceID)
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
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
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

//NewRequest creates a new request for an action (run).
func (s *scriptService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "run":
		return &ScriptCommand{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewScriptService creates a new script service
func NewScriptService() Service {
	var result = &scriptService{
		AbstractService: NewAbstractService(ScriptServiceID),
	}
	result.AbstractService.Service = result
	return result
}

//JsContextBridge represent java script context bridge.
type JsContextBridge struct {
	*Context
}

//Execute executes command
func (c *JsContextBridge) Execute(targetMap map[string]interface{}, commandMap map[string]interface{}) (*CommandResponse, error) {
	var target = &url.Resource{}
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
