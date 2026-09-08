package mobile

import (
	"context"
	"sync"
)

type CleanupError struct {
	Name  string
	Error string
}

type cleanupEntry struct {
	name string
	run  func(context.Context) error
}

// CleanupStack provides the LIFO, continue-on-error behavior that Endly's FIFO
// Context.Deffer list does not provide by itself.
type CleanupStack struct {
	mu      sync.Mutex
	entries []cleanupEntry
	closed  bool
	errors  []CleanupError
}

func (s *CleanupStack) Push(name string, cleanup func(context.Context) error) bool {
	if cleanup == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.entries = append(s.entries, cleanupEntry{name: name, run: cleanup})
	return true
}

func (s *CleanupStack) Close(ctx context.Context) []CleanupError {
	s.mu.Lock()
	if s.closed {
		result := append([]CleanupError(nil), s.errors...)
		s.mu.Unlock()
		return result
	}
	s.closed = true
	entries := append([]cleanupEntry(nil), s.entries...)
	s.mu.Unlock()

	errors := []CleanupError{}
	for i := len(entries) - 1; i >= 0; i-- {
		if err := entries[i].run(ctx); err != nil {
			errors = append(errors, CleanupError{Name: entries[i].name, Error: err.Error()})
		}
	}
	s.mu.Lock()
	s.errors = errors
	s.mu.Unlock()
	return append([]CleanupError(nil), errors...)
}
