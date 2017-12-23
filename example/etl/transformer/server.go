package transformer

import (
	"fmt"
	"github.com/viant/toolbox"
	"log"
	"net/http"
)

//Server represents http server
type Server struct {
	service       Service
	config        *Config
	serviceRouter *toolbox.ServiceRouter
}

func handlerWrapper(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.Header.Del("If-Modified-Since")
		writer.Header().Set("Cache-Control", "no-cache, private, max-age=0")
		writer.Header().Set("Pragma", "no-cache")
		handler.ServeHTTP(writer, request)
	})
}

//Start start server
func (s *Server) Start() {
	http.HandleFunc("/api/", func(writer http.ResponseWriter, reader *http.Request) {
		err := s.serviceRouter.Route(writer, reader)
		if err != nil {
			http.Error(writer, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			return
		}
	})
	fmt.Printf("Started test server on port %v\n", s.config.Port)
	log.Fatal(http.ListenAndServe(":"+s.config.Port, nil))
}

//NewServer creates a new http server
func NewServer(config *Config, service Service) (*Server, error) {
	serviceRouter := toolbox.NewServiceRouter(
		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        "/api/copy/",
			Handler:    service.Copy,
			Parameters: []string{"request"},
		},
		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        "/api/task/list/",
			Handler:    service.TaskList,
			Parameters: []string{"request"},
		},
		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        "/api/task/kill/",
			Handler:    service.KillTask,
			Parameters: []string{"request"},
		})
	var result = &Server{
		serviceRouter: serviceRouter,
		service:       service,
		config:        config,
	}
	return result, nil
}
