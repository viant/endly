package mobile

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type SegmentStarter func(context.Context, int) (*Process, error)
type SegmentFinalizer func(context.Context, int) (string, error)

type SegmentedCapture struct {
	duration time.Duration
	start    SegmentStarter
	finish   SegmentFinalizer
	stop     chan struct{}
	done     chan struct{}
	once     sync.Once
	mu       sync.Mutex
	paths    []string
	errors   []string
}

func StartSegmentedCapture(duration time.Duration, start SegmentStarter, finish SegmentFinalizer) (*SegmentedCapture, error) {
	if duration <= 0 || start == nil || finish == nil {
		return nil, fmt.Errorf("positive segment duration, starter, and finalizer are required")
	}
	result := &SegmentedCapture{
		duration: duration, start: start, finish: finish,
		stop: make(chan struct{}), done: make(chan struct{}), paths: []string{}, errors: []string{},
	}
	started := make(chan error, 1)
	go result.run(started)
	if err := <-started; err != nil {
		return nil, err
	}
	return result, nil
}

func (s *SegmentedCapture) run(started chan<- error) {
	defer close(s.done)
	for index := 0; ; index++ {
		process, err := s.start(context.Background(), index)
		if index == 0 {
			started <- err
			close(started)
		}
		if err != nil {
			s.appendError(fmt.Sprintf("start segment %d: %v", index, err))
			return
		}
		timer := time.NewTimer(s.duration)
		stopping := false
		select {
		case <-s.stop:
			stopping = true
		case <-timer.C:
		case processErr := <-process.Done:
			if processErr != nil {
				s.appendError(fmt.Sprintf("record segment %d: %v", index, processErr))
			}
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		if process.Stop != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := process.Stop(stopCtx); err != nil && err != context.DeadlineExceeded {
				s.appendError(fmt.Sprintf("stop segment %d: %v", index, err))
			}
			cancel()
		}
		path, err := s.finish(context.Background(), index)
		if err != nil {
			s.appendError(fmt.Sprintf("finish segment %d: %v", index, err))
		} else if path != "" {
			s.mu.Lock()
			s.paths = append(s.paths, path)
			s.mu.Unlock()
		}
		if stopping {
			return
		}
		select {
		case <-s.stop:
			return
		default:
		}
	}
}

func (s *SegmentedCapture) Stop() ([]string, []string) {
	if s == nil {
		return nil, nil
	}
	s.once.Do(func() { close(s.stop) })
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.paths...), append([]string(nil), s.errors...)
}

func (s *SegmentedCapture) appendError(value string) {
	s.mu.Lock()
	s.errors = append(s.errors, value)
	s.mu.Unlock()
}
