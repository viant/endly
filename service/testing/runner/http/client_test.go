package http

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/endly"
	"github.com/viant/endly/model"
)

func TestSendReusesConnectionsAndCancels(t *testing.T) {
	var connections atomic.Int64
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wait" {
			<-r.Context().Done()
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	defer server.Close()
	s := newServiceForTest()
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	for i := 0; i < 3; i++ {
		request := &SendRequest{Requests: []*Request{{Method: "GET", URL: server.URL, Repeater: &model.Repeater{Repeat: 1}}}}
		if err := request.Init(); err != nil {
			t.Fatal(err)
		}
		if _, err := s.send(ctx, request); err != nil {
			t.Fatal(err)
		}
	}
	if connections.Load() != 1 {
		t.Fatalf("wanted one reused connection, got %d", connections.Load())
	}
	operation, cancel := context.WithCancel(context.Background())
	ctx.SetBackground(operation)
	done := make(chan error, 1)
	go func() {
		_, err := s.send(ctx, &SendRequest{Requests: []*Request{{Method: "GET", URL: server.URL + "/wait", Repeater: &model.Repeater{Repeat: 1}}}})
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancellation was ignored")
		}
	case <-time.After(time.Second):
		t.Fatal("HTTP action did not observe operation cancellation")
	}
}
