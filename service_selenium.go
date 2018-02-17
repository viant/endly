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

	//SeleniumServer represents name of selenium server
	SeleniumServer = "selenium-server-standalone"
	//GeckoDriver represents name of gecko driver
	GeckoDriver = "geckodriver"

	seleniumRunnerCaller = "seleniumRunnerCaller"
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
				callResponse, err := s.callWebDriver(context, &SeleniumWebDriverCallRequest{
					SessionID: request.SessionID,
					Call:      call,
				})
				if err != nil {
					return nil, err
				}
				s.addResultIfPresent(callResponse.Result, result, call.Method)
				continue

			}

			callResponse, err := s.callWebElement(context, &SeleniumWebElementCallRequest{
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

func (s *seleniumService) callWebDriver(context *Context, request *SeleniumWebDriverCallRequest) (*SeleniumServiceCallResponse, error) {
	seleniumSession, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	return s.call(context, seleniumSession.driver, request.Call)
}

func (s *seleniumService) call(context *Context, caller interface{}, call *SeleniumMethodCall) (callResponse *SeleniumServiceCallResponse, err error) {
	callResponse = &SeleniumServiceCallResponse{
		Extracted: make(map[string]string),
	}
	repeatable := call.Wait.Get()
	var handler = func() (interface{}, error) {
		callResponse, err = s.callMethod(caller, call.Method, call.Parameters)
		if err != nil {
			return nil, err
		}
		return callResponse.Result, nil
	}
	err = repeatable.Run(s.AbstractService, seleniumRunnerCaller, context, handler, callResponse.Extracted)
	return callResponse, err
}

func (s *seleniumService) callWebElement(context *Context, request *SeleniumWebElementCallRequest) (*SeleniumWebElementCallResponse, error) {
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
	response := sdkService.Run(context, &SdkSetRequest{
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

const (
	seleniumServiceStartExample = `{
		"Target": {
			"URL": "127.0.0.1",
			"Credential": "${env.HOME}/.secret/localhost.json"
		},
		"Port": 8085,
		"Sdk": "jdk",
		"SdkVersion": "1.8",
		"Version": "3.4"
	}`

	seleniumServiceStopExample = `{
		"Target": {
			"URL": "file://127.0.0.1",
			"Credential": "${env.HOME}/.secret/localhost.json"
		},
		"Port": 8085
	}`

	seleniumServiceOpenSessionExample = ` {
		"Browser": "firefox",
		"RemoteSelenium": {
			"URL": "http://127.0.0.1:8085/"
		}
	}`

	seleniumServiceCloseExample = `{
"SessionID": "127.0.0.1:8085"
}`

	seleniumServiceCallDriverExample = `
 {
     "SessionID": "127.0.0.1:8085",
      "Calls": [
        {
          "Wait": null,
          "Method": "Get",
          "Parameters": [
            "http://127.0.0.1:8888/signin/"
          ]
        }
      ]
}
`

	seleniumServiceCallElementExample = ` {
 	"SessionID": "127.0.0.1:8085",
    "Selector": {
        "By": "xpath",
        "Value": "//SMALL[preceding-sibling::INPUT[@id='email']]",
        "Key": "email"
      },
      "Calls": [
        {
          "Method": "Text"
        }
      ]
    }`

	seleniumServiceRunAction = `{
  "SessionID": "127.0.0.1:8085",
  "Actions": [
    {
      "Calls": [
        {
          "Wait": null,
          "Method": "Get",
          "Parameters": [
            "http://127.0.0.1:8888/signin/"
          ]
        }
      ]
    },
    {
      "Selector": {
        "Value": "#email"
      },
      "Calls": [
        {
          "Method": "Clear"
        },
        {
          "Method": "SendKeys",
          "Parameters": [
            "xyz@wp.w"
          ]
        }
      ]
    },
    {
      "Selector": {
        "Value": "#password"
      },
      "Calls": [
        {
          "Method": "Clear"
        },
        {
          "Method": "SendKeys",
          "Parameters": [
            "pass1"
          ]
        }
      ]
    },
    {
      "Selector": {
        "Value": "#submit"
      },
      "Calls": [
        {
          "Wait": {
            "SleepTimeMs": 100
          },
          "Method": "Click"
        }
      ]
    },
    {
      "Selector": {
        "By": "xpath",
        "Value": "//SMALL[preceding-sibling::INPUT[@id='email']]",
        "Key": "email"
      },
      "Calls": [
        {
          "Method": "Text"
        }
      ]
    }
  ]
}
`
)

func (s *seleniumService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "start",
		RequestInfo: &ActionInfo{
			Description: "start selenium server",
			Examples: []*ExampleUseCase{
				{
					UseCase: "start server",
					Data:    seleniumServiceStartExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SeleniumServerStartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SeleniumServerStartResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SeleniumServerStartRequest); ok {
				return s.start(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "stop",
		RequestInfo: &ActionInfo{
			Description: "stop selenium server",
			Examples: []*ExampleUseCase{
				{
					UseCase: "stop server",
					Data:    seleniumServiceStopExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SeleniumServerStopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SeleniumServerStopResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SeleniumServerStopRequest); ok {
				return s.stop(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "open",
		RequestInfo: &ActionInfo{
			Description: "open selenium session",
			Examples: []*ExampleUseCase{
				{
					UseCase: "open session",
					Data:    seleniumServiceOpenSessionExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SeleniumOpenSessionRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SeleniumOpenSessionResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SeleniumOpenSessionRequest); ok {
				return s.open(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "close",
		RequestInfo: &ActionInfo{
			Description: "close selenium session",
			Examples: []*ExampleUseCase{
				{
					UseCase: "close session",
					Data:    seleniumServiceCloseExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SeleniumCloseSessionRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SeleniumCloseSessionResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SeleniumCloseSessionRequest); ok {
				return s.close(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "run",
		RequestInfo: &ActionInfo{
			Description: "run selenium requests",
			Examples: []*ExampleUseCase{
				{
					UseCase: "run",
					Data:    seleniumServiceRunAction,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SeleniumRunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SeleniumRunResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SeleniumRunRequest); ok {
				return s.run(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "call-driver",
		RequestInfo: &ActionInfo{
			Description: "call proxies request to  github.com/tebeka/selenium web driver",
			Examples: []*ExampleUseCase{
				{
					UseCase: "call driver",
					Data:    seleniumServiceCallDriverExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SeleniumWebDriverCallRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SeleniumServiceCallResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SeleniumWebDriverCallRequest); ok {
				return s.callWebDriver(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "call-element",
		RequestInfo: &ActionInfo{
			Description: "find web element and proxy request",
			Examples: []*ExampleUseCase{
				{
					UseCase: "web element call",
					Data:    seleniumServiceCallElementExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SeleniumWebElementCallRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SeleniumServiceCallResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SeleniumWebElementCallRequest); ok {
				return s.callWebElement(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//NewSeleniumService creates a new selenium service
func NewSeleniumService() Service {
	var result = &seleniumService{
		AbstractService: NewAbstractService(SeleniumServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}

var seleniumSessionsKey = (*SeleniumSessions)(nil)
