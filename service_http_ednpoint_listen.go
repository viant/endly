package endly

import (
	"github.com/pkg/errors"
	"sync"
)

//HTTPEndpointListenRequest represent HTTP endpoint listen request
type HTTPEndpointListenRequest struct {
	Port          int
	BaseDirectory string   `required:"true" description:"location with replay files (could be generate by https://github.com/viant/toolbox/blob/master/bridge/http_bridge_recording_util.go#L81"`
	IndexKeys     []string `description:"recorded requests matching keys, by default: Method,URL,Body,Cookie,Content-Type"`
}

//HTTPEndpointListenResponse represents HTTP endpoint listen response with indexed trips
type HTTPEndpointListenResponse struct {
	Trips map[string]*HTTPResponses
}

//Validate checks if request is valid.
func (r HTTPEndpointListenRequest) Validate() error {
	if r.BaseDirectory == "" {
		return errors.New("BaseDirectory was empty")
	}
	if r.Port == 0 {
		return errors.New("Port was empty")
	}
	return nil
}

//AsHTTPServerTrips return a new HTTP trips.
func (r HTTPEndpointListenRequest) AsHTTPServerTrips() *HTTPServerTrips {
	if len(r.IndexKeys) == 0 {
		r.IndexKeys = []string{MethodKey, URLKey, BodyKey, CookieKey, ContentTypeKey}
	}
	return &HTTPServerTrips{
		BaseDirectory: r.BaseDirectory,
		Trips:         make(map[string]*HTTPResponses),
		IndexKeys:     r.IndexKeys,
		Mutex:         &sync.Mutex{},
	}
}
