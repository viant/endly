package endly

import (
	"errors"
	"fmt"
	"github.com/tebeka/selenium"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
)

const (
	//SeleniumServiceID represents a SeleniumServiceID
	SeleniumServiceID = "selenium"

	//SeleniumServiceStartAction represents selenium server start action
	SeleniumServiceStartAction = "start"

	//SeleniumServiceStopAction represents selenium server stop action
	SeleniumServiceStopAction = "stop"

	//SeleniumServiceOpenAction represents selenium browser open session action
	SeleniumServiceOpenAction = "open"

	//SeleniumServiceCloseAction represents selenium close session action
	SeleniumServiceCloseAction = "close"

	//SeleniumServiceCallDriverAction represents web driver call action
	SeleniumServiceCallDriverAction = "call-driver"

	//SeleniumServiceCallElementAction represents web element call action
	SeleniumServiceCallElementAction = "call-element"

	//SeleniumServiceRunAction represents group of calls action.
	SeleniumServiceRunAction = "run"

	//SeleniumServer represents name of selenium server
	SeleniumServer = "selenium-server-standalone"
	//GeckoDriver represents name of gecko driver
	GeckoDriver = "geckodriver"
)

//SeleniumSession represents a selenium session
type SeleniumSession struct {
	ID      string
	Browser string
	driver  selenium.WebDriver
}

//SeleniumSessions reprents selenium sessions.
type SeleniumSessions map[string]*SeleniumSession
type seleniumService struct {
	*AbstractService
}

func (s *seleniumService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	var errorTemplate string

	switch actualRequest := request.(type) {

	case *SeleniumServerStartRequest:
		response.Response, err = s.start(context, actualRequest)
		errorTemplate = "failed to start selenium %v"
	case *SeleniumServerStopRequest:
		response.Response, err = s.stop(context, actualRequest)
		errorTemplate = "failed to start selenium %v"

	case *SeleniumOpenSessionRequest:
		response.Response, err = s.open(context, actualRequest)
		errorTemplate = "failed to open selenium session %v"
	case *SeleniumCloseSessionRequest:
		response.Response, err = s.close(context, actualRequest)
		errorTemplate = "failed to open selenium session %v"

	case *SeleniumWebDriverCallRequest:
		response.Response, err = s.webDriverCall(context, actualRequest)
		errorTemplate = "failed to call web driver %v"
	case *SeleniumWebElementCallRequest:
		response.Response, err = s.webElementCall(context, actualRequest)
		errorTemplate = "failed to call web element: %v"
	case *SeleniumRunRequest:
		response.Response, err = s.run(context, actualRequest)
		errorTemplate = "failed to call web element: %v"
	default:
		errorTemplate = "%v"
		err = fmt.Errorf("unsupported request type: %T", request)
	}
	if err != nil {
		response.Status = "err"
		response.Error = fmt.Sprintf(errorTemplate, err)
	}
	return response

}

func (s *seleniumService) addResultIfPresent(callResult []interface{}, result data.Map, resultPath ...string) {
	var responseData string
	var has = false
	for _, element := range callResult {
		if element == nil || !toolbox.IsString(element) {
			continue
		}
		has = true
		responseData = toolbox.AsString(element)
		break
	}
	if !has {
		return
	}
	var key = strings.Join(resultPath, ".")
	result.SetValue(key, responseData)
}

func (s *seleniumService) run(context *Context, request *SeleniumRunRequest) (*SeleniumRunResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var response = &SeleniumRunResponse{
		Data:         make(map[string]interface{}),
		LookupErrors: make([]string, 0),
	}
	var result = data.Map(response.Data)

	if request.SessionID == "" {
		openResponse, err := s.openSession(context, &SeleniumOpenSessionRequest{
			RemoteSelenium: request.RemoteSelenium,
			Browser:        request.Browser,
		})

		if err != nil {
			return nil, err
		}
		request.SessionID = openResponse.ID
	}
	response.SessionID = request.SessionID
	if len(request.Actions) == 0 {
		return response, nil
	}
	for _, action := range request.Actions {
		for _, call := range action.Calls {
			if action.Selector == nil {
				callResponse, err := s.webDriverCall(context, &SeleniumWebDriverCallRequest{
					SessionID: request.SessionID,
					Call:      call,
				})
				if err != nil {
					return nil, err
				}
				s.addResultIfPresent(callResponse.Result, result, call.Method)
				continue

			}

			callResponse, err := s.webElementCall(context, &SeleniumWebElementCallRequest{
				SessionID: request.SessionID,
				Selector:  action.Selector,
				Call:      call,
			})
			if err != nil {
				return nil, err
			}
			if callResponse.LookupError != "" {
				response.LookupErrors = append(response.LookupErrors, callResponse.LookupError)
			}
			var elementKey = action.Selector.Key
			if elementKey == "" {
				elementKey = action.Selector.Value
			}
			var elementPath = []string{elementKey, call.Method}
			if len(call.Parameters) == 1 && toolbox.IsString(call.Parameters[0]) {
				elementPath[1] = strings.Replace(elementPath[1], "Get", "", 1)
				elementPath[1] = strings.Replace(elementPath[1], "Property", "", 1)
				elementPath = append(elementPath)
			}
			s.addResultIfPresent(callResponse.Result, result, elementPath...)
		}
	}
	return response, nil
}

