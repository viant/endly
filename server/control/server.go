package control

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/viant/endly"
	managerservice "github.com/viant/endly/service/manager"
)

type Server struct {
	service       *managerservice.Service
	mux           *http.ServeMux
	authenticator Authenticator
}

func New(service *managerservice.Service) *Server {
	return NewAuthenticated(service, nil)
}

func NewAuthenticated(service *managerservice.Service, authenticator Authenticator) *Server {
	result := &Server{service: service, mux: http.NewServeMux(), authenticator: authenticator}
	result.mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})
	result.mux.HandleFunc("/readyz", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
	})
	result.mux.HandleFunc("/v1/endly/sessions", result.sessions)
	result.mux.HandleFunc("/v1/endly/sessions/", result.sessionResource)
	return result
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if s.authenticator != nil && strings.HasPrefix(request.URL.Path, "/v1/") {
		authorization := request.Header.Get("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") {
			writeError(writer, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}
		claims, err := s.authenticator.Authenticate(request.Context(), strings.TrimPrefix(authorization, "Bearer "))
		if err != nil {
			writeError(writer, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}
		request = request.WithContext(context.WithValue(request.Context(), principalContextKey{}, claims))
	}
	s.mux.ServeHTTP(writer, request)
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func Run(args []string, factory managerservice.ManagerFactory) error {
	flags := flag.NewFlagSet("endly serve", flag.ContinueOnError)
	address := flags.String("addr", "127.0.0.1:8080", "listen address")
	flags.StringVar(address, "a", "127.0.0.1:8080", "listen address")
	shutdownGrace := flags.Duration("shutdown-grace", 30*time.Second, "graceful shutdown timeout")
	allowRemote := flags.Bool("allow-remote", false, "allow a non-loopback listen address")
	jwtPublicKey := flags.String("jwt-public-key", "", "scy public-key resource, optionally URL|key")
	jwtIssuer := flags.String("jwt-issuer", "endly-client", "required JWT issuer")
	jwtAudience := flags.String("jwt-audience", "endly-service", "required JWT audience")
	jwtScope := flags.String("jwt-scope", "endly:execute", "required JWT scope")
	flags.StringVar(jwtPublicKey, "k", "", "scy public-key resource, optionally URL|key")
	flags.StringVar(jwtIssuer, "i", "endly-client", "required JWT issuer")
	flags.StringVar(jwtAudience, "d", "endly-service", "required JWT audience")
	flags.StringVar(jwtScope, "s", "endly:execute", "required JWT scope")
	maxSessions := flags.Int("max-sessions", 32, "maximum active sessions")
	sessionTTL := flags.Duration("session-ttl", 30*time.Minute, "idle session lifetime")
	operationRetention := flags.Duration("operation-retention", time.Hour, "terminal operation retention")
	maxEvents := flags.Int("max-events", 1000, "maximum retained events per operation")
	var callbackHosts stringList
	flags.Var(&callbackHosts, "allow-callback-host", "allowed callback hostname (repeatable)")
	var allowedActions stringList
	flags.Var(&allowedActions, "allow-action", "allowed service:action or service:* (repeatable)")
	allowPrivateCallbacks := flags.Bool("allow-private-callbacks", false, "allow callbacks to private/loopback addresses")
	if err := flags.Parse(args); err != nil {
		return err
	}
	var authenticator Authenticator
	if *jwtPublicKey != "" {
		jwtAuth, err := NewJWTAuthenticator(context.Background(), *jwtPublicKey, *jwtIssuer, *jwtAudience, *jwtScope)
		if err != nil {
			return err
		}
		authenticator = jwtAuth
	}
	if !isLoopbackAddress(*address) {
		if !*allowRemote {
			return errors.New("non-loopback address requires --allow-remote")
		}
		if authenticator == nil {
			return errors.New("non-loopback address requires --jwt-public-key")
		}
		if len(allowedActions) == 0 {
			return errors.New("non-loopback address requires at least one --allow-action")
		}
	}
	runtime := managerservice.New(factory,
		managerservice.WithMaxSessions(*maxSessions),
		managerservice.WithSessionTTL(*sessionTTL),
		managerservice.WithOperationRetention(*operationRetention),
		managerservice.WithMaxEvents(*maxEvents),
		managerservice.WithCallbackHosts(callbackHosts...),
		managerservice.WithPrivateCallbacks(*allowPrivateCallbacks),
		managerservice.WithAllowedActions(allowedActions...),
	)
	controlManager := factory()
	controlManager.Register(runtime)
	serverContext, stopJanitor := context.WithCancel(context.Background())
	defer stopJanitor()
	runtime.StartJanitor(serverContext, time.Minute)
	httpServer := &http.Server{Addr: *address, Handler: NewAuthenticated(runtime, authenticator)}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), *shutdownGrace)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
		_ = runtime.Shutdown(ctx)
	}()
	log.Printf("Endly service listening on %s", *address)
	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func isLoopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) sessions(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		result := s.service.Sessions()
		if principal := principalFromContext(request.Context()); principal != nil {
			filtered := result.Sessions[:0]
			for _, session := range result.Sessions {
				if session.Subject == principal.Subject {
					filtered = append(filtered, session)
				}
			}
			result.Sessions = filtered
		}
		writeJSON(writer, http.StatusOK, result)
		return
	}
	if request.Method != http.MethodPost {
		writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	input := &managerservice.OpenRequest{}
	if err := decodeJSON(request, input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if principal := principalFromContext(request.Context()); principal != nil {
		input.Subject = principal.Subject
	}
	result, err := s.service.Open(request.Context(), input)
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusCreated, result)
}

