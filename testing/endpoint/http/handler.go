package http

import (
	"fmt"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type httpHandler struct {
	running   int32
	handler   func(writer http.ResponseWriter, request *http.Request)
	thinkTime time.Duration
}

const (
	thinkTimeHeaderMs = "Think-Time-Ms"
)

func (h *httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if thinkTime := request.Header.Get(thinkTimeHeaderMs); thinkTime != "" {
		h.thinkTime = time.Duration(toolbox.AsInt(thinkTime)) * time.Millisecond
		fmt.Printf("Updated think time: %s\n", h.thinkTime)
	}
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
		if !ok {
			var errorMessage = fmt.Sprintf("key: %v not found, available: \n%v", key, strings.Join(toolbox.MapKeysToStringSlice(trips.Trips), ",\n"))
			fmt.Println(errorMessage)
			http.Error(writer, errorMessage, http.StatusNotFound)
			return
		}

		var index uint32
		for {
			index := atomic.LoadUint32(&responses.Index)
			if atomic.CompareAndSwapUint32(&responses.Index, index, index+1) {
				if int(index) >= len(trips.Trips) {
					if !trips.Rotate {
						http.NotFound(writer, request)
						return
					}
				}
				atomic.StoreUint32(&responses.Index, 0)
				index = 0
				break
			}
		}

		response := responses.Responses[index]
		for k, headerValues := range response.Header {
			for _, headerValue := range headerValues {
				writer.Header().Set(k, headerValue)
			}
		}

		if httpHandler.thinkTime > 0 {
			time.Sleep(httpHandler.thinkTime)
		}
		writer.WriteHeader(response.Code)
		if response.Body != "" {
			var body, _ = util.FromPayload(response.Body)
			_, err = writer.Write(body)
			if err != nil {
				log.Print(err)
			}
		}

		if len(trips.Trips) == 0 {
			func() { _ = httpServer.Close() }()
			go func() {
				time.Sleep(time.Second)
				if atomic.LoadInt32(&httpHandler.running) == 0 {
					_ = httpServer.Shutdown(nil)
				}
			}()

		}
	}
}
