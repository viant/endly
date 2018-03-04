package cli

import (
	"errors"
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/dsunit"
	"github.com/viant/endly"
	"github.com/viant/endly/runner/http"
	"github.com/viant/endly/runner/selenium"
	"github.com/viant/endly/testing/log"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/viant/endly/deployment/build"
	"github.com/viant/endly/deployment/deploy"
	"github.com/viant/endly/deployment/vc"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
	"github.com/viant/endly/util"
)

//OnError exit system with os.Exit with supplied code.
var OnError = func(code int) {
	os.Exit(code)
}

const (
	messageTypeAction = iota
	messageTypeTagDescription
	messageTypeError
	messageTypeSuccess
	messageTypeGeneric
)


//EventTag represents an event tag
type EventTag struct {
	Description string
	Workflow    string
	TagID       string
	Events      []*endly.Event
	Validation  []*assertly.Validation
	PassedCount int
	FailedCount int
}

//AddEvent add provided event
func (e *EventTag) AddEvent(event *endly.Event) {
	if len(e.Events) == 0 {
		e.Events = make([]*endly.Event, 0)
	}
	e.Events = append(e.Events, event)
}

//ReportSummaryEvent represents event summary
type ReportSummaryEvent struct {
	ElapsedMs      int
	TotalTagPassed int
	TotalTagFailed int
	Error          bool
}

//Runner represents command line runner
type Runner struct {
	*Renderer
	context          *endly.Context
	filter           *Filter
	manager          endly.Manager
	tags             []*EventTag
	indexedTag       map[string]*EventTag
	activities       *endly.Activities
	eventTag         *EventTag
	report           *ReportSummaryEvent
	activity         *endly.Activity
	ErrorEvent       *endly.Event
	errorCode        bool
	err              error
	lines            int
	lineRefreshCount int

	InputColor         string
	OutputColor        string
	PathColor          string
	TagColor           string
	InverseTag         bool
	ServiceActionColor string

	MessageTypeColor map[int]string
	SuccessColor     string
	ErrorColor       string

	SleepCount int
	SleepTime  time.Duration
	SleepTagID string
}

//AddTag adds reporting tag
func (r *Runner) AddTag(eventTag *EventTag) {
	r.tags = append(r.tags, eventTag)
	r.indexedTag[eventTag.TagID] = eventTag
}

//EventTag returns an event tag
func (r *Runner) EventTag() *EventTag {
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

func (r *Runner) hasActiveSession(context *endly.Context, sessionID string) bool {
	service, err := context.Service(endly.ServiceID)
	if err != nil {
		return false
	}
	var state = service.State()
	service.Mutex().RLock()
	defer service.Mutex().RUnlock()
	return state.Has(sessionID)
}

func (r *Runner) printInput(output string) {
	r.Printf("%v\n", r.ColorText(output, r.InputColor))
}

func (r *Runner) printOutput(output string) {
	r.Printf("%v\n", r.ColorText(output, r.OutputColor))
}

func (r *Runner) printError(output string) {
	r.Printf("%v\n", r.ColorText(output, r.ErrorColor))
}

func (r *Runner) printShortMessage(messageType int, message string, messageInfoType int, messageInfo string) {
	r.Printf("%v\n", r.formatShortMessage(messageType, message, messageInfoType, messageInfo))
}

func (r *Runner) overrideShortMessage(messageType int, message string, messageInfoType int, messageInfo string) {
	r.Printf("\r%v", r.formatShortMessage(messageType, message, messageInfoType, messageInfo))
}

func (r *Runner) printMessage(contextMessage string, contextMessageLength int, messageType int, message string, messageInfoType int, messageInfo string) {
	r.Printf("%v\n", r.formatMessage(contextMessage, contextMessageLength, messageType, message, messageInfoType, messageInfo))
}

func (r *Runner) formatMessage(contextMessage string, contextMessageLength int, messageType int, message string, messageInfoType int, messageInfo string) string {
	var columns = r.Columns() - 5
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
		message = r.ColorText(message, messageColor)
	}

	messageInfo = r.ColorText(messageInfo, "bold")
	if messageInfoColor, ok := r.MessageTypeColor[messageInfoType]; ok {
		messageInfo = r.ColorText(messageInfo, messageInfoColor)
	}
	return fmt.Sprintf("[%v %v %v]", contextMessage, message, messageInfo)
}

