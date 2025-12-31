package webdriver

import (
	"errors"
	"fmt"
	"github.com/viant/endly/model/msg"
	"strings"
	"time"

	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"github.com/tebeka/selenium/firefox"
	selog "github.com/tebeka/selenium/log"
	"github.com/viant/afs"
	"github.com/viant/afs/url"
	"github.com/viant/endly"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model/criteria"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/deployment/deploy"
	"github.com/viant/endly/service/deployment/sdk"
	"github.com/viant/endly/service/system/exec"
	"github.com/viant/endly/service/system/process"
	"github.com/viant/endly/service/testing/runner/webdriver/extension/html/table"
	"github.com/viant/endly/service/testing/validator"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

const (
	//ServiceID represents a ServiceID
	ServiceID = "webdriver"

	//SeleniumServer represents name of selenium server
	SeleniumServer = "selenium-server-standalone"
	//GeckoDriver represents name of gecko driver
	GeckoDriver    = "geckodriver"
	ChromeDriver   = "chromedriver"
	ChromeBrowser  = "chrome"
	FirefoxBrowser = "firefox"
	Selenium       = "webdriver"
	runnerCaller   = "runnerCaller"

	defaultFindElementTimeout = 10 * time.Second
)

type service struct {
	*endly.AbstractService
	fs afs.Service
}

func (s *service) addResultIfPresent(callResult []interface{}, result data.Map, resultPath ...string) bool {
	var responseData interface{}
	var has = false
	for _, element := range callResult {
		if element == nil {
			continue
		}
		switch actual := element.(type) {
		case string:
			responseData = actual
		case []byte:
			responseData = string(actual)
		case []interface{}:
			responseData = actual
		case []map[string]interface{}:
			responseData = actual
		case map[string]interface{}:
			responseData = actual
		default:
			fmt.Printf("unsupported type: %T\n", actual)
			continue
		}
		has = true
		break
	}
	if !has {
		return false
	}
	var key = strings.Join(resultPath, ".")
	result.SetValue(key, responseData)

	return true
}

func (s *service) getResultPath(key string, call *MethodCall, kind PathKind) []string {
	if kind == PathKindSimple {
		return []string{key}
	}
	var method = call.Method
	if len(call.Parameters) == 1 && toolbox.IsString(call.Parameters[0]) {
		method = strings.Replace(method, "Get", "", 1) + "." + toolbox.AsString(call.Parameters[0])
	}
	return []string{key, method}
}

func (s *service) run(context *endly.Context, request *RunRequest) (*RunResponse, error) {
	var response = &RunResponse{
		Data:         make(map[string]interface{}),
		LookupErrors: make([]string, 0),
	}
	navigation := navigationWithDefaults(request.Navigation)
	sessions := Sessions(context)
	session, hasSession := sessions[request.SessionID]

	if !hasSession || session.driver == nil {
		openResponse, err := s.openSession(context, &OpenSessionRequest{
			Remote:    request.RemoteSelenium,
			Browser:   request.Browser,
			SessionID: request.SessionID,
		})
		if err != nil {
			return nil, err
		}
		request.SessionID = openResponse.SessionID
		sessions = Sessions(context)
		session = sessions[request.SessionID]
	}
	response.SessionID = request.SessionID
	if len(request.Actions) == 0 {
		return response, nil
	}
	var state = context.State()

	actionDelay := time.Duration(request.ActionDelaysMs) * time.Millisecond
	for _, action := range request.Actions {
		for _, call := range action.Calls {
			if len(call.Parameters) > 0 {
				for i, item := range call.Parameters {
					call.Parameters[i] = state.Expand(item)
				}
			}
			if action.Selector == nil {
				if session != nil && isGetMethod(call.Method) && len(call.Parameters) == 1 && toolbox.IsString(call.Parameters[0]) {
					URL := toolbox.AsString(call.Parameters[0])
					if err := s.getWithGuard(context, session, URL, navigation); err != nil {
						return nil, err
					}
					if session.Capture != nil {
						session.Capture.Drain(session)
					}
					continue
				}
				callResponse, err := s.callWebDriver(context, &WebDriverCallRequest{
					Key:       action.Key,
					SessionID: request.SessionID,
					Call:      call,
					PathKind:  action.PathKind,
				})
				if err != nil {
					return nil, err
				}
				util.MergeMap(response.Data, callResponse.Data)
				if session != nil && session.Capture != nil {
					session.Capture.Drain(session)
				}
				continue
			}
			callResponse, err := s.callWebElement(context, &WebElementCallRequest{
				SessionID: request.SessionID,
				Selector:  action.Selector,
				Call:      call,
				PathKind:  action.PathKind,
			})
			if IsStaleElementError(err) {
				callResponse, err = s.callWebElement(context, &WebElementCallRequest{
					SessionID: request.SessionID,
					Selector:  action.Selector,
					Call:      call,
					PathKind:  action.PathKind,
				})
			}
			if err != nil {
				return nil, err
			}
			if callResponse.LookupError != "" {
				response.LookupErrors = append(response.LookupErrors, callResponse.LookupError)
			}
			util.MergeMap(response.Data, callResponse.Data)
			if session != nil && session.Capture != nil {
				session.Capture.Drain(session)
			}
			if actionDelay > 0 {
				time.Sleep(actionDelay)
			}
		}
	}

	var err error
	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, response.Data, "webdriver", "assert webdriver response")
	}
	if err == nil && len(response.LookupErrors) > 0 {
		err = fmt.Errorf("lookup errors: %v", strings.Join(response.LookupErrors, ","))

	}
	return response, err
}

