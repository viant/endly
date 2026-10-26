package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/viant/endly"
)

func TestGeneratedRouteUsesInvocationRequest(t *testing.T) {
	var received []network.CreateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request network.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		received = append(received, request)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"Id":"owned-network"}`))
	}))
	defer server.Close()
	dockerClient, err := client.NewClientWithOpts(client.WithHost(server.URL), client.WithVersion("1.44"))
	if err != nil {
		t.Fatal(err)
	}
	defer dockerClient.Close()
	ctx := endly.New().NewContext(nil)
	if err := ctx.Replace(clientKey, &CtxClient{Client: dockerClient, Context: context.Background()}); err != nil {
		t.Fatal(err)
	}
	routes, err := BuildRoutes(dockerClient, GetCtxClient)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range routes {
		if route.Action != "networkCreate" {
			continue
		}
		for _, name := range []string{"creative-one", "creative-two"} {
			request := route.RequestProvider().(*NetworkCreateRequest)
			request.Name = name
			request.Driver = "bridge"
			request.Labels = map[string]string{"owner": name}
			if _, err := route.Handler(ctx, request); err != nil {
				t.Fatal(err)
			}
		}
		if len(received) != 2 {
			t.Fatalf("got %d requests", len(received))
		}
		for i, name := range []string{"creative-one", "creative-two"} {
			if received[i].Name != name || received[i].Driver != "bridge" || received[i].Labels["owner"] != name {
				t.Fatalf("request %d lost invocation fields: %+v", i, received[i])
			}
		}
		return
	}
	t.Fatal("networkCreate route missing")
}