func (r *Runner) formatShortMessage(messageType int, message string, messageInfoType int, messageInfo string) string {
	var fullPath = !(messageType == messageTypeTagDescription || messageInfoType == messageTypeAction)
	var path, pathLength = "", 0
	if len(*r.activities) > 0 {
		path, pathLength = GetPath(r.activities, r, fullPath)
	}
	var result = r.formatMessage(path, pathLength, messageType, message, messageInfoType, messageInfo)
	if strings.Contains(result, message) {
		return result
	}
	return fmt.Sprintf("%v\n%v", result, message)
}

func (r *Runner) resetSleepCounterIfNeeded() {
	if r.SleepCount > 0 {
		r.Printf("\n")
		r.SleepCount = 0
	}
}

func (r *Runner) processDsunitEvent(value interface{}, filter *Filter) bool {
	switch actual := value.(type) {
	case *dsunit.InitRequest:
		r.processDsunitEvent(actual.RegisterRequest, filter)
		if actual.RunScriptRequest != nil {
			r.processDsunitEvent(actual.RunScriptRequest, filter)
		}

	case *dsunit.RegisterRequest:
		if filter.RegisterDatastore {
			var descriptor = actual.Config.SecureDescriptor
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("Datastore: %v, %v:%v", actual.Datastore, actual.Config.DriverName, descriptor), messageTypeGeneric, "register")
		}
	case *dsunit.MappingRequest:
		if filter.DataMapping {
			for _, mapping := range actual.Mappings {
				var _, name = toolbox.URLSplit(mapping.Name)
				r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v: %v", name, mapping.Name), messageTypeGeneric, "mapping")
			}
		}
	case *dsunit.SequenceRequest:
		if filter.Sequence {
			for k, v := range actual.Tables {
				r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v: %v", k, v), messageTypeGeneric, "sequence")
			}
		}
	case *dsunit.PrepareRequest:
		if filter.PopulateDatastore {
			actual.Load()
			for _, dataset := range actual.Datasets {
				r.printShortMessage(messageTypeGeneric, fmt.Sprintf("(%v) %v: %v", actual.Datastore, dataset.Table, len(dataset.Records)), messageTypeGeneric, "populate")
			}
		}
	case *dsunit.RunScriptRequest:
		if filter.SQLScript {
			for _, script := range actual.Scripts {
				r.printShortMessage(messageTypeGeneric, fmt.Sprintf("(%v) %v", actual.Datastore, script.URL), messageTypeGeneric, "sql")
			}
		}
	default:
		return false
	}
	return true
}

func (r *Runner) processHTTPEvent(event *endly.Event, filter *Filter) bool {
	switch actual := event.Value.(type) {
	case *http.Request:
		if filter.HTTPTrip {
			r.reportHTTPRequest(actual)
		}
	case *http.Response:
		if filter.HTTPTrip {
			r.reportHTTPResponse(actual)
		}
	default:
		return false
	}
	return true
}

func (r *Runner) processValidationEvent(event *endly.Event, filter *Filter) bool {
	switch response := event.Value.(type) {
	case *assertly.Validation:
		r.reportValidation(response, event)
	case *dsunit.ExpectResponse:
		if response != nil {
			for _, validation := range response.Validation {
				if validation.Validation != nil {
					r.reportValidation(validation.Validation, event)
				}
			}
		}
	case *validator.AssertResponse:
		if response != nil {
			r.reportValidation(response.Validation, event)
		}
	case *log.AssertResponse:
		r.reportLogValidation(response)
	case *selenium.RunResponse:
		r.reportLookupErrors(response)
	default:
		return false
	}
	return true
}

func (r *Runner) processWorkflowEvent(event *endly.Event, filter *Filter) bool {
	switch actual := event.Value.(type) {
	case *endly.Activity:
		r.activities.Push(actual)
		r.activity = actual
		if actual.TagDescription != "" {
			r.printShortMessage(messageTypeTagDescription, actual.TagDescription, messageTypeTagDescription, "")
			eventTag := r.EventTag()
			eventTag.Description = actual.TagDescription
		}
		var serviceAction = fmt.Sprintf("%v.%v", actual.Service, actual.Action)
		r.printShortMessage(messageTypeAction, actual.Description, messageTypeAction, serviceAction)
	case *endly.ActivityEndEvent:
		r.activity = r.activities.Pop()
	default:
		return false
	}
	r.resetSleepCounterIfNeeded()
	return true
}

