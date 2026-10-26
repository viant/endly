package webdriver

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestInteractionTimingsPreserveNestedWallSpansAndErrors(t *testing.T) {
	now := time.Unix(1, 0)
	trace := newInteractionTimings(func() time.Time { return now })
	parent := startInteractionPhase(trace, "element.invoke")
	now = now.Add(10 * time.Millisecond)
	denied := errors.New("fixture error")
	err := timedInteraction(trace, "cdp.Runtime.evaluate", func() error { now = now.Add(600 * time.Millisecond); return denied })
	now = now.Add(5 * time.Millisecond)
	parent()
	require.ErrorIs(t, err, denied)
	spans := trace.snapshot()
	require.Len(t, spans, 2)
	require.Equal(t, "cdp.Runtime.evaluate", spans[0].Phase)
	require.Equal(t, float64(10), spans[0].StartedMs)
	require.Equal(t, float64(600), spans[0].DurationMs)
	require.Equal(t, float64(615), spans[1].DurationMs)
	spans[0].Phase = "modified"
	require.Equal(t, "cdp.Runtime.evaluate", trace.snapshot()[0].Phase)
}

func TestInteractionTimingsDisabledDoesNotReadClockOrChangeErrors(t *testing.T) {
	denied := errors.New("fixture denied")
	require.ErrorIs(t, timedInteraction(nil, "element.lookup", func() error { return denied }), denied)
	startInteractionPhase(nil, "element.lookup")()
	allocations := testing.AllocsPerRun(100, func() {
		_ = timedInteraction(nil, "element.lookup", func() error { return denied })
	})
	require.Zero(t, allocations, "disabled tracing must not allocate")
}

func TestInteractionTimingsCaptureCDPWithoutPayloadsAndAllowDisable(t *testing.T) {
	fake := newFakeCDPServer(t)
	driver, err := newDirectCDPDriver(fake.server.URL)
	require.NoError(t, err)
	defer driver.Quit()
	trace := newInteractionTimings(time.Now)
	driver.timings.Store(trace)
	title, err := driver.Title()
	require.NoError(t, err)
	require.Equal(t, "Direct CDP", title)
	require.NotEmpty(t, trace.snapshot())
	require.Equal(t, "cdp.Runtime.evaluate", trace.snapshot()[0].Phase)
	before := len(trace.snapshot())
	driver.timings.Store(nil)
	_, err = driver.Title()
	require.NoError(t, err)
	require.Len(t, trace.snapshot(), before)
	require.Equal(t, "cdp.other", cdpTimingPhase("untrusted-secret-payload"))
}