func (s *seleniumService) callMethod(owner interface{}, methodName string, parameters []interface{}) (*SeleniumServiceCallResponse, error) {
	method, err := toolbox.GetFunction(owner, methodName)
	if err != nil {
		return nil, err
	}
	parameters, err = toolbox.AsCompatibleFunctionParameters(method, parameters)
	if err != nil {
		return nil, err
	}
	var response = &SeleniumServiceCallResponse{}
	response.Result = toolbox.CallFunction(method, parameters...)
	return response, nil
}

func (s *seleniumService) webDriverCall(context *Context, request *SeleniumWebDriverCallRequest) (*SeleniumServiceCallResponse, error) {
	seleniumSession, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	return s.call(context, seleniumSession.driver, request.Call)
}

func (s *seleniumService) call(context *Context, caller interface{}, call *SeleniumMethodCall) (callResponse *SeleniumServiceCallResponse, err error) {
	repeat, sleepInMs, exitCriteria := call.Wait.Data()
	for i := 0; i < repeat; i++ {
		callResponse, err = s.callMethod(caller, call.Method, call.Parameters)
		if err != nil && exitCriteria == "" {
			//if there is exit criteria error can be intermittent.
			return nil, err
		}
		if sleepInMs > 0 {
			s.Sleep(context, sleepInMs)
		}
		if exitCriteria != "" {
			var criteria = exitCriteria
			if len(callResponse.Result) > 0 {
				criteria = strings.Replace(exitCriteria, "$value", toolbox.AsString(callResponse.Result[0]), 1)
			}
			ok, err := EvaluateCriteria(context, criteria, "SeleniumWaitCriteria", true)
			if err != nil {
				return nil, err
			}
			if ok {
				break
			}
		}
	}
	return callResponse, err
}

func (s *seleniumService) webElementCall(context *Context, request *SeleniumWebElementCallRequest) (*SeleniumWebElementCallResponse, error) {
	seleniumSession, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	var response = &SeleniumWebElementCallResponse{}
	err = request.Selector.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid selector: %v", err)
	}
	var selector = request.Selector

	element, err := seleniumSession.driver.FindElement(selector.By, selector.Value)
	if err != nil || element == nil {
		response.LookupError = fmt.Sprintf("failed to lookup element: %v %v", selector.By, selector.Value)
		return response, nil
	}
	callResponse, err := s.call(context, element, request.Call)
	if err != nil {
		return nil, err
	}
	response.Result = callResponse.Result
	return response, nil
}

func (s *seleniumService) open(context *Context, request *SeleniumOpenSessionRequest) (*SeleniumOpenSessionResponse, error) {
	var response = &SeleniumOpenSessionResponse{}
	seleniumSession, err := s.openSession(context, request)
	if err != nil {
		return nil, err
	}
	response.SessionID = seleniumSession.ID
	return response, nil
}

func (s *seleniumService) close(context *Context, request *SeleniumCloseSessionRequest) (*SeleniumCloseSessionResponse, error) {
	var response = &SeleniumCloseSessionResponse{
		SessionID: request.SessionID,
	}
	seleniumSession, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	err = seleniumSession.driver.Close()
	return response, err
}

func (s *seleniumService) deployServerIfNeeded(context *Context, request *SeleniumServerStartRequest, target *url.Resource) (*SeleniumServerStartResponse, error) {
	deploymentService, _ := context.Service(DeploymentServiceID)

	deployServerResponse := deploymentService.Run(context, &DeploymentDeployRequest{
		Target:  target,
		AppName: SeleniumServer,
		Version: request.Version,
	})
	if deployServerResponse.Error != "" {
		return nil, errors.New(deployServerResponse.Error)
	}
	deployGeckoDriverResponse := deploymentService.Run(context, &DeploymentDeployRequest{
		Target:  target,
		AppName: GeckoDriver,
	})
	if deployGeckoDriverResponse.Error != "" {
		return nil, errors.New(deployGeckoDriverResponse.Error)
	}
	var response = &SeleniumServerStartResponse{}
	response.GeckodriverPath = "/opt/selenium/geckodriver"
	response.SeleniumServerPath = "/opt/selenium/selenium-server-standalone.jar"
	return response, nil
}

func (s *seleniumService) setJdk(context *Context, request *SeleniumServerStartRequest) error {
	sdkService, _ := context.Service(SdkServiceID)
	response := sdkService.Run(context, &SystemSdkSetRequest{
		Sdk:     request.Sdk,
		Version: request.SdkVersion,
		Target:  request.Target,
	})

	if response.Error != "" {
		return errors.New(response.Error)
	}
	return nil
}