func (r *Runner) processExecutionEvent(event *endly.Event, filter *Filter) bool {
	switch actual := event.Value.(type) {
	case *exec.ExecutionStartEvent:
		if filter.Stdin {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v", actual.SessionID), messageTypeGeneric, "stdin")

			r.printInput(util.EscapeStdout(actual.Stdin))
		}
	case *exec.ExecutionEndEvent:
		if filter.Stdout {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v", actual.SessionID), messageTypeGeneric, "stdout")
			r.printOutput(util.EscapeStdout(actual.Stdout))
		}
	default:
		return false
	}
	return true
}

func (r *Runner) processEndlyEvents(event *endly.Event, filter *Filter) bool {
	switch actual := event.Value.(type) {
	case *endly.ErrorEvent:
		r.report.Error = true
		r.printShortMessage(messageTypeError, fmt.Sprintf("%v", actual.Error), messageTypeError, "error")
		r.Println(r.ColorText(fmt.Sprintf("ERROR: %v\n", actual.Error), "red"))
		r.err = errors.New(actual.Error)
		return true
	case *endly.SleepEvent:
		if r.SleepCount > 0 {
			r.overrideShortMessage(messageTypeGeneric, fmt.Sprintf("%v ms x %v,  slept so far: %v", actual.SleepTimeMs, r.SleepCount, r.SleepTime), messageTypeGeneric, "Sleep")
		} else {
			r.SleepTime = 0
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v ms", actual.SleepTimeMs), messageTypeGeneric, "Sleep")
		}
		r.SleepTagID = r.eventTag.TagID
		r.SleepTime += time.Millisecond * time.Duration(actual.SleepTimeMs)
		r.SleepCount++
		return true
	}
	r.resetSleepCounterIfNeeded()
	return false
}

func (r *Runner) processEvent(event *endly.Event, filter *Filter) {
	if event.Value == nil {
		return
	}
	if r.processEndlyEvents(event, filter) {
		return
	}
	if r.processWorkflowEvent(event, filter) {
		return
	}
	if r.processExecutionEvent(event, filter) {
		return
	}
	if r.processDsunitEvent(event.Value, filter) {
		return
	}
	if r.processHTTPEvent(event, filter) {
		return
	}
	if r.processValidationEvent(event, filter) {
		return
	}

	switch value := event.Value.(type) {
	case *endly.PrintRequest:
		if value.Message != "" {
			var message = r.Renderer.ColorText(value.Message, value.Color)
			r.Renderer.Println(message)
		} else if value.Error != "" {
			var errorMessage = r.Renderer.ColorText(value.Error, r.Renderer.ErrorColor)
			r.Renderer.Println(errorMessage)
		}

	case *deploy.Request:
		if filter.Deployment {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("app: %v, forced: %v", value.AppName, value.Force), messageTypeGeneric, "deploy")
		}

	case *vc.CheckoutRequest:
		if filter.Checkout {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v %v", value.Origin.URL, value.Target.URL), messageTypeGeneric, "checkout")
		}

	case *build.Request:
		if filter.Build {
			r.printShortMessage(messageTypeGeneric, fmt.Sprintf("%v %v", value.BuildSpec.Name, value.Target.URL), messageTypeGeneric, "build")
		}

	case *storage.RemoveRequest:
		if filter.Transfer {
			for _, resource := range value.Resources {
				r.printShortMessage(messageTypeGeneric, "", messageTypeGeneric, "remove")
				r.printInput(fmt.Sprintf("SourceURL: %v", resource.URL))
			}
		}
	case *storage.UploadRequest:
		if filter.Transfer && value.Validate() == nil {
			r.printShortMessage(messageTypeGeneric, "", messageTypeGeneric, "upload")
			r.printInput(fmt.Sprintf("SourceKe: %v", value.SourceKey))
			r.printOutput(fmt.Sprintf("TargetURL: %v", value.Target.URL))
		}
	case *storage.DownloadRequest:
		if filter.Transfer && value.Validate() == nil {
			r.printShortMessage(messageTypeGeneric, "", messageTypeGeneric, "download")
			r.printInput(fmt.Sprintf("SourceURL: %v", value.Source.URL))
			r.printOutput(fmt.Sprintf("TargetKey: %v", value.TargetKey))
		}
	case *storage.CopyRequest:
		if filter.Transfer && value.Validate() == nil {
			for _, transfer := range value.Transfers {
				r.printShortMessage(messageTypeGeneric, fmt.Sprintf("expand: %v", transfer.Expand), messageTypeGeneric, "copy")
				r.printInput(fmt.Sprintf("SourceURL: %v", transfer.Source.URL))
				r.printOutput(fmt.Sprintf("TargetURL: %v", transfer.Target.URL))
			}
		}

	}
}

