package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/viant/afs"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/service/workflow"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"gopkg.in/yaml.v2"
)

type ManagerFactory func() endly.Manager

type Service struct {
	*endly.AbstractService
	factory               ManagerFactory
	fs                    afs.Service
	mu                    sync.RWMutex
	sessions              map[string]*Session
	maxSessions           int
	sessionTTL            time.Duration
	operationRetention    time.Duration
	maxEvents             int
	callbackHosts         map[string]bool
	allowPrivateCallbacks bool
	callbackClient        *http.Client
	allowedActions        map[string]bool
}

type Session struct {
	info       SessionInfo
	manager    endly.Manager
	context    *endly.Context
	workflow   *workflow.Service
	execMu     sync.Mutex
	mu         sync.RWMutex
	loaded     map[string]*LoadedWorkflow
	operations map[string]*Operation
	logging    bool
	closed     bool
	lastAccess time.Time
}

func New(factory ManagerFactory, options ...Option) *Service {
	result := &Service{
		AbstractService:    endly.NewAbstractService(ServiceID),
		factory:            factory,
		fs:                 afs.New(),
		sessions:           map[string]*Session{},
		maxSessions:        32,
		sessionTTL:         30 * time.Minute,
		operationRetention: time.Hour,
		maxEvents:          1000,
		callbackHosts:      map[string]bool{},
		callbackClient:     &http.Client{Timeout: 10 * time.Second},
		allowedActions:     map[string]bool{},
	}
	for _, option := range options {
		option(result)
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}

func (s *Service) Open(ctx context.Context, request *OpenRequest) (*SessionInfo, error) {
	if s.factory == nil {
		return nil, errors.New("manager factory was not configured")
	}
	s.mu.RLock()
	full := len(s.sessions) >= s.maxSessions
	s.mu.RUnlock()
	if full {
		return nil, fmt.Errorf("session limit %d was reached", s.maxSessions)
	}
	mgr := s.factory()
	if mgr == nil {
		return nil, errors.New("manager factory returned nil")
	}
	workflowService, err := mgr.Service(workflow.ServiceID)
	if err != nil {
		return nil, err
	}
	workflowRuntime, ok := workflowService.(*workflow.Service)
	if !ok {
		return nil, fmt.Errorf("unexpected workflow service type %T", workflowService)
	}
	id := uuid.NewString()
	endlyContext := mgr.NewContext(nil)
	endlyContext.SessionID = id
	endlyContext.CLIEnabled = true
	endlyContext.SetActionPolicy(s.authorizeAction)
	info := SessionInfo{SessionID: id, CreatedAt: time.Now().UTC()}
	if request != nil {
		info.Name = request.Name
		info.Subject = request.Subject
	}
	session := &Session{
		info:       info,
		manager:    mgr,
		context:    endlyContext,
		workflow:   workflowRuntime,
		loaded:     map[string]*LoadedWorkflow{},
		operations: map[string]*Operation{},
		logging:    true,
		lastAccess: info.CreatedAt,
	}
	s.mu.Lock()
	if len(s.sessions) >= s.maxSessions {
		s.mu.Unlock()
		endlyContext.Close()
		return nil, fmt.Errorf("session limit %d was reached", s.maxSessions)
	}
	s.sessions[id] = session
	s.mu.Unlock()
	copy := info
	return &copy, nil
}

func (s *Service) Close(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	if ok {
		delete(s.sessions, sessionID)
	}
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %q was not found", sessionID)
	}
	session.mu.Lock()
	session.closed = true
	for _, operation := range session.operations {
		if operation.cancel != nil && (operation.Status == OperationQueued || operation.Status == OperationRunning || operation.Status == OperationCancelling) {
			operation.Status = OperationCancelling
			operation.cancel()
		}
	}
	session.mu.Unlock()
	session.execMu.Lock()
	defer session.execMu.Unlock()
	session.context.Close()
	return nil
}

func (s *Service) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	s.mu.RUnlock()
	for _, id := range ids {
		if err := s.Close(ctx, id); err != nil && ctx.Err() == nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}

