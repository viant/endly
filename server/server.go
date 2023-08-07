package server

import (
	"encoding/json"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"log"
	"net/http"
)

// Request represents service request.
type Request struct {
	Data           map[string]interface{}
	ServiceRequest interface{}
}

// Response service response
type Response struct {
	Status   string
	Error    string
	Response interface{}
	Data     map[string]interface{}
}

// Server represents a server
type Server struct {
	port    string
	manager endly.Manager
}

func (s *Server) requestService(serviceName, action string, httpRequest *http.Request, httpResponse http.ResponseWriter) (*Response, error) {
	var service endly.Service
	var serviceRequest interface{}
	var err error
	service, err = s.manager.Service(serviceName)
	if err != nil {
		return nil, err
	}
	context := s.manager.NewContext(toolbox.NewContext())
	serviceRequest, err = context.NewRequest(serviceName, action, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	request := &Request{
		ServiceRequest: serviceRequest,
	}
	err = json.NewDecoder(httpRequest.Body).Decode(request)
	if err != nil {
		return nil, err
	}

	defer context.Close()
	state := context.State()
	state.Apply(request.Data)
	serviceResponse := service.Run(context, request.ServiceRequest)
	var response = &Response{
		Status:   serviceResponse.Status,
		Error:    serviceResponse.Error,
		Response: serviceResponse.Response,
		Data:     state.AsEncodableMap(),
	}
	return response, nil
}

func (s *Server) routeHandler(serviceRouting *toolbox.ServiceRouting, httpRequest *http.Request, httpResponse http.ResponseWriter, uriParameters map[string]interface{}) (err error) {
	defer func() {
		if err != nil {
			serviceResponse := &Response{
				Error: fmt.Sprintf("%v", err),
			}
			err = toolbox.WriteServiceRoutingResponse(httpResponse, httpRequest, serviceRouting, serviceResponse)
		}

	}()

	serviceName, ok := uriParameters["service"]
	if !ok {
		return fmt.Errorf("service name was missing %v", uriParameters)
	}
	action, ok := uriParameters["action"]
	if !ok {
		return fmt.Errorf("action was missing %v", uriParameters)
	}

	var response *Response
	response, err = s.requestService(toolbox.AsString(serviceName), toolbox.AsString(action), httpRequest, httpResponse)
	if err != nil {
		return err
	}
	err = toolbox.WriteServiceRoutingResponse(httpResponse, httpRequest, serviceRouting, response)

	if err != nil {
		return err
	}
	return nil

}

// Start starts server
func (s *Server) Start() error {

	router := toolbox.NewServiceRouter(
		toolbox.ServiceRouting{
			HTTPMethod:     "POST",
			URI:            "/v1/endly/service/{service}/{action}/",
			Handler:        s.requestService,
			HandlerInvoker: s.routeHandler,
			Parameters:     []string{"service", "action", "@httpRequest", "@httpResponseWriter"},
		})

	http.HandleFunc("/v1/", func(response http.ResponseWriter, reader *http.Request) {
		err := router.Route(response, reader)
		if err != nil {
			response.WriteHeader(http.StatusInternalServerError)
		}
	})
	fmt.Printf("Started test server on port %v\n", s.port)
	log.Fatal(http.ListenAndServe(":"+s.port, nil))
	return nil
}

// New createss a new server for provided port.
func New(port string) *Server {
	return &Server{
		port:    port,
		manager: endly.New(),
	}
}
