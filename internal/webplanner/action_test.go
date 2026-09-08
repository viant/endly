package webplanner

import (
	"sync"
	"testing"
	"time"

	"github.com/viant/endly"
)

type blockingManager struct {
	endly.Manager
	started chan struct{}
	once    sync.Once
}

func (m *blockingManager) Run(context *endly.Context, request interface{}) (interface{}, error) {
	m.once.Do(func() { close(m.started) })
	<-context.Background().Done()
	return nil, context.Background().Err()
}

func TestLiveCommandCanBeCancelled(t *testing.T) {
	baseManager := endly.New()
	ctx := baseManager.NewContext(nil)
	defer ctx.Close()
	manager := &blockingManager{Manager: baseManager, started: make(chan struct{})}
	service := NewService(&Config{Token: "secret"})
	service.manager = manager
	service.context = ctx
	service.started = true
	service.opened = true
	service.startLiveCommand("page.title()")
	select {
	case <-manager.started:
	case <-time.After(time.Second):
		t.Fatal("live command did not start")
	}
	service.mux.Lock()
	cancel := service.liveCancel
	service.mux.Unlock()
	if cancel == nil {
		t.Fatal("live cancel function was not installed")
	}
	cancel()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		service.mux.Lock()
		running := service.liveRunning
		service.mux.Unlock()
		if !running {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("live command did not stop after cancellation")
}
