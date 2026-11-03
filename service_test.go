package endly_test

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/model/msg"
	"sync"
	"testing"
	"time"
)

func TestAbstractService_Run(t *testing.T) {

	manager := endly.New()

	srv := newService()

	var useCases = []struct {
		Description string
		Request     interface{}
		Expected    interface{}
		HasError    bool
	}{
		{
			Description: "Init error",
			Request: &TestRequest{
				InitError: true,
			},
			HasError: true,
		},
		{
			Description: "Validation error",
			Request: &TestRequest{
				ValidationError: true,
			},
			HasError: true,
		},
		{
			Description: "Run error",
			Request: &TestRequest{
				RunError: true,
			},
			HasError: true,
		},
		{
			Description: "Validation error",
			Request: &TestRequest{
				ValidationError: true,
			},
			HasError: true,
		},
		{
			Description: "Simple request",
			Request: &TestRequest{
				Data: "test",
			},
			Expected: `{"Request":{"Data":"test","RunError":false,"InitError":false,"ValidationError":false,"Inited":true,"Validated":true}}`,
		},
		{
			Description: "Assignable request",
			Request: &TwinTestRequest{
				Data: "test",
			},
			Expected: `{"Request":{"Data":"test","RunError":false,"InitError":false,"ValidationError":false,"Inited":false,"Validated":false}}`,
		},

		{
			Description: "Unsupported request",
			Request:     &struct{}{},
			HasError:    true,
		},
	}

	for _, useCase := range useCases {
		context := manager.NewContext(nil)
		response := srv.Run(context, useCase.Request)
		if useCase.HasError {
			assert.Equal(t, "error", response.Status, useCase.Description)
			continue
		}
		assertly.AssertValues(t, useCase.Expected, response.Response, useCase.Description)

	}

}

func TestAbstractService_Route(t *testing.T) {
	srv := newService()
	route, err := srv.Route("test")
	if assert.Nil(t, err) {
		assert.NotNil(t, route)
	}
	_, err = srv.Route("test2")
	assert.NotNil(t, err)
}

func TestAbstractService_GetHostAndSSHPort(t *testing.T) {
	srv := newService()
	{
		host, port := srv.GetHostname(nil)
		assert.Equal(t, host, "")
		assert.Equal(t, port, 0)
	}
	{
		host, port := srv.GetHostname(location.NewResource("scp://127.0.0.1:22"))
		assert.Equal(t, host, "127.0.0.1")
		assert.Equal(t, port, 22)
	}
	{
		host, port := srv.GetHostname(location.NewResource("file:///avc"))
		assert.Equal(t, host, "127.0.0.1")
		assert.Equal(t, port, 22)
	}
}

func TestAbstractService_Sleep(t *testing.T) {
	manager := endly.New()
	context := manager.NewContext(nil)
	srv := newService()
	srv.Sleep(context, 1)
}

func TestRunInBackgroundCancellationStopsHeartbeatAndJoinsHandler(t *testing.T) {
	runContext := endly.New().NewContext(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runContext.SetBackground(ctx)
	logging := true
	runContext.Logging = &logging
	heartbeats := make(chan struct{}, 100)
	runContext.Listener = func(event msg.Event) {
		if event.Type() == "msg_SleepEvent" {
			select {
			case heartbeats <- struct{}{}:
			default:
			}
		}
	}
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	var releaseOnce sync.Once
	releaseHandler := func() { releaseOnce.Do(func() { close(release) }) }
	want := errors.New("handler result")
	go func() {
		finished <- newService().RunInBackground(runContext, func() error {
			close(started)
			<-release
			return want
		})
	}()
	defer releaseHandler()
	<-started
	select {
	case <-heartbeats:
	case <-time.After(time.Second):
		t.Fatal("missing wait heartbeat")
	}
	cancel()
	select {
	case <-heartbeats:
		t.Fatal("cancelled wait continued publishing heartbeats")
	case err := <-finished:
		t.Fatalf("handler was not joined: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	// Releasing the handler must return its error, even after cancellation.
	releaseHandler()
	select {
	case err := <-finished:
		assert.Same(t, want, err)
	case <-time.After(time.Second):
		t.Fatal("finished handler was not joined")
	}
}

func TestAbstractService_Actions(t *testing.T) {
	_ = endly.Registry.Register(func() endly.Service {
		return newService()
	})
}

type TestRequest struct {
	Data            string
	RunError        bool
	InitError       bool
	ValidationError bool
	Inited          bool
	Validated       bool
}

func (r *TestRequest) Init() error {
	r.Inited = true
	if r.InitError {
		return errors.New(r.Data)
	}
	return nil
}

func (r *TestRequest) Validate() error {
	r.Validated = true
	if r.ValidationError {
		return errors.New(r.Data)
	}
	return nil
}

type TwinTestRequest TestRequest

type TestResponse struct {
	Request interface{}
}

type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "test",
		RequestInfo: &endly.ActionInfo{
			Description: "no operation action, helper for separating action.Init as self descriptive steps",
		},
		RequestProvider: func() interface{} {
			return &TestRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*TestRequest); ok {
				if req.RunError {
					return nil, errors.New(req.Data)
				}
				return &TestResponse{req}, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

// newNopService creates a new NoOperation nopService.
func newService() *service {
	var result = &service{
		AbstractService: endly.NewAbstractService("test"),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