// Data returns table" data in the specified format, format uses the following values: json, csv, objects, tabular, optionally you can specify header columns after ':'
func (s *service) Data(webElement selenium.WebElement, format string) (interface{}, error) {
	//TODO add support for form
	var header = ""
	if index := strings.Index(format, ":"); index != -1 {
		header = format[index+1:]
		format = format[:index]
	}
	if format == "" {
		format = "objects"
	}
	var headers []string
	if header != "" {
		headers = strings.Split(header, ",")
	}
	tagName, err := webElement.TagName()
	if err != nil {
		return nil, err
	}
	if tagName != "table" {
		return nil, fmt.Errorf("element is not a table")
	}
	tableHtml, err := webElement.GetAttribute("outerHTML")
	if err != nil {
		return nil, fmt.Errorf("failed to get table html: %v", err)
	}
	exporter, err := table.NewExporter(tableHtml)
	if err != nil {
		return nil, fmt.Errorf("failed to create table exporter: %v", err)
	}
	ret, err := exporter.Export(headers, format)
	if err != nil {
		return nil, fmt.Errorf("failed to export table: %v", err)
	}
	return ret, nil
}

func (s *service) callMethod(owner interface{}, methodName string, response *ServiceCallResponse, parameters []interface{}) (err error) {
	switch methodName {
	case "Data":
		parameters = append([]interface{}{owner}, parameters...)
		owner = s
	}
	method, err := toolbox.GetFunction(owner, methodName)
	if err != nil {
		return err
	}
	parameters, err = toolbox.AsCompatibleFunctionParameters(method, parameters)
	if err != nil {
		return err
	}
	response.Result = toolbox.CallFunction(method, parameters...)
	value := response.Result[len(response.Result)-1]
	if value != nil {
		if err, ok := value.(error); ok {
			return err
		}
	}
	return nil
}

func (s *service) callWebDriver(context *endly.Context, request *WebDriverCallRequest) (*ServiceCallResponse, error) {
	session, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	response := &ServiceCallResponse{
		Data: make(map[string]interface{}),
	}
	var key = request.Key
	if key == "" {
		key = request.Call.Method
	}
	return response, s.call(context, session.driver, session.driver, request.Call, response, key)
}

func (s *service) call(context *endly.Context, driver selenium.WebDriver, caller interface{}, call *MethodCall, response *ServiceCallResponse, elementPath ...string) (err error) {
	if call.WaitTimeMs == 0 {
		if err = s.callMethod(caller, call.Method, response, call.Parameters); err != nil {
			return err
		}
		s.addResultIfPresent(response.Result, response.Data, elementPath...)
		if call.ThinkTimeMs > 0 {
			time.Sleep(time.Millisecond * time.Duration(call.ThinkTimeMs))
		}
		return nil
	}

	err = driver.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		err = s.callMethod(caller, call.Method, response, call.Parameters)
		if err != nil {
			return false, err
		}
		s.addResultIfPresent(response.Result, response.Data, elementPath...)
		if call.Exit == "" {
			return true, nil
		}
		evalData := data.Map{}
		util.MergeMap(evalData, response.Data)
		return criteria.Evaluate(context, evalData, call.Exit, &call.criteria, runnerCaller, true)
	}, time.Duration(call.WaitTimeMs)*time.Millisecond)

	if IsStaleElementError(err) {
		return err
	}
	if call.IgnoreTimeout {
		return nil
	}
	return err
}

