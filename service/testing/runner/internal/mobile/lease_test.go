package mobile

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

func TestLeaseFenceSurvivesGenericJSONRoundTrip(t *testing.T) {
	store := NewLeaseStore(t.TempDir())
	handle, err := store.Acquire(context.Background(), "json-safe")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(handle)
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Fatal(err)
	}
	reencoded, err := json.Marshal(generic)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &LeaseHandle{}
	if err := json.Unmarshal(reencoded, decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Fence != handle.Fence || handle.Fence > 1<<53-1 {
		t.Fatalf("fence changed across JSON: before=%d after=%d", handle.Fence, decoded.Fence)
	}
}

func TestLeaseStoreAcquireValidateRefreshRelease(t *testing.T) {
	store := NewLeaseStore(t.TempDir())
	store.TTL = time.Minute
	handle, err := store.Acquire(context.Background(), "android:avd:endly")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Validate(handle); err != nil {
		t.Fatal(err)
	}
	previousExpiry := handle.ExpiresAt
	time.Sleep(time.Millisecond)
	if err := store.Refresh(handle); err != nil || !handle.ExpiresAt.After(previousExpiry) {
		t.Fatalf("refresh failed: handle=%+v err=%v", handle, err)
	}
	if _, err := store.Acquire(context.Background(), handle.Key); !errors.Is(err, ErrLeaseHeld) {
		t.Fatalf("expected held error, got %v", err)
	}
	stale := *handle
	stale.Token = "wrong"
	if err := store.Validate(&stale); err == nil {
		t.Fatal("expected stale token validation failure")
	}
	if err := store.Release(handle); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(handle.Path); !os.IsNotExist(err) {
		t.Fatalf("lease file remains: %v", err)
	}
}

func TestLeaseStoreConcurrentAcquireHasOneWinner(t *testing.T) {
	store := NewLeaseStore(t.TempDir())
	const workers = 12
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(workers)
	winners := make(chan *LeaseHandle, workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wait.Done()
			<-start
			handle, err := store.Acquire(context.Background(), "appium:port:4723")
			if err == nil {
				winners <- handle
			}
		}()
	}
	close(start)
	wait.Wait()
	close(winners)
	var winner *LeaseHandle
	count := 0
	for candidate := range winners {
		winner = candidate
		count++
	}
	if count != 1 {
		t.Fatalf("expected one winner, got %d", count)
	}
	if err := store.Release(winner); err != nil {
		t.Fatal(err)
	}
}
