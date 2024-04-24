package webplanner

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/webdriver"
	"io"
	"net/http"
	"strings"
)

// Config holds the configuration for the server
type Config struct {
	Port int
}

// Service represents the HTTP server.
type Service struct {
	Config  *Config
	context *endly.Context
	manager endly.Manager
	started bool
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

func (s *Service) RunCommands(lines []string) (string, error) {
	var commands []interface{}
	for _, line := range lines {
		commands = append(commands, line)
	}
	ret, err := s.manager.Run(s.context, &webdriver.RunRequest{Commands: commands})
	if err != nil {
		return "", err
	}
	response, _ := ret.(*webdriver.RunResponse)
	data, _ := json.Marshal(response.Data)
	if len(response.LookupErrors) > 0 {
		return "", fmt.Errorf(response.LookupErrors[0])
	}
	return string(data), nil
}

func (s *Service) EnsureWebDriver() error {
	if s.started {
		return nil
	}
	_, err := s.manager.Run(s.context, &webdriver.StartRequest{})
	if err != nil {
		return err
	}
	s.started = true
	return nil
}

func (s *Service) handlerRequest(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "invalid method:"+request.Method, http.StatusInternalServerError)
		return
	}
	type Response struct {
		Status  string
		Message string
		Plan    string
		Output  string
	}
	response := &Response{Status: "ok"}
	plan, output, err := s.runRequest(writer, request)
	if err != nil {
		response.Status = "error"
		response.Message = err.Error()
	}
	response.Plan = plan
	response.Output = output
	data, _ := json.Marshal(response)
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(data)
}

func (s *Service) runRequest(writer http.ResponseWriter, request *http.Request) (string, string, error) {
	data, err := io.ReadAll(request.Body)
	if err != nil {
		return "", "", err
	}
	type Input struct {
		Plan string `json:"plan"`
	}
	input := &Input{}
	if err := json.Unmarshal(data, input); err != nil {
		return "", "", err
	}
	var output string
	if err = s.EnsureWebDriver(); err == nil {
		output, err = s.RunCommands(strings.Split(input.Plan, "\n"))
	}
	return input.Plan, output, err
}

func New(config *Config) *Service {
	ret := &Service{
		Config:  config,
		manager: endly.New(),
	}
	ret.context = ret.manager.NewContext(nil)
	return ret
}
