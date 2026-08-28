package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/viant/endly/internal/debug"
	managerservice "github.com/viant/endly/service/manager"
	"golang.org/x/term"
)

type Client struct {
	Endpoint      string
	SessionID     string
	TokenProvider TokenProvider
	HTTP          *http.Client
	Output        io.Writer
	ProfileName   string
	Profiles      *ProfileStore
	ProfilePath   string
	Color         bool
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func Run(args []string) error {
	global := flag.NewFlagSet("endly client", flag.ContinueOnError)
	endpoint := global.String("endpoint", os.Getenv("ENDLY_ENDPOINT"), "Endly service endpoint")
	sessionID := global.String("session", os.Getenv("ENDLY_SESSION_ID"), "Endly session ID")
	jwtPrivateKey := global.String("jwt-private-key", os.Getenv("ENDLY_JWT_PRIVATE_KEY"), "scy private-key resource, optionally URL|key")
	jwtIssuer := global.String("jwt-issuer", os.Getenv("ENDLY_JWT_ISSUER"), "JWT issuer")
	jwtAudience := global.String("jwt-audience", os.Getenv("ENDLY_JWT_AUDIENCE"), "JWT audience")
	jwtScope := global.String("jwt-scope", os.Getenv("ENDLY_JWT_SCOPE"), "JWT scope")
	jwtSubject := global.String("jwt-subject", os.Getenv("ENDLY_JWT_SUBJECT"), "JWT subject")
	jwtTTL := global.Duration("jwt-ttl", 5*time.Minute, "JWT lifetime")
	profileName := global.String("profile", envOr("ENDLY_PROFILE", "default"), "client profile")
	noColor := global.Bool("no-color", false, "disable ANSI terminal colors")
	global.StringVar(endpoint, "e", os.Getenv("ENDLY_ENDPOINT"), "Endly service endpoint")
	global.StringVar(sessionID, "s", os.Getenv("ENDLY_SESSION_ID"), "Endly session ID")
	global.StringVar(jwtPrivateKey, "k", os.Getenv("ENDLY_JWT_PRIVATE_KEY"), "scy private-key resource, optionally URL|key")
	global.StringVar(jwtSubject, "u", os.Getenv("ENDLY_JWT_SUBJECT"), "JWT subject")
	global.StringVar(profileName, "p", envOr("ENDLY_PROFILE", "default"), "client profile")
	if err := global.Parse(args); err != nil {
		return err
	}
	remaining := global.Args()
	if len(remaining) == 0 {
		return errors.New("client command was required: configure|open|sessions|load|list|run|action|operations|status|logs|stop|unload|context|logging|debug|close")
	}
	profilesPath := profilePath()
	profiles, err := loadProfiles(profilesPath)
	if err != nil {
		return fmt.Errorf("load client profiles: %w", err)
	}
	profile := profiles.Profiles[*profileName]
	if profile == nil {
		profile = &Profile{}
	}
	if *endpoint == "" {
		*endpoint = profile.Endpoint
	}
	if *endpoint == "" {
		*endpoint = "http://127.0.0.1:8080"
	}
	if *sessionID == "" {
		*sessionID = profile.SessionID
	}
	if *jwtPrivateKey == "" {
		*jwtPrivateKey = profile.JWTPrivateKey
	}
	if *jwtIssuer == "" {
		*jwtIssuer = profile.JWTIssuer
	}
	if *jwtIssuer == "" {
		*jwtIssuer = "endly-client"
	}
	if *jwtAudience == "" {
		*jwtAudience = profile.JWTAudience
	}
	if *jwtAudience == "" {
		*jwtAudience = "endly-service"
	}
	if *jwtScope == "" {
		*jwtScope = profile.JWTScope
	}
	if *jwtScope == "" {
		*jwtScope = "endly:execute"
	}
	if *jwtSubject == "" {
		*jwtSubject = profile.JWTSubject
	}
	if *jwtSubject == "" {
		*jwtSubject = os.Getenv("USER")
	}
	var tokenProvider TokenProvider
	if *jwtPrivateKey != "" {
		tokenProvider, err = NewJWTTokenProvider(context.Background(), *jwtPrivateKey, *jwtIssuer, *jwtAudience, *jwtScope, *jwtSubject, *jwtTTL)
		if err != nil {
			return err
		}
	}
	color := !*noColor && os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))
	cli := &Client{Endpoint: strings.TrimRight(*endpoint, "/"), SessionID: *sessionID, TokenProvider: tokenProvider, HTTP: &http.Client{Timeout: 30 * time.Second}, Output: os.Stdout, ProfileName: *profileName, Profiles: profiles, ProfilePath: profilesPath, Color: color}
	return cli.run(remaining[0], remaining[1:])
}

