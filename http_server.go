package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"io/ioutil"
	"net/http"
	"strings"
	"github.com/viant/toolbox/bridge"
	"time"
)

const (
	URLKey         = "URL"
	CookieKey      = "Cookie"
	ContentTypeKey = "Content-Type"
	MethodKey      = "Method"
	BodyKey        = "Body"
)

//HttpRequestKeyProvider represents request key provider to extract a request field.
type HttpRequestKeyProvider func(source interface{}) (string, error)

//HttpRequestKeyProviders rerpresents key providers
var HttpRequestKeyProviders = make(map[string]HttpRequestKeyProvider)

//HttpServerTrips represents http trips
type HttpServerTrips struct {
	BaseDirectory string
	Trips         map[string]*HttpResponses
	IndexKeys     []string
}

func (t *HttpServerTrips) LoadTripsIfNeeded() error {
	if t.BaseDirectory != "" {
		t.Trips = make(map[string]*HttpResponses)
		httpTrips, err := bridge.ReadRecordedHttpTrips(t.BaseDirectory)
		if err != nil {
			return err
		}
		if len(httpTrips) == 0 {
			fmt.Errorf("Http capautre directory was empty %v", t.BaseDirectory)
		}
		for _, trip := range httpTrips {
			key, _ := buildKeyValue(t.IndexKeys, trip.Request)
			if _, has := t.Trips[key]; ! has {
				t.Trips[key] = &HttpResponses{
					Request:   trip.Request,
					Responses: make([]*bridge.HttpResponse, 0),
				}
			}
			t.Trips[key].Responses = append(t.Trips[key].Responses, trip.Response)
		}
	}
	return nil
}

//HttpResponses represents HttpResponses
type HttpResponses struct {
	Request   *bridge.HttpRequest
	Responses []*bridge.HttpResponse
	Index     int
}

type httpHandler struct {
	handler func(writer http.ResponseWriter, request *http.Request)
}

func (h *httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.handler(writer, request)
}

//StartHttpServer starts http request
func StartHttpServer(port int, trips *HttpServerTrips) error {
	err := trips.LoadTripsIfNeeded()
	if err != nil {
		return fmt.Errorf("failed to start http server :%v, %v", port, err)
	}
	if len(trips.Trips) == 0 {
		return fmt.Errorf("failed to start http server :%v, trips were empty", port)
	}
	var httpServer *http.Server

	var handler = func(writer http.ResponseWriter, request *http.Request) {
		var key, err = buildKeyValue(trips.IndexKeys, request)
		if err != nil {
			http.Error(writer, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			return
		}
		responses, ok := trips.Trips[key]
		if ! ok {
			var errorMessage = fmt.Sprintf("key: %v not found: %v, available: [%v]", key, strings.Join(toolbox.MapKeysToStringSlice(trips.Trips), ","))
			fmt.Println(errorMessage)
			http.Error(writer, errorMessage, http.StatusNotFound)
			return
		}
		response := responses.Responses[responses.Index]
		responses.Index++
		for k, headerValues := range response.Header {
			for _, headerValue := range headerValues {
				writer.Header().Set(k, headerValue)
			}
		}
		writer.WriteHeader(response.Code)
		if response.Body != "" {
			var body []byte = ([]byte)(response.Body)
			if strings.HasPrefix(response.Body, "text:") {
				body = []byte(response.Body[5:])
			}
			writer.Write(body)
		}
		if responses.Index >= len(responses.Responses) {
			delete(trips.Trips, key)
		}
		if len(trips.Trips) == 0 {
			go func() {
				time.Sleep(1 * time.Second)
				httpServer.Shutdown(nil)
			}()

		}
	}

	httpServer = &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: &httpHandler{handler}}

	errorNotification := make(chan bool, 1)
	go func() {
		fmt.Printf("Starting server on %v\n", port)
		err = httpServer.ListenAndServe()
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
func HeaderProvider(header string) HttpRequestKeyProvider {
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
	HttpRequestKeyProviders[URLKey] = func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:

			return stripProtoAndHost(request.URL), nil
		case *http.Request:
			return stripProtoAndHost(request.URL.String()), nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
	HttpRequestKeyProviders[MethodKey] = func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:
			return request.Method, nil
		case *http.Request:
			return request.Method, nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
	HttpRequestKeyProviders[CookieKey] = HeaderProvider(CookieKey)
	HttpRequestKeyProviders[ContentTypeKey] = HeaderProvider(ContentTypeKey)
	HttpRequestKeyProviders[BodyKey] = func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:
			if strings.HasPrefix(request.Body, "text:") {
				return string(request.Body[5:]), nil
			}
			return request.Body, nil
		case *http.Request:
			if request.ContentLength == 0 {
				return "", nil
			}
			content, err := ioutil.ReadAll(request.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read body %v, %v", request.URL, err)
			}
			return string(content), nil

		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
}

func buildKeyValue(keys []string, request interface{}) (string, error) {
	var values = make([]string, 0)
	for _, key := range keys {

		provider, has := HttpRequestKeyProviders[key]
		if ! has {
			return "", fmt.Errorf("unsupported key: %v, available, [%v]", key, strings.Join(toolbox.MapKeysToStringSlice(HttpRequestKeyProviders), ","))
		}
		value, err := provider(request)
		if err != nil {
			return "", err
		}
		values = append(values, value)
	}
	return strings.Join(values, ","), nil
}