func (r *Runner) reportLogValidation(response *log.AssertResponse) {
	var passedCount, failedCount = 0, 0
	if response == nil {
		return
	}
	for _, validation := range response.Validations {
		if validation.HasFailure() {
			failedCount++
			r.errorCode = true
		} else if validation.PassedCount > 0 {
			passedCount++
			continue
		}
		if r.activity != nil {
			var tagID = validation.TagID
			if eventTag, ok := r.indexedTag[tagID]; ok {
				eventTag.AddEvent(endly.NewEvent(validation))
				eventTag.PassedCount += validation.PassedCount
				eventTag.FailedCount += validation.FailedCount
			}
		}
	}
	var total = passedCount + failedCount
	messageType := messageTypeSuccess
	messageInfo := "OK"
	var message = ""
	if total > 0 {
		message = fmt.Sprintf("Passed %v/%v %v", passedCount, total, response.Description)
		if failedCount > 0 {
			messageType = messageTypeError
			message = fmt.Sprintf("Passed %v/%v %v", passedCount, total, response.Description)
			messageInfo = "FAILED"
		}
	}
	r.printShortMessage(messageType, message, messageType, messageInfo)
}

func (r *Runner) extractHTTPTrips(eventCandidates []*endly.Event) ([]*http.Request, []*http.Response) {
	var requests = make([]*http.Request, 0)
	var responses = make([]*http.Response, 0)
	for _, event := range eventCandidates {
		request := event.Get(reflect.TypeOf(&http.Request{}))
		if request != nil {
			if httpRequest, ok := request.(*http.Request); ok {
				requests = append(requests, httpRequest)
			}
		}
		response := event.Get(reflect.TypeOf(&http.Response{}))
		if response != nil {
			if httpResponse, ok := response.(*http.Response); ok {
				responses = append(responses, httpResponse)
			}
		}
	}
	return requests, responses
}

func (r *Runner) reportFailureWithMatchSource(tag *EventTag, validation *assertly.Validation, eventCandidates []*endly.Event) {
	var theFirstFailure = validation.Failures[0]
	firstFailurePathIndex := theFirstFailure.Index()
	var requests []*http.Request
	var responses []*http.Response

	if strings.Contains(theFirstFailure.Path, "Body") || strings.Contains(theFirstFailure.Path, "Code") || strings.Contains(theFirstFailure.Path, "Cookie") || strings.Contains(theFirstFailure.Path, "Header") {
		if theFirstFailure.Index() != -1 {

			requests, responses = r.extractHTTPTrips(eventCandidates)
			if firstFailurePathIndex < len(requests) {
				r.reportHTTPRequest(requests[firstFailurePathIndex])
			}
			if firstFailurePathIndex < len(responses) {
				r.reportHTTPResponse(responses[firstFailurePathIndex])
			}
		}
	}

	var counter = 0
	for _, failure := range validation.Failures {
		failurePath := failure.Path
		if failure.Index() != -1 {
			failurePath = fmt.Sprintf("%v:%v", failure.Index(), failure.Path)
		}
		r.printMessage(failurePath, len(failurePath), messageTypeError, failure.Message, messageTypeError, "Failed")
		if firstFailurePathIndex != failure.Index() || counter >= 3 {
			break
		}
		counter++
	}
}

