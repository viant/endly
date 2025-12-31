package webdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	selog "github.com/tebeka/selenium/log"
	"github.com/viant/afs"
)

type CaptureSummary struct {
	StartedAt         time.Time
	RequestsInFlight  int
	RequestsCompleted int
	ConsoleEntries    int
	Errors            []string
}

type ConsoleEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

type CapturedBody struct {
	Encoding  string
	Data      string
	Truncated bool
}

type NetworkTransaction struct {
	RequestID string

	URL    string
	Method string

	RequestHeaders  map[string]any
	RequestBody     *CapturedBody
	ResourceType    string
	Initiator       map[string]any
	StartTimestamp  float64
	EndTimestamp    float64
	DurationMs      int64
	ErrorText       string
	WasCanceled     bool
	EncodedDataSize int64

	Status          int
	StatusText      string
	MimeType        string
	ResponseHeaders map[string]any
	ResponseBody    *CapturedBody
}

type CaptureState struct {
	mux sync.Mutex

	enabled   bool
	started   time.Time
	lastDrain time.Time

	includeBodies bool
	enableConsole bool
	enableNetwork bool

	maxBodyBytes  int
	redact        bool
	redactHeaders map[string]bool

	inflight  map[string]*NetworkTransaction
	completed []*NetworkTransaction
	console   []*ConsoleEntry

	errors []string

	sink *captureSink
}

func newCaptureState(req *CaptureStartRequest) *CaptureState {
	state := &CaptureState{
		enabled:       true,
		started:       time.Now(),
		includeBodies: true,
		enableConsole: true,
		enableNetwork: true,
		maxBodyBytes:  1_000_000,
		redact:        true,
		redactHeaders: map[string]bool{},
		inflight:      map[string]*NetworkTransaction{},
		completed:     []*NetworkTransaction{},
		console:       []*ConsoleEntry{},
		errors:        []string{},
	}
	for _, h := range []string{"authorization", "cookie", "set-cookie", "x-api-key"} {
		state.redactHeaders[h] = true
	}
	if req == nil {
		return state
	}
	if req.MaxBodyBytes > 0 {
		state.maxBodyBytes = req.MaxBodyBytes
	}
	if req.Redact != nil {
		state.redact = *req.Redact
	}
	if len(req.RedactHeaders) > 0 {
		state.redactHeaders = map[string]bool{}
		for _, h := range req.RedactHeaders {
			state.redactHeaders[strings.ToLower(strings.TrimSpace(h))] = true
		}
	}
	if req.EnableConsole != nil {
		state.enableConsole = *req.EnableConsole
	}
	if req.EnableNetwork != nil {
		state.enableNetwork = *req.EnableNetwork
	}
	if req.IncludeBodies != nil {
		state.includeBodies = *req.IncludeBodies
	}
	return state
}

func (s *CaptureState) Summary() *CaptureSummary {
	s.mux.Lock()
	defer s.mux.Unlock()

	return &CaptureSummary{
		StartedAt:         s.started,
		RequestsInFlight:  len(s.inflight),
		RequestsCompleted: len(s.completed),
		ConsoleEntries:    len(s.console),
		Errors:            append([]string(nil), s.errors...),
	}
}

func (s *CaptureState) Clear() {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.inflight = map[string]*NetworkTransaction{}
	s.completed = []*NetworkTransaction{}
	s.console = []*ConsoleEntry{}
	s.errors = []string{}
	s.started = time.Now()
	s.lastDrain = time.Time{}
	if s.sink != nil {
		s.sink.nextConsole = 0
		s.sink.nextNetwork = 0
	}
}

func (s *CaptureState) Snapshot(maxEntries int, includeConsole, includeNetwork bool) (console []*ConsoleEntry, network []*NetworkTransaction) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if maxEntries <= 0 {
		maxEntries = 10_000
	}

	if includeConsole {
		limit := min(maxEntries, len(s.console))
		console = append([]*ConsoleEntry(nil), s.console[len(s.console)-limit:]...)
	}
	if includeNetwork {
		limit := min(maxEntries, len(s.completed))
		network = append([]*NetworkTransaction(nil), s.completed[len(s.completed)-limit:]...)
	}
	return
}

