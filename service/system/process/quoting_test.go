package process

import (
	systemexec "github.com/viant/endly/service/system/exec"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestQuotedProcessArgumentsAreLiteral(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell")
	}
	dir := filepath.Join(t.TempDir(), "directory with spaces")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "result.txt")
	unexpected := filepath.Join(dir, "unexpected")
	argument := "root:pwd@tcp(127.0.0.1:18897)/db?parseTime=true&x='literal'; touch " + unexpected
	request := &StartRequest{Command: "/bin/sh", Arguments: []string{"-c", "printf '%s' \"$1\" > \"$2\"", "helper", argument, output}, QuoteArguments: true}
	request.Options = systemexec.DefaultOptions()
	request.Directory = dir
	commands := (&service{}).buildStartProcessCommand(request).Commands
	parts := []string{}
	for _, command := range commands {
		parts = append(parts, string(command))
	}
	if data, err := exec.Command("/bin/sh", "-c", strings.Join(parts, "\n")).CombinedOutput(); err != nil {
		t.Fatalf("start: %v %s", err, data)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(output)
		if err == nil {
			if string(data) != argument {
				t.Fatalf("argument changed: %q", data)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(unexpected); !os.IsNotExist(err) {
		t.Fatal("argument executed as shell code")
	}
}