func (c *Client) run(command string, args []string) error {
	switch command {
	case "open":
		return c.open(args)
	case "sessions":
		return c.sessions()
	case "load":
		return c.load(args)
	case "list":
		return c.list()
	case "run":
		return c.runWorkflow(args)
	case "action":
		return c.runAction(args)
	case "status":
		return c.status(args)
	case "operations":
		return c.operations()
	case "logs":
		return c.logs(args)
	case "stop":
		return c.stop(args)
	case "unload":
		return c.unload(args)
	case "context":
		return c.inspect(args)
	case "logging":
		return c.logging(args)
	case "debug":
		return c.debug(args)
	case "close":
		return c.close()
	case "configure":
		return c.configure(args)
	default:
		return fmt.Errorf("unknown client command %q", command)
	}
}

func (c *Client) sessions() error {
	result := &managerservice.ListSessionsResponse{}
	if err := c.do(context.Background(), http.MethodGet, "/v1/endly/sessions", nil, result); err != nil {
		return err
	}
	return c.print(result)
}

func (c *Client) operations() error {
	if err := c.requireSession(); err != nil {
		return err
	}
	result := &managerservice.ListOperationsResponse{}
	path := fmt.Sprintf("/v1/endly/sessions/%s/operations", url.PathEscape(c.SessionID))
	if err := c.do(context.Background(), http.MethodGet, path, nil, result); err != nil {
		return err
	}
	return c.print(result)
}

func (c *Client) runAction(args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return errors.New("action requires service:action before its options")
	}
	selector := strings.SplitN(args[0], ":", 2)
	if len(selector) != 2 || selector[0] == "" || selector[1] == "" {
		return errors.New("action selector must use service:action")
	}
	flags := flag.NewFlagSet("client action", flag.ContinueOnError)
	requestFile := flags.String("request", "", "JSON action request file")
	detach := flags.Bool("detach", false, "return immediately")
	timeout := flags.Duration("timeout", 0, "operation timeout")
	format := flags.String("format", "ui", "output format: ui or json")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if err := c.requireSession(); err != nil {
		return err
	}
	actionInput := &managerservice.RunActionRequest{Service: selector[0], Action: selector[1], TimeoutMillis: timeout.Milliseconds()}
	if *requestFile != "" {
		payload, err := os.ReadFile(*requestFile)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(payload, &actionInput.Request); err != nil {
			return fmt.Errorf("invalid action request JSON: %w", err)
		}
	}
	envelope := &managerservice.StartOperationRequest{Kind: "action", Action: actionInput}
	operation := &managerservice.Operation{}
	path := fmt.Sprintf("/v1/endly/sessions/%s/operations", url.PathEscape(c.SessionID))
	if err := c.do(context.Background(), http.MethodPost, path, envelope, operation); err != nil {
		return err
	}
	if *detach {
		return c.print(operation)
	}
	return c.wait(operation.ID, *format)
}

func (c *Client) open(args []string) error {
	flags := flag.NewFlagSet("client open", flag.ContinueOnError)
	name := flags.String("name", "", "session name")
	noSave := flags.Bool("no-save", false, "do not save the session to the profile")
	if err := flags.Parse(args); err != nil {
		return err
	}
	result := &managerservice.SessionInfo{}
	if err := c.do(context.Background(), http.MethodPost, "/v1/endly/sessions", &managerservice.OpenRequest{Name: *name}, result); err != nil {
		return err
	}
	c.SessionID = result.SessionID
	if !*noSave {
		if err := c.saveProfile(); err != nil {
			return err
		}
	}
	return c.print(result)
}

