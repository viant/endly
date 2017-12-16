package endly

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	messageTypeAction = iota
	messageTypeTagDescription
	messageTypeError
	messageTypeSuccess
	messageTypeGeneric
)

var reportingEventSleep = 250 * time.Millisecond

//EventTag represents an event tag
type EventTag struct {
	Description    string
	Workflow       string
	TagID          string
	Events         []*Event
	ValidationInfo []*ValidationInfo
	PassedCount    int
	FailedCount    int
}

//AddEvent add provided event
func (e *EventTag) AddEvent(event *Event) {
	if len(e.Events) == 0 {
		e.Events = make([]*Event, 0)
	}
	e.Events = append(e.Events, event)
}

var colors = map[string]func(arg interface{}) aurora.Value{
	"red":     aurora.Red,
	"green":   aurora.Green,
	"blue":    aurora.Blue,
	"bold":    aurora.Bold,
	"brown":   aurora.Brown,
	"gray":    aurora.Gray,
	"cyan":    aurora.Cyan,
	"magenta": aurora.Magenta,
	"inverse": aurora.Inverse,
}

//ReportSummaryEvent represents event summary
type ReportSummaryEvent struct {
	ElapsedMs      int
	TotalTagPassed int
	TotalTagFailed int
	Error          bool
}

//CliRunner represents command line runner
type CliRunner struct {
	manager    Manager
	tags       []*EventTag
	indexedTag map[string]*EventTag
	activities *Activities
	eventTag   *EventTag
	report     *ReportSummaryEvent
	activity   *WorkflowServiceActivity

	lines              int
	lineRefreshCount   int
	ErrorEvent         *Event
	InputColor         string
	OutputColor        string
	PathColor          string
	TagColor           string
	InverseTag         bool
	ServiceActionColor string
	MessageTypeColor   map[int]string
	SuccessColor       string
	ErrorColor         string
}

//AddTag adds reporting tag
func (r *CliRunner) AddTag(eventTag *EventTag) {
	r.tags = append(r.tags, eventTag)
	r.indexedTag[eventTag.TagID] = eventTag
}

//EventTag returns an event tag
func (r *CliRunner) EventTag() *EventTag {
	if len(*r.activities) == 0 {
		if r.eventTag == nil {
			r.eventTag = &EventTag{}
			r.tags = append(r.tags, r.eventTag)
		}
		return r.eventTag
	}

	activity := r.activities.Last()

	if _, has := r.indexedTag[activity.TagID]; !has {
		eventTag := &EventTag{
			Workflow: activity.Workflow,
			TagID:    activity.TagID,
		}
		r.AddTag(eventTag)
	}

	return r.indexedTag[activity.TagID]
}

func (r *CliRunner) hasActiveSession(context *Context, sessionID string) bool {
	service, err := context.Service(WorkflowServiceID)
	if err != nil {
		return false
	}
	var state = service.State()
	service.Mutex().RLock()
	defer service.Mutex().RUnlock()
	return state.Has(sessionID)
}

func (r *CliRunner) printInput(output string) {
	fmt.Printf("%v\n", colorText(output, r.InputColor))
}

func (r *CliRunner) printOutput(output string) {
	fmt.Printf("%v\n", colorText(output, r.OutputColor))
}

func (r *CliRunner) printShortMessage(messageType int, message string, messageInfoType int, messageInfo string) {
	fmt.Printf("%v\n", r.formatShortMessage(messageType, message, messageInfoType, messageInfo))
}

