package msg

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/viant/scy/cred"
)

func TestGCPEmulatorEmptyPullIsBounded(t *testing.T) {
	host := os.Getenv("PUBSUB_EMULATOR_HOST")
	if host == "" {
		t.Skip("local Pub/Sub emulator required")
	}
	name, _, err := net.SplitHostPort(host)
	if err != nil || !(name == "localhost" || net.ParseIP(name).IsLoopback()) {
		t.Fatal("test requires a loopback Pub/Sub emulator")
	}
	project := "viant-imp-partners-local"
	id := fmt.Sprintf("endly-empty-%d", time.Now().UnixNano())
	credentials := &cred.Generic{}
	credentials.ProjectID = project
	client, err := newCloudPubSub(credentials, "/projects/"+project+"/topics/"+id, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	topic := NewResourceSetup(ResourceTypeTopic, "/projects/"+project+"/topics/"+id, "", false, nil)
	if err = topic.Init(); err != nil {
		t.Fatal(err)
	}
	if _, err = client.SetupResource(topic); err != nil {
		t.Fatal(err)
	}
	defer client.DeleteResource(&topic.Resource)
	subscription := NewResourceSetup(ResourceTypeSubscription, "/projects/"+project+"/subscriptions/"+id, "", false, NewConfig(id))
	if err = subscription.Init(); err != nil {
		t.Fatal(err)
	}
	if _, err = client.SetupResource(subscription); err != nil {
		t.Fatal(err)
	}
	defer client.DeleteResource(&subscription.Resource)
	// The caller deadline is a test safety bound, not the service timeout.
	// The old implementation ignores it and never cancels an empty count=1 pull.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	messages, err := client.PullN(ctx, &subscription.Resource, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 || time.Since(started) >= time.Second || ctx.Err() != nil {
		t.Fatalf("empty pull did not respect service timeout: messages=%d elapsed=%s caller=%v", len(messages), time.Since(started), ctx.Err())
	}
}