func (c *Client) load(args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return errors.New("load requires a workflow URL or file before its options")
	}
	source := args[0]
	flags := flag.NewFlagSet("client load", flag.ContinueOnError)
	alias := flags.String("alias", "", "workflow alias")
	name := flags.String("name", "", "workflow name override")
	replace := flags.Bool("replace", false, "replace existing workflow")
	newSession := flags.Bool("new-session", false, "open a new session before loading")
	noSave := flags.Bool("no-save", false, "do not save a new session to the profile")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected load arguments: %s", strings.Join(flags.Args(), " "))
	}
	if *newSession {
		if c.SessionID != "" {
			return errors.New("--new-session cannot be combined with an existing --session")
		}
		opened := &managerservice.SessionInfo{}
		if err := c.do(context.Background(), http.MethodPost, "/v1/endly/sessions", &managerservice.OpenRequest{}, opened); err != nil {
			return err
		}
		c.SessionID = opened.SessionID
		if !*noSave {
			if err := c.saveProfile(); err != nil {
				return err
			}
		}
	}
	if err := c.requireSession(); err != nil {
		return err
	}
	input := &managerservice.LoadWorkflowRequest{URL: source, Name: *name, Alias: *alias, Replace: *replace}
	if info, err := os.Stat(source); err == nil && !info.IsDir() {
		content, readErr := os.ReadFile(source)
		if readErr != nil {
			return readErr
		}
		absolute, _ := filepath.Abs(source)
		input.URL = (&url.URL{Scheme: "file", Path: absolute}).String()
		input.Content = string(content)
		input.Format = strings.TrimPrefix(filepath.Ext(source), ".")
	}
	result := &managerservice.LoadedWorkflow{}
	path := fmt.Sprintf("/v1/endly/sessions/%s/workflows", url.PathEscape(c.SessionID))
	if err := c.do(context.Background(), http.MethodPost, path, input, result); err != nil {
		return err
	}
	return c.print(result)
}

func (c *Client) list() error {
	if err := c.requireSession(); err != nil {
		return err
	}
	result := &managerservice.ListWorkflowsResponse{}
	if err := c.do(context.Background(), http.MethodGet, fmt.Sprintf("/v1/endly/sessions/%s/workflows", url.PathEscape(c.SessionID)), nil, result); err != nil {
		return err
	}
	return c.print(result)
}

