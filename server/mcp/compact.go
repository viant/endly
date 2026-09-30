package mcp

import (
	"context"
	"strings"

	manager "github.com/viant/endly/service/manager"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	protocol "github.com/viant/mcp-protocol/server"
)

func compact(value interface{}) interface{} {
	switch v := value.(type) {
	case *manager.Operation:
		result := map[string]interface{}{"id": v.ID, "sessionId": v.SessionID, "status": v.Status, "kind": v.Kind, "workflow": v.Workflow, "tasks": v.Tasks, "tagIds": v.TagIDs, "passedAssertions": v.PassedAssertions, "failedAssertions": v.FailedAssertions, "eventCount": len(v.Events), "droppedEvents": v.DroppedEvents}
		if v.Error != "" {
			result["error"] = v.Error
		}
		if v.Debug != nil {
			state := *v.Debug
			state.Snapshot = nil
			result["debug"] = state
		}
		if v.Result != nil {
			keys := []string{}
			for k := range v.Result {
				keys = append(keys, k)
			}
			result["resultKeys"] = keys
		}
		return result
	case *manager.ListOperationsResponse:
		entries := []interface{}{}
		for _, op := range v.Operations {
			entries = append(entries, compact(op))
		}
		return map[string]interface{}{"sessionId": v.SessionID, "operations": entries}
	case *manager.ListTasksResponse:
		return map[string]interface{}{"sessionId": v.SessionID, "workflow": v.Workflow, "tasks": compactTasks(v.Tasks)}
	}
	return value
}

func compactTasks(tasks []*manager.TaskInfo) []interface{} {
	result := []interface{}{}
	for _, task := range tasks {
		item := map[string]interface{}{"name": task.Name, "path": task.Path}
		if len(task.Actions) > 0 {
			item["actionCount"] = len(task.Actions)
		}
		if len(task.Tasks) > 0 {
			item["tasks"] = compactTasks(task.Tasks)
		}
		if len(task.Instances) > 0 {
			instances := []interface{}{}
			for _, instance := range task.Instances {
				instances = append(instances, map[string]interface{}{"tagId": instance.TagID, "index": instance.Index, "tag": instance.Tag, "actionCount": len(instance.Actions)})
			}
			item["instances"] = instances
		}
		if len(task.Workflows) > 0 {
			nested := []interface{}{}
			for _, workflow := range task.Workflows {
				nested = append(nested, map[string]interface{}{"name": workflow.Name, "source": workflow.Source, "tasks": compactTasks(workflow.Tasks)})
			}
			item["workflows"] = nested
		}
		result = append(result, item)
	}
	return result
}

type EventsRequest struct {
	SessionID   string   `json:"sessionId"`
	OperationID string   `json:"operationId"`
	After       int64    `json:"after,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	Types       []string `json:"types,omitempty"`
	Detail      bool     `json:"detail,omitempty"`
}

func registerEvents(h *protocol.DefaultHandler, runtime *manager.Service) error {
	var input schema.ToolInputSchema
	if err := input.Load(&EventsRequest{}); err != nil {
		return err
	}
	h.RegisterToolWithSchema("endly_operation_events", "Page operation events; after is the last sequence, limit defaults to 50 (max 200), detail includes payloads", input, nil, func(ctx context.Context, req *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
		r := &EventsRequest{}
		if err := decodeArguments(req.Params.Arguments, r); err != nil {
			return nil, jsonrpc.NewInvalidParamsError(err.Error(), nil)
		}
		if r.Limit == 0 {
			r.Limit = 50
		}
		if r.Limit < 1 || r.Limit > 200 || r.After < 0 {
			return nil, jsonrpc.NewInvalidParamsError("limit must be 1..200 and after nonnegative", nil)
		}
		op, err := runtime.GetOperation(r.SessionID, r.OperationID)
		if err != nil {
			return toolResult(nil, err)
		}
		accepted := map[string]bool{}
		for _, kind := range r.Types {
			accepted[kind] = true
		}
		events := []interface{}{}
		next := r.After
		more := false
		for _, event := range op.Events {
			if event.Sequence <= r.After || len(accepted) > 0 && !accepted[event.Type] {
				continue
			}
			if len(events) == r.Limit {
				more = true
				break
			}
			next = event.Sequence
			if r.Detail {
				events = append(events, event)
			} else {
				summary := map[string]interface{}{"sequence": event.Sequence, "type": event.Type}
				if event.Activity != nil {
					summary["service"] = event.Activity.Service
					summary["action"] = event.Activity.Action
					summary["tagId"] = event.Activity.TagID
				}
				events = append(events, summary)
			}
		}
		oldest := int64(0)
		if len(op.Events) > 0 {
			oldest = op.Events[0].Sequence
		}
		return toolResult(map[string]interface{}{"operationId": op.ID, "events": events, "nextAfter": next, "hasMore": more, "oldestSequence": oldest, "droppedEvents": op.DroppedEvents}, nil)
	})
	return nil
}

func pageList(value interface{}, filter string, offset, limit int) interface{} {
	entries := []interface{}{}
	switch v := value.(type) {
	case *manager.ListSessionsResponse:
		for _, session := range v.Sessions {
			if filter == "" || strings.Contains(strings.ToLower(session.Name+" "+session.SessionID), strings.ToLower(filter)) {
				entries = append(entries, session)
			}
		}
	case *manager.ListWorkflowsResponse:
		for _, workflow := range v.Workflows {
			if filter == "" || strings.Contains(strings.ToLower(workflow.Name+" "+workflow.Alias), strings.ToLower(filter)) {
				entries = append(entries, workflow)
			}
		}
	case *manager.ListOperationsResponse:
		for _, operation := range v.Operations {
			if filter == "" || strings.Contains(strings.ToLower(operation.Status+" "+operation.Workflow+" "+operation.Tasks), strings.ToLower(filter)) {
				entries = append(entries, compact(operation))
			}
		}
	default:
		return compact(value)
	}
	start := min(offset, len(entries))
	end := min(start+limit, len(entries))
	return map[string]interface{}{"items": entries[start:end], "total": len(entries), "nextOffset": end, "hasMore": end < len(entries)}
}
