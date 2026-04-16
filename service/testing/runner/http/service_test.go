package http

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

func optionsToMap(options []*toolbox.HttpOptions) map[string]interface{} {
	out := make(map[string]interface{}, len(options))
	for _, o := range options {
		out[o.Key] = o.Value
	}
	return out
}

func newServiceForTest() *service {
	return &service{AbstractService: endly.NewAbstractService(ServiceID)}
}

func newContextWithState(values map[string]interface{}) *endly.Context {
	context := endly.New().NewContext(toolbox.NewContext())
	state := context.State()
	for k, v := range values {
		state.Put(k, v)
	}
	return context
}

// TestApplyDefaultTimeout_NilContext ensures the helper never panics when called
// without a context (keeps the function unit-testable and tolerant of callers
// that haven't yet threaded context through).
func TestApplyDefaultTimeout_NilContext(t *testing.T) {
	s := newServiceForTest()

	got := optionsToMap(s.applyDefaultTimeoutIfNeeded(nil, nil))

	assert.Equal(t, 120000, got["RequestTimeoutMs"])
	assert.Equal(t, 120000, got["TimeoutMs"])
}

// TestApplyDefaultTimeout_NoStateNoOptions preserves the pre-existing behavior:
// with nothing supplied, the hardcoded 120s floor applies.
func TestApplyDefaultTimeout_NoStateNoOptions(t *testing.T) {
	s := newServiceForTest()
	ctx := newContextWithState(nil)

	got := optionsToMap(s.applyDefaultTimeoutIfNeeded(ctx, nil))

	assert.Equal(t, 120000, got["RequestTimeoutMs"])
	assert.Equal(t, 120000, got["TimeoutMs"])
	assert.Len(t, got, 2)
}

// TestApplyDefaultTimeout_StateDefaultsApplied demonstrates the new feature:
// a map under state[httpDefaults] is merged into the options for every
// http/runner call, with any floor key not covered by state filled in.
func TestApplyDefaultTimeout_StateDefaultsApplied(t *testing.T) {
	s := newServiceForTest()
	ctx := newContextWithState(map[string]interface{}{
		HttpDefaultsKey: map[string]interface{}{
			"TimeoutMs":               300000,
			"ResponseHeaderTimeoutMs": 180000,
		},
	})

	got := optionsToMap(s.applyDefaultTimeoutIfNeeded(ctx, nil))

	assert.Equal(t, 300000, got["TimeoutMs"])
	assert.Equal(t, 180000, got["ResponseHeaderTimeoutMs"])
	assert.Equal(t, 120000, got["RequestTimeoutMs"], "floor fills in keys state didn't set")
}

// TestApplyDefaultTimeout_ActionOptionsWin confirms precedence: per-action
// options beat state defaults, which beat the hardcoded floor.
func TestApplyDefaultTimeout_ActionOptionsWin(t *testing.T) {
	s := newServiceForTest()
	ctx := newContextWithState(map[string]interface{}{
		HttpDefaultsKey: map[string]interface{}{
			"TimeoutMs":        300000,
			"RequestTimeoutMs": 300000,
		},
	})
	action := []*toolbox.HttpOptions{
		{Key: "TimeoutMs", Value: 500}, // per-action wins
	}

	got := optionsToMap(s.applyDefaultTimeoutIfNeeded(ctx, action))

	assert.Equal(t, 500, got["TimeoutMs"], "per-action option is preserved")
	assert.Equal(t, 300000, got["RequestTimeoutMs"], "state fills in where action didn't set")
}

// TestApplyDefaultTimeout_FixesSilentOverride guards against regression of a
// prior bug: setting any single option (e.g. FollowRedirects) used to drop the
// 120s floor entirely because of a len(options)>0 short-circuit.
func TestApplyDefaultTimeout_FixesSilentOverride(t *testing.T) {
	s := newServiceForTest()
	action := []*toolbox.HttpOptions{
		{Key: "FollowRedirects", Value: false},
	}

	got := optionsToMap(s.applyDefaultTimeoutIfNeeded(nil, action))

	assert.Equal(t, false, got["FollowRedirects"])
	assert.Equal(t, 120000, got["RequestTimeoutMs"], "floor still applies alongside unrelated options")
	assert.Equal(t, 120000, got["TimeoutMs"])
}

// TestApplyDefaultTimeout_ArbitraryKeys ensures httpDefaults is not limited to
// timeouts — any key accepted by toolbox.HttpOptions (e.g. MaxIdleConns,
// FollowRedirects) can be set globally.
func TestApplyDefaultTimeout_ArbitraryKeys(t *testing.T) {
	s := newServiceForTest()
	ctx := newContextWithState(map[string]interface{}{
		HttpDefaultsKey: map[string]interface{}{
			"FollowRedirects": false,
			"MaxIdleConns":    50,
		},
	})

	got := optionsToMap(s.applyDefaultTimeoutIfNeeded(ctx, nil))

	assert.Equal(t, false, got["FollowRedirects"])
	assert.Equal(t, 50, got["MaxIdleConns"])
}

// TestApplyDefaultTimeout_StateNotAMap silently ignores a misconfigured
// httpDefaults value rather than panicking, so a typo in the user's workflow
// can't take down a regression run.
func TestApplyDefaultTimeout_StateNotAMap(t *testing.T) {
	s := newServiceForTest()
	ctx := newContextWithState(map[string]interface{}{
		HttpDefaultsKey: "not-a-map",
	})

	assert.NotPanics(t, func() {
		got := s.applyDefaultTimeoutIfNeeded(ctx, nil)
		assert.Len(t, got, 2, "only the floor should remain when state is malformed")
		m := optionsToMap(got)
		assert.Equal(t, 120000, m["TimeoutMs"])
		assert.Equal(t, 120000, m["RequestTimeoutMs"])
	})
}

// TestApplyDefaultTimeout_ExplicitZeroPreserved guards the corner case where a
// user intentionally sets TimeoutMs: 0 (meaning "no client-side timeout").
// Presence is checked by key name only, so the floor must not override.
func TestApplyDefaultTimeout_ExplicitZeroPreserved(t *testing.T) {
	s := newServiceForTest()
	action := []*toolbox.HttpOptions{
		{Key: "TimeoutMs", Value: 0},
	}

	got := optionsToMap(s.applyDefaultTimeoutIfNeeded(nil, action))

	assert.Equal(t, 0, got["TimeoutMs"], "explicit zero must not be overridden by the floor")
	assert.Equal(t, 120000, got["RequestTimeoutMs"])
}

// TestTuneDefaultTransport_IdempotentAndRaceFree documents that the global
// transport tuning is one-shot (sync.Once) and safe under concurrent invocation
// — formerly an unsynchronized write on every http/runner call.
func TestTuneDefaultTransport_IdempotentAndRaceFree(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tuneDefaultTransport()
		}()
	}
	wg.Wait()

	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		assert.Equal(t, 100, transport.MaxIdleConnsPerHost)
	}
}
