package mcp

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/system/storage"
)

// ReadCaptureRequest reads a browser capture without making an MCP request.
type ReadCaptureRequest struct {
	Source          *location.Resource
	Methods         []string
	MaxTransactions int
}

func (r *ReadCaptureRequest) Validate() error {
	if r.Source == nil || r.Source.URL == "" {
		return fmt.Errorf("capture source is required")
	}
	if r.MaxTransactions < 0 || r.MaxTransactions > 100000 {
		return fmt.Errorf("MaxTransactions must be between 0 and 100000")
	}
	return nil
}

type CapturedCall struct {
	Method     string                 `json:"method"`
	Args       map[string]interface{} `json:"args"`
	Status     int                    `json:"status"`
	Response   map[string]interface{} `json:"response"`
	RequestID  interface{}            `json:"requestId"`
	DurationMs float64                `json:"durationMs"`
}

type ReadCaptureResponse struct{ Transactions []*CapturedCall }

type captureBody struct {
	Data, Encoding string
	Truncated      bool
}
type captureTransaction struct {
	RequestBody, ResponseBody *captureBody
	Status                    int
	DurationMs                float64
}
type captureEvent struct {
	Type string
	Tx   *captureTransaction
}

func (s *service) registerCaptureRoute() {
	s.Register(&endly.Route{
		Action:           "readCapture",
		RequestInfo:      &endly.ActionInfo{Description: "Decode captured MCP tool calls, pairing responses by ID within each browser network transaction"},
		RequestProvider:  func() interface{} { return &ReadCaptureRequest{} },
		ResponseProvider: func() interface{} { return &ReadCaptureResponse{} },
		Handler: func(ctx *endly.Context, value interface{}) (interface{}, error) {
			r := value.(*ReadCaptureRequest)
			source, opts, err := storage.GetResourceWithOptions(ctx, r.Source)
			if err != nil {
				return nil, fmt.Errorf("MCP capture source resolution failed")
			}
			fs, err := storage.StorageService(ctx, source)
			if err != nil {
				return nil, fmt.Errorf("MCP capture source storage unavailable")
			}
			reader, err := fs.OpenURL(ctx.Background(), source.URL, opts...)
			if err != nil {
				return nil, fmt.Errorf("MCP capture source could not be opened")
			}
			defer reader.Close()
			return readCapture(reader, r)
		},
	})
}