func (s *service) callWebElement(context *endly.Context, request *WebElementCallRequest) (*WebElementCallResponse, error) {
	session, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	var response = &WebElementCallResponse{
		Data: make(map[string]interface{}),
	}
	err = request.Selector.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid selector: %v", err)
	}
	var selector = request.Selector
	var element selenium.WebElement

	err = session.driver.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		element, err = session.driver.FindElement(selector.By, selector.Value)
		if element != nil {
			return true, nil
		}
		return false, nil
	}, defaultFindElementTimeout)

	if err != nil || element == nil {
		response.LookupError = fmt.Sprintf("failed to lookup element: %v %v, %v", selector.By, selector.Value, err)
		return response, nil
	}

	elementPath := s.getResultPath(request.Selector.Key, request.Call, request.PathKind)
	callResponse := &ServiceCallResponse{
		Data: make(map[string]interface{}),
	}
	switch request.Call.Method {
	case "Click", "SendKeys", "Clear", "Submit":
		if err = s.ensureVisible(element); err != nil {
			response.LookupError = fmt.Sprintf("element %s is not visible: %v", request.Selector.Value, err)
			return nil, err
		}
	}

	err = s.call(context, session.driver, element, request.Call, callResponse, elementPath...)
	if err != nil {
		return nil, err
	}
	util.Append(response.Data, callResponse.Data, true)
	response.Result = callResponse.Result
	return response, nil
}

func (s *service) ensureVisible(element selenium.WebElement) error {
	var err error
	var ok bool
	for i := 0; i < 10; i++ {
		if ok, err = element.IsDisplayed(); ok {
			break
		}
		if IsStaleElementError(err) {
			return err
		}
		time.Sleep(time.Millisecond * 200)
	}
	return err
}

func (s *service) open(context *endly.Context, request *OpenSessionRequest) (*OpenSessionResponse, error) {
	var response = &OpenSessionResponse{}
	seleniumSession, err := s.openSession(context, request)
	if err != nil {
		return nil, err
	}
	response.SessionID = seleniumSession.SessionID
	return response, nil
}

func (s *service) close(context *endly.Context, request *CloseSessionRequest) (*CloseSessionResponse, error) {
	var response = &CloseSessionResponse{
		SessionID: request.SessionID,
	}
	session, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	session.Close()
	return response, err
}

func (s *service) deployServerIfNeeded(context *endly.Context, request *StartRequest, target *location.Resource) (*StartResponse, error) {
	deploymentService, _ := context.Service(deploy.ServiceID)

	response := &StartResponse{}
	driver, version := pair(request.Driver)
	driverURL := url.Join(request.Target.URL, driver)
	ok, _ := s.fs.Exists(context.Background(), driverURL)

	if !ok {
		driverResponse := deploymentService.Run(context, &deploy.Request{
			Target:  target,
			Version: version,
			AppName: driver,
		})
		if driverResponse.Error != "" {
			return nil, errors.New(driverResponse.Error)
		}
	}
	response.DriverPath = url.Path(driverURL)

	if request.Server != "" { //to use with standalone selenium  server
		serverURL := url.Join(request.Target.URL, SeleniumServer)
		ok, _ := s.fs.Exists(context.Background(), serverURL)
		if !ok {
			_, version = pair(request.Server)
			driverResponse := deploymentService.Run(context, &deploy.Request{
				Target:  target,
				Version: version,
				AppName: SeleniumServer,
			})
			if driverResponse.Error != "" {
				return nil, errors.New(driverResponse.Error)
			}
		}
		response.ServerPath = url.Path(serverURL)
	}
	return response, nil
}

func (s *service) setJdk(context *endly.Context, request *StartRequest) error {
	if request.Sdk == "" {
		return nil
	}
	sdkService, _ := context.Service(sdk.ServiceID)
	_, version := pair(request.Sdk)
	response := sdkService.Run(context, &sdk.SetRequest{
		Sdk:     request.Sdk,
		Version: version,
		Target:  request.Target,
	})

	if response.Error != "" {
		return errors.New(response.Error)
	}
	return nil
}

func (s *service) stop(context *endly.Context, request *StopRequest) (*StopResponse, error) {
	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	session, _ := s.session(context, fmt.Sprintf("localhost:%v", request.Port))
	if session == nil {
		return &StopResponse{}, nil
	}
	processService, _ := context.Service(process.ServiceID)
	if session.Pid > 0 {
		serviceResponse := processService.Run(context, &process.StopRequest{
			Target: target,
			Input:  fmt.Sprintf("selenium-server-standalone.jar -port %v", toolbox.AsString(request.Port)),
		})
		if serviceResponse.Error != "" {
			return nil, errors.New(serviceResponse.Error)
		}
	}

	session.Close()

	return &StopResponse{}, nil
}

