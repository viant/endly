package http

import (
	"fmt"
	"github.com/viant/endly"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestShutdownAllowsRestartAndContextCleanup(t *testing.T) {
	directory := t.TempDir()
	for name, body := range map[string]string{"00-req.json": `{"Method":"GET","URL":"/userinfo"}`, "00-resp.json": `{"Code":200,"Body":"mock"}`} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	ctx := endly.New().NewContext(nil)
	t.Cleanup(ctx.Close)
	service := New()
	for i := 0; i < 2; i++ {
		result := service.Run(ctx, &ListenRequest{Port: port, Host: "127.0.0.1", BaseDirectory: directory, Rotate: true, IndexKeys: []string{MethodKey, URLKey}})
		if result.Err != nil {
			t.Fatal(result.Err)
		}
		response, err := http.Get("http://" + net.JoinHostPort("127.0.0.1", fmt.Sprint(port)) + "/userinfo")
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatal(response.Status)
		}
		if i == 0 {
			if result := service.Run(ctx, &ShutdownRequest{Port: port}); result.Err != nil {
				t.Fatal(result.Err)
			}
		}
	}
	ctx.Close()
	listener, err = net.Listen("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(port)))
	if err != nil {
		t.Fatalf("context did not release listener: %v", err)
	}
	listener.Close()
}