func (s *Server) sessionResource(writer http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/v1/endly/sessions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(writer, http.StatusNotFound, errors.New("session was missing"))
		return
	}
	sessionID := parts[0]
	if principal := principalFromContext(request.Context()); principal != nil {
		info, err := s.service.Session(sessionID)
		if err != nil {
			writeError(writer, http.StatusNotFound, err)
			return
		}
		if info.Subject != principal.Subject {
			writeError(writer, http.StatusForbidden, errors.New("session belongs to another subject"))
			return
		}
	}
	if len(parts) == 1 {
		s.session(writer, request, sessionID)
		return
	}
	switch parts[1] {
	case "workflows":
		s.workflows(writer, request, sessionID, parts[2:])
	case "operations":
		s.operations(writer, request, sessionID, parts[2:])
	case "context":
		s.inspectContext(writer, request, sessionID)
	case "logging":
		s.logging(writer, request, sessionID)
	default:
		writeError(writer, http.StatusNotFound, fmt.Errorf("unknown session resource %q", parts[1]))
	}
}

func (s *Server) session(writer http.ResponseWriter, request *http.Request, sessionID string) {
	switch request.Method {
	case http.MethodGet:
		result, err := s.service.Session(sessionID)
		if err != nil {
			writeError(writer, http.StatusNotFound, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	case http.MethodDelete:
		if err := s.service.Close(request.Context(), sessionID); err != nil {
			writeError(writer, http.StatusNotFound, err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	default:
		writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
	}
}

func (s *Server) workflows(writer http.ResponseWriter, request *http.Request, sessionID string, parts []string) {
	if len(parts) == 0 {
		switch request.Method {
		case http.MethodGet:
			result, err := s.service.ListWorkflows(sessionID)
			if err != nil {
				writeError(writer, http.StatusNotFound, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
		case http.MethodPost:
			input := &managerservice.LoadWorkflowRequest{SessionID: sessionID}
			if err := decodeJSON(request, input); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			input.SessionID = sessionID
			result, err := s.service.LoadWorkflow(request.Context(), input)
			if err != nil {
				status := http.StatusUnprocessableEntity
				if strings.Contains(err.Error(), "already") {
					status = http.StatusConflict
				}
				writeError(writer, status, err)
				return
			}
			writeJSON(writer, http.StatusCreated, result)
		default:
			writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		}
		return
	}
	if request.Method != http.MethodDelete {
		writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	force, _ := strconv.ParseBool(request.URL.Query().Get("force"))
	err := s.service.UnloadWorkflow(&managerservice.UnloadWorkflowRequest{SessionID: sessionID, Workflow: parts[0], Force: force})
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) operations(writer http.ResponseWriter, request *http.Request, sessionID string, parts []string) {
	if len(parts) == 0 {
		if request.Method == http.MethodGet {
			result, err := s.service.ListOperations(sessionID)
			if err != nil {
				writeError(writer, http.StatusNotFound, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
			return
		}
		if request.Method != http.MethodPost {
			writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}
		defer request.Body.Close()
		payload, err := io.ReadAll(io.LimitReader(request.Body, 8<<20))
		if err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		header := struct {
			Kind string `json:"kind"`
		}{}
		if err = json.Unmarshal(payload, &header); err != nil {
			writeError(writer, http.StatusBadRequest, fmt.Errorf("invalid JSON request: %w", err))
			return
		}
		var result *managerservice.Operation
		if header.Kind == "action" {
			envelope := &managerservice.StartOperationRequest{}
			if err = json.Unmarshal(payload, envelope); err == nil && envelope.Action != nil {
				envelope.Action.SessionID = sessionID
				result, err = s.service.StartAction(envelope.Action)
			} else if err == nil {
				err = errors.New("action operation payload was missing")
			}
		} else {
			input := &managerservice.RunWorkflowRequest{SessionID: sessionID}
			if err = json.Unmarshal(payload, input); err == nil {
				input.SessionID = sessionID
				result, err = s.service.StartWorkflow(input)
			}
		}
		if err != nil {
			writeError(writer, http.StatusUnprocessableEntity, err)
			return
		}
		writeJSON(writer, http.StatusAccepted, result)
		return
	}
	operationID := parts[0]
	if len(parts) > 1 && parts[1] == "events" {
		if request.Method != http.MethodGet {
			writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}
		result, err := s.service.GetOperation(sessionID, operationID)
		if err != nil {
			writeError(writer, http.StatusNotFound, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]interface{}{"sessionId": sessionID, "operationId": operationID, "events": result.Events})
		return
	}
	if len(parts) > 1 && parts[1] == "debug" {
		if len(parts) == 2 && request.Method == http.MethodGet {
			result, err := s.service.DebugState(sessionID, operationID)
			if err != nil {
				writeError(writer, http.StatusConflict, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
			return
		}
		if len(parts) == 3 && parts[2] == "commands" && request.Method == http.MethodPost {
			input := &managerservice.DebugCommandRequest{SessionID: sessionID, OperationID: operationID}
			if err := decodeJSON(request, input); err != nil {
				writeError(writer, http.StatusBadRequest, err)
				return
			}
			input.SessionID = sessionID
			input.OperationID = operationID
			result, err := s.service.DebugCommand(input)
			if err != nil {
				writeError(writer, http.StatusConflict, err)
				return
			}
			writeJSON(writer, http.StatusOK, result)
			return
		}
		writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	switch request.Method {
	case http.MethodGet:
		result, err := s.service.GetOperation(sessionID, operationID)
		if err != nil {
			writeError(writer, http.StatusNotFound, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	case http.MethodDelete:
		result, err := s.service.StopOperation(sessionID, operationID)
		if err != nil {
			writeError(writer, http.StatusNotFound, err)
			return
		}
		writeJSON(writer, http.StatusAccepted, result)
	default:
		writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
	}
}

func (s *Server) inspectContext(writer http.ResponseWriter, request *http.Request, sessionID string) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		return
	}
	full, _ := strconv.ParseBool(request.URL.Query().Get("full"))
	result, err := s.service.InspectContext(&managerservice.InspectContextRequest{SessionID: sessionID, OperationID: request.URL.Query().Get("operationId"), Full: full, Path: request.URL.Query().Get("path")})
	if err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if !result.Found {
		writeError(writer, http.StatusNotFound, errors.New("context path was not found"))
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) logging(writer http.ResponseWriter, request *http.Request, sessionID string) {
	switch request.Method {
	case http.MethodGet:
		result, err := s.service.Logging(sessionID)
		if err != nil {
			writeError(writer, http.StatusNotFound, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	case http.MethodPut:
		input := &managerservice.SetLoggingRequest{SessionID: sessionID}
		if err := decodeJSON(request, input); err != nil {
			writeError(writer, http.StatusBadRequest, err)
			return
		}
		input.SessionID = sessionID
		result, err := s.service.SetLogging(input)
		if err != nil {
			writeError(writer, http.StatusNotFound, err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	default:
		writeError(writer, http.StatusMethodNotAllowed, errors.New("method not allowed"))
	}
}

func decodeJSON(request *http.Request, target interface{}) error {
	defer request.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(request.Body, 8<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON request: %w", err)
	}
	return nil
}

func writeJSON(writer http.ResponseWriter, status int, value interface{}) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, err error) {
	writeJSON(writer, status, map[string]string{"error": err.Error()})
}

var _ endly.Service = (*managerservice.Service)(nil)
