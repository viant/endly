package http

import (
	"errors"
	"sync"
)

//ListenRequest represent HTTP endpoint listen request
type ListenRequest struct {
	Port          int
	Rotate        bool
	BaseDirectory string   `required:"true" description:"location with replay files (could be generate by https://github.com/viant/toolbox/blob/master/bridge/http_bridge_recording_util.go#L81"`
	IndexKeys     []string `description:"recorded requests matching keys, by default: Method,URL,Body,Cookie,Content-Type"`
}

//ListenResponse represents HTTP endpoint listen response with indexed trips
type ListenResponse struct {
	Trips map[string]*HTTPResponses
}

//Validate checks if request is valid.
func (r ListenRequest) Validate() error {
	if r.BaseDirectory == "" {
		return errors.New("baseDirectory was empty")
	}
	if r.Port == 0 {
		return errors.New("port was empty")
	}
	return nil
}

//AsHTTPServerTrips return a new HTTP trips.
func (r ListenRequest) AsHTTPServerTrips() *HTTPServerTrips {
	if len(r.IndexKeys) == 0 {
		r.IndexKeys = []string{MethodKey, URLKey, BodyKey, CookieKey, ContentTypeKey}
	}
	return &HTTPServerTrips{
		Rotate:        r.Rotate,
		BaseDirectory: r.BaseDirectory,
		Trips:         make(map[string]*HTTPResponses),
		IndexKeys:     r.IndexKeys,
		Mutex:         &sync.Mutex{},
	}
}


//ShutdownRequest represent http endpoint shutdown request
type ShutdownRequest struct {
	Port int
}