package elogger

import (
	"net/http"
	"github.com/viant/toolbox"
)

//Service represents a service event logger
type Service interface {
	Log(http.ResponseWriter, *http.Request) error
}

type service struct {
	logger *toolbox.FileLogger
}

//Log logs supplied request details
func (s *service) Log(writer http.ResponseWriter, request *http.Request) error {
	message := NewMessage(request)
	logMessage := &toolbox.LogMessage{MessageType: "elog", Message: message}
	writer.Header().Set("Eventid", message.EventID)
	writer.WriteHeader(http.StatusOK)
	return s.logger.Log(logMessage)
}

//NewService creates a new service for supplied config.
func NewService(config *Config) (Service, error) {
	logger, err := toolbox.NewFileLogger(config.LogTypes...)
	if err != nil {
		return nil, err
	}
	return &service{
		logger: logger,
	}, nil
}
