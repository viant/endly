package http

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/endly"
)

func TestListenReplacementIsExplicitAndPreservesListenerOnFailure(t *testing.T) {
	fixture := func(path, body string) string {
		t.Helper()
		dir := t.TempDir()
		for name, data := range map[string]string{
			"00-req.json":  fmt.Sprintf(`{"Method":"GET","URL":%q}`, path),
			"00-resp.json": fmt.Sprintf(`{"Code":200,"Body":%q}`, body),
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}
	first, second := fixture("/first", "first"), fixture("/second", "second")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	ctx := endly.New().NewContext(nil)
	t.Cleanup(ctx.Close)
	service := New()
	client := &http.Client{Timeout: 2 * time.Second}
	check := func(path string, code int, body string) {
		t.Helper()
		response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != code || code == 200 && string(data) != body {
			t.Fatalf("response: %s %s %v", response.Status, data, err)
		}
	}
	listen := func(directory string, replace, rotate bool) error {
		return service.Run(ctx, &ListenRequest{Host: "127.0.0.1", Port: port, BaseDirectory: directory, Replace: replace, Rotate: rotate, IndexKeys: []string{MethodKey, URLKey}}).Err
	}
	if err := listen(first, false, true); err != nil {
		t.Fatal(err)
	}
	check("/first", 200, "first")
	if err := listen(second, false, false); err != nil {
		t.Fatal(err)
	}
	check("/first", 200, "first") // default retains the old recording and rotation
	check("/second", 404, "")
	if err := listen(second, true, false); err != nil {
		t.Fatal(err)
	}
	check("/first", 404, "") // replacement must not append old fixtures
	check("/second", 200, "second")
	check("/second", 404, "")
	if err := listen(second, true, true); err != nil {
		t.Fatal(err)
	}
	check("/second", 200, "second") // replay position reset
	check("/second", 200, "second")
	if err := listen(t.TempDir(), true, true); err == nil {
		t.Fatal("invalid replacement succeeded")
	}
	check("/second", 200, "second")
	changedHost := service.Run(ctx, &ListenRequest{Host: "", Port: port, BaseDirectory: first, Replace: true})
	if changedHost.Err == nil {
		t.Fatal("replacement silently changed bind host")
	}
	check("/second", 200, "second")
	ctx.Close()
	listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("replaced listener was not cleaned up: %v", err)
	}
	listener.Close()
}
