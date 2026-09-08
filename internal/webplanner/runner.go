package webplanner

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/runner/webdriver"
	"io"
	"net/http"
	"strconv"
	"strings"
)

//go:embed content/tracker.js
var trackerCode string

type RunResponsePayload struct {
	Status         string
	Message        string
	Plan           string
	Output         string
	CaptureSinkURL string
	CaptureSummary *webdriver.CaptureSummary
}

func (s *Service) RunCommands(lines []string) (string, error) {
	return s.RunCommandsWithOptions(lines, nil)
}

func (s *Service) RunCommandsWithOptions(lines []string, nav *webdriver.NavigationOptions) (string, error) {
	return s.runCommands(s.context, lines, nav)
}

func (s *Service) runCommands(runContext *endly.Context, lines []string, nav *webdriver.NavigationOptions) (string, error) {
	var commands []interface{}
	if len(lines) == 0 {
		return "", nil
	}
	if hasGet(lines) {
		_, err := s.manager.Run(runContext, &webdriver.RunRequest{Commands: []interface{}{lines[0]}, Navigation: nav})
		if err != nil {
			return "", err
		}
		lines = lines[1:]
		if err = s.injectTrackerOnContext(runContext); err != nil {
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
	ret, err := s.manager.Run(runContext, &webdriver.RunRequest{Commands: commands, Navigation: nav})
	if err != nil {
		return "", err
	}

	response, ok := ret.(*webdriver.RunResponse)
	if !ok || response == nil {
		return "", fmt.Errorf("unexpected webdriver response %T", ret)
	}
	data, _ := json.Marshal(response)
	if len(response.LookupErrors) > 0 {
		return "", errors.New(response.LookupErrors[0])
	}
	return string(data), nil
}

func hasGet(lines []string) bool {
	if len(lines) == 0 {
		return false
	}
	command := strings.ToLower(strings.TrimSpace(lines[0]))
	return strings.HasPrefix(command, "get") || strings.HasPrefix(command, "page.goto(")
}

func (s *Service) injectTracker() error {
	return s.injectTrackerOnContext(s.context)
}

func (s *Service) injectTrackerOnContext(runContext *endly.Context) error {
	jsCode := strings.ReplaceAll(trackerCode, "${port}", strconv.Itoa(s.Config.Port))
	jsCode = strings.ReplaceAll(jsCode, "${host}", s.Config.Host)
	jsCode = strings.ReplaceAll(jsCode, "${token}", s.Config.Token)
	_, err := s.manager.Run(runContext, &webdriver.WebDriverCallRequest{
		SessionID: "localhost:4444",
		Call: &webdriver.MethodCall{
			Method:     "ExecuteScript",
			Parameters: []interface{}{jsCode, []interface{}{}},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to inject trackerCode: %v", err)
	}
	return nil
}

func (s *Service) EnsureWebDriver() error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.started {
		return nil
	}
	s.mux.Lock()
	pageLoadStrategy := s.Config.PageLoadStrategy
	directCDP := s.Config.DirectCDP
	s.mux.Unlock()
	if directCDP {
		s.started = true
		return nil
	}
	_, err := s.manager.Run(s.context, &webdriver.StartRequest{PageLoadStrategy: pageLoadStrategy})
	if err != nil {
		return err
	}
	s.started = true
	return nil
}

func (s *Service) EnsureSession() error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.opened {
		return nil
	}
	s.mux.Lock()
	config := *s.Config
	s.mux.Unlock()
	_, err := s.manager.Run(s.context, &webdriver.OpenSessionRequest{
		Browser:          config.Browser,
		Remote:           config.Remote,
		DebuggerAddress:  config.DebuggerAddress,
		DirectCDP:        config.DirectCDP,
		PageLoadStrategy: config.PageLoadStrategy,
	})
	if err != nil {
		return err
	}
	s.opened = true
	return nil
}

func (s *Service) handlerRequest(writer http.ResponseWriter, request *http.Request) {
	if !enableCors(writer, request, false) {
		http.Error(writer, "origin not allowed", http.StatusForbidden)
		return
	}
	if request.Method == http.MethodOptions {
		writer.WriteHeader(200)
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost+", "+http.MethodOptions)
		http.Error(writer, "invalid method:"+request.Method, http.StatusMethodNotAllowed)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 1<<20)
	response := &RunResponsePayload{Status: "ok"}
	plan, output, err := s.runRequest(writer, request, response)
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

func (s *Service) runRequest(writer http.ResponseWriter, request *http.Request, response *RunResponsePayload) (string, string, error) {
	data, err := io.ReadAll(request.Body)
	if err != nil {
		return "", "", err
	}
	type Input struct {
		Plan    string `json:"plan"`
		Capture struct {
			Enabled         bool   `json:"enabled"`
			SinkURL         string `json:"sinkURL"`
			FlushIntervalMs int    `json:"flushIntervalMs"`
		} `json:"capture"`
		Navigation *webdriver.NavigationOptions `json:"navigation"`
	}
	input := &Input{}
	if err := json.Unmarshal(data, input); err != nil {
		return "", "", err
	}
	var output string
	if err = s.EnsureWebDriver(); err == nil {
		if err = s.EnsureSession(); err != nil {
			return input.Plan, "", err
		}

		if input.Capture.Enabled {
			_, err = s.manager.Run(s.context, &webdriver.CaptureStartRequest{
				SinkURL:         input.Capture.SinkURL,
				FlushIntervalMs: input.Capture.FlushIntervalMs,
			})
			if err != nil {
				return input.Plan, "", err
			}
			response.CaptureSinkURL = input.Capture.SinkURL
		}

		output, err = s.RunCommandsWithOptions(strings.Split(input.Plan, "\n"), input.Navigation)

		if input.Capture.Enabled {
			stop, stopErr := s.manager.Run(s.context, &webdriver.CaptureStopRequest{})
			if stopErr == nil {
				if stopResp, ok := stop.(*webdriver.CaptureStopResponse); ok {
					response.CaptureSummary = stopResp.Summary
				}
			}
		}
	}
	return input.Plan, output, err
}
