// Package mcp exposes Endly's stateful control runtime and skill catalog via MCP.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/gops/agent"
	"github.com/viant/endly"
	"github.com/viant/endly/server/control"
	manager "github.com/viant/endly/service/manager"
	"github.com/viant/endly/skills"
	"github.com/viant/jsonrpc"
	viantmcp "github.com/viant/mcp"
	skillformat "github.com/viant/mcp-protocol/extension/skills"
	"github.com/viant/mcp-protocol/schema"
	protocol "github.com/viant/mcp-protocol/server"
	mcpserver "github.com/viant/mcp/server"
	"github.com/viant/toolbox"
)

// NewHandler registers every control action using its actual request contract.
// Both MCP and the web API share the supplied runtime, sessions and operations.
func NewHandler(ctx context.Context, runtime *manager.Service, factory manager.ManagerFactory, localSkills ...fs.FS) (*protocol.DefaultHandler, error) {
	h := protocol.NewDefaultHandler(nil, nil, nil)
	entries, err := fs.ReadDir(skills.Files, ".")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		source, err := fs.Sub(skills.Files, entry.Name())
		if err != nil {
			return nil, err
		}
		compiled, err := (skillformat.Compiler{Source: source}).Compile(ctx, "endly://skills/"+entry.Name()+"/SKILL.md")
		if err != nil {
			return nil, fmt.Errorf("compile %s: %w", entry.Name(), err)
		}
		if err = h.RegisterStaticSkill(compiled); err != nil {
			return nil, err
		}
	}
	for _, source := range localSkills {
		raw, err := fs.ReadFile(source, "SKILL.md")
		if err != nil {
			return nil, err
		}
		front, err := skillformat.Frontmatter(raw)
		if err != nil {
			return nil, err
		}
		name := front["name"].(string)
		compiled, err := (skillformat.Compiler{Source: source}).Compile(ctx, "endly://skills/"+name+"/SKILL.md")
		if err != nil {
			return nil, err
		}
		if err = h.RegisterStaticSkill(compiled); err != nil {
			return nil, err
		}
	}
	controlManager := factory()
	for _, action := range runtime.Actions() {
		route, err := runtime.Route(action)
		if err != nil {
			return nil, err
		}
		var input schema.ToolInputSchema
		if err = input.Load(route.RequestProvider()); err != nil {
			return nil, err
		}
		if input.Properties == nil {
			input.Properties = schema.ToolInputSchemaProperties{}
		}
		input.Properties["detail"] = map[string]interface{}{"type": "boolean", "description": "Return full payloads instead of concise summaries"}
		paged := action == "listOperations" || action == "listSessions" || action == "listWorkflows"
		if paged {
			input.Properties["filter"] = map[string]interface{}{"type": "string", "description": "Match name, status or ID"}
			input.Properties["offset"] = map[string]interface{}{"type": "integer", "minimum": 0}
			input.Properties["limit"] = map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 100, "default": 20}
		}
		description := route.RequestInfo.Description
		if value := toolDescriptions[action]; value != "" {
			description = value
		}
		h.RegisterToolWithSchema("endly_"+action, description, input, nil, func(ctx context.Context, req *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
			args := map[string]interface{}{}
			for key, value := range req.Params.Arguments {
				args[key] = value
			}
			detail, _ := args["detail"].(bool)
			delete(args, "detail")
			filter := ""
			offset := 0
			limit := 20
			if paged {
				filter, _ = args["filter"].(string)
				offset = toolbox.AsInt(args["offset"])
				if value, ok := args["limit"]; ok {
					limit = toolbox.AsInt(value)
				}
				delete(args, "filter")
				delete(args, "offset")
				delete(args, "limit")
				if offset < 0 || limit < 1 || limit > 100 {
					return nil, jsonrpc.NewInvalidParamsError("offset must be nonnegative and limit 1..100", nil)
				}
			}
			raw, err := json.Marshal(args)
			if err != nil {
				return nil, jsonrpc.NewInvalidParamsError(err.Error(), nil)
			}
			if string(raw) == "null" {
				raw = []byte("{}")
			}
			input := route.RequestProvider()
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.DisallowUnknownFields()
			if err = decoder.Decode(input); err != nil {
				return nil, jsonrpc.NewInvalidParamsError(err.Error(), nil)
			}
			execution := controlManager.NewContext(nil)
			execution.SetBackground(ctx)
			defer execution.Close()
			response, err := route.Handler(execution, input)
			if paged {
				response = pageList(response, filter, offset, limit)
			} else if !detail {
				response = compact(response)
			}
			return toolResult(response, err)
		})
	}
	for _, name := range []string{"endly_skill_list", "endly_skill_get"} {
		input := schema.ToolInputSchema{}
		if name == "endly_skill_list" {
			err = input.Load(&schema.ListSkillsRequestParams{})
		} else {
			err = input.Load(&schema.GetSkillRequestParams{})
		}
		if err != nil {
			return nil, err
		}
		h.RegisterToolWithSchema(name, "Discover Endly skills; get returns metadata and entrypoint content", input, nil, func(ctx context.Context, req *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
			raw, _ := json.Marshal(req.Params.Arguments)
			if string(raw) == "null" {
				raw = []byte("{}")
			}
			if name == "endly_skill_list" {
				params := &schema.ListSkillsRequest{}
				if err := json.Unmarshal(raw, &params.Params); err != nil {
					return nil, jsonrpc.NewInvalidParamsError(err.Error(), nil)
				}
				result, rpcErr := h.ListSkills(ctx, &jsonrpc.TypedRequest[*schema.ListSkillsRequest]{Request: params})
				if rpcErr != nil {
					return nil, rpcErr
				}
				entries := []interface{}{}
				for _, skill := range result.Skills {
					entries = append(entries, map[string]interface{}{"uri": skill.Uri, "name": skill.Frontmatter["name"], "description": skill.Frontmatter["description"]})
				}
				return toolResult(map[string]interface{}{"skills": entries, "nextCursor": result.NextCursor}, nil)
			}
			params := &schema.GetSkillRequest{}
			if err := json.Unmarshal(raw, &params.Params); err != nil {
				return nil, jsonrpc.NewInvalidParamsError(err.Error(), nil)
			}
			result, rpcErr := h.GetSkill(ctx, &jsonrpc.TypedRequest[*schema.GetSkillRequest]{Request: params})
			if rpcErr != nil {
				return nil, rpcErr
			}
			read := &schema.ReadResourceRequest{}
			read.Params.Uri = params.Params.Uri
			content, rpcErr, _ := h.ReadStaticSkillResource(ctx, read)
			if rpcErr != nil {
				return nil, rpcErr
			}
			return toolResult(map[string]interface{}{"skill": result.Skill, "contents": content.Contents}, nil)
		})
	}

	var resourceInput schema.ToolInputSchema
	if err := resourceInput.Load(&struct {
		URI string `json:"uri"`
	}{}); err != nil {
		return nil, err
	}
	h.RegisterToolWithSchema("endly_resource_read", "Read a skill entrypoint or reference from its published URI", resourceInput, nil, func(ctx context.Context, req *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
		raw, _ := json.Marshal(req.Params.Arguments)
		read := &schema.ReadResourceRequest{}
		params := struct {
			URI string `json:"uri"`
		}{}
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, jsonrpc.NewInvalidParamsError(err.Error(), nil)
		}
		read.Params.Uri = params.URI
		result, rpcErr, claimed := h.ReadStaticSkillResource(ctx, read)
		if !claimed {
			return nil, jsonrpc.NewInvalidParamsError("unknown skill resource", nil)
		}
		if rpcErr != nil {
			return nil, rpcErr
		}
		return toolResult(result, nil)
	})
	if err := registerSourceTools(h, runtime, controlManager); err != nil {
		return nil, err
	}
	if err := registerServiceInfo(h, controlManager); err != nil {
		return nil, err
	}
	if err := registerInstances(h, runtime); err != nil {
		return nil, err
	}
	if err := registerEvents(h, runtime); err != nil {
		return nil, err
	}
	return h, nil
}