func (r *Runner) reportSummaryEvent() {
	r.reportTagSummary()
	contextMessage := "STATUS: "
	var contextMessageColor = "green"
	contextMessageStatus := "SUCCESS"
	if r.report.Error || r.report.TotalTagFailed > 0 {
		contextMessageColor = "red"
		contextMessageStatus = "FAILED"
	}

	var contextMessageLength = len(contextMessage) + len(contextMessageStatus)
	contextMessage = fmt.Sprintf("%v%v", contextMessage, r.ColorText(contextMessageStatus, contextMessageColor))
	r.printMessage(contextMessage, contextMessageLength, messageTypeGeneric, fmt.Sprintf("Passed %v/%v", r.report.TotalTagPassed, (r.report.TotalTagPassed+r.report.TotalTagFailed)), messageTypeGeneric, fmt.Sprintf("elapsed: %v ms", r.report.ElapsedMs))
}

func (r *Runner) getValidation(event *endly.Event) *assertly.Validation {

	candidate := event.Get(reflect.TypeOf(&assertly.Validation{}))
	if candidate == nil {
		return nil
	}
	validation, ok := candidate.(*assertly.Validation)
	if !ok {
		return nil
	}
	return validation
}

func (r *Runner) getDsUnitAssertResponse(event *endly.Event) []*dsunit.DatasetValidation {
	candidate := event.Get(reflect.TypeOf(&dsunit.ExpectResponse{}))
	if candidate == nil {
		return nil
	}
	assertResponse, ok := candidate.(*dsunit.ExpectResponse)
	if !ok {
		return nil
	}
	return assertResponse.Validation
}

func (r *Runner) reportTagSummary() {
	for _, tag := range r.tags {
		if (tag.FailedCount) > 0 {
			var eventTag = tag.TagID
			r.printMessage(r.ColorText(eventTag, "red"), len(eventTag), messageTypeTagDescription, tag.Description, messageTypeError, fmt.Sprintf("failed %v/%v", tag.FailedCount, (tag.FailedCount+tag.PassedCount)))

			var minRange = 0
			for i, event := range tag.Events {

				validation := r.getValidation(event)

				if validation == nil {
					continue
				}

				if validation.HasFailure() {
					var failureSourceEvent = []*endly.Event{}
					if i-minRange > 0 {
						failureSourceEvent = tag.Events[minRange : i-1]
					}
					r.reportFailureWithMatchSource(tag, validation, failureSourceEvent)
					minRange = i + 1
				}
			}

		}
	}
}

func (r *Runner) reportLookupErrors(response *selenium.RunResponse) {
	if response == nil {
		return
	}
	if len(response.LookupErrors) > 0 {
		for _, errMessage := range response.LookupErrors {
			r.printShortMessage(messageTypeError, errMessage, messageTypeGeneric, "Selenium")
		}
	}
}

func asJSONText(source interface{}) string {
	text, _ := toolbox.AsJSONText(source)
	return text
}

func (r *Runner) reportHTTPResponse(response *http.Response) {
	r.printShortMessage(messageTypeGeneric, fmt.Sprintf("StatusCode: %v", response.Code), messageTypeGeneric, "HttpResponse")
	if len(response.Header) > 0 {
		r.printShortMessage(messageTypeGeneric, "Headers", messageTypeGeneric, "HttpResponse")

		r.printOutput(asJSONText(response.Header))
	}
	r.printShortMessage(messageTypeGeneric, "Body", messageTypeGeneric, "HttpResponse")
	r.printOutput(response.Body)
}

