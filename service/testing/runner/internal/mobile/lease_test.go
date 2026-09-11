package mobile

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

func TestLeaseRecoveryProcess(t *testing.T) {
	directory := os.Getenv("ENDLY_LEASE_RECOVERY_CHILD")
	if directory == "" {
		return
	}
	_, err := NewLeaseStore(directory).Acquire(context.Background(), "stale")
	if err == nil {
		fmt.Println("winner")
	} else if errors.Is(err, ErrLeaseHeld) {
		fmt.Println("held")
	} else {
		t.Fatal(err)
	}
	// Keep the winning owner alive until every contender has finished acquiring.
	var signal [1]byte
	_, _ = os.Stdin.Read(signal[:])
}

func TestLeaseRecoveryAcrossProcesses(t *testing.T) {
	directory := t.TempDir()
	data, _ := json.Marshal(&LeaseHandle{Key: "stale", PID: -1})
	if err := os.WriteFile(NewLeaseStore(directory).leasePath("stale"), data, 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var commands []*exec.Cmd
	var readers []*bufio.Scanner
	for i := 0; i < 8; i++ {
		cmd := exec.Command(executable, "-test.run=^TestLeaseRecoveryProcess$")
		cmd.Env = append(os.Environ(), "ENDLY_LEASE_RECOVERY_CHILD="+directory)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { stdin.Close(); _ = cmd.Wait() })
		commands = append(commands, cmd)
		readers = append(readers, bufio.NewScanner(stdout))
	}
	winners := 0
	for i, scanner := range readers {
		if !scanner.Scan() {
			t.Fatalf("child %d produced no result", commands[i].Process.Pid)
		}
		switch scanner.Text() {
		case "winner":
			winners++
		case "held":
		default:
			t.Fatalf("unexpected child result: %s", scanner.Text())
		}
	}
	if winners != 1 {
		t.Fatalf("got %d simultaneous owners", winners)
	}
}

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

func TestSeparateStoresRecoverStaleLeaseWithOneWinner(t *testing.T) {
	directory := t.TempDir()
	store := NewLeaseStore(directory)
	stale := &LeaseHandle{Key: "stale", PID: -1}
	data, _ := json.Marshal(stale)
	if err := os.WriteFile(store.leasePath("stale"), data, 0600); err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	winners := make(chan *LeaseHandle, 24)
	for i := 0; i < 24; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			handle, err := NewLeaseStore(directory).Acquire(context.Background(), "stale")
			if err == nil {
				winners <- handle
			} else if !errors.Is(err, ErrLeaseHeld) {
				t.Error(err)
			}
		}()
	}
	wait.Wait()
	close(winners)
	if len(winners) != 1 {
		t.Fatalf("got %d owners", len(winners))
	}
	for handle := range winners {
		if err := store.Validate(handle); err != nil {
			t.Fatal(err)
		}
	}
}