func toolResult(value interface{}, err error) (*schema.CallToolResult, *jsonrpc.Error) {
	result := &schema.CallToolResult{}
	if err != nil {
		failed := true
		result.IsError = &failed
		value = map[string]string{"error": err.Error()}
	}
	raw, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		return nil, jsonrpc.NewInternalError(marshalErr.Error(), nil)
	}
	result.Content = []schema.CallToolResultContentElem{schema.TextContent{Type: "text", Text: string(raw)}}
	return result, nil
}

// NewServer builds stdio/streamable MCP plus the existing REST control API.
func NewServer(ctx context.Context, runtime *manager.Service, factory manager.ManagerFactory, localSkills ...fs.FS) (*mcpserver.Server, error) {
	handler, err := NewHandler(ctx, runtime, factory, localSkills...)
	if err != nil {
		return nil, err
	}
	api := control.New(runtime)
	srv, err := viantmcp.NewServer(protocol.WithDefaultHandler(ctx, func(session *protocol.DefaultHandler) error {
		session.Registry = handler.Registry
		return nil
	}), &viantmcp.ServerOptions{
		Name: "endly", Version: strings.TrimSpace(endly.GetVersion()),
		Transport: &viantmcp.ServerTransport{Type: "streamable", CustomHandlers: map[string]http.HandlerFunc{
			"/v1/endly/skills":    SkillsHTTP(handler),
			"/v1/endly/skills/":   SkillsHTTP(handler),
			"/v1/endly/sessions":  api.ServeHTTP,
			"/v1/endly/sessions/": api.ServeHTTP,
		}},
	})
	return srv, err
}