func (r *CliRunner) reportEvenType(serviceResponse interface{}, event *Event, filter *RunnerReportingFilter) {
	switch actual := serviceResponse.(type) {
	case *ServiceResponse:
		if actual.Response != nil {
			r.reportEvenType(actual.Response, event, filter)
		}
	case *ValidationInfo:
		r.reportValidationInfo(actual, event)
	case *HTTPRequest:
		if filter.HTTPTrip {
			r.reportHTTPRequest(actual)
		}
	case *HTTPResponse:
		if filter.HTTPTrip {
			r.reportHTTPResponse(actual)
		}
	case *DeploymentDeployRequest:
		if filter.Deployment {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("app: %v, forced: %v", actual.AppName, actual.Force), messageTypeGeneric, "deploy")
		}
	case *DsUnitRegisterRequest:
		if filter.RegisterDatastore {

			var descriptor = actual.Config.Descriptor
			var password = actual.Config.Parameters["password"]
			if len(password) > 0 {
				descriptor = strings.Replace(descriptor, password, "***", len(descriptor))
			}
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("Datastore: %v, %v:%v", actual.Datastore, actual.Config.DriverName, descriptor), messageTypeGeneric, "register")
		}
	case *DsUnitMappingRequest:
		if filter.DataMapping {
			for _, mapping := range actual.Mappings {
				r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v: %v", mapping.Name, mapping.URL), messageTypeGeneric, "mapping")
			}
		}

	case *DsUnitTableSequenceResponse:
		if filter.Sequence {
			for k, v := range actual.Sequences {
				r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v: %v", k, v), messageTypeGeneric, "sequence")
			}
		}
	case *PopulateDatastoreEvent:
		if filter.PopulateDatastore {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("(%v) %v: %v", actual.Datastore, actual.Table, actual.Rows), messageTypeGeneric, "populate")
		}
	case *RunSQLScriptEvent:
		if filter.SQLScript {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("(%v) %v", actual.Datastore, actual.URL), messageTypeGeneric, "sql")
		}

	case *ErrorEventType:
		r.report.Error = true
		r.printShortMessage(messageTypeError, fmt.Sprintf("%v", actual.Error), messageTypeError, "error")
		fmt.Println(colorText(fmt.Sprintf("ERROR: %v\n", actual.Error), "red"))

	case *SleepEventType:
		r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v ms", actual.SleepTimeMs), messageTypeGeneric, "Sleep")
	case *VcCheckoutRequest:
		if filter.Checkout {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v %v", actual.Origin.URL, actual.Target.URL), messageTypeGeneric, "checkout")
		}

	case *BuildRequest:
		if filter.Build {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v %v", actual.BuildSpec.Name, actual.Target.URL), messageTypeGeneric, "build")
		}

	case *ExecutionStartEvent:
		if filter.Stdin {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v", actual.SessionID), messageTypeGeneric, "stdin")

			r.printInput(escapeStdout(actual.Stdin))
		}
	case *ExecutionEndEvent:
		if filter.Stdout {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v", actual.SessionID), messageTypeGeneric, "stdout")
			r.printOutput(escapeStdout(actual.Stdout))

		}
	case *CopyEventType:
		if filter.Transfer {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("expand: %v", actual.Expand), messageTypeGeneric, "copy")
			r.printInput(fmt.Sprintf("SourceURL: %v", actual.SourceURL))
			r.printOutput(fmt.Sprintf("TargetURL: %v", actual.TargetURL))

		}

	case *AsyncServiceActionEvent:
		r.printShortMessage(messageTypeTagDescription, fmt.Sprintf("%v[%v] %v", actual.TagID, actual.Task, actual.Description), messageTypeTagDescription, "scheduled")

	case *WorkflowServiceActivity:
		r.activities.Push(actual)
		r.activity = actual
		if actual.TagDescription != "" {
			r.printShortMessage(messageTypeTagDescription, actual.TagDescription, messageTypeTagDescription, "")
			eventTag := r.EventTag()
			eventTag.Description = actual.TagDescription
		}
		var serviceAction = fmt.Sprintf("%v.%v", actual.Service, actual.Action)
		r.printShortMessage(messageTypeAction, actual.Description, messageTypeAction, serviceAction)

	case *SeleniumRunResponse:
		r.reportLookupErrors(actual)

	case *LogValidatorAssertResponse:
		r.reportLogValidationInfo(actual)

	case *WorkflowServiceActivityEndEventType:

		r.activity = r.activities.Pop()
	case *ReportSummaryEvent:
		r.reportSummaryEvent()
	}
}

func (r *CliRunner) reportLogValidationInfo(response *LogValidatorAssertResponse) {
	var passedCount, failedCount = 0, 0
	for _, info := range response.ValidationInfo {
		if info.HasFailure() {
			failedCount++
		} else if info.TestPassed > 0 {
			passedCount++
		}
		if r.activity != nil {
			var tagID = info.TagID
			if eventTag, ok := r.indexedTag[tagID]; ok {
				eventTag.AddEvent(&Event{Type: "LogValidation", Value: Pairs("value", info)})
				eventTag.PassedCount += info.TestPassed
				eventTag.FailedCount += len(info.FailedTests)

			}
		}
	}
	var total = passedCount + failedCount
	messageType := messageTypeSuccess
	messageInfo := "OK"
	var message = fmt.Sprintf("Passed %v/%v %v", passedCount, total, response.Description)
	if failedCount > 0 {
		messageType = messageTypeError
		message = fmt.Sprintf("Passed %v/%v %v", passedCount, total, response.Description)
		messageInfo = "FAILED"
	}
	r.printShortMessage(messageType, message, messageType, messageInfo)
}

