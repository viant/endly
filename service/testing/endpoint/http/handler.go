package http

import (
	"fmt"
	"github.com/viant/endly/internal/util"
	"github.com/viant/toolbox"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type httpHandler struct {
	running   int32
	handler   func(writer http.ResponseWriter, request *http.Request)
	thinkTime int64
	mu        sync.RWMutex
}

const (
	thinkTimeHeaderMs = "Think-Time-Ms"
)

func (h *httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if thinkTime := request.Header.Get(thinkTimeHeaderMs); thinkTime != "" {
		atomic.StoreInt64(&h.thinkTime, int64(time.Duration(toolbox.AsInt(thinkTime))*time.Millisecond))
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	h.handler(writer, request)
}

func getServerHandler(httpServer *http.Server, httpHandler *httpHandler, trips *HTTPServerTrips) func(writer http.ResponseWriter, request *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		trips.Mutex.Lock()
		defer trips.Mutex.Unlock()
		if atomic.LoadInt32(&httpHandler.running) == 0 {
			return
		}

		var key, err = buildKeyValue(trips.IndexKeys, request)
		if err != nil {
			http.Error(writer, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			return
		}

		responses, ok := trips.Trips[key]
		if !ok || len(responses.Responses) == 0 {
			http.Error(writer, "mock request did not match a recorded response", http.StatusNotFound)
			return
		}
		index := responses.Index
		if int(index) >= len(responses.Responses) {
			if !trips.Rotate {
				http.NotFound(writer, request)
				return
			}
			index = 0
		}
		responses.Index = index + 1

		response := responses.Responses[index]
		for k, headerValues := range response.Header {
			for _, headerValue := range headerValues {
				writer.Header().Set(k, headerValue)
			}
		}

		if delay := time.Duration(atomic.LoadInt64(&httpHandler.thinkTime)); delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-request.Context().Done():
				return
			}
		}
		writer.WriteHeader(response.Code)
		if response.Body != "" {
			var body, _ = util.FromPayload(response.Body)
			_, err = writer.Write(body)
			if err != nil {
				log.Print(err)
			}
		}

	}
}