func (c *Client) runWorkflow(args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return errors.New("run requires a loaded workflow alias before its options")
	}
	workflowAlias := args[0]
	flags := flag.NewFlagSet("client run", flag.ContinueOnError)
	tasks := flags.String("tasks", "*", "comma-separated tasks or *")
	tagIDs := flags.String("tag-ids", "", "comma-separated action tag IDs")
	detach := flags.Bool("detach", false, "return immediately")
	debugMode := flags.Bool("debug", false, "start in step-debug mode")
	format := flags.String("format", "ui", "output format: ui or json")
	timeout := flags.Duration("timeout", 0, "operation timeout")
	paramsFile := flags.String("params", "", "JSON parameters file")
	sharedState := flags.String("shared-state", "", "share session state: true or false")
	var callbacks stringList
	var callbackEvents stringList
	var params stringList
	flags.Var(&callbacks, "callback", "callback URL (repeatable)")
	flags.Var(&callbackEvents, "callback-event", "callback event (repeatable)")
	flags.Var(&params, "param", "workflow parameter key=value (repeatable)")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if err := c.requireSession(); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected run arguments: %s", strings.Join(flags.Args(), " "))
	}
	input := &managerservice.RunWorkflowRequest{Workflow: workflowAlias, Tasks: *tasks, TagIDs: *tagIDs, TimeoutMillis: timeout.Milliseconds(), Debug: *debugMode}
	if *paramsFile != "" {
		data, err := os.ReadFile(*paramsFile)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(data, &input.Params); err != nil {
			return fmt.Errorf("invalid params JSON: %w", err)
		}
	}
	if input.Params == nil {
		input.Params = map[string]interface{}{}
	}
	for _, pair := range params {
		fragments := strings.SplitN(pair, "=", 2)
		if len(fragments) != 2 || fragments[0] == "" {
			return fmt.Errorf("invalid --param %q, expected key=value", pair)
		}
		var value interface{}
		if err := json.Unmarshal([]byte(fragments[1]), &value); err != nil {
			value = fragments[1]
		}
		input.Params[fragments[0]] = value
	}
	if *sharedState != "" {
		value, err := strconv.ParseBool(*sharedState)
		if err != nil {
			return fmt.Errorf("invalid --shared-state: %w", err)
		}
		input.SharedState = &value
	}
	for _, callbackURL := range callbacks {
		input.Callbacks = append(input.Callbacks, managerservice.Callback{URL: callbackURL, Events: append([]string(nil), callbackEvents...)})
	}
	operation := &managerservice.Operation{}
	path := fmt.Sprintf("/v1/endly/sessions/%s/operations", url.PathEscape(c.SessionID))
	if err := c.do(context.Background(), http.MethodPost, path, input, operation); err != nil {
		return err
	}
	if *detach || *debugMode {
		return c.print(operation)
	}
	return c.wait(operation.ID, *format)
}

func (c *Client) wait(operationID, format string) error {
	lastSequence := int64(0)
	listener := c.newEventListener()
	for {
		operation, err := c.getOperation(operationID)
		if err != nil {
			return err
		}
		if format != "json" {
			lastSequence = c.renderEvents(operation.Events, lastSequence, listener)
		}
		switch operation.Status {
		case managerservice.OperationSucceeded:
			if format == "json" {
				return c.print(operation)
			}
			c.renderSummary(operation)
			return nil
		case managerservice.OperationFailed, managerservice.OperationCancelled:
			if format == "json" {
				_ = c.print(operation)
			} else {
				c.renderSummary(operation)
			}
			return fmt.Errorf("operation ended with status %s", operation.Status)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *Client) status(args []string) error {
	if len(args) != 1 {
		return errors.New("status requires an operation ID")
	}
	operation, err := c.getOperation(args[0])
	if err != nil {
		return err
	}
	return c.print(operation)
}

func (c *Client) logs(args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return errors.New("logs requires an operation ID before its options")
	}
	operationID := args[0]
	flags := flag.NewFlagSet("client logs", flag.ContinueOnError)
	follow := flags.Bool("follow", false, "follow events until the operation finishes")
	format := flags.String("format", "ui", "output format: ui or json")
	since := flags.Int64("since", 0, "first event sequence to return")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if err := c.requireSession(); err != nil {
		return err
	}
	last := *since
	listener := c.newEventListener()
	for {
		operation, err := c.getOperation(operationID)
		if err != nil {
			return err
		}
		if *format == "json" && !*follow {
			return c.print(map[string]interface{}{"sessionId": c.SessionID, "operationId": operationID, "events": operation.Events})
		}
		last = c.renderEvents(operation.Events, last, listener)
		if !*follow || isTerminal(operation.Status) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *Client) stop(args []string) error {
	if len(args) != 1 {
		return errors.New("stop requires an operation ID")
	}
	if err := c.requireSession(); err != nil {
		return err
	}
	result := &managerservice.Operation{}
	path := fmt.Sprintf("/v1/endly/sessions/%s/operations/%s", url.PathEscape(c.SessionID), url.PathEscape(args[0]))
	if err := c.do(context.Background(), http.MethodDelete, path, nil, result); err != nil {
		return err
	}
	return c.print(result)
}

func (c *Client) unload(args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return errors.New("unload requires a workflow alias before its options")
	}
	workflowAlias := args[0]
	flags := flag.NewFlagSet("client unload", flag.ContinueOnError)
	force := flags.Bool("force", false, "request cancellation of active operations")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if err := c.requireSession(); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected unload arguments: %s", strings.Join(flags.Args(), " "))
	}
	path := fmt.Sprintf("/v1/endly/sessions/%s/workflows/%s?force=%t", url.PathEscape(c.SessionID), url.PathEscape(workflowAlias), *force)
	return c.do(context.Background(), http.MethodDelete, path, nil, nil)
}

