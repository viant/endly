package webplanner

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/viant/endly/service/testing/runner/webdriver"
	"log"
	"net/http"
)

func (s *Service) handleActions(writer http.ResponseWriter, request *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return isAllowedOrigin(r.Header.Get("Origin"), r.Host)
		},
	}
	ws, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	ws.SetReadLimit(64 << 10)
	s.mux.Lock()
	s.ws = ws
	s.mux.Unlock()
	defer func() {
		s.mux.Lock()
		if s.ws == ws {
			s.ws = nil
		}
		s.mux.Unlock()
	}()
	// Infinite loop to read messages from the client
	for {
		var msg *ActionRequest
		// Read in a new message as JSON and map it to a Message object
		if err := ws.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("webplanner websocket read: %v", err)
			}
			break
		}
		s.handleLiveRequest(msg)
	}
}

type ActionRequest struct {
	Type            string `json:"type,omitempty"`
	Exclusion       string `json:"exclusion"`
	Attributes      string `json:"attributes"`
	Command         string `json:"command,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
	DebuggerAddress string `json:"debuggerAddress,omitempty"`
	Remote          string `json:"remote,omitempty"`
	DirectCDP       bool   `json:"directCDP,omitempty"`
}

type LiveResponse struct {
	Type   string      `json:"type"`
	Status string      `json:"status"`
	Output string      `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func (s *Service) handleLiveRequest(request *ActionRequest) {
	if request == nil {
		return
	}
	switch request.Type {
	case "command":
		if err := s.EnsureWebDriver(); err != nil {
			s.writeLive(&LiveResponse{Type: "commandResult", Status: "error", Error: err.Error()})
			return
		}
		if err := s.EnsureSession(); err != nil {
			s.writeLive(&LiveResponse{Type: "commandResult", Status: "error", Error: err.Error()})
			return
		}
		s.startLiveCommand(request.Command)
	case "cancelCommand":
		s.mux.Lock()
		cancel := s.liveCancel
		running := s.liveRunning
		s.mux.Unlock()
		if cancel != nil {
			cancel()
			go func() {
				_, _ = s.manager.Run(s.context, &webdriver.StopLoadingRequest{})
			}()
		}
		status := "ok"
		message := ""
		if !running {
			status = "error"
			message = "no live command was running"
		}
		s.writeLive(&LiveResponse{Type: "commandCancelled", Status: status, Error: message})
	case "state":
		data, err := s.liveBrowserState()
		response := &LiveResponse{Type: "state", Status: "ok", Data: data}
		if err != nil {
			response.Status = "error"
			response.Error = err.Error()
		}
		s.writeLive(response)
	case "attachRecorder":
		response := &LiveResponse{Type: "recorder", Status: "ok"}
		if request.DebuggerAddress != "" || request.Remote != "" {
			s.lifecycleMu.Lock()
			s.mux.Lock()
			s.Config.DebuggerAddress = request.DebuggerAddress
			s.Config.Remote = request.Remote
			s.Config.Browser = webdriver.ChromeBrowser
			s.Config.DirectCDP = request.DirectCDP
			s.opened = false
			s.started = false
			s.mux.Unlock()
			s.lifecycleMu.Unlock()
		}
		if err := s.EnsureWebDriver(); err != nil {
			response.Status = "error"
			response.Error = err.Error()
		} else if err := s.EnsureSession(); err != nil {
			response.Status = "error"
			response.Error = err.Error()
		} else if err := s.startActivityCapture(); err != nil {
			response.Status = "error"
			response.Error = err.Error()
		} else if err := s.injectTracker(); err != nil {
			response.Status = "error"
			response.Error = err.Error()
		}
		s.writeLive(response)
	case "recording":
		s.mux.Lock()
		if request.Enabled != nil {
			s.recording = *request.Enabled
		}
		enabled := s.recording
		s.mux.Unlock()
		s.writeLive(&LiveResponse{Type: "recording", Status: "ok", Data: map[string]bool{"enabled": enabled}})
	case "clearRecording":
		s.mux.Lock()
		s.recorded = s.recorded[:0]
		s.mux.Unlock()
		s.writeLive(&LiveResponse{Type: "recordingCleared", Status: "ok"})
	default:
		s.mux.Lock()
		s.exclusion = request.Exclusion
		s.attributes = request.Attributes
		s.mux.Unlock()
	}
}

func (s *Service) startLiveCommand(command string) {
	if command == "" {
		s.writeLive(&LiveResponse{Type: "commandResult", Status: "error", Error: "live command was empty"})
		return
	}
	s.mux.Lock()
	if s.liveRunning {
		s.mux.Unlock()
		s.writeLive(&LiveResponse{Type: "commandResult", Status: "error", Error: "another live command is already running"})
		return
	}
	runContext := s.manager.NewContext(s.context.Context.Clone())
	runContext.SessionID = s.context.SessionID
	state := s.context.State()
	runContext.SetState(state.Clone())
	runContext.SetListener(s.context.Listener)
	background, cancel := context.WithCancel(s.context.Background())
	runContext.SetBackground(background)
	s.liveGeneration++
	generation := s.liveGeneration
	s.liveRunning = true
	s.liveCancel = cancel
	s.mux.Unlock()
	s.writeLive(&LiveResponse{Type: "commandStarted", Status: "ok", Data: map[string]interface{}{"command": command}})

	go func() {
		output, err := s.runCommands(runContext, []string{command}, nil)
		cancel()
		s.mux.Lock()
		if s.liveGeneration == generation {
			s.liveRunning = false
			s.liveCancel = nil
		}
		s.mux.Unlock()
		response := &LiveResponse{Type: "commandResult", Status: "ok", Output: output}
		if err != nil {
			response.Status = "error"
			response.Error = err.Error()
		}
		s.writeLive(response)
	}()
}

func (s *Service) liveBrowserState() (interface{}, error) {
	if err := s.EnsureWebDriver(); err != nil {
		return nil, err
	}
	if err := s.EnsureSession(); err != nil {
		return nil, err
	}
	result, err := s.manager.Run(s.context, &webdriver.RunRequest{Commands: []interface{}{
		"tabs = page.tabs()",
		"url = page.url()",
		"title = page.title()",
	}})
	if err != nil {
		return nil, err
	}
	response, ok := result.(*webdriver.RunResponse)
	if !ok {
		return nil, nil
	}
	var capture interface{}
	if value, captureErr := s.manager.Run(s.context, &webdriver.CaptureStatusRequest{}); captureErr == nil {
		capture = value
	}
	return map[string]interface{}{
		"sessionId": response.SessionID,
		"backend":   response.Backend,
		"data":      response.Data,
		"capture":   capture,
		"recorded":  s.recordedActions(),
	}, nil
}

func (s *Service) writeLive(value interface{}) {
	s.mux.Lock()
	ws := s.ws
	s.mux.Unlock()
	if ws == nil {
		return
	}
	s.writeMu.Lock()
	err := ws.WriteJSON(value)
	s.writeMu.Unlock()
	if err != nil {
		log.Printf("webplanner websocket write: %v", err)
	}
}