func (s *seleniumService) stop(context *Context, request *SeleniumServerStopRequest) (*SeleniumServerStopResponse, error) {
	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	processService, _ := context.Service(ProcessServiceID)
	serviceResponse := processService.Run(context, &ProcessStopAllRequest{
		Target: target,
		Input:  fmt.Sprintf("selenium-server-standalone.jar -port %v", toolbox.AsString(request.Port)),
	})
	if serviceResponse.Error != "" {
		return nil, errors.New(serviceResponse.Error)
	}
	return &SeleniumServerStopResponse{}, nil
}

func (s *seleniumService) start(context *Context, request *SeleniumServerStartRequest) (*SeleniumServerStartResponse, error) {
	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	response, err := s.deployServerIfNeeded(context, request, target)
	if err != nil {
		return nil, err
	}
	err = s.setJdk(context, request)
	if err != nil {
		return nil, err
	}

	s.Run(context, &SeleniumServerStopRequest{
		Target: target,
		Port:   request.Port,
	})
	processService, _ := context.Service(ProcessServiceID)
	serviceResponse := processService.Run(context, &ProcessStartRequest{
		Command:         "java",
		Target:          target,
		Directory:       "/opt/selenium",
		Arguments:       []string{"-jar", fmt.Sprintf("-Dwebdriver.gecko.driver=%v", response.GeckodriverPath), "-jar", response.SeleniumServerPath, "-port", toolbox.AsString(request.Port)},
		ImmuneToHangups: true,
	})
	if serviceResponse.Error != "" {
		return nil, errors.New(serviceResponse.Error)
	}
	if processResponse, ok := serviceResponse.Response.(*ProcessStartResponse); ok && len(processResponse.Info) > 0 {
		response.Pid = processResponse.Info[0].Pid
	}
	return response, nil
}

func (s *seleniumService) session(context *Context, sessionID string) (*SeleniumSession, error) {
	sessions := context.SeleniumSessions()
	if seleniumSession, ok := sessions[sessionID]; ok {
		return seleniumSession, nil
	}
	return nil, fmt.Errorf("failed to lookup seleniun session id: %v, make sure you first run SeleniumOpenSessionRequest", sessionID)
}

func (s *seleniumService) openSession(context *Context, request *SeleniumOpenSessionRequest) (*SeleniumSession, error) {
	resource, err := context.ExpandResource(request.RemoteSelenium)
	if err != nil {
		return nil, err
	}
	err = request.Validate()
	if err != nil {
		return nil, err
	}
	sessionID := resource.Host()
	sessions := context.SeleniumSessions()
	seleniumSession, ok := sessions[sessionID]
	if ok {
		if seleniumSession.Browser == request.Browser {
			return seleniumSession, nil
		}
		seleniumSession.driver.Close()
	} else {
		seleniumSession = &SeleniumSession{
			ID:      sessionID,
			Browser: request.Browser,
		}
	}
	caps := selenium.Capabilities{"browserName": request.Browser}
	seleniumEndpoint := fmt.Sprintf("http://%v/wd/hub", resource.ParsedURL.Host)
	seleniumSession.driver, err = selenium.NewRemote(caps, seleniumEndpoint)

	if err != nil {
		return nil, err
	}
	sessions[sessionID] = seleniumSession
	context.Deffer(func() {
		seleniumSession.driver.Quit()
	})
	return seleniumSession, nil
}

//NewRequest creates a new request for the provided action (run).
func (s *seleniumService) NewRequest(action string) (interface{}, error) {
	switch action {
	case SeleniumServiceStartAction:
		return &SeleniumServerStartRequest{}, nil
	case SeleniumServiceStopAction:
		return &SeleniumServerStopRequest{}, nil
	case SeleniumServiceOpenAction:
		return &SeleniumOpenSessionRequest{}, nil
	case SeleniumServiceCloseAction:
		return &SeleniumCloseSessionRequest{}, nil
	case SeleniumServiceCallDriverAction:
		return &SeleniumWebDriverCallRequest{}, nil
	case SeleniumServiceCallElementAction:
		return &SeleniumWebElementCallRequest{}, nil
	case SeleniumServiceRunAction:
		return &SeleniumRunRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewSeleniumService creates a new selenium service
func NewSeleniumService() Service {
	var result = &seleniumService{
		AbstractService: NewAbstractService(SeleniumServiceID,
			SeleniumServiceStartAction,
			SeleniumServiceStopAction,
			SeleniumServiceOpenAction,
			SeleniumServiceCloseAction,
			SeleniumServiceCallDriverAction,
			SeleniumServiceCallElementAction,
			SeleniumServiceRunAction,
		),
	}
	result.AbstractService.Service = result
	return result
}

var seleniumSessionsKey = (*SeleniumSessions)(nil)
