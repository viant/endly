package mobile

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrLeaseHeld = errors.New("mobile resource lease is held")

type LeaseHandle struct {
	Key       string
	Token     string
	Fence     uint64
	PID       int
	Path      string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type LeaseStore struct {
	Directory string
	TTL       time.Duration
	mu        sync.Mutex
}

func NewLeaseStore(directory string) *LeaseStore {
	if directory == "" {
		directory = os.Getenv("ENDLY_MOBILE_LEASE_DIR")
	}
	if directory == "" {
		directory = filepath.Join(os.TempDir(), "endly-mobile-leases")
	}
	return &LeaseStore{Directory: directory, TTL: 30 * time.Minute}
}

func (s *LeaseStore) Acquire(ctx context.Context, key string) (*LeaseHandle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("lease key is required")
	}
	if s.TTL <= 0 {
		s.TTL = 30 * time.Minute
	}
	if err := os.MkdirAll(s.Directory, 0o700); err != nil {
		return nil, fmt.Errorf("create mobile lease directory: %w", err)
	}
	path := s.leasePath(key)
	for attempts := 0; attempts < 4; attempts++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		handle := &LeaseHandle{
			Key: key, Token: randomLeaseToken(), Fence: randomFence(), PID: os.Getpid(), Path: path,
			CreatedAt: now, ExpiresAt: now.Add(s.TTL),
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			encoder := json.NewEncoder(file)
			encodeErr := encoder.Encode(handle)
			syncErr := file.Sync()
			closeErr := file.Close()
			if encodeErr != nil || syncErr != nil || closeErr != nil {
				_ = os.Remove(path)
				return nil, fmt.Errorf("persist mobile lease: %v %v %v", encodeErr, syncErr, closeErr)
			}
			return handle, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("create mobile lease: %w", err)
		}
		existing, readErr := readLease(path)
		if readErr != nil {
			return nil, fmt.Errorf("read existing mobile lease %s: %w", path, readErr)
		}
		if processAlive(existing.PID) {
			return nil, fmt.Errorf("%w: %s by pid %d until %s", ErrLeaseHeld, key, existing.PID, existing.ExpiresAt.Format(time.RFC3339))
		}
		stalePath := path + ".stale." + randomLeaseToken()
		if renameErr := os.Rename(path, stalePath); renameErr != nil {
			if os.IsNotExist(renameErr) {
				continue
			}
			return nil, fmt.Errorf("quarantine stale mobile lease: %w", renameErr)
		}
		_ = os.Remove(stalePath)
	}
	return nil, fmt.Errorf("acquire mobile lease %q after concurrent retries", key)
}

func (s *LeaseStore) Validate(handle *LeaseHandle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.validateUnlocked(handle)
}

func (s *LeaseStore) validateUnlocked(handle *LeaseHandle) error {
	if handle == nil || handle.Key == "" || handle.Token == "" || handle.Fence == 0 || handle.Path == "" {
		return fmt.Errorf("complete persistent lease handle is required")
	}
	current, err := readLease(handle.Path)
	if err != nil {
		return fmt.Errorf("read current mobile lease: %w", err)
	}
	if current.Key != handle.Key || current.Token != handle.Token || current.Fence != handle.Fence || current.PID != handle.PID {
		return fmt.Errorf("mobile lease fence mismatch for %q", handle.Key)
	}
	if current.PID != os.Getpid() {
		return fmt.Errorf("mobile lease %q belongs to pid %d", handle.Key, current.PID)
	}
	return nil
}

func (s *LeaseStore) Refresh(handle *LeaseHandle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.validateUnlocked(handle); err != nil {
		return err
	}
	handle.ExpiresAt = time.Now().UTC().Add(s.TTL)
	data, err := json.Marshal(handle)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(handle.Path, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(append(data, '\n')); err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func (s *LeaseStore) Release(handle *LeaseHandle) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if handle == nil {
		return nil
	}
	if err := s.validateUnlocked(handle); err != nil {
		if os.IsNotExist(errors.Unwrap(err)) {
			return nil
		}
		return err
	}
	if err := os.Remove(handle.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove mobile lease: %w", err)
	}
	return nil
}

func (s *LeaseStore) leasePath(key string) string {
	hash := sha256.Sum256([]byte(key))
	return filepath.Join(s.Directory, hex.EncodeToString(hash[:16])+".json")
}

func readLease(path string) (*LeaseHandle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	handle := &LeaseHandle{}
	if err := json.Unmarshal(data, handle); err != nil {
		return nil, err
	}
	return handle, nil
}

func randomLeaseToken() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	return hex.EncodeToString(data)
}

func randomFence() uint64 {
	const maxJSONSafeInteger = uint64(1<<53 - 1)
	data := make([]byte, 8)
	if _, err := rand.Read(data); err == nil {
		if value := binary.BigEndian.Uint64(data) & maxJSONSafeInteger; value != 0 {
			return value
		}
	}
	value := uint64(time.Now().UnixNano()) & maxJSONSafeInteger
	if value == 0 {
		return 1
	}
	return value
}