func (r *CliRunner) extractHTTPTrips(eventCandidates []*Event) ([]*HTTPRequest, []*HTTPResponse) {
	var requests = make([]*HTTPRequest, 0)
	var responses = make([]*HTTPResponse, 0)
	for _, event := range eventCandidates {
		request := event.get(reflect.TypeOf(&HTTPRequest{}))
		if request != nil {
			if httpRequest, ok := request.(*HTTPRequest); ok {
				requests = append(requests, httpRequest)
			}
		}
		response := event.get(reflect.TypeOf(&HTTPResponse{}))
		if response != nil {
			if httpResponse, ok := response.(*HTTPResponse); ok {
				responses = append(responses, httpResponse)
			}
		}

	}

	return requests, responses
}

func (r *CliRunner) reportFailureWithMatchSource(tag *EventTag, info *ValidationInfo, eventCandidates []*Event) {
	var theFirstFailure = info.FailedTests[0]
	var requests []*HTTPRequest
	var responses []*HTTPResponse

	if theFirstFailure.PathIndex != -1 && (strings.Contains(theFirstFailure.Path, "Body") || strings.Contains(theFirstFailure.Path, "Code") || strings.Contains(theFirstFailure.Path, "Cookie") || strings.Contains(theFirstFailure.Path, "Header")) {
		requests, responses = r.extractHTTPTrips(eventCandidates)
		if theFirstFailure.PathIndex < len(requests) {
			r.reportHTTPRequest(requests[theFirstFailure.PathIndex])
		}
		if theFirstFailure.PathIndex < len(responses) {
			r.reportHTTPResponse(responses[theFirstFailure.PathIndex])
		}
	}

	var counter = 0
	for _, failure := range info.FailedTests {
		path := failure.Path
		if failure.PathIndex != -1 {
			path = fmt.Sprintf("%v:%v", failure.PathIndex, failure.Path)
		}
		r.printMessage(path, len(path), messageTypeError, failure.Message, messageTypeError, "Failed")
		if theFirstFailure.PathIndex != failure.PathIndex || counter >= 3 {
			break
		}
		counter++
	}
}

func (r *CliRunner) reportSummaryEvent() {
	r.reportTagSummary()
	contextMessage := "STATUS: "
	var contextMessageColor = "green"
	contextMessageStatus := "SUCCESS"
	if r.report.Error || r.report.TotalTagFailed > 0 {
		contextMessageColor = "red"
		contextMessageStatus = "FAILED"
	}
	var contextMessageLength = len(contextMessage) + len(contextMessageStatus)
	contextMessage = fmt.Sprintf("%v%v", contextMessage, colorText(contextMessageStatus, contextMessageColor))
	r.printMessage(contextMessage, contextMessageLength, messageTypeGeneric, fmt.Sprintf("Passed %v/%v", r.report.TotalTagPassed, (r.report.TotalTagPassed+r.report.TotalTagFailed)), messageTypeGeneric, fmt.Sprintf("elapsed: %v ms", r.report.ElapsedMs))
}

func (r *CliRunner) reportTagSummary() {
	for _, tag := range r.tags {
		if tag.FailedCount > 0 {
			var eventTag = tag.TagID
			r.printMessage(colorText(eventTag, "red"), len(eventTag), messageTypeTagDescription, tag.Description, messageTypeError, fmt.Sprintf("failed %v/%v", tag.FailedCount, (tag.FailedCount+tag.PassedCount)))

			var minRange = 0
			for i, event := range tag.Events {
				candidate := event.get(reflect.TypeOf(&ValidationInfo{}))
				if info, ok := candidate.(*ValidationInfo); ok && info.HasFailure() {
					var failureSourceEvent = []*Event{}
					if i-minRange > 0 {
						failureSourceEvent = tag.Events[minRange : i-1]
					}
					r.reportFailureWithMatchSource(tag, info, failureSourceEvent)
					minRange = i + 1
				}
			}

		}
	}
}

func (r *CliRunner) reportLookupErrors(response *SeleniumRunResponse) {
	if len(response.LookupErrors) > 0 {
		for _, errMessage := range response.LookupErrors {
			r.printShortMessage(messageTypeError, errMessage, messageTypeGeneric, "Selenium")
		}
	}
}



func (r *CliRunner) reportHTTPResponse(response *HTTPResponse) {
	r.printShortMessage(messageTypeGeneric, fmt.Sprintf("StatusCode: %v", response.Code), messageTypeGeneric, "HttpResponse")
	if len(response.Header) > 0 {
		r.printShortMessage(messageTypeGeneric, "Headers", messageTypeGeneric, "HttpResponse")
		r.printOutput(asJSONText(response.Header))
	}
	r.printShortMessage(messageTypeGeneric, "Body", messageTypeGeneric, "HttpResponse")
	r.printOutput(response.Body)
}