func (s *Service) Session(sessionID string) (*SessionInfo, error) {
	session, err := s.lookup(sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.RLock()
	defer session.mu.RUnlock()
	copy := session.info
	return &copy, nil
}

func (s *Service) Sessions() *ListSessionsResponse {
	s.mu.RLock()
	result := &ListSessionsResponse{Sessions: make([]*SessionInfo, 0, len(s.sessions))}
	for _, session := range s.sessions {
		session.mu.RLock()
		info := session.info
		session.mu.RUnlock()
		result.Sessions = append(result.Sessions, &info)
	}
	s.mu.RUnlock()
	sort.Slice(result.Sessions, func(i, j int) bool { return result.Sessions[i].CreatedAt.Before(result.Sessions[j].CreatedAt) })
	return result
}

func (s *Service) LoadWorkflow(ctx context.Context, request *LoadWorkflowRequest) (*LoadedWorkflow, error) {
	if request == nil || request.SessionID == "" {
		return nil, errors.New("sessionId was empty")
	}
	if request.URL == "" && request.Content == "" {
		return nil, errors.New("url or content was required")
	}
	session, err := s.lookup(request.SessionID)
	if err != nil {
		return nil, err
	}
	session.execMu.Lock()
	defer session.execMu.Unlock()

	workflowModel, source, err := s.decodeWorkflow(ctx, request)
	if err != nil {
		return nil, err
	}
	if workflowModel.AbstractNode == nil || workflowModel.Name == "" {
		return nil, errors.New("workflow name was empty")
	}
	workflowModel.Source = source
	if err = workflowModel.Validate(); err != nil {
		return nil, err
	}
	alias := request.Alias
	if alias == "" {
		alias = workflowModel.Name
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	if previous, exists := session.loaded[alias]; exists && !request.Replace {
		return nil, fmt.Errorf("workflow alias %q already refers to %q", alias, previous.Name)
	}
	for otherAlias, item := range session.loaded {
		if otherAlias != alias && item.Name == workflowModel.Name && !request.Replace {
			return nil, fmt.Errorf("workflow name %q is already loaded as %q", workflowModel.Name, otherAlias)
		}
	}
	if previous := session.loaded[alias]; previous != nil && previous.Name != workflowModel.Name {
		session.workflow.Unregister(previous.Name)
	}
	if err = session.workflow.Register(workflowModel); err != nil {
		return nil, err
	}
	loaded := &LoadedWorkflow{
		SessionID: session.info.SessionID,
		Alias:     alias,
		Name:      workflowModel.Name,
		Source:    source.URL,
		Tasks:     taskNames(workflowModel.Tasks),
	}
	session.loaded[alias] = loaded
	return cloneLoaded(loaded), nil
}

func (s *Service) ListWorkflows(sessionID string) (*ListWorkflowsResponse, error) {
	session, err := s.lookup(sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.RLock()
	result := &ListWorkflowsResponse{SessionID: sessionID, Workflows: make([]*LoadedWorkflow, 0, len(session.loaded))}
	for _, item := range session.loaded {
		result.Workflows = append(result.Workflows, cloneLoaded(item))
	}
	session.mu.RUnlock()
	sort.Slice(result.Workflows, func(i, j int) bool { return result.Workflows[i].Alias < result.Workflows[j].Alias })
	return result, nil
}

func (s *Service) UnloadWorkflow(request *UnloadWorkflowRequest) error {
	if request == nil || request.SessionID == "" || request.Workflow == "" {
		return errors.New("sessionId and workflow were required")
	}
	session, err := s.lookup(request.SessionID)
	if err != nil {
		return err
	}
	session.mu.Lock()
	alias, loaded := session.resolveLoaded(request.Workflow)
	if loaded == nil {
		session.mu.Unlock()
		return fmt.Errorf("workflow %q was not loaded", request.Workflow)
	}
	active := make([]string, 0)
	for _, operation := range session.operations {
		if operation.Workflow == alias && (operation.Status == OperationQueued || operation.Status == OperationRunning || operation.Status == OperationCancelling) {
			active = append(active, operation.ID)
		}
	}
	session.mu.Unlock()
	if len(active) > 0 && !request.Force {
		return fmt.Errorf("workflow %q has active operation %s", alias, active[0])
	}
	if request.Force {
		for _, operationID := range active {
			_, _ = s.StopOperation(request.SessionID, operationID)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, operationID := range active {
			if _, waitErr := s.WaitOperation(ctx, request.SessionID, operationID); waitErr != nil {
				return fmt.Errorf("stop operation %s before unload: %w", operationID, waitErr)
			}
		}
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	alias, loaded = session.resolveLoaded(request.Workflow)
	if loaded == nil {
		return fmt.Errorf("workflow %q was not loaded", request.Workflow)
	}
	delete(session.loaded, alias)
	session.workflow.Unregister(loaded.Name)
	return nil
}

func (s *Service) StartWorkflow(request *RunWorkflowRequest) (*Operation, error) {
	if request == nil || request.SessionID == "" || request.Workflow == "" {
		return nil, errors.New("sessionId and workflow were required")
	}
	session, err := s.lookup(request.SessionID)
	if err != nil {
		return nil, err
	}
	if err = s.validateCallbacks(request.Callbacks); err != nil {
		return nil, err
	}
	session.mu.Lock()
	alias, loaded := session.resolveLoaded(request.Workflow)
	if loaded == nil {
		session.mu.Unlock()
		return nil, fmt.Errorf("workflow %q was not loaded", request.Workflow)
	}
	now := time.Now().UTC()
	operation := &Operation{
		ID:        uuid.NewString(),
		SessionID: request.SessionID,
		Status:    OperationQueued,
		Kind:      "workflow",
		Workflow:  alias,
		Tasks:     request.Tasks,
		TagIDs:    request.TagIDs,
		CreatedAt: now,
		Events:    []*OperationEvent{},
	}
	if operation.Tasks == "" {
		operation.Tasks = "*"
	}
	session.operations[operation.ID] = operation
	session.mu.Unlock()

	requestCopy := *request
	requestCopy.Workflow = loaded.Name
	result := cloneOperation(operation)
	go s.executeWorkflow(session, operation.ID, &requestCopy)
	return result, nil
}

func (s *Service) StartAction(request *RunActionRequest) (*Operation, error) {
	if request == nil || request.SessionID == "" || request.Service == "" || request.Action == "" {
		return nil, errors.New("sessionId, service, and action were required")
	}
	session, err := s.lookup(request.SessionID)
	if err != nil {
		return nil, err
	}
	if _, err = session.manager.Service(request.Service); err != nil {
		return nil, err
	}
	if err = s.authorizeAction(request.Service, request.Action); err != nil {
		return nil, err
	}
	if err = s.validateCallbacks(request.Callbacks); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	operation := &Operation{
		ID:        uuid.NewString(),
		SessionID: request.SessionID,
		Status:    OperationQueued,
		Kind:      "action",
		Service:   request.Service,
		Action:    request.Action,
		CreatedAt: now,
		Events:    []*OperationEvent{},
	}
	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return nil, fmt.Errorf("session %q was closed", request.SessionID)
	}
	session.operations[operation.ID] = operation
	result := cloneOperation(operation)
	session.mu.Unlock()
	requestCopy := *request
	go s.executeAction(session, operation.ID, &requestCopy)
	return result, nil
}

func (s *Service) authorizeAction(service, action string) error {
	if len(s.allowedActions) == 0 || s.allowedActions["*"] || s.allowedActions[service+":"+action] || s.allowedActions[service+":*"] {
		return nil
	}
	return fmt.Errorf("action %s:%s was not allowed", service, action)
}

func (s *Service) GetOperation(sessionID, operationID string) (*Operation, error) {
	session, err := s.lookup(sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.RLock()
	defer session.mu.RUnlock()
	operation := session.operations[operationID]
	if operation == nil {
		return nil, fmt.Errorf("operation %q was not found", operationID)
	}
	return cloneOperation(operation), nil
}

func (s *Service) ListOperations(sessionID string) (*ListOperationsResponse, error) {
	session, err := s.lookup(sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.RLock()
	result := &ListOperationsResponse{SessionID: sessionID, Operations: make([]*Operation, 0, len(session.operations))}
	for _, operation := range session.operations {
		copy := cloneOperation(operation)
		copy.Events = nil
		result.Operations = append(result.Operations, copy)
	}
	session.mu.RUnlock()
	sort.Slice(result.Operations, func(i, j int) bool { return result.Operations[i].CreatedAt.Before(result.Operations[j].CreatedAt) })
	return result, nil
}

func (s *Service) StopOperation(sessionID, operationID string) (*Operation, error) {
	session, err := s.lookup(sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	operation := session.operations[operationID]
	if operation == nil {
		return nil, fmt.Errorf("operation %q was not found", operationID)
	}
	switch operation.Status {
	case OperationQueued:
		operation.Status = OperationCancelled
		now := time.Now().UTC()
		operation.FinishedAt = &now
	case OperationRunning, OperationCancelling:
		operation.Status = OperationCancelling
		if operation.cancel != nil {
			operation.cancel()
		}
	}
	return cloneOperation(operation), nil
}

func (s *Service) InspectContext(request *InspectContextRequest) (*ContextInspection, error) {
	if request == nil || request.SessionID == "" {
		return nil, errors.New("sessionId was empty")
	}
	if request.Full == (request.Path != "") {
		return nil, errors.New("exactly one of full or path was required")
	}
	if !validReadPath(request.Path) {
		return nil, errors.New("context path contained unsupported operators")
	}
	session, err := s.lookup(request.SessionID)
	if err != nil {
		return nil, err
	}
	if request.OperationID != "" {
		session.mu.RLock()
		operation := session.operations[request.OperationID]
		if operation == nil || operation.debugger == nil {
			session.mu.RUnlock()
			return nil, fmt.Errorf("debug operation %q was not found", request.OperationID)
		}
		debugState := operation.debugger.State()
		session.mu.RUnlock()
		if debugState.Status != "paused" || debugState.Snapshot == nil {
			return nil, fmt.Errorf("operation %q was not paused", request.OperationID)
		}
		snapshot := data.Map(toolbox.AsMap(debugState.Snapshot))
		if request.Full {
			return &ContextInspection{SessionID: request.SessionID, Found: true, Value: redactMap(snapshot.AsEncodableMap())}, nil
		}
		value, found := snapshot.GetValue(request.Path)
		return &ContextInspection{SessionID: request.SessionID, Path: request.Path, Found: found, Value: redactValue(request.Path, value)}, nil
	}
	session.execMu.Lock()
	defer session.execMu.Unlock()
	state := session.context.State()
	if request.Full {
		return &ContextInspection{SessionID: request.SessionID, Found: true, Value: redactMap(state.AsEncodableMap())}, nil
	}
	value, found := state.GetValue(request.Path)
	if !found {
		return &ContextInspection{SessionID: request.SessionID, Path: request.Path, Found: false}, nil
	}
	return &ContextInspection{SessionID: request.SessionID, Path: request.Path, Found: true, Value: redactValue(request.Path, value)}, nil
}

func (s *Service) DebugState(sessionID, operationID string) (*DebugState, error) {
	session, err := s.lookup(sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.RLock()
	operation := session.operations[operationID]
	if operation == nil || operation.debugger == nil {
		session.mu.RUnlock()
		return nil, fmt.Errorf("debug operation %q was not found", operationID)
	}
	state := operation.debugger.State()
	session.mu.RUnlock()
	state.Snapshot = nil
	return state, nil
}

func (s *Service) DebugCommand(request *DebugCommandRequest) (*DebugState, error) {
	if request == nil || request.SessionID == "" || request.OperationID == "" {
		return nil, errors.New("sessionId and operationId were required")
	}
	if request.Command == "stop" {
		if _, err := s.StopOperation(request.SessionID, request.OperationID); err != nil {
			return nil, err
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			state, err := s.DebugState(request.SessionID, request.OperationID)
			if err != nil {
				return nil, err
			}
			if state.Status == "stopped" || time.Now().After(deadline) {
				return state, nil
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	session, err := s.lookup(request.SessionID)
	if err != nil {
		return nil, err
	}
	session.mu.RLock()
	operation := session.operations[request.OperationID]
	if operation == nil || operation.debugger == nil {
		session.mu.RUnlock()
		return nil, fmt.Errorf("debug operation %q was not found", request.OperationID)
	}
	debugger := operation.debugger
	session.mu.RUnlock()
	before := debugger.State()
	switch request.Command {
	case "addBreakpoint":
		if request.Breakpoint == nil {
			return nil, errors.New("breakpoint was required")
		}
		debugger.SetBreakpoint(*request.Breakpoint)
	case "removeBreakpoint":
		if request.Breakpoint == nil {
			return nil, errors.New("breakpoint was required")
		}
		debugger.RemoveBreakpoint(*request.Breakpoint)
	default:
		if err := debugger.Control(request.Command); err != nil {
			return nil, err
		}
	}
	if request.Command == "addBreakpoint" || request.Command == "removeBreakpoint" {
		state := debugger.State()
		state.Snapshot = nil
		return state, nil
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		state := debugger.State()
		transitioned := false
		switch request.Command {
		case "step", "next":
			transitioned = state.Status == "paused" && (state.Point != before.Point || !sameTime(state.PausedAt, before.PausedAt))
		case "continue":
			transitioned = state.Status != "paused"
		case "pause":
			transitioned = state.Status == "paused"
		case "stop":
			transitioned = state.Status == "stopped"
		}
		if transitioned || time.Now().After(deadline) {
			state.Snapshot = nil
			return state, nil
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func (s *Service) SetLogging(request *SetLoggingRequest) (*LoggingState, error) {
	if request == nil || request.SessionID == "" {
		return nil, errors.New("sessionId was empty")
	}
	session, err := s.lookup(request.SessionID)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	session.logging = request.Enabled
	session.mu.Unlock()
	return &LoggingState{SessionID: request.SessionID, Enabled: request.Enabled}, nil
}

func (s *Service) Logging(sessionID string) (*LoggingState, error) {
	session, err := s.lookup(sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.RLock()
	enabled := session.logging
	session.mu.RUnlock()
	return &LoggingState{SessionID: sessionID, Enabled: enabled}, nil
}

func (s *Service) lookup(sessionID string) (*Session, error) {
	s.mu.RLock()
	session := s.sessions[sessionID]
	s.mu.RUnlock()
	if session == nil {
		return nil, fmt.Errorf("session %q was not found", sessionID)
	}
	session.mu.Lock()
	closed := session.closed
	if !closed {
		session.lastAccess = time.Now().UTC()
	}
	session.mu.Unlock()
	if closed {
		return nil, fmt.Errorf("session %q was closed", sessionID)
	}
	return session, nil
}

func (s *Service) decodeWorkflow(ctx context.Context, request *LoadWorkflowRequest) (*model.Workflow, *location.Resource, error) {
	sourceURL := request.URL
	if sourceURL == "" {
		name := request.Alias
		if name == "" {
			name = "workflow"
		}
		ext := request.Format
		if ext == "" {
			ext = "yaml"
		}
		sourceURL = "mem://endly/manager/" + name + "." + strings.TrimPrefix(ext, ".")
	}
	resource := location.NewResource(sourceURL)
	runRequest := &workflow.RunRequest{}
	if request.Content != "" {
		format := strings.ToLower(request.Format)
		if format == "" {
			format = strings.TrimPrefix(strings.ToLower(filepath.Ext(sourceURL)), ".")
		}
		if format == "yaml" || format == "yml" {
			ordered := yaml.MapSlice{}
			if err := toolbox.NewYamlDecoderFactory().Create(strings.NewReader(request.Content)).Decode(&ordered); err != nil {
				return nil, nil, fmt.Errorf("failed to decode workflow content: %w", err)
			}
			if err := toolbox.DefaultConverter.AssignConverted(runRequest, ordered); err != nil {
				return nil, nil, fmt.Errorf("failed to convert workflow content: %w", err)
			}
		} else if err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(request.Content)).Decode(runRequest); err != nil {
			return nil, nil, fmt.Errorf("failed to decode workflow content: %w", err)
		}
	} else {
		var err error
		if extension := strings.ToLower(filepath.Ext(sourceURL)); extension == ".yaml" || extension == ".yml" {
			err = resource.YAMLDecode(ctx, s.fs, runRequest)
		} else {
			err = resource.DecodeWith(ctx, s.fs, runRequest, resource.DecoderFactory())
		}
		if err != nil {
			return nil, nil, err
		}
	}
	runRequest.Source = resource
	runRequest.AssetURL = sourceURL
	name := request.Name
	if name == "" {
		name = model.WorkflowSelector(sourceURL).Name()
	}
	runRequest.Name = name
	if err := runRequest.Init(); err != nil {
		return nil, nil, err
	}
	if runRequest.Inlined == nil {
		return nil, nil, errors.New("workflow pipeline was empty")
	}
	baseURL, _ := toolbox.URLSplit(sourceURL)
	result, err := runRequest.AsWorkflow(name, baseURL)
	if err != nil {
		return nil, nil, err
	}
	result.Source = resource
	return result, resource, nil
}

func taskNames(tasks model.Tasks) []string {
	result := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		result = append(result, task.Name)
	}
	return result
}

func (s *Session) resolveLoaded(candidate string) (string, *LoadedWorkflow) {
	if loaded := s.loaded[candidate]; loaded != nil {
		return candidate, loaded
	}
	for alias, loaded := range s.loaded {
		if loaded.Name == candidate {
			return alias, loaded
		}
	}
	return "", nil
}

func cloneLoaded(source *LoadedWorkflow) *LoadedWorkflow {
	if source == nil {
		return nil
	}
	copy := *source
	copy.Tasks = append([]string(nil), source.Tasks...)
	return &copy
}

func cloneOperation(source *Operation) *Operation {
	if source == nil {
		return nil
	}
	copy := &Operation{
		ID:            source.ID,
		SessionID:     source.SessionID,
		Status:        source.Status,
		Kind:          source.Kind,
		Workflow:      source.Workflow,
		Service:       source.Service,
		Action:        source.Action,
		Tasks:         source.Tasks,
		TagIDs:        source.TagIDs,
		CreatedAt:     source.CreatedAt,
		StartedAt:     source.StartedAt,
		FinishedAt:    source.FinishedAt,
		Error:         source.Error,
		DroppedEvents: atomic.LoadInt64(&source.droppedEvents),
	}
	if source.debugger != nil {
		copy.Debug = source.debugger.State()
		if copy.Debug.Status == "paused" && copy.Status == OperationRunning {
			copy.Status = "paused"
		}
	}
	copy.Events = append([]*OperationEvent(nil), source.Events...)
	if source.Result != nil {
		copy.Result = map[string]interface{}{}
		for key, value := range source.Result {
			copy.Result[key] = value
		}
	}
	return copy
}

func validReadPath(path string) bool {
	if path == "" {
		return true
	}
	return !strings.ContainsAny(path, "$+<>(){}")
}

func redactMap(source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(source))
	for key, value := range source {
		if isSecretKey(key) {
			result[key] = "[REDACTED]"
			continue
		}
		if nested, ok := value.(map[string]interface{}); ok {
			result[key] = redactMap(nested)
			continue
		}
		result[key] = value
	}
	return result
}

func redactValue(path string, value interface{}) interface{} {
	fragments := strings.FieldsFunc(path, func(r rune) bool { return r == '.' || r == '[' || r == ']' })
	for _, fragment := range fragments {
		if isSecretKey(fragment) {
			return "[REDACTED]"
		}
	}
	if nested, ok := value.(map[string]interface{}); ok {
		return redactMap(nested)
	}
	return value
}

func isSecretKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "secret") || strings.Contains(key, "password") || strings.Contains(key, "token") || strings.Contains(key, "credential")
}

func encodable(value interface{}) interface{} {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	var result interface{}
	if err = json.Unmarshal(data, &result); err != nil {
		return fmt.Sprintf("%v", value)
	}
	return result
}

func callbackEventName(event msg.Event) string {
	switch value := event.Value().(type) {
	case *workflow.WorkflowStartEvent:
		return "workflow.started"
	case *workflow.WorkflowEndEvent:
		if value.Status == "error" {
			return "workflow.failed"
		}
		return "workflow.completed"
	case *model.TaskStartEvent:
		return "task.started"
	case *model.TaskEndEvent:
		if value.Status == "error" {
			return "task.failed"
		}
		return "task.completed"
	case *model.Activity:
		return "action.started"
	case *model.ActivityEndEvent:
		return "action.completed"
	default:
		return event.Type()
	}
}

func callbackMatches(callback Callback, eventName string) bool {
	if len(callback.Events) == 0 {
		return eventName == "task.completed" || eventName == "task.failed" || eventName == "workflow.completed" || eventName == "workflow.failed"
	}
	for _, candidate := range callback.Events {
		if candidate == eventName {
			return true
		}
	}
	return false
}

func (s *Service) deliverCallback(callback Callback, operationID, sessionID string, event *OperationEvent) {
	parsed, err := url.Parse(callback.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return
	}
	payload, err := json.Marshal(map[string]interface{}{
		"operationId": operationID,
		"sessionId":   sessionID,
		"event":       event,
	})
	if err != nil {
		return
	}
	for attempt := 0; attempt < 3; attempt++ {
		request, requestErr := http.NewRequest(http.MethodPost, callback.URL, strings.NewReader(string(payload)))
		if requestErr != nil {
			return
		}
		request.Header.Set("Content-Type", "application/json")
		for key, value := range callback.Headers {
			request.Header.Set(key, value)
		}
		response, requestErr := s.callbackClient.Do(request)
		if requestErr == nil {
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return
			}
		}
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
}
