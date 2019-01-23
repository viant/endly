package http

import (
	"bytes"
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/model/criteria"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//ServiceID represents http runner service id.
const ServiceID = "http/runner"
const RunnerID = "HttpRunner"

type service struct {
	*endly.AbstractService
}

func (s *service) send(context *endly.Context, sendGroupRequest *SendRequest) (*SendResponse, error) {
	client, err := toolbox.NewHttpClient(s.applyDefaultTimeoutIfNeeded(sendGroupRequest.options_)...)
	if err != nil {
		return nil, fmt.Errorf("failed to send req: %v", err)
	}
	initializeContext(context)
	defer s.resetContext(context, sendGroupRequest)

	var sendGroupResponse = &SendResponse{
		Responses: make([]*Response, 0),
		Data:      make(map[string]interface{}),
	}
	var sessionCookies Cookies = make([]*http.Cookie, 0)
	for _, req := range sendGroupRequest.Requests {
		err = s.sendRequest(context, client, req, &sessionCookies, sendGroupRequest, sendGroupResponse)
		if err != nil {
			return nil, err
		}
	}
	if sendGroupRequest.Expect != nil {

		var actual = map[string]interface{}{
			"Responses": sendGroupResponse.Responses,
			"Data":      sendGroupResponse.Data,
		}
		sendGroupResponse.Assert, err = validator.Assert(context, sendGroupRequest, sendGroupRequest.Expect, actual, "HTTP.responses", "assert http responses")
	}
	return sendGroupResponse, nil

}

func (s *service) sendRequest(context *endly.Context, client *http.Client, request *Request, sessionCookies *Cookies, sendGroupRequest *SendRequest, sendGroupResponse *SendResponse) error {
	var err error
	var state = context.State()
	cookies := state.GetMap("cookies")
	trips := Trips(state.GetMap(TripsKey))
	request.Expand(state)

	canRun, err := criteria.Evaluate(context, context.State(), request.When, fmt.Sprintf("%v.When", "HttpRequest"), true)
	if err != nil || !canRun {
		return err
	}
	httpRequest, expectBinary, err := request.Build(context, *sessionCookies)
	if err != nil {
		return err
	}
	_ = trips.addRequest(request)
	startEvent := s.Begin(context, request)
	repeater := request.Repeater.Init()
	var response *Response
	bodyProvider, err := getRequestBodyReader(httpRequest, repeater.Repeat)

	handler := func() (interface{}, error) {
		httpRequest.Body = bodyProvider()
		httpResponse, err := client.Do(httpRequest)
		if err != nil {
			return nil, err
		}
		if response == nil { //if request is repeated only the allocated one, and keep overriding it to see the last snapshot
			response = sendGroupResponse.NewResponse()
		}
		response.Merge(httpResponse, expectBinary)
		response.UpdateCookies(cookies)
		sessionCookies.AddCookies(httpResponse.Cookies()...)
		err = response.TransformBodyIfNeeded(context, request)
		if request.DataSource == "response" {
			return toolbox.AsMap(response), err
		}
		return response.Body, err
	}

	err = repeater.Run(s.AbstractService, RunnerID, context, handler, sendGroupResponse.Data)
	if err != nil {
		return err
	}
	if toolbox.IsCompleteJSON(response.Body) {
		response.JSONBody, err = toolbox.JSONToMap(response.Body)
	}

	sendGroupResponse.Expand(sendGroupResponse.Data)

	trips.setData(sendGroupResponse.Data)

	_ = trips.addResponse(response)
	endEvent := s.End(context)(startEvent, response)
	response.TimeTakenMs = int(endEvent.Timestamp().Sub(startEvent.Timestamp()) / time.Millisecond)
	return nil
}

func (s *service) applyDefaultTimeoutIfNeeded(options []*toolbox.HttpOptions) []*toolbox.HttpOptions {
	if len(options) > 0 {
		return options
	}

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100
	return []*toolbox.HttpOptions{
		{
			Key:   "RequestTimeoutMs",
			Value: 120000,
		},
		{
			Key:   "TimeoutMs",
			Value: 120000,
		},
	}
}

//resetContext resets context for variables with Reset flag set, and removes PreviousTripStateKey
func (s *service) resetContext(context *endly.Context, request *SendRequest) {
	state := context.State()
	state.Delete(TripsKey)
	for _, request := range request.Requests {
		if request.Repeater != nil && len(request.Extract) > 0 {
			request.Extract.Reset(state)
		}
	}
}

func (s *service) handleRequest(client *http.Client, metric *runtimeMetric, trip *stressTestTrip) {
	defer func() {
		trip.waitGroup.Done()
	}()

	trip.requestTime = time.Now()
	var response *http.Response
	var err error
	if atomic.LoadInt64(&metric.startTime) == 0 {
		atomic.CompareAndSwapInt64(&metric.startTime, 0, trip.requestTime.UnixNano())
	}
	response, err = client.Do(trip.request)
	if err, ok := err.(net.Error); ok && err.Timeout() {
		trip.timeout = true
		atomic.AddUint32(&metric.timeouts, 1)
		return
	} else if err != nil {
		trip.err = err
		metric.err = err
		atomic.AddUint32(&metric.errors, 1)
		return
	}
	defer response.Body.Close()
	atomic.AddUint32(&metric.count, 1)
	trip.responseTime = time.Now()
	if trip.err != nil || trip.timeout {
		return
	}
	var content []byte
	if response.ContentLength > 0 {
		content, err = ioutil.ReadAll(response.Body)
	}

	if trip.expected {
		trip.response = &http.Response{
			Header:        response.Header,
			Status:        response.Status,
			StatusCode:    response.StatusCode,
			ContentLength: response.ContentLength,
			Body:          ioutil.NopCloser(strings.NewReader("")),
		}
		if len(content) > 0 {
			trip.response.Body = ioutil.NopCloser(bytes.NewReader(content))
		}
	}

}

func (s *service) handleRequests(client *http.Client, sendChannel chan *stressTestTrip, metric *runtimeMetric, done *uint32) {
	for {
		select {
		case trip := <-sendChannel:
			s.handleRequest(client, metric, trip)
		case <-time.After(15 * time.Second):
			return
		}
		if atomic.LoadUint32(done) == 1 {
			return
		}
	}
}

type runtimeMetric struct {
	count     uint32
	startTime int64
	timeouts  uint32
	errors    uint32
	err       error
}

func (s *service) stressTest(context *endly.Context, request *LoadRequest) (*LoadResponse, error) {
	var waitGroup = &sync.WaitGroup{}
	capacity := 1024 * request.ThreadCount
	var sendChannel = make(chan *stressTestTrip, capacity)
	var done uint32 = 0
	metrics := &runtimeMetric{}

	go s.emitMetrics(context, metrics, &done, request.Message)
	if _, err := s.initClients(request, sendChannel, metrics, &done); err != nil {
		return nil, err
	}
	partialTrips := newPartialStressTrips(capacity, sendChannel, waitGroup)
	trips, err := buildStressTestTrip(request, context, partialTrips)
	if err != nil {
		return nil, err
	}
	waitGroup.Wait()
	atomic.StoreUint32(&done, 1)
	var response = &LoadResponse{
		Status: "ok",
	}
	if err = collectTripResponses(context, trips, response, request); err != nil {
		return nil, err
	}

	response.Assert = &validator.AssertResponse{Validation: &assertly.Validation{}}
	var actual = make([]interface{}, 0)
	var expected = make([]interface{}, 0)

	if request.Expect != nil {
		for _, trip := range trips {
			if !trip.expected {
				continue
			}
			if trip.index >= len(request.Requests) {
				continue
			}
			expect := request.Requests[trip.index].Expect
			if expect == nil {
				continue
			}
			actualResponse := response.NewResponse()
			var index = trip.index
			if trip.response != nil {
				response.StatusCodes[trip.response.StatusCode]++
				actualResponse.Merge(trip.response, trip.expectBinary)
				err := actualResponse.TransformBodyIfNeeded(context, request.Requests[index])
				if err != nil {
					continue
				}
				if toolbox.IsCompleteJSON(actualResponse.Body) {
					actualResponse.JSONBody, _ = toolbox.JSONToMap(actualResponse.Body)
				}
			}
			actual = append(actual, actualResponse)
			expected = append(expected, expect)

		}
		response.Assert, err = validator.Assert(context, request, expected, actual, "HTTP.Responses", "assert http responses")
	}
	return response, err
}

func collectTripResponses(context *endly.Context, trips []*stressTestTrip, response *LoadResponse, request *LoadRequest) error {
	startTime := trips[0].requestTime
	endTime := trips[0].responseTime
	minResponse := trips[0].elapsed
	maxResponse := trips[0].elapsed

	response.StatusCodes = make(map[int]int)
	var cumulativeResponse time.Duration
	//collect responses and build validation collection
	for _, trip := range trips {
		if trip.err != nil {
			response.ErrorCount++
			response.Error = trip.err.Error()
		}

		if trip.timeout {
			response.TimeoutCount++
		}
		if !trip.expected {
			continue
		}
		if trip.requestTime.Before(startTime) {
			startTime = trip.requestTime
		}
		if trip.responseTime.After(endTime) {
			endTime = trip.responseTime
		}
		if trip.elapsed < minResponse {
			minResponse = trip.elapsed
		}
		if trip.elapsed > maxResponse {
			maxResponse = trip.elapsed
		}
		cumulativeResponse += trip.elapsed
	}

	response.MinResponseTimeInMs = float64(minResponse) / float64(time.Millisecond)
	response.MaxResponseTimeInMs = float64(maxResponse) / float64(time.Millisecond)
	avg := float64(cumulativeResponse) / float64(len(trips))
	response.AvgResponseTimeInMs = avg / float64(time.Millisecond)
	testDuration := endTime.Sub(startTime)
	response.TestDurationSec = float64(testDuration) / float64(time.Second)
	response.RequestCount = len(trips)
	response.QPS = float64(len(trips)) / response.TestDurationSec
	return nil
}

type stressTestTrip struct {
	index        int
	err          error
	timeout      bool
	expectBinary bool
	request      *http.Request
	response     *http.Response
	expected     bool
	waitGroup    *sync.WaitGroup
	requestTime  time.Time
	responseTime time.Time
	elapsed      time.Duration
}

func buildStressTestTrip(request *LoadRequest, context *endly.Context, partials *partialStressTrips) ([]*stressTestTrip, error) {
	var sessionCookies = []*http.Cookie{}
	var err error
	var trips = make([]*stressTestTrip, 0)

	var expectedResponses []interface{}
	if len(request.Expect) > 0 {
		responses, ok := request.Expect["Responses"]
		if !ok {
			responses, ok = request.Expect["responses"]
		}
		if ok {
			expectedResponses = toolbox.AsSlice(responses)
		}
	}

	for index, req := range request.Requests {
		var state = context.State()
		req.Expand(state)
		for i := 0; i < req.Repeat; i++ {
			trip := &stressTestTrip{
				waitGroup: partials.WaitGroup,
				index:     index,
			}
			if (i == 0 || (i%request.AssertMod) == 0) && index < len(expectedResponses) { //add validation to the first response from repeated
				trip.expected = true
			}
			if trip.request, trip.expectBinary, err = req.Build(context, sessionCookies); err != nil {
				return nil, err
			}
			trips = append(trips, trip)
			partials.append(trip)

		}
	}

	if partials.index > 0 {
		partials.flush(partials.index + 1)
	}
	return trips, nil
}

func (s *service) initClients(request *LoadRequest, sendChannel chan *stressTestTrip, metric *runtimeMetric, done *uint32) ([]*http.Client, error) {
	var clients = make([]*http.Client, request.ThreadCount)
	var err error
	for i := 0; i < request.ThreadCount; i++ {
		var client *http.Client
		options := s.applyDefaultTimeoutIfNeeded(request.options_)
		if client, err = toolbox.NewHttpClient(options...); err != nil {
			return nil, err
		}

		go s.handleRequests(client, sendChannel, metric, done)
		clients[i] = client
	}
	return clients, nil
}

const httpRunnerSendRequestExample = `{
  "Requests": [
    {
      "Method": "GET",
      "URL": "http://127.0.0.1:8777/event1/?k1=v1\u0026k2=v2"
    },
    {
      "Method": "GET",
      "URL": "http://127.0.0.1:8777/event1/?k10=v1\u0026k2=v2"
    },
    {
      "When": "$httpTrips[0].Body:/pixel/",
      "Method": "POST",
      "URL": "http://127.0.0.1:8777/event4/",
      "Body": "Lorem Ipsum is simply dummy text of the printing and typesetting industry."
    }
  ]
}`

const httpRunnerLoadRequestExample = `{
  "ThreadCount":2,
  "Requests": [
    {
      "Method": "GET",
      "URL": "http://127.0.0.1:8777/event1/?k1=v1\u0026k2=v2"
    },
    {
      "Method": "GET",
      "URL": "http://127.0.0.1:8777/event1/?k10=v1\u0026k2=v2",
      "Repeat":10
    },
    {
      "Method": "POST",
      "URL": "http://127.0.0.1:8777/event4/",
      "Body": "Lorem Ipsum is simply dummy text of the printing and typesetting industry."
    },
	{
      "Method": "POST",
      "URL": "http://127.0.0.1:8777/event4/",
      "Body": "Lorem Ipsum is simply dummy text of the printing and typesetting industry."
    }
  ]
}`

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "send",
		RequestInfo: &endly.ActionInfo{
			Description: "send http request(s)",
			Examples: []*endly.UseCase{
				{
					Description: "send",
					Data:        httpRunnerSendRequestExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SendRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SendResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SendRequest); ok {

				return s.send(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "load",
		RequestInfo: &endly.ActionInfo{
			Description: "http endpoint stress test",
			Examples: []*endly.UseCase{
				{
					Description: "stress test",
					Data:        httpRunnerLoadRequestExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &LoadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &LoadResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*LoadRequest); ok {
				return s.stressTest(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) emitMetrics(context *endly.Context, metric *runtimeMetric, done *uint32, message string) {
	if message == "" {
		return
	}
	state := context.State()
	private := state.Clone()
	for atomic.LoadUint32(done) == 0 {
		count := atomic.LoadUint32(&metric.count)
		if count == 0 {
			time.Sleep(time.Second)
			continue
		}
		timeTakenNs := time.Now().UnixNano() - atomic.LoadInt64(&metric.startTime)
		timeTakenSec := float64(timeTakenNs) / float64(time.Second)
		qps := 0.0
		if timeTakenSec > 0 {
			qps = float64(count) / timeTakenSec
		}

		loadInfo := data.NewMap()
		loadInfo.Put("QPS", fmt.Sprintf("%9v", float64(int(qps*10.0))/10.0))
		loadInfo.Put("Count", fmt.Sprintf("%9v", count))
		loadInfo.Put("Elapsed", fmt.Sprintf("%9v", fmt.Sprintf("%s", (time.Duration(timeTakenSec)*time.Second))))
		loadInfo.Put("Errors", fmt.Sprintf("%4v", atomic.LoadUint32(&metric.errors)))
		loadInfo.Put("Timeouts", fmt.Sprintf("%4v", atomic.LoadUint32(&metric.timeouts)))
		errMessage := ""
		if metric.err != nil {
			errMessage = metric.err.Error()
		}
		loadInfo.Put("Error", errMessage)
		private.Put("load", loadInfo)
		eventMessage := private.ExpandAsText(message)
		context.Publish(msg.NewRepeatedEvent(eventMessage, "loadTest"))
		time.Sleep(time.Second)
	}
}

//New creates a new http runner service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
