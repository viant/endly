package http

import (
	"github.com/viant/toolbox/bridge"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestReplaySequenceAndExhaustion(t *testing.T) {
	for _, rotate := range []bool{false, true} {
		trips := &HTTPServerTrips{Rotate: rotate, IndexKeys: []string{MethodKey, URLKey}, Mutex: &sync.Mutex{}, Trips: map[string]*HTTPResponses{
			"GET,/userinfo": {Responses: []*bridge.HttpResponse{{Code: 200, Body: `{"roles":["reader"]}`}, {Code: 403, Body: `{"roles":[]}`}}},
		}}
		h := &httpHandler{running: 1}
		h.handler = getServerHandler(&http.Server{}, h, trips)
		want := []int{200, 403, 404}
		if rotate {
			want[2] = 200
		}
		for i, code := range want {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/userinfo", nil))
			if w.Code != code {
				t.Fatalf("rotate=%v request=%d status=%d want=%d", rotate, i, w.Code, code)
			}
		}
	}
}

func TestUnmatchedBearerDoesNotDiscloseRecordedCredentials(t *testing.T) {
	trips := &HTTPServerTrips{IndexKeys: []string{MethodKey, URLKey, "Header.Authorization"}, Mutex: &sync.Mutex{}, Trips: map[string]*HTTPResponses{"GET,/userinfo,Bearer private-fixture-token": {Responses: []*bridge.HttpResponse{{Code: 200}}}}}
	h := &httpHandler{running: 1}
	h.handler = getServerHandler(&http.Server{}, h, trips)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/userinfo", nil))
	if w.Code != 404 || w.Body.String() != "mock request did not match a recorded response\n" {
		t.Fatalf("unexpected unmatched response: %d %s", w.Code, w.Body.String())
	}
}

func TestAppendRetainsEarlierAppends(t *testing.T) {
	h := &httpHandler{running: 1}
	makeTrips := func(path string) *HTTPServerTrips {
		return &HTTPServerTrips{Rotate: true, IndexKeys: []string{MethodKey, URLKey}, Mutex: &sync.Mutex{}, Trips: map[string]*HTTPResponses{"GET," + path: {Responses: []*bridge.HttpResponse{{Code: 200, Body: path}}}}}
	}
	first := makeTrips("/first")
	server := &Server{httpHandler: h, trips: first.Trips}
	h.handler = getServerHandler(&server.Server, h, first)
	server.Append(makeTrips("/second"))
	server.Append(makeTrips("/third"))
	for _, path := range []string{"/first", "/second", "/third"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || w.Body.String() != path {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
	}
}

func TestHeaderMatchingConcurrentAndCaseInsensitive(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			request := httptest.NewRequest("GET", "/userinfo", nil)
			request.Header.Set("Authorization", "Bearer fixture")
			key, err := buildKeyValue([]string{"Header.authorization"}, request)
			if err != nil || key != "Bearer fixture" {
				t.Errorf("header matching failed: %v", err)
			}
		}()
	}
	wg.Wait()
}
