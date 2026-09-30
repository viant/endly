package mcp

import (
	"context"
	"strings"

	manager "github.com/viant/endly/service/manager"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	protocol "github.com/viant/mcp-protocol/server"
)

type InstancesRequest struct {
	SessionID string `json:"sessionId"`
	Workflow  string `json:"workflow"`
	Path      string `json:"path,omitempty"`
	Filter    string `json:"filter,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func registerInstances(h *protocol.DefaultHandler, runtime *manager.Service) error {
	var input schema.ToolInputSchema
	if err := input.Load(&InstancesRequest{}); err != nil {
		return err
	}
	h.RegisterToolWithSchema("endly_listInstances", "Discover exact template IDs in the actual nested run context. Filter by tag, ID or group; page with offset/limit (default 20, max 100)", input, nil, func(ctx context.Context, req *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
		r := &InstancesRequest{}
		if err := decodeArguments(req.Params.Arguments, r); err != nil {
			return nil, jsonrpc.NewInvalidParamsError(err.Error(), nil)
		}
		if r.Limit == 0 {
			r.Limit = 20
		}
		if r.Offset < 0 || r.Limit < 1 || r.Limit > 100 {
			return nil, jsonrpc.NewInvalidParamsError("offset must be nonnegative and limit 1..100", nil)
		}
		tree, err := runtime.ListTasks(&manager.ListTasksRequest{SessionID: r.SessionID, Workflow: r.Workflow, Path: r.Path, ExpandWorkflows: true})
		if err != nil {
			return toolResult(nil, err)
		}
		entries := []interface{}{}
		var visit func([]*manager.TaskInfo, string)
		visit = func(tasks []*manager.TaskInfo, owner string) {
			for _, task := range tasks {
				for _, instance := range task.Instances {
					if r.Filter != "" && !strings.Contains(strings.ToLower(instance.TagID+" "+instance.Tag+" "+task.Path), strings.ToLower(r.Filter)) {
						continue
					}
					item := map[string]interface{}{"workflow": owner, "group": task.Path, "tagId": instance.TagID, "index": instance.Index, "tag": instance.Tag, "actionCount": len(instance.Actions)}
					for _, action := range instance.Actions {
						if action.Skip != "" {
							item["skipExpression"] = action.Skip
						}
					}
					entries = append(entries, item)
				}
				visit(task.Tasks, owner)
				for _, child := range task.Workflows {
					visit(child.Tasks, child.Source)
				}
			}
		}
		visit(tree.Tasks, r.Workflow)
		start := min(r.Offset, len(entries))
		end := min(start+r.Limit, len(entries))
		return toolResult(map[string]interface{}{"instances": entries[start:end], "total": len(entries), "nextOffset": end, "hasMore": end < len(entries)}, nil)
	})
	return nil
}
