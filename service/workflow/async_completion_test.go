package workflow

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/model/msg"
)

func TestAsyncCompletionWaitsForFinalEventListener(t *testing.T) {
	for _, missing := range []bool{false, true} {
		name := "success"
		if missing {
			name = "error"
		}
		t.Run(name, func(t *testing.T) {
			_, ctx := runtimeURLFixture(t)
			filename := filepath.Join(t.TempDir(), "run.yaml")
			if !missing {
				require.NoError(t, os.WriteFile(filename, []byte("pipeline:\n  check:\n    action: nop\n"), 0600))
			}
			entered, release, completed := make(chan struct{}), make(chan struct{}), make(chan struct{})
			ctx.SetListener(func(event msg.Event) {
				if _, ok := event.Value().(*EndEvent); ok {
					close(entered)
					<-release
				}
			})
			defer func() {
				close(release)
				select {
				case <-completed:
				case <-time.After(3 * time.Second):
					t.Error("async completion did not return after listener release")
				}
			}()
			var result RunResponse
			require.NoError(t, endly.Run(ctx, &RunRequest{URL: filename, Async: true}, &result))
			go func() { ctx.Wait.Wait(); close(completed) }()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("final event listener not invoked")
			}
			select {
			case <-completed:
				t.Fatal("async completion returned before final event listener completed")
			case <-time.After(25 * time.Millisecond):
			}
		})
	}
}
