package cli

import (
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly/model/msg"
)

func TestCLIListenerRetainsConcurrentEvents(t *testing.T) {
	runner := New()
	runner.Renderer = NewRenderer(io.Discard, 120)
	runner.context = runner.manager.NewContext(nil)
	defer runner.context.Close()
	runner.report = &ReportSummaryEvent{}
	listener := runner.AsListener()
	const workers, perWorker = 8, 20
	start := make(chan struct{})
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			<-start
			for index := 0; index < perWorker; index++ {
				listener(msg.NewEvent(worker*perWorker + index))
			}
		}(worker)
	}
	close(start)
	group.Wait()
	seen := map[int]bool{}
	count := 0
	for _, tag := range runner.tags {
		for _, event := range tag.Events {
			id, ok := event.Value().(int)
			require.True(t, ok)
			require.False(t, seen[id], "each published event must appear once")
			seen[id] = true
			count++
		}
	}
	require.Equal(t, workers*perWorker, count)
	require.Len(t, seen, workers*perWorker)
}