// Each error identifies the event and stage, never body contents or headers.
func readCapture(reader io.Reader, request *ReadCaptureRequest) (*ReadCaptureResponse, error) {
	limit := request.MaxTransactions
	if limit == 0 {
		limit = 10000
	}
	out := &ReadCaptureResponse{Transactions: make([]*CapturedCall, 0)}
	methods := map[string]bool{}
	for _, name := range request.Methods {
		methods[name] = true
	}
	scan := bufio.NewScanner(reader)
	scan.Buffer(make([]byte, 4096), 64*1024*1024)
	line, total := 0, 0
	for scan.Scan() {
		line++
		if len(bytes.TrimSpace(scan.Bytes())) == 0 {
			continue
		}
		var event captureEvent
		if json.Unmarshal(scan.Bytes(), &event) != nil {
			return nil, fmt.Errorf("MCP capture event %d: invalid event JSON", line)
		}
		if event.Type != "network" {
			continue
		}
		if event.Tx == nil {
			return nil, fmt.Errorf("MCP capture event %d: missing network transaction", line)
		}
		tx := event.Tx
		if tx.RequestBody == nil {
			continue
		} // Non-RPC requests have no body.
		requests, err := captureMessages(tx.RequestBody)
		if err != nil {
			return nil, fmt.Errorf("MCP capture event %d: request %s", line, err)
		}
		var calls []map[string]interface{}
		for _, message := range requests {
			if message["method"] == "tools/call" {
				calls = append(calls, message)
			}
		}
		if len(calls) == 0 {
			continue
		}
		responses, err := captureMessages(tx.ResponseBody)
		if err != nil {
			return nil, fmt.Errorf("MCP capture event %d: response %s", line, err)
		}
		seen := map[string]bool{}
		for _, call := range calls {
			id, err := captureID(call)
			if err != nil || seen[id] {
				return nil, fmt.Errorf("MCP capture event %d: missing, invalid or duplicate request ID", line)
			}
			seen[id] = true
			params, ok := call["params"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("MCP capture event %d: missing tool parameters", line)
			}
			name, ok := params["name"].(string)
			if !ok || name == "" {
				return nil, fmt.Errorf("MCP capture event %d: missing tool name", line)
			}
			args, ok := params["arguments"].(map[string]interface{})
			if params["arguments"] != nil && !ok {
				return nil, fmt.Errorf("MCP capture event %d: tool arguments must be an object", line)
			}
			var response map[string]interface{}
			for _, candidate := range responses {
				candidateID, e := captureID(candidate)
				if e != nil || candidateID != id {
					continue
				}
				_, result := candidate["result"]
				_, protocolError := candidate["error"]
				if !result && !protocolError {
					continue
				}
				if result && protocolError {
					return nil, fmt.Errorf("MCP capture event %d: response has both result and error", line)
				}
				if response != nil {
					return nil, fmt.Errorf("MCP capture event %d: duplicate response ID", line)
				}
				response = candidate
			}
			if response == nil {
				return nil, fmt.Errorf("MCP capture event %d: no matching response ID", line)
			}
			total++
			if total > limit {
				return nil, fmt.Errorf("MCP capture exceeds MaxTransactions (%d)", limit)
			}
			if len(methods) == 0 || methods[name] {
				out.Transactions = append(out.Transactions, &CapturedCall{Method: name, Args: args, Status: tx.Status, Response: response, RequestID: call["id"], DurationMs: tx.DurationMs})
			}
		}
	}
	if scan.Err() != nil {
		return nil, fmt.Errorf("MCP capture event %d: read failed or event exceeds 64 MiB", line+1)
	}
	if total == 0 {
		return nil, fmt.Errorf("MCP capture contains no complete tool calls")
	}
	return out, nil
}

func captureID(message map[string]interface{}) (string, error) {
	if message["jsonrpc"] != "2.0" {
		return "", fmt.Errorf("invalid JSON-RPC version")
	}
	switch value := message["id"].(type) {
	case string:
		return "string:" + value, nil
	case json.Number:
		return "number:" + string(value), nil
	default:
		return "", fmt.Errorf("invalid JSON-RPC ID")
	}
}

func captureMessages(body *captureBody) ([]map[string]interface{}, error) {
	if body == nil || body.Data == "" {
		return nil, fmt.Errorf("body missing")
	}
	if body.Truncated {
		return nil, fmt.Errorf("body truncated")
	}
	data := []byte(body.Data)
	switch body.Encoding {
	case "", "utf8", "utf-8":
	case "base64":
		var err error
		data, err = base64.StdEncoding.Strict().DecodeString(body.Data)
		if err != nil {
			return nil, fmt.Errorf("body has invalid base64")
		}
	default:
		return nil, fmt.Errorf("body has unsupported encoding")
	}
	decode := func(raw []byte) ([]map[string]interface{}, error) {
		var value interface{}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if d.Decode(&value) != nil {
			return nil, fmt.Errorf("body has invalid JSON")
		}
		if d.Decode(new(interface{})) != io.EOF {
			return nil, fmt.Errorf("body has trailing JSON")
		}
		if object, ok := value.(map[string]interface{}); ok {
			return []map[string]interface{}{object}, nil
		}
		if batch, ok := value.([]interface{}); ok {
			out := make([]map[string]interface{}, 0, len(batch))
			for _, entry := range batch {
				object, ok := entry.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("body has invalid JSON-RPC batch")
				}
				out = append(out, object)
			}
			return out, nil
		}
		return nil, fmt.Errorf("body is not a JSON-RPC object or batch")
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return decode(trimmed)
	}
	var result []map[string]interface{}
	var parts []string
	flush := func() error {
		if len(parts) == 0 {
			return nil
		}
		messages, err := decode([]byte(strings.Join(parts, "\n")))
		if err != nil {
			return err
		}
		result = append(result, messages...)
		parts = nil
		return nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if line == "" {
			if err := flush(); err != nil {
				return nil, err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			parts = append(parts, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("body has no JSON or SSE messages")
	}
	return result, nil
}
