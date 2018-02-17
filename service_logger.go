package endly

import (
	"fmt"
	"os"
)

const (
	//LogServiceID backward compatible id
	LogServiceID = "log"
	//LoggerServiceID represents log service id.
	LoggerServiceID = "logger"
)

//LoggerPrintRequest represent print request
type LoggerPrintRequest struct {
	Message string
	Color   string
	Error   string
}

//LoggerService represents no operation service
type LoggerService struct {
	*AbstractService
	*Renderer
}

func (s *LoggerService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "print",
		RequestInfo: &ActionInfo{
			Description: "print log message",
		},
		RequestProvider: func() interface{} {
			return &LoggerPrintRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if _, ok := request.(*LoggerPrintRequest); ok {
				//actual printing happened in runner (it is async)
				return nil, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewLogService creates a new log service.
func NewLogService() Service {
	var result = &LoggerService{
		Renderer:        NewRenderer(os.Stdout, 120),
		AbstractService: NewAbstractService(LoggerServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
