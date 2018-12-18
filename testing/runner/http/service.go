package http

import (
	"bytes"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/criteria"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox"
	"io/ioutil"
	"net/http"
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
	client, err := toolbox.NewHttpClient(s.applyDefaultTimeoutIfNeeded(sendGroupRequest.Options)...)
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
	_= trips.addRequest(request)
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

	trips.addResponse(response)
	endEvent := s.End(context)(startEvent, response)
	response.TimeTakenMs = int(endEvent.Timestamp().Sub(startEvent.Timestamp()) / time.Millisecond)
	return nil
}

func (s *service) applyDefaultTimeoutIfNeeded(options []*toolbox.HttpOptions) []*toolbox.HttpOptions {
	if len(options) > 0 {
		return options
	}
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
		if request.Repeater != nil && len(request.Extraction) > 0 {
			request.Extraction.Reset(state)
		}
	}
}

func (s *service) handleRequest(client *http.Client, trip *stressTestTrip) {
	defer func() {
		trip.waitGroup.Done()
	}()
	trip.requestTime = time.Now()
	var response *http.Response
	response, trip.err = client.Do(trip.request)

	trip.responseTime = time.Now()
	trip.elapsed = trip.responseTime.Sub(trip.requestTime)
	if trip.err != nil {
		return
	}
	if trip.expected {
		if response.ContentLength > 0 {
			content, _ := ioutil.ReadAll(response.Body)
			response.Body.Close()
			response.Body = ioutil.NopCloser(bytes.NewReader(content))
		}
		trip.response = response
	}
}

func (s *service) handleRequests(client *http.Client, sendChannel chan *stressTestTrip, done *uint32) {

	for {
		select {
		case trip := <-sendChannel:
			s.handleRequest(client, trip)
		case <-time.After(15 * time.Second):
			return
		}
		if atomic.LoadUint32(done) == 1 {
			return
		}
	}
}

func (s *service) stressTest(context *endly.Context, request *LoadRequest) (*LoadResponse, error) {
	var waitGroup = &sync.WaitGroup{}
	var sendChannel = make(chan *stressTestTrip, 2*request.ThreadCount)
	var done uint32 = 0
	if _, err := s.initClients(request, sendChannel, &done); err != nil {
		return nil, err
	}
	trips, err := buildStressTestTrip(request, context, waitGroup)
	if err != nil {
		return nil, err
	}
	waitGroup.Add(len(trips))
	for _, trip := range trips {
		select {
		case sendChannel <- trip:
		}
	}
	waitGroup.Wait()
	atomic.StoreUint32(&done, 1)
	var response = &LoadResponse{
		Status: "ok",
	}
	if err = collectTripResponses(context, trips, response, request); err != nil {
		return nil, err
	}
	if request.Expect != nil {
		var actual = map[string]interface{}{
			"Responses": response.Responses,
		}
		response.Assert, err = validator.Assert(context, request, request.Expect, actual, "HTTP.responses", "assert http responses")
	}
	return response, nil
}

func collectTripResponses(context *endly.Context, trips []*stressTestTrip, response *LoadResponse, request *LoadRequest) error {
	startTime := trips[0].requestTime
	endTime := trips[0].responseTime
	minResponse := trips[0].elapsed
	maxResponse := trips[0].elapsed
	var cumulativeResponse time.Duration
	//collect responses and build validation collection
	for _, trip := range trips {
		if trip.err != nil {
			response.Status = "error"
			response.Error = trip.err.Error()
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
		tripResponse := response.NewResponse()
		var index = trip.index
		if trip.response != nil {
			tripResponse.Merge(trip.response, trip.expectBinary)
			err := tripResponse.TransformBodyIfNeeded(context, request.Requests[index])
			if err != nil {
				return err
			}
			if toolbox.IsCompleteJSON(tripResponse.Body) {
				tripResponse.JSONBody, _ = toolbox.JSONToMap(tripResponse.Body)
			}
		}
		response.Responses[trip.index] = tripResponse
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
	expectBinary bool
	request      *http.Request
	response     *http.Response
	expected     bool
	waitGroup    *sync.WaitGroup
	requestTime  time.Time
	responseTime time.Time
	elapsed      time.Duration
}

func buildStressTestTrip(request *LoadRequest, context *endly.Context, waitGroup *sync.WaitGroup) ([]*stressTestTrip, error) {
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
				waitGroup: waitGroup,
				index:     index,
			}
			if i == 0 && index < len(expectedResponses) { //add validation to the first response from repeated
				trip.expected = true
			}
			if trip.request, trip.expectBinary, err = req.Build(context, sessionCookies); err != nil {
				return nil, err
			}
			trips = append(trips, trip)
		}
	}
	return trips, nil
}

func (s *service) initClients(request *LoadRequest, sendChannel chan *stressTestTrip, done *uint32) ([]*http.Client, error) {
	var clients = make([]*http.Client, request.ThreadCount)
	var err error
	for i := 0; i < request.ThreadCount; i++ {
		var client *http.Client
		if client, err = toolbox.NewHttpClient(s.applyDefaultTimeoutIfNeeded(request.Options)...); err != nil {
			return nil, err
		}
		go s.handleRequests(client, sendChannel, done)
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

//New creates a new http runner service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
