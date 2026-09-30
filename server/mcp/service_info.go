package mcp

import (
	"context"
	"sort"

	"github.com/viant/endly"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	protocol "github.com/viant/mcp-protocol/server"
)

type ServiceInfoRequest struct {
	Service string `json:"service,omitempty"`
	Action  string `json:"action,omitempty"`
}

func registerServiceInfo(h *protocol.DefaultHandler, manager endly.Manager) error {
	var input schema.ToolInputSchema
	if err := input.Load(&ServiceInfoRequest{}); err != nil {
		return err
	}
	h.RegisterToolWithSchema("endly_service_info", "Discover service IDs, action names, or one live request schema; no example payloads", input, nil, func(ctx context.Context, req *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
		r := &ServiceInfoRequest{}
		if err := decodeArguments(req.Params.Arguments, r); err != nil {
			return nil, jsonrpc.NewInvalidParamsError(err.Error(), nil)
		}
		if r.Service == "" {
			ids := []string{}
			for id := range endly.Services(manager) {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			return toolResult(map[string]interface{}{"services": ids}, nil)
		}
		service, err := manager.Service(r.Service)
		if err != nil {
			return toolResult(nil, err)
		}
		if r.Action == "" {
			return toolResult(map[string]interface{}{"service": r.Service, "actions": service.Actions()}, nil)
		}
		route, err := service.Route(r.Action)
		if err != nil {
			return toolResult(nil, err)
		}
		var request schema.ToolInputSchema
		if err = request.Load(route.RequestProvider()); err != nil {
			return toolResult(nil, err)
		}
		return toolResult(map[string]interface{}{"service": r.Service, "action": r.Action, "requestSchema": request}, nil)
	})
	return nil
}
