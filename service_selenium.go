package endly

import (
	"errors"
	"fmt"
	"github.com/tebeka/selenium"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"time"
	"github.com/viant/toolbox/data"
)

//represents a SeleniumServiceID
const SeleniumServiceID = "selenium"
const SeleniumServer = "selenium-server-standalone"
const GeckoDriver = "geckodriver"






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

	switch actualRequest := request.(type) {

	case *SeleniumServerStartRequest:
		response.Response, err = s.start(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to start selenium %v", err)
		}

	case *SeleniumServerStopRequest:
		response.Response, err = s.stop(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to start selenium %v", err)
		}

	case *SeleniumOpenSessionRequest:
		response.Response, err = s.open(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to open selenium session %v", err)
		}
	case *SeleniumWebDriverCallRequest:
		response.Response, err = s.webDriverCall(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to call web driver %v", err)
		}
	case *SeleniumWebElementCallRequest:
		response.Response, err = s.webElementCall(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to call web element: %v", err)
		}
	case *SeleniumRunRequest:
		response.Response, err = s.run(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to call web element: %v", err)
		}
	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}

	if response.Error != "" {
		response.Status = "err"
	}
	return response


	}



func (s *seleniumService) run(context *Context, request *SeleniumRunRequest) (*SeleniumRunResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	var response = &SeleniumRunResponse{
		Data :make(map[string]*ElementResponse),
	}


	if request.SessionID == "" {
		openResponse, err:= s.openSession(context, &SeleniumOpenSessionRequest{
			RemoteSelenium:request.RemoteSelenium,
			Browser:request.Browser,
		})


		if err != nil {
			return nil, err
		}
		request.SessionID = openResponse.ID
	}
	response.SessionID = request.SessionID
	if request.PageURL != "" {
		seleniumSession, err := s.session(context, request.SessionID)
		if err != nil {
			return nil, err
		}
		err = seleniumSession.driver.Get(request.PageURL)
		if err != nil {
			return nil, err
		}
	}

	if len(request.Actions) == 0 {
		return response, nil
	}
	for _, action := range request.Actions {
		for _, call := range action.Calls {
			callResponse, err := s.webElementCall(context, &SeleniumWebElementCallRequest{
				SessionID:request.SessionID,
				Selector: action.Selector,
				Call:call,
			})
			if err != nil {
				return nil,err
			}

			var responseData string
			var has = false
			for _, element := range callResponse.Result {
				if element == nil || ! toolbox.IsString(element) {
					continue
				}
				has  = true
				responseData = toolbox.AsString(element)
				break;
			}
			if ! has {
				continue
			}

			if _, has := response.Data[action.Selector.Value]; ! has {
				response.Data[action.Selector.Value] = &ElementResponse{
					Selector:action.Selector,
					Data:make(map[string]string),
				}
			}
			response.Data[action.Selector.Value].Data[call.Method] = responseData

		}
	}
	return response, nil
}

func (s *seleniumService) webDriverCall(context *Context, request *SeleniumWebDriverCallRequest) (*SeleniumServiceCallResponse, error) {
	seleniumSession, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	return s.callMethod(seleniumSession.driver, request.Call.Method, request.Call.Parameters)
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

func (s *seleniumService) webElementCall(context *Context, request *SeleniumWebElementCallRequest) (*SeleniumWebElementCallResponse, error) {
	seleniumSession, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	var response = &SeleniumWebElementCallResponse{}
	err = request.Selector.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid selector: %v",err)
	}
	var selector = request.Selector

	element, err := seleniumSession.driver.FindElement(selector.By, selector.Value)
	if err != nil || element == nil {
		return nil, fmt.Errorf("failed to lookup element: %v %v", selector.By, selector.Value)
	}

	var callResponse *SeleniumServiceCallResponse
	repeat, sleepMs, exitCriteria := request.Data()
	for i := 0; i < repeat; i++ {
		callResponse, err = s.callMethod(element, request.Call.Method, request.Call.Parameters)
		if err != nil {
			return nil, err
		}
		if sleepMs > 0 {
			time.Sleep(sleepMs)
		}
		if exitCriteria != "" {
			var criteriaState = data.NewMap()
			if len(callResponse.Result) > 0 {
				criteriaState.Put("value", callResponse.Result[0])
			}
			criteria := criteriaState.ExpandAsText(exitCriteria)
			ok, err := EvaluateCriteria(context, criteria, "SeleniumWaitCriteria", true)
			if err != nil {
				return nil, err
			}
			if ok {
				break
			}
		}
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
		Input:  fmt.Sprintf("selenium-server-standalone.jar -port %d", toolbox.AsString(request.Port)),
	})
	if serviceResponse.Error != "" {
		return nil, errors.New(serviceResponse.Error)
	}
	return &SeleniumServerStopResponse{

	}, nil
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
		Target:target,
		Port:request.Port,
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
	return nil, fmt.Errorf("failed to lookup seleniun session id: %v, make sure you first run SeleniumOpenSessionRequest\n", sessionID)
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
	case "start":
		return &SeleniumServerStartRequest{}, nil
	case "stop":
		return &SeleniumServerStopRequest{}, nil
	case "open":
		return &SeleniumOpenSessionRequest{}, nil
	case "call-driver":
		return &SeleniumWebDriverCallRequest{}, nil
	case "call-element":
		return &SeleniumWebElementCallRequest{}, nil
	case "run":
		return SeleniumRunRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewScriptService creates a new selenium service
func NewSeleniumService() Service {
	var result = &seleniumService{
		AbstractService: NewAbstractService(SeleniumServiceID),
	}
	result.AbstractService.Service = result
	return result
}

var seleniumSessionsKey = (*SeleniumSessions)(nil)