func (r *CliRunner) reportHTTPRequest(request *HTTPRequest) {
	r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v %v", request.Method, request.URL), messageTypeGeneric, "HttpRequest")
	r.printInput(asJSONText(request.URL))
	if len(request.Header) > 0 {
		r.printShortMessage(messageTypeGeneric, "Headers", messageTypeGeneric, "HttpRequest")
		r.printInput(asJSONText(request.Header))
	}
	if len(request.Cookies) > 0 {
		r.printShortMessage(messageTypeGeneric, "Cookies", messageTypeGeneric, "HttpRequest")
		r.printInput(asJSONText(request.Cookies))
	}
	r.printShortMessage(messageTypeGeneric, "Body", messageTypeGeneric, "HttpRequest")
	r.printInput(request.Body)
}

func (r *CliRunner) reportValidationInfo(info *ValidationInfo, event *Event) {
	var total = info.TestPassed + len(info.FailedTests)
	var description = info.Description

	var activity = r.activities.Last()
	if activity != nil {
		var tagID = info.TagID
		eventTag, ok := r.indexedTag[tagID]
		if !ok {
			eventTag = r.EventTag()
		}
		eventTag.FailedCount += len(info.FailedTests)
		eventTag.PassedCount += info.TestPassed

	}
	messageType := messageTypeSuccess
	messageInfo := "OK"
	var message = fmt.Sprintf("Passed %v/%v %v", info.TestPassed, total, description)
	if len(info.FailedTests) > 0 {
		messageType = messageTypeError
		message = fmt.Sprintf("Passed %v/%v %v", info.TestPassed, total, description)
		messageInfo = "FAILED"
	}
	r.printShortMessage(messageType, message, messageType, messageInfo)
}

func (r *CliRunner) reportEvent(context *Context, event *Event, filter *RunnerReportingFilter) error {
	defer func() {
		eventTag := r.EventTag()
		eventTag.AddEvent(event)
	}()
	if event.Level > Debug {
		return nil
	}
	r.processEvents(event, filter)
	return nil
}

func (r *CliRunner) processEvents(event *Event, filter *RunnerReportingFilter) {
	for _, value := range event.Value {
		r.reportEvenType(value, event, filter)
	}
}

func (r *CliRunner) getReportedEvents(context *Context, service Service, sessionID string) (*EventReporterResponse, error) {
	response := service.Run(context, &EventReporterRequest{
		SessionID: sessionID,
	})
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	reporterResponse, ok := response.Response.(*EventReporterResponse)
	if !ok {
		return nil, fmt.Errorf("failed to check event - unexpected reponse type: %T", response.Response)
	}
	return reporterResponse, nil
}

func (r *CliRunner) reportEvents(context *Context, sessionID string, filter *RunnerReportingFilter) error {
	service, err := context.Service(EventReporterServiceID)
	if err != nil {
		return err
	}

	r.report = &ReportSummaryEvent{}
	defer func(){
		_ = r.reportEvent(context, &Event{Type: "ReportSummaryEvent", Value: Pairs("value", r.report)}, filter)
	}()

	time.Sleep(time.Second)
	var firstEvent *Event
	var lastEvent *Event

	if context.Workflow() != nil {
		var workflow = context.Workflow().Name
		var workflowLength = len(workflow)
		r.printMessage(colorText(workflow, r.TagColor), workflowLength, messageTypeGeneric, fmt.Sprintf("%v", time.Now()), messageTypeGeneric, "started")
	}
	for {
		reporterResponse, err := r.getReportedEvents(context, service, sessionID)
		if err != nil {
			return err
		}
		if len(reporterResponse.Events) == 0 {
			if !r.hasActiveSession(context, sessionID) {
				break
			}
			time.Sleep(reportingEventSleep)
			continue
		}

		for _, event := range reporterResponse.Events {
			if firstEvent == nil {
				firstEvent = event
			} else {
				lastEvent = event
				r.report.ElapsedMs = int(lastEvent.Timestamp.UnixNano()-firstEvent.Timestamp.UnixNano()) / int(time.Millisecond)
			}
			err = r.reportEvent(context, event, filter)
			if err != nil {
				return err
			}
		}

	}
	r.processEventTags()
	return nil
}

func (r *CliRunner) processEventTags() {
	for _, eventTag := range r.tags {
		if eventTag.FailedCount > 0 {
			r.report.TotalTagFailed++
		} else if eventTag.PassedCount > 0 {
			r.report.TotalTagPassed++
		}
	}
}

