package cli

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/cli/xunit"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/service/system/exec"
	"github.com/viant/endly/service/testing/runner/webdriver"
	"github.com/viant/endly/service/workflow"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

// OnError exit system with os.Exit with supplied code.
var OnError = func(code int) {
	os.Exit(code)
}

const (
	messageTypeAction = iota + 10
	messageTypeTagDescription
)

// ReportSummaryEvent represents event xUnitSummary
type ReportSummaryEvent struct {
	ElapsedMs      int
	TotalTagPassed int
	TotalTagFailed int
	Error          bool
}

// Testing represents command line runner
type Runner struct {
	*Style
	*Renderer
	*Events
	request               *workflow.RunRequest
	xUnitSummary          *xunit.Testsuite
	context               *endly.Context
	filter                map[string]bool
	manager               endly.Manager
	report                *ReportSummaryEvent
	repeated              *msg.Repeated
	activityEnded         bool
	hasValidationFailures bool
	err                   error
	group                 *MessageGroup
}

func (r *Runner) printInput(output string) {
	r.Printf("%v\n", r.ColorText(output, r.InputColor))
}

func (r *Runner) printOutput(output string) {
	r.Printf("%v\n", r.ColorText(output, r.OutputColor))
}

func (r *Runner) printError(output string) {
	r.Printf("%v\n", r.ColorText(output, r.Renderer.ErrorColor))
}

func (r *Runner) printShortMessage(messageType int, message string, messageInfoType int, messageInfo string) {
	r.Printf("%v\n", r.formatShortMessage(messageType, message, messageInfoType, messageInfo))
}

func (r *Runner) overrideShortMessage(messageType int, message string, messageInfoType int, messageInfo string) {
	r.setPendingLine(false)
	defer r.setPendingLine(true)
	r.Printf("\r%v", r.formatShortMessage(messageType, message, messageInfoType, messageInfo))
}

func (r *Runner) printMessage(contextMessage string, messageType int, message string, messageInfoType int, messageInfo string) {
	r.Printf("%v\n", r.formatMessage(contextMessage, messageType, message, messageInfoType, messageInfo))
}

func (r *Runner) formatMessage(contextMessage string, messageType int, message string, messageInfoType int, messageInfo string) string {
	var columns = r.Columns() - 5
	var infoLength = len(messageInfo)
	var messageLength = columns - len(vtclean.Clean(contextMessage, false)) - infoLength

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
	var path = "[/]"
	if r.Len() > 0 {
		path = GetPath(r.Activities, r, fullPath)
	}

	var result = r.formatMessage(path, messageType, message, messageInfoType, messageInfo)
	if strings.Contains(result, message) {
		return result
	}
	return fmt.Sprintf("%v\n%v", result, message)
}

func (r *Runner) getRepeated(event msg.Event) *msg.Repeated {
	var repeatedType = fmt.Sprintf("%T", event.Value)
	if r.repeated != nil && r.repeated.Type == repeatedType {
		return r.repeated
	}
	r.repeated = &msg.Repeated{
		Type: repeatedType,
	}
	return r.repeated
}

func (r *Runner) processRepeated(reporter msg.RepeatedReporter, event msg.Event) {
	repeated := r.getRepeated(event)
	message := reporter.Message(repeated)
	tag := message.Tag
	header := message.Header
	if header != nil {
		if repeated.Count == 0 {
			r.printShortMessage(header.Style, header.Text, tag.Style, tag.Text)
		} else {
			r.overrideShortMessage(header.Style, fmt.Sprintf("%v", header.Text), tag.Style, tag.Text)
		}
		repeated.Count++
	}
}

func (r *Runner) processMessages(reporter msg.Reporter) {
	messages := reporter.Messages()
	for i := range messages {
		message := messages[i]
		tag := message.Tag
		header := message.Header
		if header == nil {
			r.group.Reset()
		}

		if header != nil && !r.group.EnableIfMatched(message) {
			r.printShortMessage(header.Style, header.Text, tag.Style, tag.Text)
		}
		if len(message.Items) == 0 {
			continue
		}

		for _, item := range message.Items {
			suffix := "\n"
			if strings.Count(item.Text, "\n")+strings.Count(item.Text, "\r") > 0 {
				r.pendingNewLine = false
				suffix = ""
			}

			if color, ok := r.MessageStyleColor[item.Style]; ok {
				r.Printf("%v%v", r.ColorText(item.Text, color), suffix)
			} else {
				r.Printf("%v%v", item.Text, suffix)
			}
			r.group.item = item
		}
	}
}

