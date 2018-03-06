package cli

import (
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/runner/http"
	"github.com/viant/endly/workflow"
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
	messageTypeAction         = iota + 10
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
	context       *endly.Context
	filter        map[string]bool
	manager       endly.Manager
	tags          []*EventTag
	indexedTag    map[string]*EventTag
	activities    *endly.Activities
	eventTag      *EventTag
	report        *ReportSummaryEvent
	activity      *endly.Activity
	repeated      *endly.RepeatedMessage
	activityEnded bool

	hasValidationFailures bool
	err                   error

	MessageStyleColor  map[int]string
	InputColor         string
	OutputColor        string
	TagColor           string
	InverseTag         bool
	ServiceActionColor string
	PathColor          string
	SuccessColor       string
	ErrorColor         string
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

func (r *Runner) getRepeated(event *endly.Event) *endly.RepeatedMessage {
	var repeatedType = fmt.Sprintf("%T", event.Value)
	if r.repeated != nil && r.repeated.Type == repeatedType {
		return r.repeated
	}
	r.repeated = &endly.RepeatedMessage{
		Type: repeatedType,
	}
	return r.repeated
}

func (r *Runner) resetRepeated() {
	r.repeated = nil
}

func (r *Runner) processRepeatedReporter(reporter endly.RepeatedReporter, event *endly.Event) {
	repeated := r.getRepeated(event)
	message := reporter.Message(repeated)
	tag := message.Tag
	header := message.Header
	if header != nil {
		if repeated.Count == 0 {
			r.printShortMessage(header.Style, header.Text, tag.Style, tag.Text)
		} else {
			r.overrideShortMessage(header.Style, header.Text, tag.Style, tag.Text)
		}
	}
}

func (r *Runner) processMessageReporter(reporter endly.MessageReporter) {
	for _, message := range reporter.Messages() {
		tag := message.Tag
		header := message.Header
		if header != nil {
			r.printShortMessage(header.Style, header.Text, tag.Style, tag.Text)
		}
		if len(message.Items) > 0 {
			for _, item := range message.Items {
				if color, ok := r.MessageStyleColor[item.Style]; ok {
					r.Printf("%v\n", r.ColorText(item.Text, color))
				} else {
					r.Printf("%v\n", item.Text)
				}
			}
		}
	}
}

func (r *Runner) canReport(event *endly.Event, filter map[string]bool) bool {
	if filter["*"] {
		return true
	}
	var eventType = strings.ToLower(event.Type())
	index := strings.Index(eventType, "_")
	var packageName, eventName, shortName string
	if index != -1 {
		packageName = string(eventType[:index])
		eventName = strings.ToLower(string(eventType[index+1:]))
		shortName = eventName
		if shortName != "request" {
			shortName = strings.Replace(shortName, "request", "", 1)
		}
		shortName = strings.Replace(shortName, "event", "", 1)
		shortName = packageName + "." + shortName
		eventName = packageName + "." + eventName
	}

	for _, candidate := range []string{shortName, eventName, packageName} {
		if value, has := filter[candidate]; has {
			return value
		}
	}
	return false
}

func (r *Runner) processReporter(event *endly.Event, filter map[string]bool) bool {
	if event.Value == nil {
		return false
	}
	messageReporter, isMessageReporter := event.Value.(endly.MessageReporter)
	reptedReporter, isRepeatReporter := event.Value.(endly.RepeatedReporter)

	if !isMessageReporter || isRepeatReporter {
		return false
	}
	if !r.canReport(event, filter) {
		return true
	}
	if isRepeatReporter {
		r.processRepeatedReporter(reptedReporter, event)
		if isMessageReporter {
			r.processMessageReporter(messageReporter)
		}
		return true
	}
	r.resetRepeated()
	if isMessageReporter {
		r.processMessageReporter(messageReporter)
	}
	return true
}

func (r *Runner) processAssertable(event *endly.Event) bool {
	assertable, ok := event.Value.(Assertable)
	if !ok {
		return false
	}
	validations := assertable.Assertion()
	if len(validations) == 0 {
		return true
	}
	r.reportAssertion(event, validations...)
	return true
}

func (r *Runner) processActivityStart(event *endly.Event) bool {
	activity, ok := event.Value.(*endly.Activity)
	if !ok {
		return false
	}
	r.activities.Push(activity)
	r.activity = activity
	if activity.TagDescription != "" {
		r.printShortMessage(messageTypeTagDescription, activity.TagDescription, messageTypeTagDescription, "")
		eventTag := r.EventTag()
		eventTag.Description = activity.TagDescription
	}
	var serviceAction = fmt.Sprintf("%v.%v", activity.Service, activity.Action)
	r.printShortMessage(messageTypeAction, activity.Description, messageTypeAction, serviceAction)
	return true

}

func (r *Runner) processActivityEnd(event *endly.Event) {
	if r.activityEnded {
		r.activities.Pop()
	}
	_, r.activityEnded = event.Value.(*endly.ActivityEndEvent)
}

func (r *Runner) processEvent(event *endly.Event, filter map[string]bool) {
	if event.Value == nil {
		return
	}
	if r.processActivityStart(event) {
		return
	}
	if r.processErrorEvent(event) {
		return
	}
	if r.processAssertable(event) {
		r.resetRepeated()
		return
	}
	if r.processReporter(event, filter) {
		return
	}

	r.resetRepeated()
	r.processActivityEnd(event)
}

func (r *Runner) extractHTTPTrips(eventCandidates []*endly.Event) ([]*http.Request, []*http.Response) {
	var requests = make([]*http.Request, 0)
	var responses = make([]*http.Response, 0)
	for _, event := range eventCandidates {
		if event.Value == nil {
			continue
		}
		switch value := event.Value.(type) {
		case *http.Request:
			requests = append(requests, value)
		case *http.Response:
			responses = append(responses, value)
		}
	}
	return requests, responses
}

func (r *Runner) reportFailureWithMatchSource(tag *EventTag, validation *assertly.Validation, eventCandidates []*endly.Event) {
	var theFirstFailure = validation.Failures[0]
	firstFailurePathIndex := theFirstFailure.Index()
	var requests []*http.Request
	var responses []*http.Response
	var wildcardFilter = WildcardFilter()
	if strings.Contains(theFirstFailure.Path, "Body") || strings.Contains(theFirstFailure.Path, "Code") || strings.Contains(theFirstFailure.Path, "Cookie") || strings.Contains(theFirstFailure.Path, "Header") {
		if theFirstFailure.Index() != -1 {
			requests, responses = r.extractHTTPTrips(eventCandidates)
			if firstFailurePathIndex < len(requests) {
				r.processReporter(endly.NewEvent(requests[firstFailurePathIndex]), wildcardFilter)
			}
			if firstFailurePathIndex < len(responses) {
				r.processReporter(endly.NewEvent(responses[firstFailurePathIndex]), wildcardFilter)
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
	var totalTagValidated = (r.report.TotalTagPassed + r.report.TotalTagFailed)
	var validationInfo = fmt.Sprintf("Passed %v/%v (TagIDs).", r.report.TotalTagPassed, totalTagValidated)
	if totalTagValidated == 0 {
		validationInfo = ""
	}
	r.printMessage(contextMessage, contextMessageLength, endly.MessageStyleGeneric, validationInfo, endly.MessageStyleGeneric, fmt.Sprintf("elapsed: %v ms", r.report.ElapsedMs))
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

func (r *Runner) reportAssertion(event *endly.Event, validations ...*assertly.Validation) {
	if len(validations) == 0 {
		return
	}

	var passedCount, failedCount = 0, 0
	var failedValidation *assertly.Validation

	for i, validation := range validations {
		var tagID = validation.TagID
		_, ok := r.indexedTag[tagID];
		if ! ok {
			r.AddTag(&EventTag{TagID: tagID})
		}
		eventTag := r.indexedTag[tagID];

		if validation.HasFailure() {
			failedCount += validation.FailedCount
			r.hasValidationFailures = true
			failedValidation = validations[i]
			eventTag.FailedCount += validation.FailedCount
		} else if validation.PassedCount > 0 {
			passedCount += validation.PassedCount
			eventTag.PassedCount += validation.PassedCount
		}
		eventTag.AddEvent(endly.NewEvent(validation))
	}
	var total = passedCount + failedCount
	messageType := endly.MessageStyleSuccess
	messageInfo := "OK"
	var message = ""
	if total > 0 {
		message = fmt.Sprintf("Passed %v/%v %v", passedCount, total, validations[0].Description)
		if failedCount > 0 {
			messageType = endly.MessageStyleError
			message = fmt.Sprintf("Passed %v/%v %v", passedCount, total, failedValidation.Description)
			messageInfo = "FAILED"
		}
	}
	r.printShortMessage(messageType, message, messageType, messageInfo)
}

func (r *Runner) reportTagSummary() {
	for _, tag := range r.tags {
		if (tag.FailedCount) > 0 {
			var eventTag = tag.TagID
			r.printMessage(r.ColorText(eventTag, "red"), len(eventTag), messageTypeTagDescription, tag.Description, endly.MessageStyleError, fmt.Sprintf("failed %v/%v", tag.FailedCount, (tag.FailedCount + tag.PassedCount)))
			var minRange = 0
			for i, event := range tag.Events {
				validation := r.getValidation(event)
				if validation == nil {
					continue
				}
				if validation.HasFailure() {
					var beforeValidationEvents = []*endly.Event{}
					if i-minRange > 0 {
						beforeValidationEvents = tag.Events[minRange: i-1]
					}
					r.reportFailureWithMatchSource(tag, validation, beforeValidationEvents)
					minRange = i + 1
				}
			}
		}
	}
}

func (r *Runner) reportEvent(context *endly.Context, event *endly.Event, filter map[string]bool) error {
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
			r.hasValidationFailures = true
		} else if eventTag.PassedCount > 0 {
			r.report.TotalTagPassed++
		}
	}
}

func (r *Runner) onWorkflowStart() {
	if r.context.Workflow() != nil {
		var workflowName = r.context.Workflow().Name
		var workflowLength = len(workflowName)
		r.printMessage(r.ColorText(workflowName, r.TagColor), workflowLength, endly.MessageStyleGeneric, fmt.Sprintf("%v", time.Now()), endly.MessageStyleGeneric, "started")
	}
}

func (r *Runner) onWorkflowEnd() {
	r.processEventTags()
	r.reportSummaryEvent()
}

//Run run workflow for the supplied run request and runner options.
func (r *Runner) Run(request *workflow.RunRequest) (err error) {
	r.context = r.manager.NewContext(toolbox.NewContext())
	r.report = &ReportSummaryEvent{}
	r.context.CLIEnabled = true
	r.filter = request.EventFilter
	if len(r.filter) == 0 {
		r.filter = DefaultFilter()
	}
	if request.Name == "" {
		name, URL, err := getWorkflowURL(request.WorkflowURL)
		if err != nil {
			return fmt.Errorf("failed to locate workflow: %v %v", request.WorkflowURL, err)
		}
		request.WorkflowURL = URL
		request.Name = name
	}
	defer func() {
		r.onWorkflowEnd()
		if r.err != nil {
			err = r.err
		}
		r.context.Close()
		if r.hasValidationFailures || err != nil {
			OnError(1)
		}
	}()

	var service endly.Service
	if service, err = r.context.Service(workflow.ServiceID); err != nil {
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
	_, ok := response.Response.(*workflow.RunResponse)
	if !ok {
		return fmt.Errorf("failed to run workflow: %v invalid response type %T,  %v", request.Name, response.Response, response.Error)
	}
	r.context.Wait.Wait()
	return err
}

func (r *Runner) processErrorEvent(event *endly.Event) bool {
	if errorEvent, ok := event.Value.(*endly.ErrorEvent); ok {
		r.err = fmt.Errorf("%v", errorEvent.Error)
		r.report.Error = true
		r.processReporter(event, WildcardFilter())
		return true
	}
	return false
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
		TagColor:           "brown",
		PathColor:          "brown",
		ServiceActionColor: "gray",
		ErrorColor:         "red",
		InverseTag:         true,
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
