package control

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/viant/endly"
	managerservice "github.com/viant/endly/service/manager"
	_ "github.com/viant/endly/service/testing/runner/android"
	scyjwt "github.com/viant/scy/auth/jwt"
)

type mobileWorkerAuthenticator struct{}

func (mobileWorkerAuthenticator) Authenticate(_ context.Context, token string) (*scyjwt.Claims, error) {
	if token != "mobile-worker-token" {
		return nil, fmt.Errorf("invalid token")
	}
	return &scyjwt.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "mobile-ci"}}, nil
}

func TestRemoteWorkerPreservesAndroidLifecycleAcrossAuthenticatedOperations(t *testing.T) {
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
			value = map[string]interface{}{"sessionId": "worker-backend", "capabilities": map[string]interface{}{}}
		case "POST /session/worker-backend/elements":
			value = []map[string]interface{}{{"element-6066-11e4-a52e-4f735466cecf": "title"}}
		case "GET /session/worker-backend/element/title/text":
			value = "Remote worker"
		default:
			value = nil
		}
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{"value": value})
	}))
	defer appium.Close()

	runtime := managerservice.New(endly.New, managerservice.WithAllowedActions("android:*"))
	worker := httptest.NewServer(NewAuthenticated(runtime, mobileWorkerAuthenticator{}))
	defer worker.Close()
	client := worker.Client()
	session := &managerservice.SessionInfo{}
	workerJSON(t, client, http.MethodPost, worker.URL+"/v1/endly/sessions", map[string]interface{}{"name": "android-worker"}, session, http.StatusCreated)

	registered := workerAction(t, client, runtime, worker.URL, session.SessionID, "android", "device-register", map[string]interface{}{
		"Provider": "remote-worker", "DeviceID": "worker-device", "PlatformVersion": "15",
	})
	lease := mapField(t, registered, "Lease")
	serverStarted := workerAction(t, client, runtime, worker.URL, session.SessionID, "android", "server-start", map[string]interface{}{
		"Lease": lease, "Mode": "external", "ServerURL": appium.URL,
	})
	serverHandle := mapField(t, serverStarted, "Server")
	opened := workerAction(t, client, runtime, worker.URL, session.SessionID, "android", "open", map[string]interface{}{
		"SessionID": "remote-worker-session", "Lease": lease, "Server": serverHandle,
		"AppReference": "worker://artifacts/app.apk", "TestIDStrategy": "accessibilityId",
		"Capabilities": map[string]interface{}{"worker:options": map[string]interface{}{"lane": "one"}},
	})
	sessionHandle := mapField(t, opened, "Session")
	workerAction(t, client, runtime, worker.URL, session.SessionID, "android", "run", map[string]interface{}{
		"SessionID":       mapScalar(t, sessionHandle, "ID"),
		"Commands":        []interface{}{`expect(app.getByTestId("title")).toHaveText("Remote worker", 1000)`},
		"ActionTimeoutMs": 1000, "PollIntervalMs": 10,
	})
	workerAction(t, client, runtime, worker.URL, session.SessionID, "android", "close", map[string]interface{}{
		"SessionID": mapScalar(t, sessionHandle, "ID"),
	})
	workerAction(t, client, runtime, worker.URL, session.SessionID, "android", "server-stop", map[string]interface{}{"Server": serverHandle})
	workerAction(t, client, runtime, worker.URL, session.SessionID, "android", "device-release", map[string]interface{}{"Lease": lease})

	alwaysMatch := mapField(t, mapField(t, appiumPayload, "capabilities"), "alwaysMatch")
	if mapScalar(t, alwaysMatch, "appium:app") != "worker://artifacts/app.apk" || mapScalar(t, alwaysMatch, "appium:udid") != "worker-device" {
		t.Fatalf("remote worker Appium payload: %+v", alwaysMatch)
	}
	entries, err := os.ReadDir(leaseDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			t.Fatalf("remote worker lease remains: %s", entry.Name())
		}
	}
	workerJSON(t, client, http.MethodDelete, worker.URL+"/v1/endly/sessions/"+session.SessionID, nil, nil, http.StatusNoContent)
}

func workerAction(t *testing.T, client *http.Client, runtime *managerservice.Service, endpoint, sessionID, service, action string, input map[string]interface{}) map[string]interface{} {
	t.Helper()
	operation := &managerservice.Operation{}
	workerJSON(t, client, http.MethodPost, endpoint+"/v1/endly/sessions/"+sessionID+"/operations", map[string]interface{}{
		"kind":   "action",
		"action": map[string]interface{}{"service": service, "action": action, "request": input, "timeoutMillis": 30_000},
	}, operation, http.StatusAccepted)
	waitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	finished, err := runtime.WaitOperation(waitCtx, sessionID, operation.ID)
	if err != nil {
		t.Fatalf("wait for %s:%s: %v", service, action, err)
	}
	if finished.Status != managerservice.OperationSucceeded {
		t.Fatalf("remote %s:%s status=%s error=%s input=%+v events=%+v", service, action, finished.Status, finished.Error, input, finished.Events)
	}
	response, ok := fieldCI(finished.Result, "response").(map[string]interface{})
	if !ok {
		t.Fatalf("remote %s:%s response missing: %+v", service, action, finished.Result)
	}
	return response
}

func workerJSON(t *testing.T, client *http.Client, method, URL string, input, target interface{}, expectedStatus int) {
	t.Helper()
	var body *bytes.Reader
	if input == nil {
		body = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, URL, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer mobile-worker-token")
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != expectedStatus {
		t.Fatalf("%s %s returned %s", method, URL, response.Status)
	}
	if target != nil && response.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
}

func mapField(t *testing.T, value map[string]interface{}, name string) map[string]interface{} {
	t.Helper()
	result, ok := fieldCI(value, name).(map[string]interface{})
	if !ok {
		t.Fatalf("field %s was not a map in %+v", name, value)
	}
	return result
}

func mapScalar(t *testing.T, value map[string]interface{}, name string) interface{} {
	t.Helper()
	result := fieldCI(value, name)
	if result == nil {
		t.Fatalf("field %s was missing in %+v", name, value)
	}
	return result
}

func fieldCI(value map[string]interface{}, name string) interface{} {
	for key, item := range value {
		if strings.EqualFold(key, name) {
			return item
		}
	}
	return nil
}