func (r *Runner) canReport(event msg.Event, filter map[string]bool) bool {
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

func (r *Runner) processReporter(event msg.Event, filter map[string]bool) bool {
	if event == nil {
		return true
	}
	var eventValue = event.Value()
	if eventValue == nil {
		return false
	}
	messageReporter, isMessageReporter := eventValue.(msg.Reporter)
	repeatedReporter, isRepeatedReporter := eventValue.(msg.RepeatedReporter)
	if !(isMessageReporter || isRepeatedReporter) {
		return false
	}

	if !r.canReport(event, filter) {
		return true
	}

	if isRepeatedReporter {
		r.processRepeated(repeatedReporter, event)
		if isMessageReporter {
			r.processMessages(messageReporter)
		}
		return true
	}
	if isMessageReporter {
		r.processMessages(messageReporter)
	}
	return true
}

func (r *Runner) processAssertable(event msg.Event) bool {
	asserted, ok := event.Value().(Asserted)
	if !ok {
		return false
	}
	r.repeated = nil
	validations := asserted.Assertion()
	if len(validations) == 0 {
		return true
	}
	r.reportAssertion(event, validations...)
	return true
}

func (r *Runner) processActivityStart(event msg.Event) bool {
	if r.activityEnded {
		r.Pop()
		r.activityEnded = false
	}
	var eventValue = event.Value()
	activity, ok := eventValue.(*model.Activity)
	if !ok {
		return false
	}
	event.SetLoggable(true)
	if r.activity != nil && (activity.Caller != r.activity.Caller) {
		r.activityEnded = false
	}
	r.Push(activity)
	if activity.Logging != nil && !*activity.Logging {
		return true
	}
	if activity.TagIndex != "" {
		r.repeated.Reset()
		r.printShortMessage(messageTypeAction, activity.TagID, messageTypeAction, "tag.id")
		if activity.TagDescription != "" {
			r.printShortMessage(messageTypeTagDescription, activity.TagDescription, messageTypeTagDescription, "use case")
			eventTag := r.EventTag()
			eventTag.Description = activity.TagDescription
		}
	}
	info := activity.Description
	if info == "" {
		info = activity.Comments
	}
	serviceAction := fmt.Sprintf("%v.%v", activity.Service, activity.Action)
	r.printShortMessage(messageTypeAction, info, messageTypeAction, serviceAction)
	return true
}

func (r *Runner) processActivityEnd(event msg.Event) {
	if _, ended := event.Value().(*model.ActivityEndEvent); ended {
		r.activityEnded = ended
		event.SetLoggable(true)
	}
}

func (r *Runner) processEvent(event msg.Event, filter map[string]bool) {
	if event.Value() == nil {
		return
	}
	r.processActivityEnd(event)
	if r.processActivityStart(event) {
		return
	}
	if r.processErrorEvent(event) {
		return
	}

	if r.processAssertable(event) {
		return
	}
	if !event.IsLoggable() {
		return
	}
	if r.processReporter(event, filter) {
		return
	}

}

type runnerLog struct {
	In         msg.Event
	Out        msg.Event
	JSONOutput string
}

func (r *Runner) createRunnerLogIfNeeded(logs map[string][]*runnerLog, key string) {
	if _, has := logs[key]; !has {
		logs[key] = make([]*runnerLog, 0)
	}
}

func (r *Runner) extractRunnerLogs(candidates []msg.Event, offset, maxIndex int) map[string][]*runnerLog {
	var result = make(map[string][]*runnerLog)
	for i := offset; i < len(candidates); i++ {
		var candidate = candidates[i]
		var eventValue = candidate.Value()
		if eventValue == nil {
			continue
		}
		if i <= maxIndex {
			_, ok := eventValue.(msg.RunnerInput)
			if ok {
				key := candidate.Package()
				r.createRunnerLogIfNeeded(result, key)
				result[key] = append(result[key], &runnerLog{
					In: candidate,
				})
				continue
			}
		}

		if _, ok := eventValue.(msg.RunnerOutput); ok {
			key := candidate.Package()
			lastIndex := len(result[key]) - 1
			if lastIndex == -1 {
				continue
			}
			result[key][lastIndex].Out = candidate
			result[key][lastIndex].JSONOutput, _ = toolbox.AsJSONText(candidate.Value)
			continue
		}
	}
	return result
}

func (r *Runner) hasFailureMatch(failure *assertly.Failure, runnerLogs map[string][]*runnerLog) bool {
	var leafKey = failure.LeafKey()
	for _, logs := range runnerLogs {
		for _, log := range logs {
			var matchable = log.JSONOutput
			if matchable == "" {
				var value = log.Out.Value()
				if toolbox.IsMap(value) || toolbox.IsStruct(value) {
					matchable, _ = toolbox.AsJSONText(log.Out.Value())
				} else {
					matchable = toolbox.AsString(log.Out.Value())
				}
			}
			if strings.Contains(matchable, leafKey) {
				return true
			}
		}
	}
	return false
}

func (r *Runner) reportFailureWithMatchSource(tag *Event, event msg.Event, validation *assertly.Validation, eventCandidates []msg.Event, offset, maxIndex int) *runnerLog {
	var runnerLog *runnerLog
	var theFirstFailure = validation.Failures[0]
	firstFailurePathIndex := theFirstFailure.Index()
	var runnerLogs = r.extractRunnerLogs(eventCandidates, offset, maxIndex)

	if r.hasFailureMatch(theFirstFailure, runnerLogs) {
		var wildcardFilter = WildcardFilter()
		var matched = false
		if theFirstFailure.Index() != -1 {
			for _, logs := range runnerLogs {
				if firstFailurePathIndex < len(logs) {
					runnerLog = logs[firstFailurePathIndex]
					if runnerLog.In != nil && runnerLog.Out != nil {
						matched = true
						r.processReporter(runnerLog.In, wildcardFilter)
						r.processReporter(runnerLog.Out, wildcardFilter)
					}
				}
			}
		}
		if !matched {
			for _, logs := range runnerLogs {
				runnerLog = logs[0]
				r.processReporter(runnerLog.In, wildcardFilter)
				r.processReporter(runnerLog.Out, wildcardFilter)
			}
		}
	}
	var counter = 1
	for _, failure := range validation.Failures {
		failurePath := failure.Path
		if failure.Index() != -1 {
			failurePath = fmt.Sprintf("%v:%v", failure.Index(), failure.Path)
		}
		r.printMessage(r.ColorText(failurePath, r.InputColor), msg.MessageStyleError, "", msg.MessageStyleError, "Failed")
		r.Printf("%v\n", r.ColorText(failure.Message, r.Style.ErrorColor))
		//TODO match input for various failure index group: firstFailurePathIndex != failure.Index()
		if r.request.FailureCount > 0 && counter >= r.request.FailureCount {
			break
		}
		counter++
	}
	return runnerLog
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

	contextMessage = fmt.Sprintf("%v%v", contextMessage, r.ColorText(contextMessageStatus, contextMessageColor))
	var totalTagValidated = (r.report.TotalTagPassed + r.report.TotalTagFailed)
	var validationInfo = fmt.Sprintf("Passed %v/%v.", r.report.TotalTagPassed, totalTagValidated)
	if totalTagValidated == 0 {
		validationInfo = ""
	}
	r.printMessage(contextMessage, msg.MessageStyleGeneric, validationInfo, msg.MessageStyleGeneric, fmt.Sprintf("elapsed: %v ms", r.report.ElapsedMs))
}

func (r *Runner) getValidation(event msg.Event) *assertly.Validation {
	var eventValue = event.Value()
	validation, ok := eventValue.(*assertly.Validation)
	if !ok {
		return nil
	}
	return validation
}

func (r *Runner) reportAssertion(event msg.Event, validations ...*assertly.Validation) {
	if len(validations) == 0 {
		return
	}
	for _, validation := range validations {

		if validation.PassedCount+validation.FailedCount == 0 {
			continue
		}
		eventTag := r.TemplateEvent(r.context, validation.TagID)
		if validation.HasFailure() {
			r.hasValidationFailures = true
			eventTag.FailedCount += validation.FailedCount
		} else if validation.PassedCount > 0 {
			eventTag.PassedCount += validation.PassedCount
		}
		eventTag.AddEvent(msg.NewEventWithInit(validation, event.Init()))
		messageInfo := "OK"
		messageType := msg.MessageStyleSuccess
		if validation.FailedCount > 0 {
			messageInfo = "FAILED"
			messageType = msg.MessageStyleError
		}
		var aMap = data.NewMap()
		aMap.Put("tagIndex", eventTag.Index)
		aMap.Put("tagID", eventTag.TagID)
		message := fmt.Sprintf("Passed %v/%v %v", validation.PassedCount, validation.PassedCount+validation.FailedCount, aMap.ExpandAsText(validation.Description))
		r.printShortMessage(messageType, message, messageType, messageInfo)
	}

}

func (r *Runner) reportTagSummary() {
	var useCaseCount = 0
	for _, tag := range r.tags {

		if (tag.FailedCount) > 0 || tag.PassedCount > 0 {
			useCaseCount++
		} else {
			continue
		}
		useCase := xunit.NewTestCase()
		r.xUnitSummary.TestCase = append(r.xUnitSummary.TestCase, useCase)
		useCase.Label = tag.TagID
		description := strings.Split(tag.Description, "\n")[0]
		if description == "" {
			description = tag.TagID
		}
		useCase.Name = description
		useCase.Tests = fmt.Sprintf("%d", tag.PassedCount+tag.FailedCount)
		useCase.Failures = fmt.Sprintf("%d", tag.FailedCount)

		var failureLog *runnerLog
		var validation *assertly.Validation
		if (tag.FailedCount) > 0 {
			var eventTag = tag.TagID
			r.printMessage(r.ColorText(eventTag, "red"), messageTypeTagDescription, tag.Description, msg.MessageStyleError, fmt.Sprintf("failed %v/%v", tag.FailedCount, (tag.FailedCount+tag.PassedCount)))
			var offset = 0
			for i, event := range tag.Events {
				validation = r.getValidation(event)
				if validation == nil {
					continue
				}
				if validation.HasFailure() {
					var maxIndex = i - 1
					candidates := tag.Events
					if tag.subEvent != nil {
						candidates = tag.subEvent.Events
					}
					runnerLog := r.reportFailureWithMatchSource(tag, event, validation, candidates, offset, maxIndex)
					if runnerLog != nil {
						failureLog = runnerLog
					}
					offset = i + 1
					nodes := xunit.NewNodes()
					useCase.Nodes = nodes
					nodes.Expected = "/"
					nodes.Result = "/"
					for _, failure := range validation.Failures {
						node := xunit.NewNodes()
						node.Expected = fmt.Sprintf("%s: %s", failure.Path, failure.Expected)
						node.Result = fmt.Sprintf("%s: %s", failure.Path, failure.Actual)
						node.Error = &xunit.Error{
							Type:  failure.Reason,
							Value: failure.Message,
						}
						nodes.Nodes = append(nodes.Nodes, node)
					}
				}
			}
		}
		if validation != nil {
			useCase.FailuresDetail = validation.Report()
		}
		if len(tag.Events) > 0 {
			useCase.Time = tag.Events[0].Timestamp().String()
		}
		if failureLog != nil {
			useCase.Sysout = failureLog.JSONOutput
		}
	}
	r.xUnitSummary.TestCases = fmt.Sprintf("%d", useCaseCount)
	r.xUnitSummary.Reports = fmt.Sprintf("%d", useCaseCount)
	r.xUnitSummary.Tests = fmt.Sprintf("%d", r.report.TotalTagPassed+r.report.TotalTagFailed)
	r.xUnitSummary.Failures = fmt.Sprintf("%d", +r.report.TotalTagFailed)
	if r.request != nil && len(r.request.Params) > 0 {
		if val, ok := r.request.Params["app"]; ok {
			r.xUnitSummary.Name = toolbox.AsString(val)
		} else if r.request.Source != nil {
			workflowPath := r.request.Source.Path()
			if strings.HasSuffix(workflowPath, "/") {
				workflowPath = string(workflowPath[:len(workflowPath)-1])
			}
			_, r.xUnitSummary.Name = path.Split(workflowPath)
		}
	}
}

func (r *Runner) reportEvent(context *endly.Context, event msg.Event, filter map[string]bool) error {
	eventTag := r.EventTag()
	r.processEvent(event, filter)
	eventTag.AddEvent(event)
	return nil
}

func (r *Runner) AsListener() msg.Listener {
	var firstEvent, lastEvent msg.Event
	return func(event msg.Event) {
		if firstEvent == nil {
			firstEvent = event
		} else {
			lastEvent = event
			r.report.ElapsedMs = int(lastEvent.Timestamp().UnixNano()-firstEvent.Timestamp().UnixNano()) / int(time.Millisecond)
		}
		_ = r.reportEvent(r.context, event, r.filter)
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

func (r *Runner) onCallerStart() {
	process := workflow.Last(r.context)
	if process != nil {
		var CallerName = process.Owner
		if CallerName == "" {
			CallerName = "noname"
		}
		r.printMessage(r.ColorText(CallerName, r.TagColor), msg.MessageStyleGeneric, fmt.Sprintf("%v", time.Now()), msg.MessageStyleGeneric, "started")
	}
}

func (r *Runner) onCallerEnd() {
	r.processEventTags()
	r.reportSummaryEvent()
	r.printSummary()
}

func (r *Runner) printSummary() {

	if r.request == nil || r.request.SummaryFormat == "" {
		return
	}
	var err error
	buf := new(bytes.Buffer)
	switch r.request.SummaryFormat {
	case "xml":
		encoder := xml.NewEncoder(buf)
		encoder.Indent("  ", "    ")
		err = encoder.EncodeElement(r.xUnitSummary, xml.StartElement{Name: xml.Name{Local: "test-suite"}})
	case "yaml":
		err = yaml.NewEncoder(buf).Encode(r.xUnitSummary)
	case "json":
		encoder := json.NewEncoder(buf)
		encoder.SetIndent("  ", "    ")
		err = encoder.Encode(r.xUnitSummary)
	}
	if err == nil {
		err = ioutil.WriteFile(fmt.Sprintf("summary.%v", r.request.SummaryFormat), buf.Bytes(), 0644)
	}
	if err != nil {
		log.Fatal(err)
	}

}

// Run run Caller for the supplied run request and runner options.
func (r *Runner) Run(request *workflow.RunRequest) (err error) {
	r.request = request
	r.context = r.manager.NewContext(toolbox.NewContext())
	//init shared session
	exec.TerminalSessions(r.context)
	webdriver.Sessions(r.context)

	r.report = &ReportSummaryEvent{}
	r.context.CLIEnabled = true
	r.filter = request.EventFilter
	if len(r.filter) == 0 {
		r.filter = DefaultFilter()
	}
	defer func() {
		r.onCallerEnd()
		if r.err != nil {
			err = r.err
		}
		if !request.Interactive {
			r.context.Close()
		}
		if r.hasValidationFailures || err != nil {
			OnError(1)
		}
	}()
	r.context.SetListener(r.AsListener())
	request.Async = true
	var response = &workflow.RunResponse{}
	err = endly.Run(r.context, request, response)
	r.onCallerStart()
	if err != nil {
		r.context.Publish(msg.NewErrorEvent(err.Error()))
		return err
	}
	r.context.Wait.Wait()
	return err
}

func (r *Runner) processErrorEvent(event msg.Event) bool {

	if _, ok := event.Value().(*msg.ResetError); ok {
		r.report.Error = false
		r.err = nil
		r.xUnitSummary.Errors = ""
		r.xUnitSummary.ErrorsDetail = ""
		return true
	}
	if errorEvent, ok := event.Value().(*msg.ErrorEvent); ok {
		event.SetLoggable(true)
		r.err = fmt.Errorf("%v", errorEvent.Error)
		r.xUnitSummary.Errors = "1"
		r.xUnitSummary.ErrorsDetail = errorEvent.Error
		r.report.Error = true
		r.processReporter(event, WildcardFilter())
		return true
	}
	return false
}

// New creates a new command line runner
func New() *Runner {
	return &Runner{
		manager:      endly.New(),
		Events:       NewEventTags(),
		Renderer:     NewRenderer(os.Stdout, 120),
		group:        &MessageGroup{},
		xUnitSummary: xunit.NewTestsuite(),
		Style:        NewStyle(),
	}
}
