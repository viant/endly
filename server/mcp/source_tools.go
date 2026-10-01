package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/viant/endly"
	manager "github.com/viant/endly/service/manager"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	protocol "github.com/viant/mcp-protocol/server"
)

// Source actions dispatch through the same stateful manager as runAction.
// Request schemas come from registered Endly routes, without example payloads.
var extendedServices = []string{"dsunit", "http/runner", "http/endpoint", "validator", "validator/log"}

type sourceActionRequest struct {
	SessionID     string                 `json:"sessionId"`
	Request       map[string]interface{} `json:"request,omitempty"`
	TimeoutMillis int64                  `json:"timeoutMillis,omitempty"`
}

func registerSourceTools(h *protocol.DefaultHandler, runtime *manager.Service, registry endly.Manager) error {
	for _, id := range extendedServices {
		service, err := registry.Service(id)
		if err != nil {
			continue
		} // A minimal/custom registry may omit optional services.
		actions := append([]string(nil), service.Actions()...)
		sort.Strings(actions)
		for _, action := range actions {
			route, err := service.Route(action)
			if err != nil {
				return err
			}
			var source schema.ToolInputSchema
			if err = source.Load(route.RequestProvider()); err != nil {
				return fmt.Errorf("%s:%s schema: %w", id, action, err)
			}
			raw, err := json.Marshal(source)
			if err != nil {
				return err
			}
			requestSchema := map[string]interface{}{}
			if err = json.Unmarshal(raw, &requestSchema); err != nil {
				return err
			}
			input := schema.ToolInputSchema{Type: "object", Required: []string{"sessionId"}, Properties: schema.ToolInputSchemaProperties{
				"sessionId":     {"type": "string", "description": "Existing Endly session"},
				"request":       requestSchema,
				"timeoutMillis": {"type": "integer", "minimum": 0},
			}}
			name := "endly_" + strings.ReplaceAll(id, "/", "_") + "_" + action
			description := fmt.Sprintf("Dispatch native %s:%s; returns operation ID. Inspect status/state/events with control tools", id, action)
			if _, exists := h.ToolRegistry.Get(name); exists {
				return fmt.Errorf("duplicate source tool %s", name)
			}
			h.RegisterToolWithSchema(name, description, input, nil, func(ctx context.Context, req *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
				input := &sourceActionRequest{}
				if err := decodeArguments(req.Params.Arguments, input); err != nil {
					return nil, jsonrpc.NewInvalidParamsError(err.Error(), nil)
				}
				if input.SessionID == "" || input.TimeoutMillis < 0 {
					return nil, jsonrpc.NewInvalidParamsError("sessionId is required and timeoutMillis must be nonnegative", nil)
				}
				operation, err := runtime.StartAction(&manager.RunActionRequest{SessionID: input.SessionID, Service: id, Action: action, Request: input.Request, TimeoutMillis: input.TimeoutMillis})
				if err != nil {
					return toolResult(nil, err)
				}
				return toolResult(compact(operation), nil)
			})
		}
	}
	return nil
}
