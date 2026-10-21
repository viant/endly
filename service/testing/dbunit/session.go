package dbunit

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"

	core "github.com/viant/dbunit"
	"github.com/viant/dbunit/fixture"
	"github.com/viant/endly"
)

var sessionKey = (*session)(nil)

var holderMu sync.Mutex

type connection struct {
	db         *sql.DB
	driver     string
	owned      bool
	nativeLoad *core.NativeLoadOptions
}

type session struct {
	mu        sync.RWMutex
	stateMu   sync.Mutex
	closed    bool
	cancel    context.CancelFunc
	done      context.Context
	databases map[string]*connection
	fixtures  map[string]fixture.Resolver
	once      sync.Once
}

func sessionFor(ctx *endly.Context) (*session, error) {
	holderMu.Lock()
	defer holderMu.Unlock()
	var result *session
	if ctx.Contains(sessionKey) {
		ctx.GetInto(sessionKey, &result)
	}
	if result != nil {
		return result, nil
	}
	if ctx.IsClosed() {
		return nil, fmt.Errorf("dbunit session is closed")
	}
	done, cancel := context.WithCancel(context.Background())
	result = &session{done: done, cancel: cancel, databases: map[string]*connection{}, fixtures: map[string]fixture.Resolver{}}
	if err := ctx.Put(sessionKey, result); err != nil {
		cancel()
		return nil, err
	}
	ctx.Deffer(result.close)
	return result, nil
}

func (s *session) close() {
	s.once.Do(func() {
		s.cancel()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.closed = true
		names := make([]string, 0, len(s.databases))
		for name := range s.databases {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			item := s.databases[name]
			if item.owned {
				_ = item.db.Close()
			}
		}
	})
}

// BindDatabase registers a caller-owned pool. The session never closes this pool.
// An optional native loader capability is an explicit caller certification, copied per binding.
func BindDatabase(ctx *endly.Context, name string, db *sql.DB, nativeLoad ...core.NativeLoadOptions) error {
	if len(nativeLoad) > 1 {
		return fmt.Errorf("at most one explicit native loader capability is allowed")
	}
	if name == "" || db == nil {
		return fmt.Errorf("database name and pool are required")
	}
	s, err := sessionFor(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.done.Err() != nil {
		return fmt.Errorf("dbunit session is closed")
	}
	if _, ok := s.databases[name]; ok {
		return fmt.Errorf("database %q is already registered", name)
	}
	binding := &connection{db: db}
	if len(nativeLoad) == 1 {
		capability := nativeLoad[0]
		binding.nativeLoad = &capability
	}
	s.databases[name] = binding
	return nil
}

// BindFixtures registers an explicit session-local fixture resolver, including embedded filesystems.
func BindFixtures(ctx *endly.Context, name string, resolver fixture.Resolver) error {
	if name == "" || resolver == nil {
		return fmt.Errorf("fixture name and resolver are required")
	}
	s, err := sessionFor(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.done.Err() != nil {
		return fmt.Errorf("dbunit session is closed")
	}
	if _, ok := s.fixtures[name]; ok {
		return fmt.Errorf("fixtures %q are already registered", name)
	}
	s.fixtures[name] = resolver
	return nil
}

// operation holds the session read lock until the operation completes; shutdown cancels first.
func (s *session) operation(ctx *endly.Context) (context.Context, func(), error) {
	s.mu.RLock()
	if s.closed || s.done.Err() != nil {
		s.mu.RUnlock()
		return nil, nil, fmt.Errorf("dbunit session is closed")
	}
	op, cancel := context.WithCancel(ctx.Background())
	stop := context.AfterFunc(s.done, cancel)
	return op, func() { stop(); cancel(); s.mu.RUnlock() }, nil
}
