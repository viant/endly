package endly

import (
	"fmt"
	_ "github.com/viant/endly/internal/unsafe"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/model/msg"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// AbstractService represenst an abstract service.
type (
	AbstractService struct {
		Service
		*sync.RWMutex
		routeByAction  map[string]*Route
		routeByRequest map[reflect.Type]*Route
		actions        []string
		id             string
		state          data.Map
	}

	Informer interface {
		Info() string
	}
)

// Mutex returns a mutex.
func (s *AbstractService) Mutex() *sync.RWMutex {
	return s.RWMutex
}

// Register register action routes
func (s *AbstractService) Register(routes ...*Route) {
	for _, route := range routes {
		s.routeByAction[route.Action] = route
		s.routeByRequest[reflect.TypeOf(route.RequestProvider())] = route
		s.actions = append(s.actions, route.Action)
	}
}

func (s *AbstractService) addRouteIfConvertible(request interface{}) *Route {
	var requestType = reflect.TypeOf(request)
	if requestType != nil {
		for k, v := range s.routeByRequest {
			if requestType.Kind() == reflect.Ptr && requestType.Elem().ConvertibleTo(k.Elem()) {

				s.routeByRequest[requestType] = &Route{
					Action:           v.Action,
					RequestInfo:      v.RequestInfo,
					ResponseInfo:     v.ResponseInfo,
					RequestProvider:  v.RequestProvider,
					ResponseProvider: v.ResponseProvider,
					Handler: func(context *Context, convertibleRequest interface{}) (interface{}, error) {
						var request = v.RequestProvider()
						var requestValue = reflect.ValueOf(request)
						var convertibleValue = reflect.ValueOf(convertibleRequest)
						requestValue.Elem().Set(convertibleValue.Elem().Convert(k.Elem()))
						return v.Handler(context, request)
					},
				}
				return s.routeByRequest[requestType]
			}
		}
	}
	return nil
}

// Run returns a service action for supplied action
func (s *AbstractService) Run(context *Context, request interface{}) (response *ServiceResponse) {
	response = &ServiceResponse{Status: "ok"}
	startEvent := s.Begin(context, request)
	var err error
	defer func() {
		s.End(context)(startEvent, response.Response)
		if err != nil {
			response.Err = err
			response.Status = "error"
			response.Error = fmt.Sprintf("%v", err)
		}
	}()
	service, ok := s.routeByRequest[reflect.TypeOf(request)]
	if !ok {

		service = s.addRouteIfConvertible(request)
		if service == nil {
			err = NewError(s.ID(), fmt.Sprintf("%T", request), fmt.Errorf("failed to lookup service route: %T", request))
			return response
		}
	}

	if initializer, ok := request.(Initializer); ok {
		if err = initializer.Init(); err != nil {
			err = NewError(s.ID(), service.Action, fmt.Errorf("init %T failed: %v", request, err))
			return response
		}
	}

	if validator, ok := request.(Validator); ok {
		if err = validator.Validate(); err != nil {
			err = NewError(s.ID(), service.Action, fmt.Errorf("validation %T failed: %v", request, err))
			return response
		}
	}

	response.Response, err = service.Handler(context, request)
	if err != nil {
		var previous = err
		err = NewError(s.ID(), service.Action, err)
		if previous != err {
			context.Publish(msg.NewErrorEvent(fmt.Sprintf("%v", err)))
		}
		response.Err = err
	}
	return response
}

// Route returns a service action route for supplied action
func (s *AbstractService) Route(action string) (*Route, error) {
	if result, ok := s.routeByAction[action]; ok {
		return result, nil
	}
	return nil, fmt.Errorf("unknown %v.%v service action", s.id, action)
}

// Sleep sleeps for provided time in ms
func (s *AbstractService) Sleep(context *Context, sleepTimeMs int) {
	if sleepTimeMs == 0 {
		return
	}
	sleepTime := time.Millisecond * time.Duration(sleepTimeMs)
	if sleepTime < time.Minute {
		if context.IsLoggingEnabled() {
			context.Publish(msg.NewSleepEvent(sleepTimeMs))
		}
		time.Sleep(sleepTime)
		return
	}

	startTime := time.Now()
	for {
		if context.IsLoggingEnabled() {
			context.Publish(msg.NewSleepEvent(1000))
		}
		if time.Now().Sub(startTime) >= sleepTime {
			break
		}
		time.Sleep(time.Second)
	}
}

// GetHostname return host and ssh port
func (s *AbstractService) GetHostname(target *location.Resource) (string, int) {
	if target == nil {
		return "", 0
	}
	port := toolbox.AsInt(target.Port())
	if port == 0 {
		port = 22
	}
	hostname := target.Hostname()
	return hostname, port
}

// Actions returns service actions
func (s *AbstractService) Actions() []string {
	return s.actions
}

// Begin add starting event
func (s *AbstractService) Begin(context *Context, value interface{}) msg.Event {
	return context.Publish(value)
}

// End adds finishing event.
func (s *AbstractService) End(context *Context) func(startEvent msg.Event, value interface{}) msg.Event {
	return func(startEvent msg.Event, value interface{}) msg.Event {
		return context.PublishWithStartEvent(value, startEvent)
	}
}

// ID returns this service id.
func (s *AbstractService) ID() string {
	return s.id
}

// State returns this service state map.
func (s *AbstractService) State() data.Map {
	return s.state
}

func (s *AbstractService) RunInBackground(context *Context, handler func() error) (err error) {
	wait := &sync.WaitGroup{}
	wait.Add(1)
	var done uint32 = 0
	go func() {
		for {
			if atomic.LoadUint32(&done) == 1 {
				break
			}
			s.Sleep(context, 2000)
		}
	}()

	go func() {
		defer wait.Done()
		err = handler()

	}()
	wait.Wait()
	atomic.StoreUint32(&done, 1)
	return err
}

func (s *AbstractService) Info() string {
	if description, ok := serviceDescriptor[s.id]; ok {
		return description
	}
	return ""
}

// NewAbstractService creates a new abstract service.
func NewAbstractService(id string) *AbstractService {
	return &AbstractService{
		id:             id,
		actions:        make([]string, 0),
		RWMutex:        &sync.RWMutex{},
		state:          data.NewMap(),
		routeByAction:  make(map[string]*Route),
		routeByRequest: make(map[reflect.Type]*Route),
	}
}

// NopRequest represent no operation to be deprecated
type NopRequest struct {
	In interface{}
}

// nopService represents no operation nopService (deprecated, use workflow, nop instead)
type nopService struct {
	*AbstractService
}

func (s *nopService) registerRoutes() {
	s.Register(&Route{
		Action: "nop",
		RequestInfo: &ActionInfo{
			Description: "no operation action, helper for separating action.Init as self descriptive steps",
		},
		RequestProvider: func() interface{} {
			return &NopRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*NopRequest); ok {
				return req.In, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

// newNopService creates a new NoOperation nopService.
func newNopService() Service {
	var result = &nopService{
		AbstractService: NewAbstractService("nop"),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}

var serviceDescriptor = map[string]string{
	"aws/apigateway":       "Manages API endpoints that allow HTTP integration for AWS services.",
	"aws/cloudwatch":       "Monitors AWS resources and applications in real-time.",
	"aws/cloudwatchevents": "Triggers AWS Lambda functions, streams, or notifications based on specific events.",
	"aws/dynamodb":         "Provides operations on DynamoDB databases including CRUD and table management.",
	"aws/ec2":              "Manages AWS EC2 instances including their lifecycle, configuration, and networking.",
	"aws/iam":              "Handles AWS Identity and Access Management for securely controlling access to AWS services.",
	"aws/kinesis":          "Offers operations for Amazon Kinesis handling real-time data streams.",
	"aws/kms":              "Manages keys and performs cryptographic operations using AWS Key Management Service.",
	"aws/lambda":           "Automates AWS Lambda functions deployment and management.",
	"aws/logs":             "Works with CloudWatch Logs for monitoring, storing, and accessing log data.",
	"aws/rds":              "Manages AWS relational database service instances including setup, operations, and scaling.",
	"aws/s3":               "Operates on Amazon S3 objects and buckets for storage management.",
	"aws/ses":              "Integrates with Amazon Simple Email Service for sending emails.",
	"aws/sns":              "Manages Amazon Simple Notification Service for pub/sub, notifications, and alerts.",
	"aws/sqs":              "Provides access to Amazon Simple Queue Service for message queuing.",
	"aws/ssm":              "Manages AWS System Manager for resource grouping, configuration, and automation.",
	"gcp/bigquery":         "Manages Google BigQuery operations for data querying and storage.",
	"gcp/cloudfunctions":   "Deploys and manages Google Cloud Functions.",
	"gcp/cloudscheduler":   "Manages cron jobs in Google Cloud.",
	"gcp/compute":          "Manages Google Compute Engine resources including VMs and networks.",
	"gcp/container":        "Operates Google Kubernetes Engine for deploying and managing containers.",
	"gcp/kms":              "Manages cryptographic keys in Google Cloud.",
	"gcp/pubsub":           "Handles interactions with Google Cloud Pub/Sub for asynchronous messaging services.",
	"gcp/run":              "Manages Google Cloud Run applications for containerized apps.",
	"gcp/storage":          "Provides operations on Google Cloud Storage for object storage management.",
	"http/endpoint":        "Manages HTTP services including endpoints and requests.",
	"http/runner":          "Manages HTTP request sending and handling.",
	"rest/runner":          "Executes RESTful requests.",
	"docker":               "Manages Docker containers, including lifecycle operations and image management.",
	"dsunit":               "Handles database operations for testing, including setups, assertions, and data management.",
	"storage":              "Manages file storage operations across different storage services including local and cloud.",
	"secret":               "Handles secrets management including secure storage and retrieval.",
	"smtp":                 "Manages SMTP services for sending and receiving emails.",
	"validator":            "Provides data validation and log assertion functionalities.",
	"version/control":      "Manages version control operations including checkouts and commits.",
	"webdriver":            "Manages browser automation for testing web applications.",
	"slack":                "Integrates with Slack for messaging and notifications.",
	"process":              "Manages system processes including starting, stopping, and monitoring.",
	"msg":                  "Handles message queuing and pub/sub operations.",
	"migrator":             "Automates the migration of collections and workflows.",
	"sdk":                  "Manages software development kits within SSH sessions.",
	"daemon":               "Manages background services on host machines.",
	"build":                "Handles the build processes of applications according to specified parameters.",
	"deployment":           "Manages the deployment of applications to various environments.",
	"udf":                  "Supports user-defined functions for specific operations or processes.",
	"workflow":             "Manages the execution and lifecycle of defined workflows.",
}
