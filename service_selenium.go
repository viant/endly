package endly

import (
	"errors"
	"fmt"
	"github.com/tebeka/selenium"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
)

//represents a SeleniumServiceID
const SeleniumServiceID = "selenium"

type SeleniumOpenSessionRequest struct {
	Browser        string
	RemoteSelenium *url.Resource //remote selenium resource
}

func (r *SeleniumOpenSessionRequest) Validate() error {
	if r.RemoteSelenium == nil {
		return errors.New("Remote (remote selenium endpoint) was empty")
	}
	if r.RemoteSelenium.URL == "" {
		return errors.New("Remote.URL (selenium resource URL) was empty")
	}
	if r.RemoteSelenium.Name == "" {
		return errors.New("Remote.Name (selenium browser) was empty")
	}
	return nil
}

type SeleniumOpenSessionResponse struct {
	SessionID string
}

type WebElementSelector struct {
	By    string
	Value string
}

type SeleniumWebElementCallRequest struct {
	SessionID  string
	Selector   *WebElementSelector
	Method     string
	Parameters []interface{}
}

type SeleniumCallResponse struct {
	Result []interface{}
}

type SeleniumWebDriverCallRequest struct {
	SessionID  string
	Method     string
	Parameters []interface{}
}

type SeleniumSession struct {
	ID      string
	Browser string
	driver  selenium.WebDriver
}

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
	case *SeleniumOpenSessionRequest:
		response.Response, err = s.open(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}
	case *SeleniumWebDriverCallRequest:
		response.Response, err = s.webDriverCall(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}
	case *SeleniumWebElementCallRequest:
		response.Response, err = s.webElementCall(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}

	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *seleniumService) webDriverCall(context *Context, request *SeleniumWebDriverCallRequest) (*SeleniumCallResponse, error) {
	seleniumSession, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	return s.callMethod(seleniumSession.driver, request.Method, request.Parameters)
}

func (s *seleniumService) callMethod(owner interface{}, methodName string, parameters []interface{}) (*SeleniumCallResponse, error) {
	method, err := toolbox.GetFunction(owner, methodName)
	if err != nil {
		return nil, err
	}
	parameters, err = toolbox.AsCompatibleFunctionParameters(method, parameters)
	if err != nil {
		return nil, err
	}
	var response = &SeleniumCallResponse{}
	response.Result = toolbox.CallFunction(method, parameters)
	return response, nil
}

func (s *seleniumService) webElementCall(context *Context, request *SeleniumWebElementCallRequest) (interface{}, error) {
	seleniumSession, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	var selector = request.Selector
	element, err := seleniumSession.driver.FindElement(selector.By, selector.Value)
	if err != nil {
		return nil, err
	}
	if element == nil {
		return nil, fmt.Errorf("Failed to lookup element: %v %v", selector.By, selector.Value)
	}
	return s.callMethod(element, request.Method, request.Parameters)
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

func (s *seleniumService) session(context *Context, sessionID string) (*SeleniumSession, error) {
	sessions := context.SeleniumSessions()
	if seleniumSession, ok := sessions[sessionID]; ok {
		return seleniumSession, nil
	}
	return nil, fmt.Errorf("Failed to lookup seleniun session id: %v, make sure you first run SeleniumOpenSessionRequest\n", sessionID)
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
		if seleniumSession.Browser == resource.Name {
			return seleniumSession, nil
		}
		seleniumSession.driver.Close()
	} else {
		seleniumSession = &SeleniumSession{
			ID:      sessionID,
			Browser: resource.Name,
		}
	}

	//TODO deploy and start remote selenium server if needed
	caps := selenium.Capabilities{"browserName": resource.Name}
	seleniumSession.driver, err = selenium.NewRemote(caps, fmt.Sprintf("http://%v:%v/webDriver/hub", resource.ParsedURL.Host, resource.ParsedURL.Port()))
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
	case "open":
		return &SeleniumOpenSessionRequest{}, nil
	case "web-driver":
		return &SeleniumOpenSessionRequest{}, nil
	case "web-element":
		return &SeleniumOpenSessionRequest{}, nil
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