func (s *CaptureState) Drain(sess *Session) {
	if sess == nil || sess.driver == nil {
		return
	}
	now := time.Now()
	s.mux.Lock()
	last := s.lastDrain
	s.mux.Unlock()
	if !last.IsZero() && now.Sub(last) < 100*time.Millisecond {
		return
	}
	s.mux.Lock()
	s.lastDrain = now
	s.mux.Unlock()

	s.drainConsole(sess.driver)
	s.drainPerformance(sess)
	_ = s.FlushSink()
}

func (s *CaptureState) drainConsole(driver any) {
	if !s.enableConsole {
		return
	}
	wd, ok := driver.(interface {
		Log(typ selog.Type) ([]selog.Message, error)
	})
	if !ok {
		return
	}
	messages, err := wd.Log(selog.Browser)
	if err != nil {
		s.appendErr(fmt.Sprintf("browser log: %v", err))
		return
	}
	if len(messages) == 0 {
		return
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	for _, msg := range messages {
		s.console = append(s.console, &ConsoleEntry{
			Timestamp: msg.Timestamp,
			Level:     string(msg.Level),
			Message:   msg.Message,
		})
	}
}

func (s *CaptureState) drainPerformance(sess *Session) {
	if !s.enableNetwork && !s.enableConsole {
		return
	}
	wd, ok := sess.driver.(interface {
		Log(typ selog.Type) ([]selog.Message, error)
		SessionID() string
	})
	if !ok {
		return
	}
	messages, err := wd.Log(selog.Performance)
	if err != nil {
		s.appendErr(fmt.Sprintf("performance log: %v", err))
		return
	}
	for _, msg := range messages {
		method, params, perr := parsePerformanceLogMessage(msg.Message)
		if perr != nil {
			s.appendErr(fmt.Sprintf("performance parse: %v", perr))
			continue
		}
		switch method {
		case "Network.requestWillBeSent":
			if !s.enableNetwork {
				continue
			}
			s.onRequestWillBeSent(params)
		case "Network.requestWillBeSentExtraInfo":
			if !s.enableNetwork {
				continue
			}
			s.onRequestExtraInfo(params)
		case "Network.responseReceived":
			if !s.enableNetwork {
				continue
			}
			s.onResponseReceived(params)
		case "Network.responseReceivedExtraInfo":
			if !s.enableNetwork {
				continue
			}
			s.onResponseExtraInfo(params)
		case "Network.loadingFinished":
			if !s.enableNetwork {
				continue
			}
			s.onLoadingFinished(sess, params)
		case "Network.loadingFailed":
			if !s.enableNetwork {
				continue
			}
			s.onLoadingFailed(params)
		case "Runtime.consoleAPICalled":
			s.onRuntimeConsole(params)
		case "Runtime.exceptionThrown":
			s.onRuntimeException(params)
		default:
			continue
		}
	}
}

func (s *CaptureState) appendErr(err string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.errors = append(s.errors, err)
}

func (s *CaptureState) onRequestWillBeSent(params json.RawMessage) {
	type request struct {
		URL      string         `json:"url"`
		Method   string         `json:"method"`
		Headers  map[string]any `json:"headers"`
		PostData string         `json:"postData,omitempty"`
	}
	type input struct {
		RequestID string         `json:"requestId"`
		Timestamp float64        `json:"timestamp"`
		Type      string         `json:"type,omitempty"`
		Initiator map[string]any `json:"initiator,omitempty"`
		Request   request        `json:"request"`
		Extra     map[string]any `json:"extra,omitempty"`
	}
	in := &input{}
	if err := json.Unmarshal(params, in); err != nil {
		s.appendErr(fmt.Sprintf("requestWillBeSent: %v", err))
		return
	}
	tx := &NetworkTransaction{
		RequestID:      in.RequestID,
		URL:            in.Request.URL,
		Method:         in.Request.Method,
		RequestHeaders: redactIfNeeded(in.Request.Headers, s.redact, s.redactHeaders),
		ResourceType:   in.Type,
		Initiator:      in.Initiator,
		StartTimestamp: in.Timestamp,
	}
	if s.includeBodies && in.Request.PostData != "" {
		tx.RequestBody = capBody(in.Request.PostData, false, s.maxBodyBytes)
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.inflight[in.RequestID] = tx
}

func (s *CaptureState) onRequestExtraInfo(params json.RawMessage) {
	type input struct {
		RequestID string         `json:"requestId"`
		Headers   map[string]any `json:"headers"`
	}
	in := &input{}
	if err := json.Unmarshal(params, in); err != nil {
		return
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	tx := s.inflight[in.RequestID]
	if tx == nil {
		return
	}
	if tx.RequestHeaders == nil {
		tx.RequestHeaders = map[string]any{}
	}
	for k, v := range redactIfNeeded(in.Headers, s.redact, s.redactHeaders) {
		tx.RequestHeaders[k] = v
	}
}

func (s *CaptureState) onResponseReceived(params json.RawMessage) {
	type response struct {
		Status     int            `json:"status"`
		StatusText string         `json:"statusText"`
		MimeType   string         `json:"mimeType"`
		Headers    map[string]any `json:"headers"`
	}
	type input struct {
		RequestID string   `json:"requestId"`
		Timestamp float64  `json:"timestamp"`
		Type      string   `json:"type,omitempty"`
		Response  response `json:"response"`
	}
	in := &input{}
	if err := json.Unmarshal(params, in); err != nil {
		return
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	tx := s.inflight[in.RequestID]
	if tx == nil {
		tx = &NetworkTransaction{RequestID: in.RequestID}
		s.inflight[in.RequestID] = tx
	}
	tx.Status = in.Response.Status
	tx.StatusText = in.Response.StatusText
	tx.MimeType = in.Response.MimeType
	tx.ResponseHeaders = redactIfNeeded(in.Response.Headers, s.redact, s.redactHeaders)
	if tx.ResourceType == "" {
		tx.ResourceType = in.Type
	}
}

func (s *CaptureState) onResponseExtraInfo(params json.RawMessage) {
	type input struct {
		RequestID string         `json:"requestId"`
		Headers   map[string]any `json:"headers"`
	}
	in := &input{}
	if err := json.Unmarshal(params, in); err != nil {
		return
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	tx := s.inflight[in.RequestID]
	if tx == nil {
		return
	}
	if tx.ResponseHeaders == nil {
		tx.ResponseHeaders = map[string]any{}
	}
	for k, v := range redactIfNeeded(in.Headers, s.redact, s.redactHeaders) {
		tx.ResponseHeaders[k] = v
	}
}

func (s *CaptureState) onLoadingFailed(params json.RawMessage) {
	type input struct {
		RequestID string  `json:"requestId"`
		Timestamp float64 `json:"timestamp"`
		ErrorText string  `json:"errorText"`
		Canceled  bool    `json:"canceled"`
	}
	in := &input{}
	if err := json.Unmarshal(params, in); err != nil {
		return
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	tx := s.inflight[in.RequestID]
	if tx == nil {
		tx = &NetworkTransaction{RequestID: in.RequestID}
		s.inflight[in.RequestID] = tx
	}
	tx.ErrorText = in.ErrorText
	tx.WasCanceled = in.Canceled
	tx.EndTimestamp = in.Timestamp
	s.finishLocked(in.RequestID, tx)
}

func (s *CaptureState) onLoadingFinished(sess *Session, params json.RawMessage) {
	type input struct {
		RequestID         string  `json:"requestId"`
		Timestamp         float64 `json:"timestamp"`
		EncodedDataLength float64 `json:"encodedDataLength"`
	}
	in := &input{}
	if err := json.Unmarshal(params, in); err != nil {
		return
	}

	var tx *NetworkTransaction
	s.mux.Lock()
	tx = s.inflight[in.RequestID]
	if tx == nil {
		tx = &NetworkTransaction{RequestID: in.RequestID}
		s.inflight[in.RequestID] = tx
	}
	tx.EndTimestamp = in.Timestamp
	tx.EncodedDataSize = int64(in.EncodedDataLength)
	start := tx.StartTimestamp
	end := tx.EndTimestamp
	s.mux.Unlock()

	if s.includeBodies {
		body, enc, truncated, berr := getResponseBody(sess, in.RequestID, s.maxBodyBytes)
		if berr != nil {
			s.appendErr(fmt.Sprintf("getResponseBody(%s): %v", in.RequestID, berr))
		} else if body != "" || enc != "" {
			s.mux.Lock()
			if tx.ResponseBody == nil {
				tx.ResponseBody = &CapturedBody{Encoding: enc, Data: body, Truncated: truncated}
			}
			s.mux.Unlock()
		}
	}

	s.mux.Lock()
	if start > 0 && end >= start {
		tx.DurationMs = int64((end - start) * 1000)
	}
	s.finishLocked(in.RequestID, tx)
	s.mux.Unlock()
}

func (s *CaptureState) finishLocked(requestID string, tx *NetworkTransaction) {
	if tx == nil {
		return
	}
	s.completed = append(s.completed, tx)
	delete(s.inflight, requestID)
}

func (s *CaptureState) onRuntimeConsole(params json.RawMessage) {
	if !s.enableConsole {
		return
	}
	type arg struct {
		Type        string `json:"type"`
		Value       any    `json:"value,omitempty"`
		Description string `json:"description,omitempty"`
	}
	type input struct {
		Type      string  `json:"type"`
		Timestamp float64 `json:"timestamp"`
		Args      []arg   `json:"args"`
	}
	in := &input{}
	if err := json.Unmarshal(params, in); err != nil {
		return
	}
	parts := make([]string, 0, len(in.Args))
	for _, a := range in.Args {
		if a.Description != "" {
			parts = append(parts, a.Description)
			continue
		}
		if a.Value != nil {
			parts = append(parts, fmt.Sprintf("%v", a.Value))
			continue
		}
		parts = append(parts, a.Type)
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.console = append(s.console, &ConsoleEntry{
		Timestamp: time.Now(),
		Level:     in.Type,
		Message:   strings.Join(parts, " "),
	})
}

func (s *CaptureState) onRuntimeException(params json.RawMessage) {
	if !s.enableConsole {
		return
	}
	type input struct {
		Timestamp float64 `json:"timestamp"`
		Details   struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	in := &input{}
	if err := json.Unmarshal(params, in); err != nil {
		return
	}
	msg := in.Details.Text
	if in.Details.Exception.Description != "" {
		msg = in.Details.Exception.Description
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.console = append(s.console, &ConsoleEntry{
		Timestamp: time.Now(),
		Level:     "exception",
		Message:   msg,
	})
}

func parsePerformanceLogMessage(raw string) (string, json.RawMessage, error) {
	// Chrome performance log entry is JSON, with "message" either object or string.
	var outer struct {
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &outer); err != nil {
		return "", nil, err
	}
	if len(outer.Message) == 0 {
		return "", nil, errors.New("missing message field")
	}

	var inner struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	// message can be a JSON string containing another JSON.
	if outer.Message[0] == '"' {
		var msgStr string
		if err := json.Unmarshal(outer.Message, &msgStr); err != nil {
			return "", nil, err
		}
		if err := json.Unmarshal([]byte(msgStr), &inner); err != nil {
			return "", nil, err
		}
		return inner.Method, inner.Params, nil
	}
	if err := json.Unmarshal(outer.Message, &inner); err != nil {
		return "", nil, err
	}
	return inner.Method, inner.Params, nil
}

func redactIfNeeded(headers map[string]any, redact bool, redactHeaders map[string]bool) map[string]any {
	if headers == nil {
		return nil
	}
	if !redact || len(redactHeaders) == 0 {
		dup := map[string]any{}
		for k, v := range headers {
			dup[k] = v
		}
		return dup
	}
	out := map[string]any{}
	for k, v := range headers {
		if redactHeaders[strings.ToLower(k)] {
			out[k] = "<redacted>"
			continue
		}
		out[k] = v
	}
	return out
}

func capBody(body string, base64Encoded bool, maxBytes int) *CapturedBody {
	if maxBytes <= 0 {
		maxBytes = 1_000_000
	}
	encoding := ""
	if base64Encoded {
		encoding = "base64"
	}
	truncated := false
	if len(body) > maxBytes {
		body = body[:maxBytes]
		truncated = true
	}
	return &CapturedBody{Encoding: encoding, Data: body, Truncated: truncated}
}

func getResponseBody(sess *Session, requestID string, maxBytes int) (data string, encoding string, truncated bool, err error) {
	if sess == nil || sess.driver == nil || sess.Remote == "" {
		return "", "", false, errors.New("missing session remote")
	}
	wdSession := sess.driver.SessionID()
	if wdSession == "" {
		return "", "", false, errors.New("missing webdriver session id")
	}

	type result struct {
		Body          string `json:"body"`
		Base64Encoded bool   `json:"base64Encoded"`
	}
	raw, err := cdpExecute(sess.Remote, wdSession, "Network.getResponseBody", map[string]any{"requestId": requestID})
	if err != nil {
		return "", "", false, err
	}
	out := &result{}
	if err := json.Unmarshal(raw, out); err != nil {
		return "", "", false, err
	}
	capped := capBody(out.Body, out.Base64Encoded, maxBytes)
	return capped.Data, capped.Encoding, capped.Truncated, nil
}

func cdpExecute(remote, wdSession, cmd string, params map[string]any) (json.RawMessage, error) {
	payload := map[string]any{"cmd": cmd, "params": params}
	// Try modern endpoint first.
	if raw, err := postW3C(remote, wdSession, "goog/cdp/execute", payload); err == nil {
		return raw, nil
	}
	// Fallback for older chromedriver.
	return postW3C(remote, wdSession, "chromium/send_command_and_get_result", payload)
}

func postW3C(remote, wdSession, endpoint string, payload map[string]any) (json.RawMessage, error) {
	u, err := url.Parse(remote)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "session", wdSession, endpoint)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cdp %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var decoded struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.Value) == 0 {
		decoded.Value = json.RawMessage(`{}`)
	}
	return decoded.Value, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type captureSink struct {
	url           string
	writer        io.WriteCloser
	flushInterval time.Duration
	lastSync      time.Time
	nextConsole   int
	nextNetwork   int
}

func (s *CaptureState) StartSink(fs afs.Service, sinkURL string, flushIntervalMs int) error {
	if sinkURL == "" {
		return nil
	}
	if fs == nil {
		fs = afs.New()
	}
	if flushIntervalMs <= 0 {
		flushIntervalMs = 500
	}
	w, err := fs.NewWriter(context.Background(), sinkURL, os.FileMode(0644))
	if err != nil {
		return err
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.sink != nil && s.sink.writer != nil {
		_ = s.sink.writer.Close()
	}
	s.sink = &captureSink{
		url:           sinkURL,
		writer:        w,
		flushInterval: time.Duration(flushIntervalMs) * time.Millisecond,
		lastSync:      time.Now(),
		nextConsole:   0,
		nextNetwork:   0,
	}
	return nil
}

func (s *CaptureState) CloseSink() error {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.sink == nil || s.sink.writer == nil {
		return nil
	}
	err := s.sink.writer.Close()
	s.sink.writer = nil
	return err
}

func (s *CaptureState) FlushSink() error {
	s.mux.Lock()
	sink := s.sink
	if sink == nil || sink.writer == nil {
		s.mux.Unlock()
		return nil
	}
	pendingConsole := append([]*ConsoleEntry(nil), s.console[sink.nextConsole:]...)
	pendingNetwork := append([]*NetworkTransaction(nil), s.completed[sink.nextNetwork:]...)
	s.mux.Unlock()

	if len(pendingConsole) == 0 && len(pendingNetwork) == 0 {
		return nil
	}

	writeLine := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := sink.writer.Write(append(b, '\n')); err != nil {
			return err
		}
		return nil
	}

	for _, entry := range pendingConsole {
		if err := writeLine(map[string]any{"type": "console", "entry": entry}); err != nil {
			s.appendErr(fmt.Sprintf("sink write console: %v", err))
			return err
		}
	}
	for _, tx := range pendingNetwork {
		if err := writeLine(map[string]any{"type": "network", "tx": tx}); err != nil {
			s.appendErr(fmt.Sprintf("sink write network: %v", err))
			return err
		}
	}

	s.mux.Lock()
	if s.sink != nil {
		s.sink.nextConsole += len(pendingConsole)
		s.sink.nextNetwork += len(pendingNetwork)
	}
	shouldSync := s.sink != nil && s.sink.flushInterval > 0 && time.Since(s.sink.lastSync) >= s.sink.flushInterval
	writer := io.WriteCloser(nil)
	if shouldSync && s.sink != nil {
		s.sink.lastSync = time.Now()
		writer = s.sink.writer
	}
	s.mux.Unlock()

	if shouldSync {
		if syncer, ok := writer.(interface{ Sync() error }); ok {
			_ = syncer.Sync()
		}
	}
	return nil
}
