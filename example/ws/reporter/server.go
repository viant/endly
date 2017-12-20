package reporter

import (
	"fmt"
	"github.com/viant/toolbox"
	"log"
	"net/http"
)

type Server struct {
	port          string
	serviceRouter *toolbox.ServiceRouter
}

func (s *Server) Start() {

	http.HandleFunc("/v1/", func(writer http.ResponseWriter, reader *http.Request) {
		err := s.serviceRouter.Route(writer, reader)
		if err != nil {
			log.Fatal(err)
		}
	})
	fmt.Printf("Started test server on port %v\n", s.port)
	log.Fatal(http.ListenAndServe(":"+s.port, nil))
}

func NewServer(port string, service Service) *Server {

	serviceRouter := toolbox.NewServiceRouter(
		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        "/v1/reporter/register/",
			Handler:    service.Register,
			Parameters: []string{"request"},
		},

		toolbox.ServiceRouting{
			HTTPMethod: "POST",
			URI:        "/v1/reporter/run/",
			Handler:    service.Run,
			Parameters: []string{"request"},
		},
	)
	return &Server{
		port:          port,
		serviceRouter: serviceRouter,
	}
}
