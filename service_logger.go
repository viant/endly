package endly

import (
	"fmt"
	"os"
)

const (
	//backward compatible
	LogServiceID = "log"
	//LoggerServiceID represents log service id.
	LoggerServiceID = "logger"

	//LogServicePrintAction represents log action
	LogServicePrintAction = "print"
)

//Log represent no operation
type LogPrintRequest struct {
	Message string
	Color   string
	Error   string
}

//LoggerService represents no operation service
type LoggerService struct {
	*AbstractService
	*Renderer
}

//Run run supplied request
func (s *LoggerService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok", Response: struct{}{}}
	switch actualRequest := request.(type) {
	case *LogPrintRequest:
		if actualRequest.Message != "" {
			var message = s.Renderer.ColorText(actualRequest.Message, actualRequest.Color)
			s.Renderer.Println(message)
		} else if actualRequest.Error != "" {
			var errorMessage = s.Renderer.ColorText(actualRequest.Error, s.Renderer.ErrorColor)
			s.Renderer.Println(errorMessage)
		}

	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}

	if response.Error != "" {
		response.Status = "error"
	}

	defer s.End(context)(startEvent, Pairs("response", response))
	return response
}

//NewRequest returns a new request for supplied action
func (s *LoggerService) NewRequest(action string) (interface{}, error) {
	switch action {
	case LogServicePrintAction:
		return &LogPrintRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)
}

//NewRequest returns a new request for supplied action
func (s *LoggerService) NewResponse(action string) (interface{}, error) {
	switch action {
	case LogServicePrintAction:
		return struct{}{}, nil
	}
	return s.AbstractService.NewResponse(action)
}

//NewLogService creates a new log service.
func NewLogService() Service {
	var result = &LoggerService{
		Renderer: NewRenderer(os.Stdout, 120),
		AbstractService: NewAbstractService(LoggerServiceID,
			LogServicePrintAction,
		),
	}
	result.AbstractService.Service = result
	return result
}
