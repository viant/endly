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
		Handler: func(context *Context, handlerRequest interface{}) (interface{}, error) {
			if request, ok := handlerRequest.(*LoggerPrintRequest); ok {
				if !context.cliRunnner { //actual printing happened in runner
					if request.Message != "" {
						var message = s.Renderer.ColorText(request.Message, request.Color)
						s.Renderer.Println(message)
					} else if request.Error != "" {
						var errorMessage = s.Renderer.ColorText(request.Error, s.Renderer.ErrorColor)
						s.Renderer.Println(errorMessage)
					}
				}
				return nil, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", handlerRequest)
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
