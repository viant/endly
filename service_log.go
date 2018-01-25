package endly

import (
	"fmt"
)

const (
	//LogServiceID represents log service id.
	LogServiceID = "log"

	//LogServicePrintAction represents log action
	LogServicePrintAction = "print"
)

//Log represent no operation
type LogPrintRequest struct {
	Message string
	Error   string
}

//LogService represents no operation service
type LogService struct {
	*AbstractService
}

//Run run supplied request
func (s *LogService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok", Response: struct{}{}}
	switch request.(type) {
	case *LogPrintRequest:
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
func (s *LogService) NewRequest(action string) (interface{}, error) {
	switch action {
	case LogServicePrintAction:
		return &LogPrintRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)
}

//NewRequest returns a new request for supplied action
func (s *LogService) NewResponse(action string) (interface{}, error) {
	switch action {
	case LogServicePrintAction:
		return struct{}{}, nil
	}
	return s.AbstractService.NewResponse(action)
}

//NewLogService creates a new log service.
func NewLogService() Service {
	var result = &LogService{
		AbstractService: NewAbstractService(LogServiceID,
			LogServicePrintAction,
		),
	}
	result.AbstractService.Service = result
	return result
}