func (c *Client) inspect(args []string) error {
	flags := flag.NewFlagSet("client context", flag.ContinueOnError)
	full := flags.Bool("full", false, "return full context")
	pathValue := flags.String("path", "", "return one context path")
	operationID := flags.String("operation", "", "inspect a paused operation snapshot")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if err := c.requireSession(); err != nil {
		return err
	}
	query := url.Values{}
	if *full {
		query.Set("full", "true")
	}
	if *pathValue != "" {
		query.Set("path", *pathValue)
	}
	if *operationID != "" {
		query.Set("operationId", *operationID)
	}
	result := &managerservice.ContextInspection{}
	requestPath := fmt.Sprintf("/v1/endly/sessions/%s/context?%s", url.PathEscape(c.SessionID), query.Encode())
	if err := c.do(context.Background(), http.MethodGet, requestPath, nil, result); err != nil {
		return err
	}
	return c.print(result)
}

func (c *Client) debug(args []string) error {
	if err := c.requireSession(); err != nil {
		return err
	}
	if len(args) < 2 {
		return errors.New("debug requires an operation ID and status|pause|step|next|continue|stop")
	}
	operationID, command := args[0], args[1]
	basePath := fmt.Sprintf("/v1/endly/sessions/%s/operations/%s/debug", url.PathEscape(c.SessionID), url.PathEscape(operationID))
	result := &managerservice.DebugState{}
	if command == "status" {
		if err := c.do(context.Background(), http.MethodGet, basePath, nil, result); err != nil {
			return err
		}
		return c.print(result)
	}
	input := &managerservice.DebugCommandRequest{Command: command}
	if command == "breakpoint" {
		if len(args) < 3 || (args[2] != "add" && args[2] != "remove") {
			return errors.New("debug breakpoint requires add or remove")
		}
		flags := flag.NewFlagSet("client debug breakpoint", flag.ContinueOnError)
		workflowName := flags.String("workflow", "", "workflow name")
		taskName := flags.String("task", "", "task name")
		actionName := flags.String("action", "", "action name")
		tagID := flags.String("tag-id", "", "action tag ID")
		if err := flags.Parse(args[3:]); err != nil {
			return err
		}
		input.Command = map[bool]string{true: "addBreakpoint", false: "removeBreakpoint"}[args[2] == "add"]
		input.Breakpoint = &debug.Step{Workflow: *workflowName, TaskName: *taskName, Action: *actionName, TagID: *tagID}
	}
	if err := c.do(context.Background(), http.MethodPost, basePath+"/commands", input, result); err != nil {
		return err
	}
	return c.print(result)
}

func (c *Client) logging(args []string) error {
	if err := c.requireSession(); err != nil {
		return err
	}
	if len(args) != 1 {
		return errors.New("logging requires enable, disable, or status")
	}
	path := fmt.Sprintf("/v1/endly/sessions/%s/logging", url.PathEscape(c.SessionID))
	result := &managerservice.LoggingState{}
	if args[0] == "status" {
		if err := c.do(context.Background(), http.MethodGet, path, nil, result); err != nil {
			return err
		}
		return c.print(result)
	}
	if args[0] != "enable" && args[0] != "disable" {
		return errors.New("logging requires enable, disable, or status")
	}
	if err := c.do(context.Background(), http.MethodPut, path, &managerservice.SetLoggingRequest{Enabled: args[0] == "enable"}, result); err != nil {
		return err
	}
	return c.print(result)
}