func (s *service) start(context *endly.Context, request *StartRequest) (*StartResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	response, err := s.deployServerIfNeeded(context, request, target)
	if err != nil {
		return nil, err
	}
	sessionID := fmt.Sprintf("localhost:%v", request.Port)
	sessions := Sessions(context)
	session, ok := sessions[sessionID]
	if ok {
		session.Close()
	} else {
		session = &Session{SessionID: sessionID}
		sessions[sessionID] = session
	}
	useSelenium := request.Server != ""
	if !useSelenium {
		session.Capabilities = request.Capabilities
		switch request.Driver {
		case ChromeDriver:
			if session.service, err = selenium.NewChromeDriverService(response.DriverPath, request.Port); err != nil {
				return nil, fmt.Errorf("failed to start chromedriver service %w", err)
			}
			session.Browser = ChromeBrowser
		case GeckoDriver:
			if session.service, err = selenium.NewGeckoDriverService(response.DriverPath, request.Port); err != nil {
				return nil, fmt.Errorf("failed to start geckodriver service %w", err)
			}
			session.Browser = FirefoxBrowser
		default:
			if request.Server == "" {
				return nil, fmt.Errorf("invalid driver %v", request.Driver)
			}
		}
	}
	session.SessionID = sessionID
	if request.Server == "" {
		return response, nil
	}
	session.Server = request.Server
	err = s.setJdk(context, request)
	if err != nil {
		return nil, err
	}

	s.Run(context, &StopRequest{
		Target: target,
		Port:   request.Port,
	})
	processService, _ := context.Service(process.ServiceID)
	session.Browser = FirefoxBrowser
	serviceResponse := processService.Run(context, &process.StartRequest{
		Command: "java",
		Target:  target,
		Options: &exec.Options{
			Directory:  defaultTarget,
			CheckError: true,
		},
		Arguments:       []string{fmt.Sprintf("-Dwebdriver.gecko.driver=%v", response.DriverPath), "-jar", response.ServerPath, "-port", toolbox.AsString(request.Port)},
		ImmuneToHangups: true,
	})
	if serviceResponse.Error != "" {
		return nil, errors.New(serviceResponse.Error)
	}

	if processResponse, ok := serviceResponse.Response.(*process.StartResponse); ok && len(processResponse.Info) > 0 {
		response.Pid = processResponse.Info[0].Pid
		session.Pid = response.Pid
	}
	return response, nil
}

func (s *service) session(context *endly.Context, sessionID string) (*Session, error) {
	sessions := Sessions(context)
	if seleniumSession, ok := sessions[sessionID]; ok {
		return seleniumSession, nil
	}
	return nil, fmt.Errorf("failed to lookup seleniun session id: %v, make sure you first run SeleniumOpenSessionRequest", sessionID)
}

