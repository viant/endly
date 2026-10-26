package mcp

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
)

const captureRequest = `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"namespaces/preview","arguments":{"resource":{"uri":"window://platform/advertiser"}}}}`
const captureResponse = `{"jsonrpc":"2.0","id":7,"result":{"structuredContent":{"sessionId":"child-1"}}}`

func captureLine(t *testing.T, request, response *captureBody) string {
	t.Helper()
	data, err := json.Marshal(captureEvent{Type: "network", Tx: &captureTransaction{RequestBody: request, ResponseBody: response, Status: 200, DurationMs: 12.5}})
	if err != nil {
		t.Fatal(err)
	}
	return string(data) + "\n"
}
func textBody(value string) *captureBody { return &captureBody{Data: value} }

func TestReadCapturePairsWithinNetworkTransaction(t *testing.T) {
	text := captureLine(t, textBody(captureRequest), textBody(captureResponse)) +
		captureLine(t, textBody(captureRequest), textBody(strings.ReplaceAll(captureResponse, "child-1", "child-2")))
	out, err := readCapture(strings.NewReader(text), &ReadCaptureRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Transactions) != 2 {
		t.Fatalf("calls=%d", len(out.Transactions))
	}
	for i, call := range out.Transactions {
		if call.Method != "namespaces/preview" || call.Status != 200 || call.DurationMs != 12.5 {
			t.Fatalf("metadata mismatch: %+v", call)
		}
		if call.Response["result"].(map[string]interface{})["structuredContent"].(map[string]interface{})["sessionId"] != []string{"child-1", "child-2"}[i] {
			t.Fatal("response crossed transaction boundary")
		}
	}
}

func TestReadCaptureBase64SSEAndBatch(t *testing.T) {
	request := "[" + captureRequest + "," + strings.ReplaceAll(captureRequest, `"id":7`, `"id":"7"`) + "]"
	response := "event: message\r\ndata: " + strings.ReplaceAll(captureResponse, `"id":7`, `"id":"7"`) + "\r\n\r\n: heartbeat\r\n\r\ndata: " + captureResponse + "\r\n\r\n"
	line := captureLine(t, &captureBody{Encoding: "base64", Data: base64.StdEncoding.EncodeToString([]byte(request))}, textBody(response))
	out, err := readCapture(strings.NewReader(line), &ReadCaptureRequest{Methods: []string{"namespaces/preview"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Transactions) != 2 {
		t.Fatal("batch calls were lost")
	}
	if _, ok := out.Transactions[0].RequestID.(json.Number); !ok {
		t.Fatal("numeric ID changed type")
	}
	if out.Transactions[1].RequestID != "7" {
		t.Fatal("string ID changed type")
	}
}

func TestReadCapturePreservesProtocolAndToolErrors(t *testing.T) {
	for _, response := range []string{`{"jsonrpc":"2.0","id":7,"error":{"code":-32602,"message":"invalid"}}`, `{"jsonrpc":"2.0","id":7,"result":{"isError":true,"structuredContent":{"status":403}}}`} {
		out, err := readCapture(strings.NewReader(captureLine(t, textBody(captureRequest), textBody(response))), &ReadCaptureRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Transactions) != 1 {
			t.Fatal("error result silently dropped")
		}
	}
}

func TestReadCaptureRejectsIncompleteOrAmbiguousEvidence(t *testing.T) {
	for _, tc := range []struct {
		name              string
		request, response *captureBody
		want              string
	}{
		{"missing response", textBody(captureRequest), nil, "response body missing"},
		{"truncated request", &captureBody{Data: captureRequest, Truncated: true}, textBody(captureResponse), "request body truncated"},
		{"truncated response", textBody(captureRequest), &captureBody{Data: captureResponse, Truncated: true}, "response body truncated"},
		{"wrong response", textBody(captureRequest), textBody(strings.ReplaceAll(captureResponse, `"id":7`, `"id":8`)), "no matching response ID"},
		{"duplicate response", textBody(captureRequest), textBody("[" + captureResponse + "," + captureResponse + "]"), "duplicate response ID"},
		{"duplicate request", textBody("[" + captureRequest + "," + captureRequest + "]"), textBody(captureResponse), "duplicate request ID"},
		{"invalid body", textBody("SECRET-SENTINEL"), textBody(captureResponse), "request body has no JSON"},
		{"invalid base64", &captureBody{Encoding: "base64", Data: "SECRET-SENTINEL"}, textBody(captureResponse), "invalid base64"},
		{"invalid arguments", textBody(strings.ReplaceAll(captureRequest, `{"resource":{"uri":"window://platform/advertiser"}}`, `[]`)), textBody(captureResponse), "arguments must be an object"},
		{"mixed result and error", textBody(captureRequest), textBody(`{"jsonrpc":"2.0","id":7,"result":{},"error":{}}`), "both result and error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readCapture(strings.NewReader(captureLine(t, tc.request, tc.response)), &ReadCaptureRequest{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v; want %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "SECRET-SENTINEL") {
				t.Fatal("capture error leaked body")
			}
		})
	}
}

func TestReadCaptureRequiresCallsAndEnforcesLimit(t *testing.T) {
	for _, text := range []string{"", `{"type":"console"}`, `{"type":"network","tx":null}`} {
		if _, err := readCapture(strings.NewReader(text), &ReadCaptureRequest{}); err == nil {
			t.Fatal("empty or invalid evidence accepted")
		}
	}
	line := captureLine(t, textBody(captureRequest), textBody(captureResponse))
	if _, err := readCapture(strings.NewReader(line+line), &ReadCaptureRequest{MaxTransactions: 1}); err == nil {
		t.Fatal("limit silently truncated capture")
	}
	out, err := readCapture(strings.NewReader(line), &ReadCaptureRequest{Methods: []string{"other"}})
	if err != nil || len(out.Transactions) != 0 {
		t.Fatalf("filter=%+v %v", out, err)
	}
}

func TestReadCaptureNativeRoute(t *testing.T) {
	file := filepath.Join(t.TempDir(), "capture.jsonl")
	if err := os.WriteFile(file, []byte(captureLine(t, textBody(captureRequest), textBody(captureResponse))), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := endly.New().NewContext(nil)
	var response ReadCaptureResponse
	if err := endly.Run(ctx, &ReadCaptureRequest{Source: location.NewResource(file)}, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Transactions) != 1 {
		t.Fatal("native route lost transaction")
	}
}
