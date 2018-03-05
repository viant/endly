package cli

import (
	"errors"
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/dsunit"
	"github.com/viant/endly"
	"github.com/viant/endly/deployment/build"
	"github.com/viant/endly/deployment/deploy"
	"github.com/viant/endly/deployment/vc"
	"github.com/viant/endly/runner/http"
	"github.com/viant/endly/runner/selenium"
	"github.com/viant/endly/testing/log"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox"
	"os"
	"reflect"
	"strings"
	"time"
)

//OnError exit system with os.Exit with supplied code.
var OnError = func(code int) {
	os.Exit(code)
}

const (
	messageTypeAction = iota + 10
	messageTypeTagDescription
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

	MessageStyleColor map[int]string
	SuccessColor      string
	ErrorColor        string

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

	if messageColor, ok := r.MessageStyleColor[messageType]; ok {
		message = r.ColorText(message, messageColor)
	}

	messageInfo = r.ColorText(messageInfo, "bold")
	if messageInfoColor, ok := r.MessageStyleColor[messageInfoType]; ok {
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

func (r *Runner) processReporter(event *endly.Event, filter *Filter) bool {
	if event.Value == nil {
		return false
	}
	filteredReporter, isFilterReporter := event.Value.(endly.FilteredReporter)
	messageReporter, isMessageReporter := event.Value.(endly.MessageReporter)

	if !(isFilterReporter || isMessageReporter) {
		return false
	}

	if isFilterReporter {
		if !filteredReporter.CanReport(filter.Report) {
			return true
		}
	}
	if isMessageReporter {
		for _, message := range messageReporter.Messages() {
			tag := message.Tag
			header := message.Header
			if header != nil {
				r.printShortMessage(header.Style, header.Text, tag.Style, tag.Text)
			}
			if len(message.Messages) > 0 {
				for _, subMessage := range message.Messages {
					if color, ok := r.MessageStyleColor[subMessage.Style]; ok {
						r.Printf("%v\n", r.ColorText(subMessage.Text, color))
					} else {
						r.Printf("%v\n", subMessage.Text)
					}
				}
			}
		}
	}
	return false
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
			r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("Datastore: %v, %v:%v", actual.Datastore, actual.Config.DriverName, descriptor), endly.MessageStyleGeneric, "register")
		}
	case *dsunit.MappingRequest:
		if filter.DataMapping {
			for _, mapping := range actual.Mappings {
				var _, name = toolbox.URLSplit(mapping.Name)
				r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v: %v", name, mapping.Name), endly.MessageStyleGeneric, "mapping")
			}
		}
	case *dsunit.SequenceRequest:
		if filter.Sequence {
			for k, v := range actual.Tables {
				r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v: %v", k, v), endly.MessageStyleGeneric, "sequence")
			}
		}
	case *dsunit.PrepareRequest:
		if filter.PopulateDatastore {
			actual.Load()
			for _, dataset := range actual.Datasets {
				r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("(%v) %v: %v", actual.Datastore, dataset.Table, len(dataset.Records)), endly.MessageStyleGeneric, "populate")
			}
		}
	case *dsunit.RunScriptRequest:
		if filter.SQLScript {
			for _, script := range actual.Scripts {
				r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("(%v) %v", actual.Datastore, script.URL), endly.MessageStyleGeneric, "sql")
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

func (r *Runner) processEndlyEvents(event *endly.Event, filter *Filter) bool {
	switch actual := event.Value.(type) {
	case *endly.ErrorEvent:
		r.report.Error = true
		r.printShortMessage(endly.MessageStyleError, fmt.Sprintf("%v", actual.Error), endly.MessageStyleError, "error")
		r.Println(r.ColorText(fmt.Sprintf("ERROR: %v\n", actual.Error), "red"))
		r.err = errors.New(actual.Error)
		return true
	case *endly.SleepEvent:
		if r.SleepCount > 0 {
			r.overrideShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v ms x %v,  slept so far: %v", actual.SleepTimeMs, r.SleepCount, r.SleepTime), endly.MessageStyleGeneric, "Sleep")
		} else {
			r.SleepTime = 0
			r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v ms", actual.SleepTimeMs), endly.MessageStyleGeneric, "Sleep")
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
	if r.processReporter(event, filter) {
		return
	}
	if r.processEndlyEvents(event, filter) {
		return
	}
	if r.processWorkflowEvent(event, filter) {
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
			r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("app: %v, forced: %v", value.AppName, value.Force), endly.MessageStyleGeneric, "deploy")
		}

	case *vc.CheckoutRequest:
		if filter.Checkout {
			r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v %v", value.Origin.URL, value.Target.URL), endly.MessageStyleGeneric, "checkout")
		}

	case *build.Request:
		if filter.Build {
			r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v %v", value.BuildSpec.Name, value.Target.URL), endly.MessageStyleGeneric, "build")
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
	messageType := endly.MessageStyleSuccess
	messageInfo := "OK"
	var message = ""
	if total > 0 {
		message = fmt.Sprintf("Passed %v/%v %v", passedCount, total, response.Description)
		if failedCount > 0 {
			messageType = endly.MessageStyleError
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
		r.printMessage(failurePath, len(failurePath), endly.MessageStyleError, failure.Message, endly.MessageStyleError, "Failed")
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
	r.printMessage(contextMessage, contextMessageLength, endly.MessageStyleGeneric, fmt.Sprintf("Passed %v/%v", r.report.TotalTagPassed, (r.report.TotalTagPassed+r.report.TotalTagFailed)), endly.MessageStyleGeneric, fmt.Sprintf("elapsed: %v ms", r.report.ElapsedMs))
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
			r.printMessage(r.ColorText(eventTag, "red"), len(eventTag), messageTypeTagDescription, tag.Description, endly.MessageStyleError, fmt.Sprintf("failed %v/%v", tag.FailedCount, (tag.FailedCount+tag.PassedCount)))

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
			r.printShortMessage(endly.MessageStyleError, errMessage, endly.MessageStyleGeneric, "Selenium")
		}
	}
}

func asJSONText(source interface{}) string {
	text, _ := toolbox.AsJSONText(source)
	return text
}

func (r *Runner) reportHTTPResponse(response *http.Response) {
	r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("StatusCode: %v", response.Code), endly.MessageStyleGeneric, "HttpResponse")
	if len(response.Header) > 0 {
		r.printShortMessage(endly.MessageStyleGeneric, "Headers", endly.MessageStyleGeneric, "HttpResponse")

		r.printOutput(asJSONText(response.Header))
	}
	r.printShortMessage(endly.MessageStyleGeneric, "Body", endly.MessageStyleGeneric, "HttpResponse")
	r.printOutput(response.Body)
}

func (r *Runner) reportHTTPRequest(request *http.Request) {
	r.printShortMessage(endly.MessageStyleGeneric, fmt.Sprintf("%v %v", request.Method, request.URL), endly.MessageStyleGeneric, "HttpRequest")
	r.printInput(asJSONText(request.URL))
	if len(request.Header) > 0 {
		r.printShortMessage(endly.MessageStyleGeneric, "Headers", endly.MessageStyleGeneric, "HttpRequest")
		r.printInput(asJSONText(request.Header))
	}
	if len(request.Cookies) > 0 {
		r.printShortMessage(endly.MessageStyleGeneric, "Cookies", endly.MessageStyleGeneric, "HttpRequest")
		r.printInput(asJSONText(request.Cookies))
	}
	r.printShortMessage(endly.MessageStyleGeneric, "Body", endly.MessageStyleGeneric, "HttpRequest")
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

	messageType := endly.MessageStyleSuccess
	messageInfo := "OK"
	var message = fmt.Sprintf("Passed %v/%v %v", validation.PassedCount, total, description)
	if validation.FailedCount > 0 {
		r.errorCode = true
		messageType = endly.MessageStyleError
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

func (r *Runner) updateFilterReporSettings() {
	if len(r.filter.Report) == 0 {
		r.filter.Report = make(map[string]bool)
	}
	//temporary transition during refactoring
	if r.filter.Stdin {
		r.filter.Report["stdin"] = true
	}
	if r.filter.Stdout {
		r.filter.Report["stdout"] = true
	}
	if r.filter.Transfer {
		r.filter.Report["storage"] = true
	}
}

func (r *Runner) AsListener() endly.EventListener {
	var firstEvent, lastEvent *endly.Event
	if r.filter == nil {
		r.filter = DefaultRunnerReportingOption().Filter
	}
	r.updateFilterReporSettings()

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
		r.printMessage(r.ColorText(workflow, r.TagColor), workflowLength, endly.MessageStyleGeneric, fmt.Sprintf("%v", time.Now()), endly.MessageStyleGeneric, "started")
	}
}

func (r *Runner) onWorkflowEnd() {
	r.processEventTags()
	r.reportSummaryEvent()
}

//Run run workflow for the supplied run request and runner options.
func (r *Runner) Run(request *endly.RunRequest, options *RunnerReportingOptions) (err error) {
	r.context = r.manager.NewContext(toolbox.NewContext())
	r.report = &ReportSummaryEvent{}
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
		MessageStyleColor: map[int]string{
			messageTypeTagDescription: "cyan",
			endly.MessageStyleError:   "red",
			endly.MessageStyleSuccess: "green",
			endly.MessageStyleGeneric: "black",
			endly.MessageStyleInput:   "blue",
			endly.MessageStyleOutput:  "green",
		},
	}
}