func (r *Runner) reportHTTPRequest(request *http.Request) {
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

func (r *Runner) reportValidation(validation *assertly.Validation, event *endly.Event) {
	if validation == nil {
		return
	}
	var total = validation.PassedCount + validation.FailedCount
	var description = validation.Description
	var activity = r.activities.Last()
	if activity != nil {
		var tagID = validation.TagID
		eventTag, ok := r.indexedTag[tagID]
		if !ok {
			eventTag = r.EventTag()
		}
		eventTag.FailedCount += validation.FailedCount
		eventTag.PassedCount += validation.PassedCount
		if validation.FailedCount > 0 {
			eventTag.AddEvent(endly.NewEvent(validation))
		}
	}

	messageType := messageTypeSuccess
	messageInfo := "OK"
	var message = fmt.Sprintf("Passed %v/%v %v", validation.PassedCount, total, description)
	if validation.FailedCount > 0 {
		r.errorCode = true
		messageType = messageTypeError
		message = fmt.Sprintf("Passed %v/%v %v", validation.PassedCount, total, description)
		messageInfo = "FAILED"
	}
	r.printShortMessage(messageType, message, messageType, messageInfo)
}

func (r *Runner) reportEvent(context *endly.Context, event *endly.Event, filter *Filter) error {
	defer func() {
		eventTag := r.EventTag()
		eventTag.AddEvent(event)
	}()
	r.processEvent(event, filter)
	return nil
}

func (r *Runner) AsListener() endly.EventListener {
	var firstEvent, lastEvent *endly.Event
	return func(event *endly.Event) {
		if firstEvent == nil {
			firstEvent = event
		} else {
			lastEvent = event
			r.report.ElapsedMs = int(lastEvent.Timestamp.UnixNano()-firstEvent.Timestamp.UnixNano()) / int(time.Millisecond)
		}
		r.reportEvent(r.context, event, r.filter)
	}
}

func (r *Runner) processEventTags() {
	for _, eventTag := range r.tags {
		if eventTag.FailedCount > 0 {
			r.report.TotalTagFailed++
			r.errorCode = true
		} else if eventTag.PassedCount > 0 {
			r.report.TotalTagPassed++
		}
	}
}

func (r *Runner) onWorkflowStart() {
	if r.context.Workflow() != nil {
		var workflow = r.context.Workflow().Name
		var workflowLength = len(workflow)
		r.printMessage(r.ColorText(workflow, r.TagColor), workflowLength, messageTypeGeneric, fmt.Sprintf("%v", time.Now()), messageTypeGeneric, "started")
	}
}

func (r *Runner) onWorkflowEnd() {
	r.processEventTags()
	r.reportSummaryEvent()
}

//Run run workflow for the supplied run request and runner options.
func (r *Runner) Run(request *endly.RunRequest, options *RunnerReportingOptions) (err error) {
	r.context = r.manager.NewContext(toolbox.NewContext())
	if request.Name == "" {
		name, URL, err := getWorkflowURL(request.WorkflowURL)
		if err != nil {
			return fmt.Errorf("failed to locate workflow: %v %v", request.WorkflowURL, err)
		}
		request.WorkflowURL = URL
		request.Name = name
	}
	r.context.CLIEnabled = true
	defer func() {
		r.onWorkflowEnd()
		if r.err != nil {
			err = r.err
		}
		r.context.Close()
		if r.errorCode || err != nil {
			OnError(1)
		}
	}()

	if options == nil {
		options = DefaultRunnerReportingOption()
	}

	var service endly.Service
	if service, err = r.context.Service(endly.ServiceID); err != nil {
		return err
	}
	r.context.SetListener(r.AsListener())
	request.Async = true
	response := service.Run(r.context, request)
	r.onWorkflowStart()
	if response.Err != nil {
		err = response.Err
		return err
	}
	_, ok := response.Response.(*endly.RunResponse)
	if !ok {
		return fmt.Errorf("failed to run workflow: %v invalid response type %T,  %v", request.Name, response.Response, response.Error)
	}
	r.context.Wait.Wait()
	return err
}

//New creates a new command line runner
func New() *Runner {
	var activities endly.Activities = make([]*endly.Activity, 0)
	return &Runner{
		manager:            endly.NewManager(),
		Renderer:           NewRenderer(os.Stdout, 120),
		tags:               make([]*EventTag, 0),
		indexedTag:         make(map[string]*EventTag),
		activities:         &activities,
		InputColor:         "blue",
		OutputColor:        "green",
		PathColor:          "brown",
		TagColor:           "brown",
		ErrorColor:         "red",
		InverseTag:         true,
		ServiceActionColor: "gray",

		MessageTypeColor: map[int]string{
			messageTypeTagDescription: "cyan",
			messageTypeError:          "red",
			messageTypeSuccess:        "green",
		},
	}
}