//Run run workflow for the specified URL
func (r *CliRunner) Run(workflowRunRequestURL string) error {
	request := &WorkflowRunRequest{}
	resource := url.NewResource(workflowRunRequestURL)
	err := resource.JSONDecode(request)
	if err != nil {
		return err
	}
	ctx := r.manager.NewContext(toolbox.NewContext())
	defer ctx.Close()
	service, err := ctx.Service(WorkflowServiceID)
	if err != nil {
		return err
	}
	runnerOption := &RunnerReportingOption{}
	err = resource.JSONDecode(runnerOption)
	if err != nil {
		return err
	}
	if runnerOption.Filter == nil {
		runnerOption.Filter = &RunnerReportingFilter{}
	}
	request.Async = true
	response := service.Run(ctx, request)
	if response.Error != "" {
		return errors.New(response.Error)
	}
	workflowResponse, ok := response.Response.(*WorkflowRunResponse)
	if !ok {
		return fmt.Errorf("failed to run workflow: %v invalid response type %T", workflowRunRequestURL, response.Response)
	}
	return r.reportEvents(ctx, workflowResponse.SessionID, runnerOption.Filter)
}

func colorText(text, color string) string {
	if color, has := colors[color]; has {
		return fmt.Sprintf("%v", color(text))
	}
	return text
}

var minColumns = 120

func (r *CliRunner) columns() int {

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "tput", "cols")
	output, err := cmd.CombinedOutput()
	if err == nil {
		r.lines, err = strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			r.lines = minColumns
		}
	}
	if r.lines < minColumns {
		r.lines = minColumns
	}
	return r.lines
}

func (r *CliRunner) printMessage(contextMessage string, contextMessageLength int, messageType int, message string, messageInfoType int, messageInfo string) {
	fmt.Printf("%v\n", r.formatMessage(contextMessage, contextMessageLength, messageType, message, messageInfoType, messageInfo))
}

func (r *CliRunner) formatMessage(contextMessage string, contextMessageLength int, messageType int, message string, messageInfoType int, messageInfo string) string {
	var columns = r.columns() - 5
	var infoLength = len(messageInfo)
	var messageLength = columns - contextMessageLength - infoLength

	if messageLength < len(message) {
		if messageLength > 1 {
			message = message[:messageLength]
		} else {
			message = "."
		}
	}
	message = fmt.Sprintf("%-"+toolbox.AsString(messageLength)+"v", message)
	messageInfo = fmt.Sprintf("%"+toolbox.AsString(infoLength)+"v", messageInfo)

	if messageColor, ok := r.MessageTypeColor[messageType]; ok {
		message = colorText(message, messageColor)
	}

	messageInfo = colorText(messageInfo, "bold")
	if messageInfoColor, ok := r.MessageTypeColor[messageInfoType]; ok {
		messageInfo = colorText(messageInfo, messageInfoColor)
	}
	return fmt.Sprintf("[%v %v %v]", contextMessage, message, messageInfo)
}

func (r *CliRunner) formatShortMessage(messageType int, message string, messageInfoType int, messageInfo string) string {
	var fullPath = !(messageType == messageTypeTagDescription || messageInfoType == messageTypeAction)
	var path, pathLength = "", 0
	if len(*r.activities) > 0 {
		path, pathLength = r.activities.GetPath(r, fullPath)
	}
	var result = r.formatMessage(path, pathLength, messageType, message, messageInfoType, messageInfo)
	if strings.Contains(result, message) {
		return result
	}
	return fmt.Sprintf("%v\n%v", result, message)
}

//NewCliRunner creates a new command line runner
func NewCliRunner() *CliRunner {
	var activities Activities = make([]*WorkflowServiceActivity, 0)
	return &CliRunner{
		manager:            NewManager(),
		tags:               make([]*EventTag, 0),
		indexedTag:         make(map[string]*EventTag),
		activities:         &activities,
		InputColor:         "blue",
		OutputColor:        "green",
		PathColor:          "brown",
		TagColor:           "brown",
		InverseTag:         true,
		ServiceActionColor: "gray",
		MessageTypeColor: map[int]string{
			messageTypeTagDescription: "cyan",
			messageTypeError:          "red",
			messageTypeSuccess:        "green",
		},
	}
}

func asJSONText(source interface{}) string {
	if source == nil {
		return ""
	}
	var buf = new(bytes.Buffer)
	toolbox.NewJSONEncoderFactory().Create(buf).Encode(source)
	return buf.String()
}