func (c *Client) close() error {
	if err := c.requireSession(); err != nil {
		return err
	}
	if err := c.do(context.Background(), http.MethodDelete, fmt.Sprintf("/v1/endly/sessions/%s", url.PathEscape(c.SessionID)), nil, nil); err != nil {
		return err
	}
	c.SessionID = ""
	return c.saveProfile()
}

func (c *Client) configure(args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return errors.New("configure requires a profile name")
	}
	name := args[0]
	flags := flag.NewFlagSet("client configure", flag.ContinueOnError)
	endpoint := flags.String("endpoint", "", "Endly service endpoint")
	jwtPrivateKey := flags.String("jwt-private-key", "", "scy private-key resource")
	jwtIssuer := flags.String("jwt-issuer", "endly-client", "JWT issuer")
	jwtAudience := flags.String("jwt-audience", "endly-service", "JWT audience")
	jwtScope := flags.String("jwt-scope", "endly:execute", "JWT scope")
	jwtSubject := flags.String("jwt-subject", os.Getenv("USER"), "JWT subject")
	flags.StringVar(endpoint, "e", "", "Endly service endpoint")
	flags.StringVar(jwtPrivateKey, "k", "", "scy private-key resource")
	flags.StringVar(jwtSubject, "u", os.Getenv("USER"), "JWT subject")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if *endpoint == "" {
		return errors.New("configure requires --endpoint")
	}
	if c.Profiles == nil {
		c.Profiles = &ProfileStore{Profiles: map[string]*Profile{}}
	}
	c.Profiles.Profiles[name] = &Profile{
		Endpoint:      strings.TrimRight(*endpoint, "/"),
		JWTPrivateKey: *jwtPrivateKey,
		JWTIssuer:     *jwtIssuer,
		JWTAudience:   *jwtAudience,
		JWTScope:      *jwtScope,
		JWTSubject:    *jwtSubject,
	}
	if err := c.Profiles.save(c.ProfilePath); err != nil {
		return err
	}
	return c.print(map[string]interface{}{"profile": name, "endpoint": *endpoint, "jwtPrivateKey": *jwtPrivateKey, "jwtIssuer": *jwtIssuer, "jwtAudience": *jwtAudience, "jwtScope": *jwtScope, "jwtSubject": *jwtSubject})
}

func (c *Client) saveProfile() error {
	if c.Profiles == nil || c.ProfileName == "" {
		return nil
	}
	profile := c.Profiles.Profiles[c.ProfileName]
	if profile == nil {
		profile = &Profile{}
		c.Profiles.Profiles[c.ProfileName] = profile
	}
	profile.Endpoint = c.Endpoint
	profile.SessionID = c.SessionID
	return c.Profiles.save(c.ProfilePath)
}

func (c *Client) getOperation(operationID string) (*managerservice.Operation, error) {
	if err := c.requireSession(); err != nil {
		return nil, err
	}
	result := &managerservice.Operation{}
	path := fmt.Sprintf("/v1/endly/sessions/%s/operations/%s", url.PathEscape(c.SessionID), url.PathEscape(operationID))
	if err := c.do(context.Background(), http.MethodGet, path, nil, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) requireSession() error {
	if c.SessionID == "" {
		return errors.New("session was required; pass --session, set ENDLY_SESSION_ID, or use load --new-session")
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, input, output interface{}) error {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.Endpoint+path, body)
	if err != nil {
		return err
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.TokenProvider != nil {
		token, tokenErr := c.TokenProvider(ctx)
		if tokenErr != nil {
			return tokenErr
		}
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		payload := struct {
			Error string `json:"error"`
		}{}
		_ = json.NewDecoder(response.Body).Decode(&payload)
		if payload.Error == "" {
			payload.Error = response.Status
		}
		return errors.New(payload.Error)
	}
	if output == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(output)
}

func (c *Client) print(value interface{}) error {
	encoder := json.NewEncoder(c.Output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func isTerminal(status string) bool {
	return status == managerservice.OperationSucceeded || status == managerservice.OperationFailed || status == managerservice.OperationCancelled
}
