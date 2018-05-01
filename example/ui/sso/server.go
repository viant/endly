package sso

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

//Start start server
func (s *Server) Start() {
	http.HandleFunc("/api/", func(writer http.ResponseWriter, reader *http.Request) {
		err := s.serviceRouter.Route(writer, reader)
		if err != nil {
			http.Error(writer, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			return
		}
	})
	for _, route := range s.config.StaticRoutes {
		fileServer := http.FileServer(http.Dir(route.Directory))
		http.Handle(route.URI, fileServer)
	}
	fmt.Printf("Started test server on port %v\n", s.config.Port)
	log.Fatal(http.ListenAndServe(":"+s.config.Port, nil))
}

//NewServer creates a new http server
func NewServer(config *Config, service Service) (*Server, error) {
	serviceRouter := toolbox.NewServiceRouter(
		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        "/api/singup/",
			Handler:    service.SignUp,
			Parameters: []string{"request"},
		},
		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        "/api/signin/",
			Handler:    service.SignIn,
			Parameters: []string{"request"},
		})
	var result = &Server{
		serviceRouter: serviceRouter,
		service:       service,
		config:        config,
	}
	return result, nil
}
