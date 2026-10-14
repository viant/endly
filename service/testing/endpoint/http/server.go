package http

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

// HTTPRequestKeyProvider represents request key provider to extract a request field.
type HTTPRequestKeyProvider func(source interface{}) (string, error)

// HTTPRequestKeyProviders represents key providers
var HTTPRequestKeyProviders = make(map[string]HTTPRequestKeyProvider)

type Server struct {
	http.Server
	*httpHandler
	trips            map[string]*HTTPResponses
	mux              sync.Mutex
	rotate           bool
	indexKeys        []string
	requestTemplate  string
	responseTemplate string
	bindHost         string
}

// Replace installs a fully loaded recording without releasing the listening
// socket. In-flight requests finish against the previous recording.
func (s *Server) Replace(trips *HTTPServerTrips, requestTemplate, responseTemplate string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.httpHandler.mu.Lock()
	defer s.httpHandler.mu.Unlock()
	s.trips, s.rotate, s.indexKeys = trips.Trips, trips.Rotate, trips.IndexKeys
	s.requestTemplate, s.responseTemplate = requestTemplate, responseTemplate
	atomic.StoreInt64(&s.httpHandler.thinkTime, 0)
	s.httpHandler.handler = getServerHandler(&s.Server, s.httpHandler, trips)
}

func (s *Server) Append(trips *HTTPServerTrips) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.httpHandler.mu.Lock()
	defer s.httpHandler.mu.Unlock()
	if len(s.trips) > 0 {
		for k, v := range s.trips {
			if _, ok := trips.Trips[k]; ok {
				continue
			}
			trips.Trips[k] = v
		}
	}
	s.trips = trips.Trips
	s.httpHandler.handler = getServerHandler(&s.Server, s.httpHandler, trips)
}

// StartServer starts http request, the server has ability to replay recorded  HTTP trips with https://github.com/viant/toolbox/blob/master/bridge/http_bridge_recording_util.go#L82
func StartServer(port int, trips *HTTPServerTrips, reqTemplate, respTemplate string) (*Server, error) {
	return startServer("", port, trips, reqTemplate, respTemplate)
}

func startServer(host string, port int, trips *HTTPServerTrips, reqTemplate, respTemplate string) (*Server, error) {
	err := trips.Init(reqTemplate, respTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to start http server :%v, %v", port, err)
	}

	var httpHandler = &httpHandler{
		running: 1,
	}

	server := &Server{
		bindHost:         host,
		rotate:           trips.Rotate,
		indexKeys:        trips.IndexKeys,
		httpHandler:      httpHandler,
		trips:            trips.Trips,
		Server:           http.Server{Addr: net.JoinHostPort(host, fmt.Sprint(port)), Handler: httpHandler},
		requestTemplate:  reqTemplate,
		responseTemplate: respTemplate,
	}
	httpHandler.handler = getServerHandler(&server.Server, httpHandler, trips)

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start http server on port %v: %w", port, err)
	}
	server.Addr = listener.Addr().String()
	go func() {
		_ = server.Server.Serve(listener)
		atomic.StoreInt32(&httpHandler.running, 0)
	}()
	return server, nil
}
