package elogger

import (
	"fmt"
	"log"
	"net/http"
)

//Server represents a server
type Server struct {
	service Service
	config  *Config
}

//Start start server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		err := s.service.Log(writer, request)
		if err != nil {
			http.Error(writer, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			return
		}
	})
	fmt.Printf("Started test server on port %v\n", s.config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", s.config.Port), mux))
	return nil
}

//NewServer creates a new server with supplied config.
func NewServer(config *Config, service Service) (*Server, error) {
	return &Server{
		service: service,
		config:  config,
	}, nil
}
