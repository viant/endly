package webplanner

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/viant/endly"
	"github.com/viant/endly/internal/webplanner/httputil"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const Separator = ", "

// Config holds the configuration for the server
type Config struct {
	Port             int
	Host             string
	Token            string
	Browser          string
	Remote           string
	DebuggerAddress  string
	DirectCDP        bool
	PageLoadStrategy string
}

// Service represents the HTTP server.
type Service struct {
	Config             *Config
	context            *endly.Context
	manager            endly.Manager
	exclusion          string
	attributes         string
	ws                 *websocket.Conn
	mux                sync.Mutex
	writeMu            sync.Mutex
	lifecycleMu        sync.Mutex
	Keys               string
	Target             string
	started            bool
	opened             bool
	recording          bool
	recorded           []*Action
	recorderGeneration uint64
	seenEvents         map[string]struct{}
	seenEventOrder     []string
	pollerStarted      bool
	liveCancel         func()
	liveRunning        bool
	liveGeneration     uint64
}

// NewService creates a new instance of Service with the provided config.
func NewService(config *Config) *Service {
	config = ensureConfig(config)
	return &Service{
		Config:     config,
		recording:  true,
		recorded:   make([]*Action, 0),
		seenEvents: make(map[string]struct{}),
	}
}

// Start starts the HTTP server.
func (s *Service) Start() error {
	host := strings.TrimSpace(s.Config.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleContent)
	mux.HandleFunc("/run", s.handlerRequest)
	mux.HandleFunc("/event", s.handleEvent)
	mux.HandleFunc("/ws", s.handleActions)

	address := net.JoinHostPort(host, fmt.Sprintf("%d", s.Config.Port))
	fmt.Printf("Server is running at http://%s/\n", address)
	return http.ListenAndServe(address, mux)
}

//go:embed content/index.html
var content string

// handleContent handles the web requests.
func (s *Service) handleContent(w http.ResponseWriter, r *http.Request) {
	_, _ = io.WriteString(w, content)
}

func enableCors(writer http.ResponseWriter, request *http.Request, allowCrossOrigin bool) bool {
	origin := request.Header.Get("Origin")
	if !allowCrossOrigin && !isAllowedOrigin(origin, request.Host) {
		return false
	}
	if origin != "" {
		writer.Header().Set(httputil.AllowOriginHeader, origin)
		writer.Header().Set("Vary", "Origin")
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
	return true
}

func (s *Service) authorize(request *http.Request) bool {
	if s.Config.Token == "" {
		return false
	}
	candidate := request.Header.Get("X-Endly-Planner-Token")
	if candidate == "" {
		candidate = request.URL.Query().Get("token")
	}
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(s.Config.Token)) == 1
}

func isAllowedOrigin(origin, requestHost string) bool {
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	originHost := strings.ToLower(parsed.Hostname())
	requestHostname, _, splitErr := net.SplitHostPort(requestHost)
	if splitErr != nil {
		requestHostname = requestHost
	}
	requestHostname = strings.ToLower(strings.Trim(requestHostname, "[]"))
	if originHost == requestHostname {
		return true
	}
	return isLoopbackHost(originHost) && isLoopbackHost(requestHostname)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	IP := net.ParseIP(strings.Trim(host, "[]"))
	return IP != nil && IP.IsLoopback()
}

func New(config *Config) *Service {
	config = ensureConfig(config)
	ret := &Service{
		Config:     config,
		manager:    endly.New(),
		recording:  true,
		recorded:   make([]*Action, 0),
		seenEvents: make(map[string]struct{}),
	}
	ret.context = ret.manager.NewContext(nil)
	return ret
}

func ensureConfig(config *Config) *Config {
	if config == nil {
		config = &Config{}
	}
	if config.Host == "" {
		config.Host = "127.0.0.1"
	}
	if config.Token == "" {
		buffer := make([]byte, 24)
		if _, err := rand.Read(buffer); err == nil {
			config.Token = hex.EncodeToString(buffer)
		}
	}
	return config
}
