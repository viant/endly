package http

import (
	"errors"
	"fmt"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/bridge"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	//URLKey represents url key
	URLKey = "URL"
	//CookieKey represents cookie header key
	CookieKey = "Cookie"
	//ContentTypeKey represents content type header key
	ContentTypeKey = "Content-Type"
	//MethodKey represents http method key
	MethodKey = "Method"
	//BodyKey represents http body key
	BodyKey = "Body"
)

//HTTPRequestKeyProvider represents request key provider to extract a request field.
type HTTPRequestKeyProvider func(source interface{}) (string, error)

//HTTPRequestKeyProviders represents key providers
var HTTPRequestKeyProviders = make(map[string]HTTPRequestKeyProvider)

//HTTPServerTrips represents http trips
type HTTPServerTrips struct {
	BaseDirectory string
	Rotate        bool
	Trips         map[string]*HTTPResponses
	IndexKeys     []string
	Mutex         *sync.Mutex
}

func (t *HTTPServerTrips) loadTripsIfNeeded() error {
	if t.BaseDirectory != "" {
		t.Trips = make(map[string]*HTTPResponses)
		httpTrips, err := bridge.ReadRecordedHttpTrips(t.BaseDirectory)
		if err != nil {
			return err
		}
		if len(httpTrips) == 0 {
			return fmt.Errorf("http capautre directory was empty %v", t.BaseDirectory)
		}
		for _, trip := range httpTrips {
			key, err := buildKeyValue(t.IndexKeys, trip.Request)
			if err != nil {
				return fmt.Errorf("failed to build request key: %v, %v", trip.Request.URL, err)
			}

			if _, has := t.Trips[key]; !has {
				t.Trips[key] = &HTTPResponses{
					Request:   trip.Request,
					Responses: make([]*bridge.HttpResponse, 0),
				}
			}
			t.Trips[key].Responses = append(t.Trips[key].Responses, trip.Response)
		}
	}
	return nil
}

//Init initialises trips
func (t *HTTPServerTrips) Init() error {
	if t.Mutex == nil {
		t.Mutex = &sync.Mutex{}
	}
	err := t.loadTripsIfNeeded()
	if err != nil {
		return fmt.Errorf("failed to load trips: %v", err)
	}
	if len(t.Trips) == 0 {
		return errors.New("trips were empty")
	}
	return nil
}

//HTTPResponses represents HTTPResponses
type HTTPResponses struct {
	Request   *bridge.HttpRequest
	Responses []*bridge.HttpResponse
	Index     uint32
}

type httpHandler struct {
	running int32
	handler func(writer http.ResponseWriter, request *http.Request)
}

func (h *httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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

		index := int(atomic.LoadUint32(&responses.Index))

		if index >= len(responses.Responses) {
			if !trips.Rotate {
				http.NotFound(writer, request)
				return
			}
			index = 0
			atomic.StoreUint32(&responses.Index, 0)
		}

		response := responses.Responses[responses.Index]
		defer atomic.AddUint32(&responses.Index, 1)
		for k, headerValues := range response.Header {
			for _, headerValue := range headerValues {
				writer.Header().Set(k, headerValue)
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

//StartServer starts http request, the server has ability to replay recorded  HTTP trips with https://github.com/viant/toolbox/blob/master/bridge/http_bridge_recording_util.go#L82
func StartServer(port int, trips *HTTPServerTrips) error {
	err := trips.Init()
	if err != nil {
		return fmt.Errorf("failed to start http server :%v, %v", port, err)
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
	return err
}

//HeaderProvider return a header value for supplied source
func HeaderProvider(header string) HTTPRequestKeyProvider {
	return func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:
			return strings.Join(request.Header[header], "\n"), nil
		case *http.Request:
			return strings.Join(request.Header[header], "\n"), nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
}

func stripProtoAndHost(URL string) string {
	if index := strings.Index(URL, "://"); index != -1 {
		URL = string(URL[index+3:])
	}
	if index := strings.Index(URL, "/"); index > 0 {
		URL = string(URL[index:])
	}
	return URL
}

func init() {
	HTTPRequestKeyProviders[URLKey] = func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:

			return stripProtoAndHost(request.URL), nil
		case *http.Request:
			return stripProtoAndHost(request.URL.String()), nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
	HTTPRequestKeyProviders[MethodKey] = func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:
			return request.Method, nil
		case *http.Request:
			return request.Method, nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
	HTTPRequestKeyProviders[CookieKey] = HeaderProvider(CookieKey)
	HTTPRequestKeyProviders[ContentTypeKey] = HeaderProvider(ContentTypeKey)
	HTTPRequestKeyProviders[BodyKey] = func(source interface{}) (string, error) {

		switch request := source.(type) {
		case *bridge.HttpRequest:
			body, _ := util.FromPayload(request.Body)
			return util.AsPayload(body), nil
		case *http.Request:
			if request.ContentLength == 0 {
				return "", nil
			}
			content, err := ioutil.ReadAll(request.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read body %v, %v", request.URL, err)
			}
			encoded := string(content)
			if strings.HasPrefix(encoded , "base64:") {
				content, _ = util.FromPayload(encoded)
			}
			return util.AsPayload(content), nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
}

func buildKeyValue(keys []string, request interface{}) (string, error) {
	var values = make([]string, 0)
	for _, key := range keys {
		provider, has := HTTPRequestKeyProviders[key]
		if !has {
			return "", fmt.Errorf("unsupported key: %v, available, [%v]", key, strings.Join(toolbox.MapKeysToStringSlice(HTTPRequestKeyProviders), ","))
		}
		value, err := provider(request)
		if err != nil {
			return "", fmt.Errorf("unable to get value for %v, %v", key, err)
		}
		values = append(values, value)
	}
	return strings.Join(values, ","), nil
}
