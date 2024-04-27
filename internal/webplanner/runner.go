package webplanner

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/viant/endly/service/testing/runner/webdriver"
	"io"
	"net/http"
	"strconv"
	"strings"
)

//go:embed content/tracker.js
var trackerCode string

func (s *Service) RunCommands(lines []string) (string, error) {
	var commands []interface{}
	if len(lines) == 0 {
		return "", nil
	}
	if hasGet(lines) {
		_, err := s.manager.Run(s.context, &webdriver.RunRequest{Commands: []interface{}{lines[0]}})
		if err != nil {
			return "", err
		}
		lines = lines[1:]
		if err = s.injectTracker(); err != nil {
			return "", err
		}
	}
	if len(lines) == 0 {
		return "", nil
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		commands = append(commands, line)
	}
	if len(commands) == 0 {
		return "", nil
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

func hasGet(lines []string) bool {
	if len(lines) == 0 {
		return false
	}
	return strings.HasPrefix(strings.ToLower(lines[0]), "get")
}

func (s *Service) injectTracker() error {
	jsCode := strings.ReplaceAll(trackerCode, "${port}", strconv.Itoa(s.Config.Port))
	sessions := webdriver.Sessions(s.context)
	if session := sessions["localhost:4444"]; session != nil {
		if _, err := session.Driver().ExecuteScript(jsCode, nil); err != nil {
			return fmt.Errorf("failed to inject trackerCode: %v", err)
		}
	}
	return nil
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
	enableCors(writer, request)
	if request.Method == http.MethodOptions {
		writer.WriteHeader(200)
		return
	}
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
