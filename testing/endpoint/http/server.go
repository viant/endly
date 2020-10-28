package http

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)


//HTTPRequestKeyProvider represents request key provider to extract a request field.
type HTTPRequestKeyProvider func(source interface{}) (string, error)

//HTTPRequestKeyProviders represents key providers
var HTTPRequestKeyProviders = make(map[string]HTTPRequestKeyProvider)




//StartServer starts http request, the server has ability to replay recorded  HTTP trips with https://github.com/viant/toolbox/blob/master/bridge/http_bridge_recording_util.go#L82
func StartServer(port int, trips *HTTPServerTrips) (*http.Server, error) {
	err := trips.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to start http server :%v, %v", port, err)
	}

	var httpServer *http.Server
	var httpHandler = &httpHandler{
		running: 1,
	}
	httpHandler.handler = getServerHandler(httpServer, httpHandler, trips)
	httpServer = &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: httpHandler}

	errorNotification := make(chan bool, 1)
	go func() {
		fmt.Printf("Starting server on %v\n", port)
		err = httpServer.ListenAndServe()
		atomic.StoreInt32(&httpHandler.running, 0)
		errorNotification <- true
		if err != nil {
			err = fmt.Errorf("failed to start http server on port %v, %v", port, err)
		}
	}()

	//if there is error in starting server quite immediately
	select {
	case <-errorNotification:
	case <-time.After(time.Second * 2):
	}
	return httpServer, err
}

