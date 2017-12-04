package endly

import (
	"fmt"
)

//EventReporterServiceID represents event reporter service id
const EventReporterServiceID = "event/reporter"

//EventReporterFilter represents event reporter fitler
type EventReporterFilter struct {
	EventType string
	Workflow  string
	Task      string
	Action    string
	Level     int
}

//EventReporterRequest represents  an event reporter request
type EventReporterRequest struct {
	SessionID string
	Level     int
	Filters   []*EventReporterFilter
}

//EventReporterResponse represents an event reporter response
type EventReporterResponse struct {
	Events []*Event
}

type eventReporterService struct {
	*AbstractService
}

func (s *eventReporterService) getWorkFlowContext(context *Context, sessionID string) (*Context, error) {
	service, err := context.Service(WorkflowServiceID)
	if err != nil {
		return nil, err
	}
	service.Mutex().RLock()
	defer service.Mutex().RUnlock()
	var serviceState = service.State()
	value := serviceState.Get(sessionID)
	if value == nil {
		return nil, nil
	}
	workflowContext, _ := value.(*Context)
	return workflowContext, nil

}

func (s *eventReporterService) report(context *Context, request *EventReporterRequest) (interface{}, error) {
	var response = &EventReporterResponse{}
	var events = make([]*Event, 0)
	workflowContext, err := s.getWorkFlowContext(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	if workflowContext == nil {
		return response, nil
	}
	var eventCount = len(workflowContext.Events.Events)
	for i := 0; i < eventCount; i++ {
		var event = workflowContext.Events.Shift()
		if event == nil {
			break
		}
		events = append(events, event)
	}
	response.Events = events
	return response, nil
}

func (s *eventReporterService) Run(context *Context, request interface{}) *ServiceResponse {
	var err error
	var response = &ServiceResponse{Status: "ok"}
	switch actualRequest := request.(type) {
	case *EventReporterRequest:
		response.Response, err = s.report(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to run eventReporter: %v, %v", actualRequest.SessionID, err)
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *eventReporterService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "report":
		return &EventReporterRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewEventReporterService creates a new event reporter service.
func NewEventReporterService() Service {
	var result = &eventReporterService{
		AbstractService: NewAbstractService(EventReporterServiceID),
	}
	result.AbstractService.Service = result
	return result
}