func SkillsHTTP(h *protocol.DefaultHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/v1/endly/skills")
		if name == "" || name == "/" {
			entries := h.ListRegisteredSkills()
			offset, offsetErr := strconv.Atoi(r.URL.Query().Get("offset"))
			if r.URL.Query().Get("offset") == "" {
				offset = 0
				offsetErr = nil
			}
			limit := 20
			var limitErr error
			if value := r.URL.Query().Get("limit"); value != "" {
				limit, limitErr = strconv.Atoi(value)
			}
			if offsetErr != nil || limitErr != nil || offset < 0 || limit < 1 || limit > 100 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			filtered := []interface{}{}
			filter := strings.ToLower(r.URL.Query().Get("filter"))
			for _, entry := range entries {
				name, _ := entry.Frontmatter["name"].(string)
				description, _ := entry.Frontmatter["description"].(string)
				if filter != "" && !strings.Contains(strings.ToLower(name+" "+description), filter) {
					continue
				}
				filtered = append(filtered, map[string]interface{}{"uri": entry.Uri, "name": name, "description": description})
			}
			start := min(offset, len(filtered))
			end := min(start+limit, len(filtered))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"skills": filtered[start:end], "total": len(filtered), "nextOffset": end, "hasMore": end < len(filtered)})
			return
		}
		name = strings.TrimPrefix(name, "/")
		params := &schema.GetSkillRequest{}
		params.Params.Uri = "endly://skills/" + name + "/SKILL.md"
		result, rpcErr := h.GetSkill(r.Context(), &jsonrpc.TypedRequest[*schema.GetSkillRequest]{Request: params})
		if rpcErr != nil {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unknown skill"})
			return
		}
		read := &schema.ReadResourceRequest{}
		read.Params.Uri = params.Params.Uri
		content, _, _ := h.ReadStaticSkillResource(r.Context(), read)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"skill": result.Skill, "contents": content.Contents})
	}
}

func Run(args []string, factory manager.ManagerFactory) error {
	flags := flag.NewFlagSet("endly mcp", flag.ContinueOnError)
	diagnostics := flags.Bool("diagnostics", false, "enable gops runtime diagnostics")
	transport := flags.String("transport", "stdio", "stdio or streamable")
	var skillDirs stringList
	flags.Var(&skillDirs, "skill-dir", "additional local skill folder (repeatable)")
	maxEvents := flags.Int("max-events", 1000, "retained tail events per operation")
	address := flags.String("addr", "127.0.0.1:4981", "loopback HTTP address")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *transport != "stdio" && *transport != "streamable" {
		return fmt.Errorf("unknown transport: %s", *transport)
	}
	host, _, err := net.SplitHostPort(*address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("MCP HTTP must bind to a loopback address")
	}
	if *diagnostics {
		if err := agent.Listen(agent.Options{}); err != nil {
			return err
		}
		defer agent.Close()
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	runtime := manager.New(factory, manager.WithMaxEvents(*maxEvents))
	defer func() {
		shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_ = runtime.Shutdown(shutdown)
	}()
	runtime.StartJanitor(ctx, time.Minute)
	sources := []fs.FS{}
	for _, dir := range skillDirs {
		sources = append(sources, os.DirFS(dir))
	}
	server, err := NewServer(ctx, runtime, factory, sources...)
	if err != nil {
		return err
	}
	if *transport == "stdio" {
		stdio := server.Stdio(ctx)
		// The transport captures its writer before workflow print actions are redirected.
		original := os.Stdout
		os.Stdout = os.Stderr
		defer func() { os.Stdout = original }()
		return stdio.ListenAndServe()
	}
	httpServer := server.HTTP(ctx, *address)
	httpServer.ReadHeaderTimeout = 10 * time.Second
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdown)
	}()
	err = httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func decodeArguments(args map[string]interface{}, target interface{}) error {
	raw, err := json.Marshal(args)
	if err != nil {
		return err
	}
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

var toolDescriptions = map[string]string{
	"open":           "Create a reusable session for loaded workflows, state and operations",
	"loadWorkflow":   "Load YAML/JSON and resolve Endly assets/defaults before running; returns alias",
	"runWorkflow":    "Queue a loaded workflow; select tasks/tagIds or enable debug; poll getOperation",
	"startWorkflow":  "Alias of runWorkflow",
	"getOperation":   "Poll concise status and assertion counts; detail=true includes result/events",
	"listTasks":      "Discover task paths; expandWorkflows includes nested run boundaries; detail=true shows actions",
	"inspectContext": "Read one state path, or full=true; operationId reads a paused debug snapshot",
	"debugCommand":   "Control pause/step/next/continue/stop or addBreakpoint/removeBreakpoint",
	"getDebugState":  "Read current debug point, pause status and breakpoints",
	"runAction":      "Queue a service action in the session; poll getOperation",
	"listOperations": "Filter and page operations by status, workflow or tasks",
	"listWorkflows":  "Filter and page loaded workflows by name or alias",
	"listSessions":   "Filter and page sessions by name or ID",
}

type stringList []string

func (s *stringList) String() string         { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error { *s = append(*s, value); return nil }
