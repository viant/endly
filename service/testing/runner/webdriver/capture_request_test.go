package webdriver

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	selog "github.com/tebeka/selenium/log"
)

func TestCaptureRequestEntriesBoundedAndRedacted(t *testing.T) {
	yes := true
	s := newCaptureState(&CaptureStartRequest{IncludeBodies: &yes, MaxBodyBytes: 4})
	raw, _ := json.Marshal(map[string]any{"requestId": "one", "request": map[string]any{"url": "http://fixture.test/", "method": "POST", "headers": map[string]any{"Authorization": "secret"}, "postDataEntries": []map[string]any{{"bytes": base64.StdEncoding.EncodeToString([]byte("abcdef"))}}}})
	s.onRequestWillBeSent(raw)
	require.Equal(t, "abcd", s.inflight["one"].RequestBody.Data)
	require.True(t, s.inflight["one"].RequestBody.Truncated)
	require.Equal(t, "<redacted>", s.inflight["one"].RequestHeaders["Authorization"])
	no := false
	off := newCaptureState(&CaptureStartRequest{IncludeBodies: &no})
	off.onRequestWillBeSent(raw)
	require.Nil(t, off.inflight["one"].RequestBody)
}
func TestCaptureRealPostBodyLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome endpoint required")
	}
	body := `{"jsonrpc":"2.0","id":17,"method":"tools/call","params":{"name":"fixture","arguments":{"Id":[123]}}}`
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			data, _ := io.ReadAll(r.Body)
			received <- string(data)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		_, _ = w.Write([]byte(`<title>Endly native request capture fixture</title>`))
	}))
	defer server.Close()
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	_, err = driver.OpenTab(server.URL)
	require.NoError(t, err)
	defer driver.Close()
	yes := true
	s := newCaptureState(&CaptureStartRequest{IncludeBodies: &yes, URLIncludes: []string{server.URL}})
	_, err = driver.CDPCommand("Network.enable", map[string]any{"maxPostDataSize": s.maxBodyBytes})
	require.NoError(t, err)
	_, err = driver.ExecuteScript(`return fetch(arguments[0],{method:'POST',headers:{'Content-Type':'application/json'},body:arguments[1]}).then(r=>r.json());`, []interface{}{server.URL + "/mcp", body})
	require.NoError(t, err)
	require.Equal(t, body, <-received)
	sess := &Session{driver: driver, Backend: "cdp"}
	logs, err := driver.Log(selog.Performance)
	require.NoError(t, err)
	var fallbackID string
	for _, entry := range logs {
		method, params, err := parsePerformanceLogMessage(entry.Message)
		require.NoError(t, err)
		switch method {
		case "Network.requestWillBeSent":
			var event map[string]any
			require.NoError(t, json.Unmarshal(params, &event))
			request := event["request"].(map[string]any)
			if request["method"] == "POST" {
				t.Logf("Chrome request body fields: postData=%v entries=%v hasPostData=%v", request["postData"] != nil, request["postDataEntries"] != nil, request["hasPostData"])
				// Exercise recovery using the real browser-owned request ID with inline
				// fields withheld, as Chrome does for bodies excluded from events.
				delete(request, "postData")
				delete(request, "postDataEntries")
				request["hasPostData"] = true
				params, err = json.Marshal(event)
				require.NoError(t, err)
				fallbackID = event["requestId"].(string)
			}
			s.onRequestWillBeSent(params, sess)
		case "Network.responseReceived":
			s.onResponseReceived(params)
		case "Network.loadingFinished":
			s.onLoadingFinished(sess, params)
		}
	}
	require.NotEmpty(t, fallbackID)
	require.Empty(t, s.Summary().Errors)

	_, network := s.Snapshot(100, false, true)
	var found bool
	for _, tx := range network {
		if tx.Method == "POST" && strings.HasSuffix(tx.URL, "/mcp") {
			found = true
			require.NotNil(t, tx.RequestBody)
			require.Equal(t, body, tx.RequestBody.Data)
			require.NotNil(t, tx.ResponseBody)
		}
	}
	require.True(t, found, "actual local fetch transaction must be captured")
}
