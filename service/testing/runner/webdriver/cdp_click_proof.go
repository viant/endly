package webdriver

import (
	"context"
	"time"
)

// A native click acknowledgement belongs to the connection, not to the page's
// window object: a successful submit may destroy that object before cleanup.
func (c *directCDPConnection) beginClickProof(name string) (<-chan struct{}, func(), error) {
	proof := make(chan struct{})
	c.mu.Lock()
	if c.clickProofs == nil {
		c.clickProofs = map[string]chan struct{}{}
	}
	c.clickProofs[name] = proof
	c.mu.Unlock()
	release := func() {
		c.mu.Lock()
		delete(c.clickProofs, name)
		c.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = c.command(ctx, "Runtime.removeBinding", map[string]interface{}{"name": name})
	}
	if _, err := c.command(context.Background(), "Runtime.addBinding", map[string]interface{}{"name": name}); err != nil {
		release()
		return nil, nil, err
	}
	return proof, release, nil
}
func clickProofSeen(proof <-chan struct{}, timeout time.Duration) bool {
	select {
	case <-proof:
		return true
	default:
	}
	if timeout <= 0 {
		return false
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-proof:
		return true
	case <-timer.C:
		return false
	}
}
