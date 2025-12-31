package webplanner

import (
	_ "embed"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/viant/endly"
	"github.com/viant/endly/internal/webplanner/httputil"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

const Separator = ", "

// Config holds the configuration for the server
type Config struct {
	Port int
}

// Service represents the HTTP server.
type Service struct {
	Config     *Config
	context    *endly.Context
	manager    endly.Manager
	exclusion  string
	attributes string
	ws         *websocket.Conn
	mux        sync.Mutex
	Keys       string
	Target     string
	started    bool
	opened     bool
}

// NewService creates a new instance of Service with the provided config.
func NewService(config *Config) *Service {
	return &Service{
		Config: config,
	}
}

// Start starts the HTTP server.
func (s *Service) Start() {
	http.HandleFunc("/", s.handleContent)
	http.HandleFunc("/run", s.handlerRequest)
	http.HandleFunc("/event", s.handleEvent)
	http.HandleFunc("/ws", s.handleActions)

	address := fmt.Sprintf(":%d", s.Config.Port)
	fmt.Printf("Server is running at http://localhost%s/\n", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}

//go:embed content/index.html
var content string

// handleContent handles the web requests.
func (s *Service) handleContent(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, content)
}

func enableCors(writer http.ResponseWriter, request *http.Request) {
	origins := request.Header["Origin"]
	origin := ""
	if len(origins) > 0 {
		origin = origins[0]
	}
	//Access-Control-Allow-Origin
	if origin == "" {
		writer.Header().Set(httputil.AllowOriginHeader, "*")
	} else {
		writer.Header().Set(httputil.AllowOriginHeader, origin)
	}

	if request.Method == "OPTIONS" {
		requestMethod := request.Header.Get(httputil.ControlRequestHeader)
		if requestMethod != "" {
			writer.Header().Set(httputil.AllowMethodsHeader, requestMethod)
		}
		if requestHeaders := request.Header.Get(httputil.AccessRequestHeader); requestHeaders != "" {
			writer.Header().Set(httputil.AllowRequestHeader, requestHeaders)
		}
	}
}

func New(config *Config) *Service {
	ret := &Service{
		Config:  config,
		manager: endly.New(),
	}
	ret.context = ret.manager.NewContext(nil)
	return ret
}