func (s *service) openSession(context *endly.Context, request *OpenSessionRequest) (*Session, error) {
	ensureOpenSessionDefaults(request)
	sessionID := request.SessionID
	sessions := Sessions(context)
	session, ok := sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("webdriver service not running - start ?")
	}
	if session.driver != nil {
		_ = session.driver.Close()
	}

	caps := selenium.Capabilities{}
	if session.Pid == 0 {
		if len(session.Capabilities) > 0 && len(request.Capabilities) == 0 {
			request.Capabilities = session.Capabilities
		}
		switch session.Browser {
		case ChromeBrowser:
			caps.AddChrome(chrome.Capabilities{Args: request.Capabilities})
			caps.SetLogLevel(selog.Performance, selog.All)
			caps.SetLogLevel(selog.Browser, selog.All)
		case FirefoxBrowser:
			caps.AddFirefox(firefox.Capabilities{Args: request.Capabilities})
			caps.SetLogLevel(selog.Browser, selog.All)
		}
	} else {
		caps["browserName"] = request.Browser
	}

	var err error
	session.driver, err = selenium.NewRemote(caps, request.Remote)
	if err != nil {
		return nil, err
	}
	session.Remote = request.Remote
	sessions[sessionID] = session
	context.Deffer(func() {
		session.driver.Quit()
	})
	return session, nil
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "start",
		RequestInfo: &endly.ActionInfo{
			Description: "start selenium server",
		},
		RequestProvider: func() interface{} {
			return &StartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StartResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StartRequest); ok {
				return s.start(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "stop",
		RequestInfo: &endly.ActionInfo{
			Description: "stop selenium server",
		},
		RequestProvider: func() interface{} {
			return &StopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StopResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StopRequest); ok {
				return s.stop(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "open",
		RequestInfo: &endly.ActionInfo{
			Description: "open selenium session",
		},
		RequestProvider: func() interface{} {
			return &OpenSessionRequest{}
		},
		ResponseProvider: func() interface{} {
			return &OpenSessionResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*OpenSessionRequest); ok {
				return s.open(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "close",
		RequestInfo: &endly.ActionInfo{
			Description: "close selenium session",
		},
		RequestProvider: func() interface{} {
			return &CloseSessionRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CloseSessionResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CloseSessionRequest); ok {
				return s.close(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "run",
		RequestInfo: &endly.ActionInfo{
			Description: "run selenium requests",
		},
		RequestProvider: func() interface{} {
			return &RunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RunRequest); ok {
				return s.run(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "call-driver",
		RequestInfo: &endly.ActionInfo{
			Description: "call proxies request to  github.com/tebeka/selenium web driver",
		},
		RequestProvider: func() interface{} {
			return &WebDriverCallRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ServiceCallResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*WebDriverCallRequest); ok {
				return s.callWebDriver(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "call-element",
		RequestInfo: &endly.ActionInfo{
			Description: "find web element and proxy request",
		},
		RequestProvider: func() interface{} {
			return &WebElementCallRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ServiceCallResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*WebElementCallRequest); ok {
				return s.callWebElement(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-start",
		RequestInfo: &endly.ActionInfo{
			Description: "start capturing console and network (Chrome/Edge CDP)",
		},
		RequestProvider: func() interface{} {
			return &CaptureStartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureStartResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureStartRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureStart(context, req)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-stop",
		RequestInfo: &endly.ActionInfo{
			Description: "stop capturing console and network (Chrome/Edge CDP)",
		},
		RequestProvider: func() interface{} {
			return &CaptureStopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureStopResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureStopRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureStop(context, req)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-status",
		RequestInfo: &endly.ActionInfo{
			Description: "capture status and counters",
		},
		RequestProvider: func() interface{} {
			return &CaptureStatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureStatusResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureStatusRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureStatus(context, req)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-clear",
		RequestInfo: &endly.ActionInfo{
			Description: "clear captured console and network buffers",
		},
		RequestProvider: func() interface{} {
			return &CaptureClearRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureClearResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureClearRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureClear(context, req)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-export",
		RequestInfo: &endly.ActionInfo{
			Description: "export captured console and network buffers",
		},
		RequestProvider: func() interface{} {
			return &CaptureExportRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureExportResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureExportRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureExport(context, req)
		},
	})

}

// New creates a new webdriver service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
		fs:              afs.New(),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}

func pair(value string) (string, string) {
	pair := strings.SplitN(value, ":", 2)
	if len(pair) == 2 {
		return pair[0], pair[1]
	}
	return value, ""
}

func ensureOpenSessionDefaults(request *OpenSessionRequest) {
	if request == nil {
		return
	}
	if request.SessionID == "" {
		request.SessionID = "localhost:4444"
	}
	if request.Remote == "" {
		host, port := pair(request.SessionID)
		request.Remote = fmt.Sprintf("http://%v:%v/wd/hub", host, port)
	}
}

func isGetMethod(method string) bool {
	return strings.EqualFold(method, "Get")
}

func navigationWithDefaults(nav *NavigationOptions) NavigationOptions {
	if nav == nil {
		nav = &NavigationOptions{}
	}
	out := *nav
	if out.TimeoutMs <= 0 {
		out.TimeoutMs = 45000
	}
	if out.AutoScrollMs < 0 {
		out.AutoScrollMs = 0
	}
	if out.ScrollDelayMs <= 0 {
		out.ScrollDelayMs = 300
	}
	if out.StableWindowMs <= 0 {
		out.StableWindowMs = 1500
	}
	if out.MaxScrollSteps <= 0 {
		out.MaxScrollSteps = 30
	}
	if out.IdleThreshold < 0 {
		out.IdleThreshold = 0
	}
	if out.IdleWindowMs <= 0 {
		out.IdleWindowMs = 1500
	}
	if out.IdleMaxWaitMs < 0 {
		out.IdleMaxWaitMs = 0
	}
	return out
}

func (s *service) getWithGuard(context *endly.Context, session *Session, URL string, nav NavigationOptions) error {
	if session == nil || session.driver == nil {
		return fmt.Errorf("webdriver session not open")
	}
	_ = session.driver.SetPageLoadTimeout(time.Duration(nav.TimeoutMs) * time.Millisecond)
	err := session.driver.Get(URL)
	if err == nil {
		return nil
	}
	if !isPageLoadTimeout(err) {
		return err
	}
	context.Publish(msg.NewOutputEvent("Navigation timeout (continuing)", "webdriver.get", map[string]any{
		"url":   URL,
		"error": err.Error(),
	}))
	if nav.AutoScrollMs > 0 {
		if session.Capture == nil && session.Net == nil && isChromeLike(session.Browser) {
			session.Net = &netTracker{}
		}
		s.autoScrollStabilize(context, session, nav)
	}
	return nil
}

func isPageLoadTimeout(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return true
	}
	var sErr *selenium.Error
	if errors.As(err, &sErr) {
		if sErr.LegacyCode == 21 { // legacy "timeout"
			return true
		}
		if strings.Contains(strings.ToLower(sErr.Message), "timeout") {
			return true
		}
	}
	return false
}

func (s *service) autoScrollStabilize(context *endly.Context, session *Session, nav NavigationOptions) {
	maxWaitMs := nav.IdleMaxWaitMs
	if maxWaitMs == 0 {
		if nav.AutoScrollMs > 0 {
			maxWaitMs = nav.AutoScrollMs
		} else {
			maxWaitMs = nav.TimeoutMs
		}
	}
	deadline := time.Now().Add(time.Duration(maxWaitMs) * time.Millisecond)
	delay := time.Duration(nav.ScrollDelayMs) * time.Millisecond
	stableWindow := time.Duration(nav.StableWindowMs) * time.Millisecond
	idleWindow := time.Duration(nav.IdleWindowMs) * time.Millisecond

	var lastHeight float64 = -1
	stableSince := time.Now()
	idleSince := time.Time{}

	for step := 0; step < nav.MaxScrollSteps && time.Now().Before(deadline); step++ {
		height := s.scrollHeight(session)
		if height >= 0 && height == lastHeight {
			if time.Since(stableSince) >= stableWindow {
				if s.isNetworkIdle(session, nav.IdleThreshold, idleWindow, &idleSince) {
					return
				}
			}
		} else {
			stableSince = time.Now()
			lastHeight = height
		}

		_, _ = session.driver.ExecuteScript("window.scrollBy(0, window.innerHeight || 800);", nil)
		time.Sleep(delay)
		if session.Capture != nil {
			session.Capture.Drain(session)
		} else if session.Net != nil {
			session.Net.Drain(session.driver)
		}
		if time.Since(stableSince) >= stableWindow && s.isNetworkIdle(session, nav.IdleThreshold, idleWindow, &idleSince) {
			return
		}
	}
	_ = context // reserved for potential future warning publishing
}

func (s *service) isNetworkIdle(session *Session, threshold int, window time.Duration, idleSince *time.Time) bool {
	if threshold < 0 {
		threshold = 0
	}
	if window <= 0 {
		window = 1500 * time.Millisecond
	}

	inflight := -1
	if session.Capture != nil {
		inflight = session.Capture.Summary().RequestsInFlight
	} else if session.Net != nil {
		inflight = session.Net.Inflight()
	}
	if inflight < 0 {
		return true // cannot observe; do not block
	}
	if networkIdle(inflight, threshold) {
		if idleSince != nil && idleSince.IsZero() {
			*idleSince = time.Now()
		}
		if idleSince == nil {
			return true
		}
		return time.Since(*idleSince) >= window
	}
	if idleSince != nil {
		*idleSince = time.Time{}
	}
	return false
}

func (s *service) scrollHeight(session *Session) float64 {
	if session == nil || session.driver == nil {
		return -1
	}
	v, err := session.driver.ExecuteScript("return (document.body && document.body.scrollHeight) ? document.body.scrollHeight : 0;", nil)
	if err != nil {
		return -1
	}
	switch actual := v.(type) {
	case float64:
		return actual
	case int:
		return float64(actual)
	case int64:
		return float64(actual)
	case uint64:
		return float64(actual)
	case string:
		return toolbox.AsFloat(actual)
	default:
		return toolbox.AsFloat(actual)
	}
}
