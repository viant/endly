package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	managerservice "github.com/viant/endly/service/manager"
	scyjwt "github.com/viant/scy/auth/jwt"
)

type testAuthenticator struct{}

func (t *testAuthenticator) Authenticate(_ context.Context, token string) (*scyjwt.Claims, error) {
	if token != "test-token" {
		return nil, errors.New("invalid token")
	}
	return &scyjwt.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "test-user"}}, nil
}

type subjectAuthenticator struct{}

func (s *subjectAuthenticator) Authenticate(_ context.Context, token string) (*scyjwt.Claims, error) {
	return &scyjwt.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: token}}, nil
}

func TestServerSessionLifecycle(t *testing.T) {
	runtime := managerservice.New(endly.New)
	handler := New(runtime)
	request := httptest.NewRequest(http.MethodPost, "/v1/endly/sessions", bytes.NewBufferString(`{"name":"test"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	require.Equal(t, http.StatusCreated, response.StatusCode)
	session := &managerservice.SessionInfo{}
	require.NoError(t, json.NewDecoder(response.Body).Decode(session))
	response.Body.Close()
	require.NotEmpty(t, session.SessionID)

	request = httptest.NewRequest(http.MethodDelete, "/v1/endly/sessions/"+session.SessionID, nil)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	response = recorder.Result()
	require.Equal(t, http.StatusNoContent, response.StatusCode)
	response.Body.Close()
}

func TestServerAuthorization(t *testing.T) {
	runtime := managerservice.New(endly.New)
	handler := NewAuthenticated(runtime, &testAuthenticator{})
	request := httptest.NewRequest(http.MethodPost, "/v1/endly/sessions", bytes.NewBufferString(`{}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	request = httptest.NewRequest(http.MethodPost, "/v1/endly/sessions", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer test-token")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusCreated, recorder.Code)
}

func TestLoopbackAddress(t *testing.T) {
	require.True(t, isLoopbackAddress("127.0.0.1:8080"))
	require.True(t, isLoopbackAddress("[::1]:8080"))
	require.False(t, isLoopbackAddress(":8080"))
	require.False(t, isLoopbackAddress("0.0.0.0:8080"))
}

func TestAuthenticatedSessionOwnership(t *testing.T) {
	runtime := managerservice.New(endly.New)
	handler := NewAuthenticated(runtime, &subjectAuthenticator{})
	request := httptest.NewRequest(http.MethodPost, "/v1/endly/sessions", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer alice")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusCreated, recorder.Code)
	info := &managerservice.SessionInfo{}
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(info))

	request = httptest.NewRequest(http.MethodGet, "/v1/endly/sessions/"+info.SessionID, nil)
	request.Header.Set("Authorization", "Bearer bob")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestHTTPRunPreservesTaskOrderAndDuplicates(t *testing.T) {
	runtime := managerservice.New(endly.New)
	session, err := runtime.Open(context.Background(), &managerservice.OpenRequest{})
	require.NoError(t, err)
	_, err = runtime.LoadWorkflow(context.Background(), &managerservice.LoadWorkflowRequest{
		SessionID: session.SessionID,
		URL:       "ordered.yaml",
		Content:   "pipeline:\n  task1:\n    action: nop\n  task2:\n    action: nop\n",
		Format:    "yaml",
	})
	require.NoError(t, err)
	handler := New(runtime)
	request := httptest.NewRequest(http.MethodPost, "/v1/endly/sessions/"+session.SessionID+"/operations", bytes.NewBufferString(`{"workflow":"ordered","tasks":"task1,task2,task1"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusAccepted, recorder.Code)
	operation := &managerservice.Operation{}
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(operation))
	finished, err := runtime.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	var tasks []string
	for _, event := range finished.Events {
		if event.Type != "task.started" {
			continue
		}
		value := event.Value.(map[string]interface{})
		tasks = append(tasks, value["TaskName"].(string))
	}
	require.Equal(t, []string{"task1", "task2", "task1"}, tasks)
}
