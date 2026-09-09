//go:build darwin

package control

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/endly"
	managerservice "github.com/viant/endly/service/manager"
	_ "github.com/viant/endly/service/testing/runner/ios"
)

func TestRemoteWorkerPreservesIOSLifecycleAcrossAuthenticatedOperations(t *testing.T) {
	leaseDirectory := t.TempDir()
	t.Setenv("ENDLY_MOBILE_LEASE_DIR", leaseDirectory)
	var appiumPayload map[string]interface{}
	appium := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		var value interface{}
		switch request.Method + " " + request.URL.Path {
		case "GET /status":
			value = map[string]interface{}{"ready": true}
		case "POST /session":
			if err := json.NewDecoder(request.Body).Decode(&appiumPayload); err != nil {
				t.Error(err)
			}
			value = map[string]interface{}{"sessionId": "ios-worker-backend", "capabilities": map[string]interface{}{}}
		case "POST /session/ios-worker-backend/elements":
			value = []map[string]interface{}{{"element-6066-11e4-a52e-4f735466cecf": "title"}}
		case "GET /session/ios-worker-backend/element/title/text":
			value = "Remote iOS worker"
		default:
			value = nil
		}
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{"value": value})
	}))
	defer appium.Close()

	runtime := managerservice.New(endly.New, managerservice.WithAllowedActions("ios:*"))
	worker := httptest.NewServer(NewAuthenticated(runtime, mobileWorkerAuthenticator{}))
	defer worker.Close()
	client := worker.Client()
	session := &managerservice.SessionInfo{}
	workerJSON(t, client, http.MethodPost, worker.URL+"/v1/endly/sessions", map[string]interface{}{"name": "ios-worker"}, session, http.StatusCreated)

	registered := workerAction(t, client, runtime, worker.URL, session.SessionID, "ios", "destination-register", map[string]interface{}{
		"Provider": "remote-mac", "DeviceID": "ios-worker-device", "PlatformVersion": "18.5",
	})
	lease := mapField(t, registered, "Lease")
	serverStarted := workerAction(t, client, runtime, worker.URL, session.SessionID, "ios", "server-start", map[string]interface{}{
		"Destination": lease, "Mode": "external", "ServerURL": appium.URL,
	})
	serverHandle := mapField(t, serverStarted, "Server")
	opened := workerAction(t, client, runtime, worker.URL, session.SessionID, "ios", "open", map[string]interface{}{
		"SessionID": "remote-ios-session", "Destination": lease, "Server": serverHandle,
		"AppReference": "worker://artifacts/Fixture.app",
		"Capabilities": map[string]interface{}{"worker:options": map[string]interface{}{"lane": "ios"}},
	})
	sessionHandle := mapField(t, opened, "Session")
	workerAction(t, client, runtime, worker.URL, session.SessionID, "ios", "run", map[string]interface{}{
		"SessionID":       mapScalar(t, sessionHandle, "ID"),
		"Commands":        []interface{}{`expect(app.getByTestId("title")).toHaveText("Remote iOS worker", 1000)`},
		"ActionTimeoutMs": 1000, "PollIntervalMs": 10,
	})
	workerAction(t, client, runtime, worker.URL, session.SessionID, "ios", "close", map[string]interface{}{
		"SessionID": mapScalar(t, sessionHandle, "ID"),
	})
	workerAction(t, client, runtime, worker.URL, session.SessionID, "ios", "server-stop", map[string]interface{}{"Server": serverHandle})
	workerAction(t, client, runtime, worker.URL, session.SessionID, "ios", "destination-release", map[string]interface{}{"Lease": lease})

	alwaysMatch := mapField(t, mapField(t, appiumPayload, "capabilities"), "alwaysMatch")
	if mapScalar(t, alwaysMatch, "appium:app") != "worker://artifacts/Fixture.app" || mapScalar(t, alwaysMatch, "appium:udid") != "ios-worker-device" {
		t.Fatalf("remote iOS worker Appium payload: %+v", alwaysMatch)
	}
	entries, err := os.ReadDir(leaseDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			t.Fatalf("remote iOS worker lease remains: %s", entry.Name())
		}
	}
	workerJSON(t, client, http.MethodDelete, worker.URL+"/v1/endly/sessions/"+session.SessionID, nil, nil, http.StatusNoContent)
}
