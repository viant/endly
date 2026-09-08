package webdriver

import (
	"context"
	"errors"
	"fmt"
	"github.com/viant/endly/model/msg"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"github.com/tebeka/selenium/firefox"
	selog "github.com/tebeka/selenium/log"
	"github.com/viant/afs"
	"github.com/viant/afs/url"
	"github.com/viant/endly"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model/criteria"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/deployment/deploy"
	"github.com/viant/endly/service/deployment/sdk"
	"github.com/viant/endly/service/system/exec"
	"github.com/viant/endly/service/system/process"
	"github.com/viant/endly/service/testing/runner/webdriver/extension/html/table"
	"github.com/viant/endly/service/testing/validator"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

const (
	//ServiceID represents a ServiceID
	ServiceID = "webdriver"

	//SeleniumServer represents name of selenium server
	SeleniumServer = "selenium-server-standalone"
	//GeckoDriver represents name of gecko driver
	GeckoDriver    = "geckodriver"
	ChromeDriver   = "chromedriver"
	ChromeBrowser  = "chrome"
	FirefoxBrowser = "firefox"
	Selenium       = "webdriver"
	runnerCaller   = "runnerCaller"

	defaultFindElementTimeout = 10 * time.Second
)

type service struct {
	*endly.AbstractService
	fs afs.Service
}

// expandContextText resolves workflow values that are intentionally composed
// from other values, such as webdriverURL -> webdriverRoot -> repoPath.
func expandContextText(context *endly.Context, value string) string {
	for range 8 {
		expanded := context.Expand(value)
		if expanded == value {
			return value
		}
		value = expanded
	}
	return value
}

func (s *service) addResultIfPresent(callResult []interface{}, result data.Map, resultPath ...string) bool {
	var responseData interface{}
	var has = false
	for _, element := range callResult {
		if element == nil {
			continue
		}
		switch actual := element.(type) {
		case string:
			responseData = actual
		case []byte:
			responseData = string(actual)
		case []interface{}:
			responseData = actual
		case []map[string]interface{}:
			responseData = actual
		case map[string]interface{}:
			responseData = actual
		case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			responseData = actual
		default:
			fmt.Printf("unsupported type: %T\n", actual)
			continue
		}
		has = true
		break
	}
	if !has {
		return false
	}
	var key = strings.Join(resultPath, ".")
	result.SetValue(key, responseData)

	return true
}

func (s *service) getResultPath(key string, call *MethodCall, kind PathKind) []string {
	if kind == PathKindSimple {
		return []string{key}
	}
	var method = call.Method
	if len(call.Parameters) == 1 && toolbox.IsString(call.Parameters[0]) {
		method = strings.Replace(method, "Get", "", 1) + "." + toolbox.AsString(call.Parameters[0])
	}
	return []string{key, method}
}

func (s *service) run(context *endly.Context, request *RunRequest) (response *RunResponse, err error) {
	response = &RunResponse{
		Data:         make(map[string]interface{}),
		LookupErrors: make([]string, 0),
		Navigations:  make([]*NavigationReport, 0),
		Failures:     make([]*FailureArtifact, 0),
		Assertions:   make([]*AssertionResult, 0),
	}
	navigation := navigationWithDefaults(request.Navigation)
	navigation.ScrollSelector = context.Expand(navigation.ScrollSelector)
	session, hasSession := lookupSession(context, request.SessionID)
	autoStart := request.AutoStart == nil || *request.AutoStart
	if !hasSession && autoStart && request.SessionID == "localhost:4444" && request.RemoteSelenium == "" && request.DebuggerAddress == "" {
		startRequest := &StartRequest{PageLoadStrategy: request.PageLoadStrategy}
		if initErr := startRequest.Init(); initErr != nil {
			return response, initErr
		}
		if _, startErr := s.start(context, startRequest); startErr != nil {
			return response, fmt.Errorf("auto-start webdriver: %w", startErr)
		}
		session, hasSession = lookupSession(context, request.SessionID)
	}

	if !hasSession || session.driver == nil {
		openResponse, err := s.openSession(context, &OpenSessionRequest{
			Remote:           request.RemoteSelenium,
			Browser:          request.Browser,
			SessionID:        request.SessionID,
			DebuggerAddress:  request.DebuggerAddress,
			DirectCDP:        request.DirectCDP,
			PageLoadStrategy: request.PageLoadStrategy,
			BlockedURLs:      request.BlockedURLs,
		})
		if err != nil {
			return response, fmt.Errorf("open webdriver session %s at %s: %w", request.SessionID, request.RemoteSelenium, err)
		}
		request.SessionID = openResponse.SessionID
		session, _ = lookupSession(context, request.SessionID)
	}
	response.SessionID = request.SessionID
	response.Backend = session.Backend
	session.mu.Lock()
	defer session.mu.Unlock()
	var currentAction *Action
	var currentCall *MethodCall
	defer func() {
		failureErr := err
		if failureErr == nil && response.Assert != nil && response.Assert.Validation != nil && response.Assert.Validation.HasFailure() {
			failureErr = errors.New(response.Assert.Validation.Report())
		}
		if failureErr == nil || request.FailureArtifacts == nil {
			return
		}
		if artifact := s.captureFailure(context, session, request.FailureArtifacts, currentAction, currentCall, failureErr, response.Navigations); artifact != nil {
			response.Failures = append(response.Failures, artifact)
		}
	}()
	if len(request.Actions) == 0 {
		return response, nil
	}
	responseBaseline := 0
	if hasResponseWait(request.Actions) {
		if err = s.ensureNetworkCaptureLocked(session); err != nil {
			return response, err
		}
		responseBaseline = session.Capture.Summary().RequestsCompleted
	}
	var state = context.State()

	actionDelay := time.Duration(request.ActionDelaysMs) * time.Millisecond
	for _, action := range request.Actions {
		currentAction = action
		actionTimeoutMs := request.ActionTimeoutMs
		for _, call := range action.Calls {
			if call.WaitTimeMs > actionTimeoutMs {
				actionTimeoutMs = call.WaitTimeMs
			}
		}
		actionDeadline := time.Now().Add(time.Duration(actionTimeoutMs) * time.Millisecond)
		for _, call := range action.Calls {
			runtimeCall := *call
			currentCall = &runtimeCall
			callStarted := time.Now()
			if len(call.Parameters) > 0 {
				runtimeCall.Parameters = make([]interface{}, len(call.Parameters))
				for i, item := range call.Parameters {
					runtimeCall.Parameters[i] = state.Expand(item)
				}
			}
			if action.Selector != nil {
				remainingMs := max(1, int(time.Until(actionDeadline)/time.Millisecond))
				if runtimeCall.WaitTimeMs <= 0 || runtimeCall.WaitTimeMs > remainingMs {
					runtimeCall.WaitTimeMs = remainingMs
				}
				if runtimeCall.PollIntervalMs <= 0 {
					runtimeCall.PollIntervalMs = request.PollIntervalMs
				}
			}
			if action.Selector == nil {
				if runtimeCall.Method == "StopLoading" {
					stopResponse, stopErr := s.stopLoading(context, &StopLoadingRequest{SessionID: request.SessionID})
					if stopErr != nil {
						return response, stopErr
					}
					response.Data["stopLoading"] = stopResponse
					continue
				}
				if runtimeCall.Method == "WaitForResponse" {
					if len(runtimeCall.Parameters) != 3 {
						return response, fmt.Errorf("WaitForResponse requires URL pattern, status, and timeout")
					}
					pattern := toolbox.AsString(runtimeCall.Parameters[0])
					status := toolbox.AsInt(runtimeCall.Parameters[1])
					timeoutMs := toolbox.AsInt(runtimeCall.Parameters[2])
					transaction, waitErr := s.waitForResponse(context, session, pattern, status, timeoutMs, responseBaseline)
					if waitErr != nil {
						return response, waitErr
					}
					responseBaseline = transaction.Sequence
					key := action.Key
					if key == "" {
						key = "response"
					}
					responseData := data.Map(response.Data)
					responseData.SetValue(key, transaction)
					continue
				}
				if session != nil && isGetMethod(runtimeCall.Method) && len(runtimeCall.Parameters) == 1 && toolbox.IsString(runtimeCall.Parameters[0]) {
					URL := toolbox.AsString(runtimeCall.Parameters[0])
					report, err := s.getWithGuard(context, session, URL, navigation)
					if report != nil {
						response.Navigations = append(response.Navigations, report)
					}
					if err != nil {
						return response, err
					}
					if session.Capture != nil {
						session.Capture.Drain(session)
					}
					continue
				}
				callResponse, err := s.callWebDriver(context, &WebDriverCallRequest{
					Key:           action.Key,
					SessionID:     request.SessionID,
					Call:          &runtimeCall,
					PathKind:      action.PathKind,
					sessionLocked: true,
				})
				var driverResult []interface{}
				if callResponse != nil {
					driverResult = callResponse.Result
				}
				appendAssertionResult(response, action, &runtimeCall, driverResult, err, callStarted)
				if err != nil {
					return response, err
				}
				util.MergeMap(response.Data, callResponse.Data)
				if session != nil && session.Capture != nil {
					session.Capture.Drain(session)
				}
				continue
			}
			callResponse, err := s.callWebElement(context, &WebElementCallRequest{
				SessionID:            request.SessionID,
				Selector:             action.Selector,
				Call:                 &runtimeCall,
				PathKind:             action.PathKind,
				AllowJavaScriptClick: request.AllowJavaScriptClick,
				Strict:               selectorStrictness(action.Strict, request.StrictSelectors),
				sessionLocked:        true,
			})
			var callResult []interface{}
			if callResponse != nil {
				callResult = callResponse.Result
			}
			appendAssertionResult(response, action, &runtimeCall, callResult, err, callStarted)
			if err != nil {
				return response, err
			}
			if callResponse.LookupError != "" {
				response.LookupErrors = append(response.LookupErrors, callResponse.LookupError)
			}
			util.MergeMap(response.Data, callResponse.Data)
			if session != nil && session.Capture != nil {
				session.Capture.Drain(session)
			}
			if actionDelay > 0 {
				if err := waitForContext(context.Background(), actionDelay); err != nil {
					return response, err
				}
			}
		}
	}

	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, response.Data, "webdriver", "assert webdriver response")
	}
	if err == nil && len(response.LookupErrors) > 0 {
		err = fmt.Errorf("lookup errors: %v", strings.Join(response.LookupErrors, ","))

	}
	return response, err
}

// Data returns table" data in the specified format, format uses the following values: json, csv, objects, tabular, optionally you can specify header columns after ':'
func (s *service) Data(webElement selenium.WebElement, format string) (interface{}, error) {
	//TODO add support for form
	var header = ""
	if index := strings.Index(format, ":"); index != -1 {
		header = format[index+1:]
		format = format[:index]
	}
	if format == "" {
		format = "objects"
	}
	var headers []string
	if header != "" {
		headers = strings.Split(header, ",")
	}
	tagName, err := webElement.TagName()
	if err != nil {
		return nil, err
	}
	if tagName != "table" {
		return nil, fmt.Errorf("element is not a table")
	}
	tableHtml, err := webElement.GetAttribute("outerHTML")
	if err != nil {
		return nil, fmt.Errorf("failed to get table html: %v", err)
	}
	exporter, err := table.NewExporter(tableHtml)
	if err != nil {
		return nil, fmt.Errorf("failed to create table exporter: %v", err)
	}
	ret, err := exporter.Export(headers, format)
	if err != nil {
		return nil, fmt.Errorf("failed to export table: %v", err)
	}
	return ret, nil
}

func (s *service) callMethod(owner interface{}, methodName string, response *ServiceCallResponse, parameters []interface{}) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("webdriver method %s is not supported by %T: %v", methodName, owner, recovered)
		}
	}()
	switch methodName {
	case "Data", "Tabs", "SwitchTab", "NewTab", "CloseTab", "Check", "Uncheck", "SelectOption", "SwitchFrameBySelector", "MainFrame", "ElementCount":
		parameters = append([]interface{}{owner}, parameters...)
		owner = s
	}
	method, err := toolbox.GetFunction(owner, methodName)
	if err != nil {
		return err
	}
	parameters, err = toolbox.AsCompatibleFunctionParameters(method, parameters)
	if err != nil {
		return err
	}
	response.Result = toolbox.CallFunction(method, parameters...)
	value := response.Result[len(response.Result)-1]
	if value != nil {
		if err, ok := value.(error); ok {
			return err
		}
	}
	return nil
}

func (s *service) callWebDriver(context *endly.Context, request *WebDriverCallRequest) (*ServiceCallResponse, error) {
	session, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	if !request.sessionLocked {
		session.mu.Lock()
		defer session.mu.Unlock()
	}
	response := &ServiceCallResponse{
		Data: make(map[string]interface{}),
	}
	var key = request.Key
	if key == "" {
		key = request.Call.Method
	}
	return response, s.call(context, session.driver, request.Call, response, key)
}

// missingStringValue normalizes ChromeDriver's JSON null for an optional DOM
// string. Older Selenium clients report it as an error, which prevents an
// Endly repeat action from polling until a dynamically-added attribute exists.
func missingStringValue(call *MethodCall, response *ServiceCallResponse, err error) bool {
	if err == nil || err.Error() != "nil return value" {
		return false
	}
	switch call.Method {
	case "Text", "GetAttribute", "GetProperty":
		response.Result = []interface{}{""}
		return true
	}
	return false
}

func (s *service) call(context *endly.Context, caller interface{}, call *MethodCall, response *ServiceCallResponse, elementPath ...string) (err error) {
	if call.WaitTimeMs == 0 {
		if err = s.callMethod(caller, call.Method, response, call.Parameters); err != nil && !missingStringValue(call, response, err) {
			return fmt.Errorf("webdriver call %s: %w", call.Method, err)
		}
		s.addResultIfPresent(response.Result, response.Data, elementPath...)
		if call.Expectation != nil {
			matched, actual, matchErr := matchesExpectation(response.Result, call.Expectation)
			if matchErr != nil {
				return matchErr
			}
			if !matched {
				return fmt.Errorf("webdriver expectation failed: expected %s %v, actual %v", call.Expectation.Matcher, call.Expectation.Value, actual)
			}
		}
		if call.ThinkTimeMs > 0 {
			if err := waitForContext(context.Background(), time.Millisecond*time.Duration(call.ThinkTimeMs)); err != nil {
				return err
			}
		}
		return nil
	}

	deadline := time.Now().Add(time.Duration(call.WaitTimeMs) * time.Millisecond)
	pollInterval := time.Duration(call.PollIntervalMs) * time.Millisecond
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}
	for {
		if contextErr := context.Background().Err(); contextErr != nil {
			return contextErr
		}
		response.Result = nil
		err = s.callMethod(caller, call.Method, response, call.Parameters)
		if err != nil && !missingStringValue(call, response, err) {
			return fmt.Errorf("webdriver call %s: %w", call.Method, err)
		}
		s.addResultIfPresent(response.Result, response.Data, elementPath...)
		matchedExpectation := true
		var actual interface{}
		if call.Expectation != nil {
			var matchErr error
			matchedExpectation, actual, matchErr = matchesExpectation(response.Result, call.Expectation)
			if matchErr != nil {
				return matchErr
			}
		}
		if call.Exit == "" && matchedExpectation {
			return nil
		}
		if call.Exit != "" {
			evalData := data.Map{}
			util.MergeMap(evalData, response.Data)
			matched, evalErr := criteria.Evaluate(context, evalData, call.Exit, &call.criteria, runnerCaller, true)
			if evalErr != nil {
				return evalErr
			}
			if matched && matchedExpectation {
				return nil
			}
		}
		if !time.Now().Before(deadline) {
			if call.IgnoreTimeout {
				return nil
			}
			if call.Expectation != nil {
				return fmt.Errorf("webdriver expectation timed out after %dms: expected %s %v, actual %v", call.WaitTimeMs, call.Expectation.Matcher, call.Expectation.Value, actual)
			}
			return fmt.Errorf("webdriver call %s timed out after %dms waiting for %q", call.Method, call.WaitTimeMs, call.Exit)
		}
		remaining := time.Until(deadline)
		if pollInterval > remaining {
			pollInterval = remaining
		}
		if err := waitForContext(context.Background(), pollInterval); err != nil {
			return err
		}
	}
}

func matchesExpectation(results []interface{}, expectation *CallExpectation) (bool, interface{}, error) {
	actual := firstCallResult(results)
	if expectation == nil {
		return true, actual, nil
	}
	switch expectation.Matcher {
	case "equal":
		if actualText, ok := actual.(string); ok {
			expectedText := toolbox.AsString(expectation.Value)
			if len(expectedText) >= 2 && strings.HasPrefix(expectedText, "/") && strings.HasSuffix(expectedText, "/") {
				pattern, err := regexp.Compile(expectedText[1 : len(expectedText)-1])
				if err != nil {
					return false, actual, fmt.Errorf("invalid expectation regexp %q: %w", expectedText, err)
				}
				return pattern.MatchString(actualText), actual, nil
			}
			return normalizeDOMText(actualText) == normalizeDOMText(expectedText), actual, nil
		}
		return reflect.DeepEqual(actual, expectation.Value), actual, nil
	case "contains":
		return strings.Contains(normalizeDOMText(toolbox.AsString(actual)), normalizeDOMText(toolbox.AsString(expectation.Value))), actual, nil
	default:
		return false, actual, fmt.Errorf("unsupported expectation matcher %q", expectation.Matcher)
	}
}

func firstCallResult(results []interface{}) interface{} {
	for _, candidate := range results {
		if candidate == nil {
			continue
		}
		if _, isError := candidate.(error); isError {
			continue
		}
		return candidate
	}
	return nil
}

func appendAssertionResult(response *RunResponse, action *Action, call *MethodCall, results []interface{}, err error, started time.Time) {
	if response == nil || call == nil || call.Expectation == nil {
		return
	}
	selector := "page"
	if action != nil && action.Selector != nil {
		selector = action.Selector.By + ":" + action.Selector.Value
	}
	result := &AssertionResult{
		Method:    call.Method,
		Selector:  selector,
		Matcher:   call.Expectation.Matcher,
		Expected:  call.Expectation.Value,
		Actual:    firstCallResult(results),
		Passed:    err == nil,
		ElapsedMs: int(time.Since(started) / time.Millisecond),
	}
	if err != nil {
		result.Error = err.Error()
	}
	response.Assertions = append(response.Assertions, result)
}

func normalizeDOMText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func (s *service) callWebElement(context *endly.Context, request *WebElementCallRequest) (*WebElementCallResponse, error) {
	session, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	err = request.Selector.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid selector: %v", err)
	}
	if !request.sessionLocked {
		session.mu.Lock()
		defer session.mu.Unlock()
	}
	var deadline time.Time
	if request.Call.WaitTimeMs > 0 {
		deadline = time.Now().Add(time.Duration(request.Call.WaitTimeMs) * time.Millisecond)
	}
	for {
		runtimeRequest := *request
		runtimeCall := *request.Call
		if !deadline.IsZero() {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return nil, fmt.Errorf("webdriver element wait timed out after %dms: %s", request.Call.WaitTimeMs, request.Selector.Value)
			}
			runtimeCall.WaitTimeMs = max(1, int(remaining/time.Millisecond))
		}
		runtimeRequest.Call = &runtimeCall
		response, callErr := s.callWebElementOnce(context, session, &runtimeRequest)
		if response != nil && response.LookupError != "" && request.Call.AllowMissing {
			response.LookupError = ""
			return response, nil
		}
		lookupPending := response != nil && response.LookupError != "" && !deadline.IsZero()
		retryTransient := IsRetryableElementError(callErr) && !deadline.IsZero()
		if !lookupPending && !retryTransient {
			return response, callErr
		}
		if time.Now().After(deadline) {
			if callErr != nil {
				return response, callErr
			}
			return response, fmt.Errorf("webdriver element wait timed out after %dms: %s", request.Call.WaitTimeMs, response.LookupError)
		}
		pollInterval := time.Duration(request.Call.PollIntervalMs) * time.Millisecond
		if pollInterval <= 0 {
			pollInterval = 100 * time.Millisecond
		}
		if remaining := time.Until(deadline); pollInterval > remaining {
			pollInterval = remaining
		}
		if err := waitForContext(context.Background(), pollInterval); err != nil {
			return response, err
		}
	}
}

func (s *service) callWebElementOnce(context *endly.Context, session *Session, request *WebElementCallRequest) (*WebElementCallResponse, error) {
	var response = &WebElementCallResponse{
		Data: make(map[string]interface{}),
	}
	var selector = request.Selector
	var element selenium.WebElement

	findTimeout := defaultFindElementTimeout
	if request.Call.WaitTimeMs > 0 {
		// The outer wait loop owns the total deadline and retries missing or
		// stale elements without nesting another ten-second wait.
		findTimeout = 0
	}
	var err error
	if request.Strict {
		elements, findErr := session.driver.FindElements(selector.By, selector.Value)
		err = findErr
		switch len(elements) {
		case 0:
		case 1:
			element = elements[0]
		default:
			return response, fmt.Errorf("strict locator matched %d elements: %s:%s", len(elements), selector.By, selector.Value)
		}
	} else {
		element, err = findElement(context.Background(), session.driver, selector, findTimeout)
	}

	if err != nil || element == nil {
		response.LookupError = fmt.Sprintf("failed to lookup element: %v %v, %v", selector.By, selector.Value, err)
		return response, nil
	}

	elementPath := s.getResultPath(request.Selector.Key, request.Call, request.PathKind)
	callResponse := &ServiceCallResponse{
		Data: make(map[string]interface{}),
	}
	switch request.Call.Method {
	case "Click", "SendKeys", "Clear", "Submit", "Check", "Uncheck", "SelectOption", "MoveTo":
		if err = s.ensureVisible(context.Background(), element, time.Duration(request.Call.WaitTimeMs)*time.Millisecond); err != nil {
			// ChromeDriver 150 can return a JSON null for IsDisplayed even
			// when the element lookup succeeded. Let the actual interaction
			// report visibility/interactability in that compatibility case.
			if err.Error() == "nil return value" {
				err = nil
			} else {
				response.LookupError = fmt.Sprintf("element %s is not visible: %v", request.Selector.Value, err)
				return response, err
			}
		}
		if err = s.scrollIntoView(element); err != nil {
			response.LookupError = fmt.Sprintf("element %s could not be scrolled into view: %v", request.Selector.Value, err)
			return response, err
		}
	}

	err = s.call(context, element, request.Call, callResponse, elementPath...)
	response.Result = callResponse.Result
	util.Append(response.Data, callResponse.Data, true)
	if err != nil {
		if request.AllowJavaScriptClick && request.Call.Method == "Click" && isJavaScriptClickCandidate(err) {
			if _, fallbackErr := session.driver.ExecuteScript("arguments[0].click();", []interface{}{element}); fallbackErr == nil {
				return response, nil
			}
		}
		return response, fmt.Errorf("%w; selector=%s; %s", err, request.Selector.Value, s.elementDiagnostics(element))
	}
	return response, nil
}

func selectorStrictness(actionDefault bool, override *bool) bool {
	if override != nil {
		return *override
	}
	return actionDefault
}

func findElement(ctx context.Context, driver selenium.WebDriver, selector *WebElementSelector, timeout time.Duration) (selenium.WebElement, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		element, err := driver.FindElement(selector.By, selector.Value)
		if err == nil && element != nil {
			return element, nil
		}
		lastErr = err
		if timeout <= 0 || !time.Now().Before(deadline) {
			return nil, lastErr
		}
		if err := waitForContext(ctx, 100*time.Millisecond); err != nil {
			return nil, err
		}
	}
}

func (s *service) ensureVisible(ctx context.Context, element selenium.WebElement, timeout time.Duration) error {
	if timeout <= 0 || timeout > 2*time.Second {
		timeout = 2 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		ok, err := element.IsDisplayed()
		if ok {
			return nil
		}
		if IsStaleElementError(err) {
			return err
		}
		if !time.Now().Before(deadline) {
			if err != nil {
				return err
			}
			return errors.New("element is not displayed")
		}
		wait := 200 * time.Millisecond
		if remaining := time.Until(deadline); remaining < wait {
			wait = remaining
		}
		if waitErr := waitForContext(ctx, wait); waitErr != nil {
			return waitErr
		}
	}
}

// scrollIntoView makes keyboard and pointer actions deterministic for elements
// that are rendered below the current viewport. ChromeDriver does not always
// scroll small form controls into view before SendKeys.
func (s *service) scrollIntoView(element selenium.WebElement) error {
	_, err := element.LocationInView()
	if err != nil && err.Error() == "nil return value" {
		return nil
	}
	return err
}

func (s *service) elementDiagnostics(element selenium.WebElement) string {
	displayed, displayedErr := element.IsDisplayed()
	enabled, enabledErr := element.IsEnabled()
	location, locationErr := element.Location()
	size, sizeErr := element.Size()
	return fmt.Sprintf(
		"displayed=%t displayedErr=%v enabled=%t enabledErr=%v location=%v locationErr=%v size=%v sizeErr=%v",
		displayed,
		displayedErr,
		enabled,
		enabledErr,
		location,
		locationErr,
		size,
		sizeErr,
	)
}

func (s *service) open(context *endly.Context, request *OpenSessionRequest) (*OpenSessionResponse, error) {
	var response = &OpenSessionResponse{}
	seleniumSession, err := s.openSession(context, request)
	if err != nil {
		return nil, err
	}
	response.SessionID = seleniumSession.SessionID
	response.Attached = seleniumSession.Attached
	response.Backend = seleniumSession.Backend
	return response, nil
}

func (s *service) close(context *endly.Context, request *CloseSessionRequest) (*CloseSessionResponse, error) {
	var response = &CloseSessionResponse{
		SessionID: request.SessionID,
	}
	session, err := s.session(context, request.SessionID)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.Close()
	return response, err
}

func (s *service) deployServerIfNeeded(context *endly.Context, request *StartRequest, target *location.Resource) (*StartResponse, error) {
	deploymentService, _ := context.Service(deploy.ServiceID)

	response := &StartResponse{}
	driver, version := pair(request.Driver)
	driverURL := url.Join(request.Target.URL, driver)
	ok, _ := s.fs.Exists(context.Background(), driverURL)

	if !ok {
		driverResponse := deploymentService.Run(context, &deploy.Request{
			Target:       target,
			Version:      version,
			AppName:      driver,
			BaseLocation: request.BaseLocation,
		})
		if driverResponse.Error != "" {
			return nil, errors.New(driverResponse.Error)
		}
	}
	response.DriverPath = url.Path(driverURL)

	if request.Server != "" { //to use with standalone selenium  server
		serverURL := url.Join(request.Target.URL, SeleniumServer)
		ok, _ := s.fs.Exists(context.Background(), serverURL)
		if !ok {
			_, version = pair(request.Server)
			driverResponse := deploymentService.Run(context, &deploy.Request{
				Target:       target,
				Version:      version,
				AppName:      SeleniumServer,
				BaseLocation: request.BaseLocation,
			})
			if driverResponse.Error != "" {
				return nil, errors.New(driverResponse.Error)
			}
		}
		response.ServerPath = url.Path(serverURL)
	}
	return response, nil
}

func (s *service) setJdk(context *endly.Context, request *StartRequest) error {
	if request.Sdk == "" {
		return nil
	}
	sdkService, _ := context.Service(sdk.ServiceID)
	_, version := pair(request.Sdk)
	response := sdkService.Run(context, &sdk.SetRequest{
		Sdk:     request.Sdk,
		Version: version,
		Target:  request.Target,
	})

	if response.Error != "" {
		return errors.New(response.Error)
	}
	return nil
}

func (s *service) stop(context *endly.Context, request *StopRequest) (*StopResponse, error) {
	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	session, _ := s.session(context, fmt.Sprintf("localhost:%v", request.Port))
	if session == nil {
		return &StopResponse{}, nil
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	processService, _ := context.Service(process.ServiceID)
	if session.Pid > 0 {
		serviceResponse := processService.Run(context, &process.StopRequest{
			Target: target,
			Input:  fmt.Sprintf("selenium-server-standalone.jar -port %v", toolbox.AsString(request.Port)),
		})
		if serviceResponse.Error != "" {
			return nil, errors.New(serviceResponse.Error)
		}
	}

	session.Close()

	return &StopResponse{}, nil
}

func (s *service) start(context *endly.Context, request *StartRequest) (*StartResponse, error) {
	request.BaseLocation = expandContextText(context, request.BaseLocation)
	if request.URL != "" {
		request.Target = location.NewResource(expandContextText(context, request.URL))
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	response, err := s.deployServerIfNeeded(context, request, target)
	if err != nil {
		return nil, err
	}
	sessionID := fmt.Sprintf("localhost:%v", request.Port)
	session, ok := lookupSession(context, sessionID)
	if ok {
		session.mu.Lock()
		session.Close()
		session.mu.Unlock()
	} else {
		session = &Session{SessionID: sessionID}
		putSession(context, sessionID, session)
	}
	registerSessionCleanup(context, session)
	useSelenium := request.Server != ""
	if !useSelenium {
		session.Capabilities = request.Capabilities
		session.PageLoadStrategy = request.PageLoadStrategy
		switch request.Driver {
		case ChromeDriver:
			if session.service, err = selenium.NewChromeDriverService(response.DriverPath, request.Port); err != nil {
				return nil, fmt.Errorf("failed to start chromedriver service %w", err)
			}
			session.Browser = ChromeBrowser
		case GeckoDriver:
			if session.service, err = selenium.NewGeckoDriverService(response.DriverPath, request.Port); err != nil {
				return nil, fmt.Errorf("failed to start geckodriver service %w", err)
			}
			session.Browser = FirefoxBrowser
		default:
			if request.Server == "" {
				return nil, fmt.Errorf("invalid driver %v", request.Driver)
			}
		}
	}
	session.SessionID = sessionID
	if request.Server == "" {
		return response, nil
	}
	session.Server = request.Server
	err = s.setJdk(context, request)
	if err != nil {
		return nil, err
	}

	s.Run(context, &StopRequest{
		Target: target,
		Port:   request.Port,
	})
	processService, _ := context.Service(process.ServiceID)
	session.Browser = FirefoxBrowser
	serviceResponse := processService.Run(context, &process.StartRequest{
		Command: "java",
		Target:  target,
		Options: &exec.Options{
			Directory:  defaultTarget,
			CheckError: true,
		},
		Arguments:       []string{fmt.Sprintf("-Dwebdriver.gecko.driver=%v", response.DriverPath), "-jar", response.ServerPath, "-port", toolbox.AsString(request.Port)},
		ImmuneToHangups: true,
	})
	if serviceResponse.Error != "" {
		return nil, errors.New(serviceResponse.Error)
	}

	if processResponse, ok := serviceResponse.Response.(*process.StartResponse); ok && len(processResponse.Info) > 0 {
		response.Pid = processResponse.Info[0].Pid
		session.Pid = response.Pid
	}
	return response, nil
}

func (s *service) session(context *endly.Context, sessionID string) (*Session, error) {
	if seleniumSession, ok := lookupSession(context, sessionID); ok {
		return seleniumSession, nil
	}
	return nil, fmt.Errorf("failed to lookup seleniun session id: %v, make sure you first run SeleniumOpenSessionRequest", sessionID)
}

func (s *service) openSession(context *endly.Context, request *OpenSessionRequest) (*Session, error) {
	ensureOpenSessionDefaults(request)
	sessionID := request.SessionID
	session, ok := lookupSession(context, sessionID)
	if request.DirectCDP {
		if !ok {
			session = &Session{SessionID: sessionID, Browser: ChromeBrowser}
			putSession(context, sessionID, session)
		}
		registerSessionCleanup(context, session)
		session.mu.Lock()
		defer session.mu.Unlock()
		session.Close()
		driver, err := newDirectCDPDriver(request.DebuggerAddress)
		if err != nil {
			return nil, fmt.Errorf("connect direct CDP at %s: %w", request.DebuggerAddress, err)
		}
		driver.pageLoadStrategy = request.PageLoadStrategy
		if len(request.BlockedURLs) > 0 {
			if _, blockErr := driver.CDPCommand("Network.setBlockedURLs", map[string]interface{}{"urls": request.BlockedURLs}); blockErr != nil {
				driver.Quit()
				return nil, fmt.Errorf("configure direct CDP blocked URLs: %w", blockErr)
			}
		}
		session.driver = driver
		session.Browser = ChromeBrowser
		session.Attached = true
		session.Backend = "cdp"
		session.CDPRemote = driver.address
		session.DriverSessionID = driver.SessionID()
		putSession(context, sessionID, session)
		return session, nil
	}
	if !ok {
		if request.DebuggerAddress == "" {
			return nil, fmt.Errorf("webdriver service not running - start ?")
		}
		session = &Session{SessionID: sessionID, Browser: ChromeBrowser, Attached: true}
		putSession(context, sessionID, session)
	}
	registerSessionCleanup(context, session)
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.driver != nil {
		_ = session.driver.Quit()
		session.driver = nil
		session.CDPRemote = ""
		session.DriverSessionID = ""
	}

	caps := selenium.Capabilities{}
	pageLoadStrategy := request.PageLoadStrategy
	if pageLoadStrategy == "" {
		pageLoadStrategy = session.PageLoadStrategy
	}
	if pageLoadStrategy != "" {
		caps["pageLoadStrategy"] = pageLoadStrategy
	}
	if session.Pid == 0 {
		if len(session.Capabilities) > 0 && len(request.Capabilities) == 0 {
			request.Capabilities = session.Capabilities
		}
		switch session.Browser {
		case ChromeBrowser:
			chromeCaps := chrome.Capabilities{Args: request.Capabilities, DebuggerAddr: request.DebuggerAddress}
			if request.DebuggerAddress != "" {
				detach := true
				chromeCaps.Detach = &detach
				session.Attached = true
			}
			caps.AddChrome(chromeCaps)
			caps.SetLogLevel(selog.Performance, selog.All)
			caps.SetLogLevel(selog.Browser, selog.All)
		case FirefoxBrowser:
			caps.AddFirefox(firefox.Capabilities{Args: request.Capabilities})
			caps.SetLogLevel(selog.Browser, selog.All)
		}
	} else {
		caps["browserName"] = request.Browser
	}

	driver, err := selenium.NewRemote(caps, request.Remote)
	if err != nil {
		return nil, err
	}
	session.driver = driver
	session.Backend = "selenium"
	session.Remote = request.Remote
	session.CDPRemote = request.Remote
	session.DriverSessionID = driver.SessionID()
	if len(request.BlockedURLs) > 0 {
		if !isChromeLike(session.Browser) {
			_ = driver.Quit()
			session.driver = nil
			return nil, fmt.Errorf("blockedURLs requires Chrome")
		}
		_, _ = cdpExecute(session.Remote, driver.SessionID(), "Network.enable", map[string]any{})
		if _, blockErr := cdpExecute(session.Remote, driver.SessionID(), "Network.setBlockedURLs", map[string]any{"urls": request.BlockedURLs}); blockErr != nil {
			_ = driver.Quit()
			session.driver = nil
			return nil, fmt.Errorf("configure blocked URLs: %w", blockErr)
		}
	}
	putSession(context, sessionID, session)
	return session, nil
}

func registerSessionCleanup(context *endly.Context, session *Session) {
	if context == nil || session == nil {
		return
	}
	session.mu.Lock()
	if session.cleanupRegistered {
		session.mu.Unlock()
		return
	}
	session.cleanupRegistered = true
	session.mu.Unlock()
	context.Deffer(func() {
		session.mu.Lock()
		session.Close()
		session.mu.Unlock()
	})
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "start",
		RequestInfo: &endly.ActionInfo{
			Description: "start selenium server",
		},
		RequestProvider: func() interface{} {
			return &StartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StartResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StartRequest); ok {
				return s.start(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "stop",
		RequestInfo: &endly.ActionInfo{
			Description: "stop selenium server",
		},
		RequestProvider: func() interface{} {
			return &StopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StopResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StopRequest); ok {
				return s.stop(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "open",
		RequestInfo: &endly.ActionInfo{
			Description: "open selenium session",
		},
		RequestProvider: func() interface{} {
			return &OpenSessionRequest{}
		},
		ResponseProvider: func() interface{} {
			return &OpenSessionResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*OpenSessionRequest); ok {
				return s.open(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "close",
		RequestInfo: &endly.ActionInfo{
			Description: "close selenium session",
		},
		RequestProvider: func() interface{} {
			return &CloseSessionRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CloseSessionResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CloseSessionRequest); ok {
				return s.close(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "stop-loading",
		RequestInfo: &endly.ActionInfo{
			Description: "interrupt an active Chrome page load through CDP",
		},
		RequestProvider: func() interface{} {
			return &StopLoadingRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StopLoadingResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StopLoadingRequest); ok {
				return s.stopLoading(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "run",
		RequestInfo: &endly.ActionInfo{
			Description: "run selenium requests",
		},
		RequestProvider: func() interface{} {
			return &RunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RunRequest); ok {
				return s.run(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "call-driver",
		RequestInfo: &endly.ActionInfo{
			Description: "call proxies request to  github.com/tebeka/selenium web driver",
		},
		RequestProvider: func() interface{} {
			return &WebDriverCallRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ServiceCallResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*WebDriverCallRequest); ok {
				return s.callWebDriver(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "call-element",
		RequestInfo: &endly.ActionInfo{
			Description: "find web element and proxy request",
		},
		RequestProvider: func() interface{} {
			return &WebElementCallRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ServiceCallResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*WebElementCallRequest); ok {
				return s.callWebElement(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-start",
		RequestInfo: &endly.ActionInfo{
			Description: "start capturing console and network (Chrome/Edge CDP)",
		},
		RequestProvider: func() interface{} {
			return &CaptureStartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureStartResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureStartRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureStart(context, req)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-stop",
		RequestInfo: &endly.ActionInfo{
			Description: "stop capturing console and network (Chrome/Edge CDP)",
		},
		RequestProvider: func() interface{} {
			return &CaptureStopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureStopResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureStopRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureStop(context, req)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-status",
		RequestInfo: &endly.ActionInfo{
			Description: "capture status and counters",
		},
		RequestProvider: func() interface{} {
			return &CaptureStatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureStatusResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureStatusRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureStatus(context, req)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-clear",
		RequestInfo: &endly.ActionInfo{
			Description: "clear captured console and network buffers",
		},
		RequestProvider: func() interface{} {
			return &CaptureClearRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureClearResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureClearRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureClear(context, req)
		},
	})

	s.Register(&endly.Route{
		Action: "capture-export",
		RequestInfo: &endly.ActionInfo{
			Description: "export captured console and network buffers",
		},
		RequestProvider: func() interface{} {
			return &CaptureExportRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CaptureExportResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*CaptureExportRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			return s.captureExport(context, req)
		},
	})

}

// New creates a new webdriver service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
		fs:              afs.New(),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}

func pair(value string) (string, string) {
	pair := strings.SplitN(value, ":", 2)
	if len(pair) == 2 {
		return pair[0], pair[1]
	}
	return value, ""
}

func ensureOpenSessionDefaults(request *OpenSessionRequest) {
	if request == nil {
		return
	}
	if request.SessionID == "" {
		request.SessionID = "localhost:4444"
	}
	if request.Remote == "" {
		host, port := pair(request.SessionID)
		request.Remote = fmt.Sprintf("http://%v:%v/wd/hub", host, port)
	}
}

func isGetMethod(method string) bool {
	return strings.EqualFold(method, "Get")
}

func navigationWithDefaults(nav *NavigationOptions) NavigationOptions {
	if nav == nil {
		nav = &NavigationOptions{}
	}
	out := *nav
	if out.TimeoutMs <= 0 {
		out.TimeoutMs = 45000
	}
	if out.AutoScrollMs < 0 {
		out.AutoScrollMs = 0
	}
	if out.ScrollDelayMs <= 0 {
		out.ScrollDelayMs = 300
	}
	if out.StableWindowMs <= 0 {
		out.StableWindowMs = 1500
	}
	if out.MaxScrollSteps <= 0 {
		out.MaxScrollSteps = 30
	}
	if out.MaxScrollGrowthPx <= 0 {
		out.MaxScrollGrowthPx = 50_000
	}
	if out.IdleThreshold < 0 {
		out.IdleThreshold = 0
	}
	if out.IdleWindowMs <= 0 {
		out.IdleWindowMs = 1500
	}
	if out.IdleMaxWaitMs < 0 {
		out.IdleMaxWaitMs = 0
	}
	return out
}

func (s *service) getWithGuard(context *endly.Context, session *Session, URL string, nav NavigationOptions) (*NavigationReport, error) {
	started := time.Now()
	report := &NavigationReport{URL: URL, StopReason: "loaded"}
	defer func() {
		report.ElapsedMs = int(time.Since(started) / time.Millisecond)
	}()
	if session == nil || session.driver == nil {
		return report, fmt.Errorf("webdriver session not open")
	}
	if err := session.driver.SetPageLoadTimeout(time.Duration(nav.TimeoutMs) * time.Millisecond); err != nil {
		return report, fmt.Errorf("set page-load timeout: %w", err)
	}
	err := session.driver.Get(URL)
	if err != nil {
		if !isPageLoadTimeout(err) {
			return report, err
		}
		report.TimedOut = true
		report.StopReason = "timeout-continued"
		stopLoading := nav.StopLoadingOnTimeout == nil || *nav.StopLoadingOnTimeout
		if stopLoading {
			if _, stopErr := session.driver.ExecuteScript("window.stop();", nil); stopErr != nil {
				report.Warning = "stop loading: " + stopErr.Error()
			} else {
				report.LoadingStopped = true
			}
		}
		continueOnTimeout := nav.ContinueOnTimeout == nil || *nav.ContinueOnTimeout
		if !continueOnTimeout {
			report.StopReason = "timeout"
			return report, err
		}
		context.Publish(msg.NewOutputEvent("Navigation timeout (continuing)", "webdriver.get", map[string]any{
			"url":   URL,
			"error": err.Error(),
		}))
	}
	if nav.AutoScrollMs > 0 {
		if session.Capture == nil && session.Net == nil && isChromeLike(session.Browser) {
			session.Net = &netTracker{}
		}
		if err := s.autoScrollStabilize(context, session, nav, report); err != nil {
			return report, err
		}
	}
	return report, nil
}

func isPageLoadTimeout(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return true
	}
	var sErr *selenium.Error
	if errors.As(err, &sErr) {
		if sErr.LegacyCode == 21 { // legacy "timeout"
			return true
		}
		if strings.Contains(strings.ToLower(sErr.Message), "timeout") {
			return true
		}
	}
	return false
}

func (s *service) autoScrollStabilize(context *endly.Context, session *Session, nav NavigationOptions, report *NavigationReport) error {
	maxWaitMs := nav.AutoScrollMs
	if nav.IdleMaxWaitMs > 0 && nav.IdleMaxWaitMs < maxWaitMs {
		maxWaitMs = nav.IdleMaxWaitMs
	}
	deadline := time.Now().Add(time.Duration(maxWaitMs) * time.Millisecond)
	delay := time.Duration(nav.ScrollDelayMs) * time.Millisecond
	stableWindow := time.Duration(nav.StableWindowMs) * time.Millisecond
	idleWindow := time.Duration(nav.IdleWindowMs) * time.Millisecond

	report.ScrollTarget = nav.ScrollSelector
	if report.ScrollTarget == "" {
		report.ScrollTarget = "document"
	}
	metrics, err := s.pageMetrics(session, nav.ScrollSelector)
	if err != nil {
		report.StopReason = "metrics-error"
		return err
	}
	report.Scrolled = true
	report.StartHeightPx = int(metrics.Height)
	report.FinalHeightPx = int(metrics.Height)
	if nav.ReturnToTop {
		defer func() {
			_, _ = session.driver.ExecuteScript(`const target = arguments[0] ? document.querySelector(arguments[0]) : null;
			if (target) target.scrollTo(0, 0); else window.scrollTo(0, 0);`, []interface{}{nav.ScrollSelector})
		}()
	}
	lastHeight := metrics.Height
	lastY := metrics.Y
	lastContent := metrics.Content
	stableSince := time.Now()
	idleSince := time.Time{}

	for report.Steps < nav.MaxScrollSteps {
		if err := context.Background().Err(); err != nil {
			report.StopReason = "cancelled"
			return err
		}
		if !time.Now().Before(deadline) {
			report.StopReason = "time-budget"
			return nil
		}
		metrics, err = s.pageMetrics(session, nav.ScrollSelector)
		if err != nil {
			report.StopReason = "metrics-error"
			return err
		}
		report.FinalHeightPx = int(metrics.Height)
		if nav.MaxScrollHeightPx > 0 && metrics.Height >= float64(nav.MaxScrollHeightPx) {
			report.StopReason = "height-limit"
			return nil
		}
		if nav.MaxScrollGrowthPx > 0 && metrics.Height-float64(report.StartHeightPx) >= float64(nav.MaxScrollGrowthPx) {
			report.StopReason = "growth-limit"
			return nil
		}
		stable := metrics.Height == lastHeight && metrics.Y == lastY && metrics.Content == lastContent
		if !stable {
			stableSince = time.Now()
			lastHeight = metrics.Height
			lastY = metrics.Y
			lastContent = metrics.Content
		}
		if stable && metrics.AtBottom() && time.Since(stableSince) >= stableWindow && s.isNetworkIdle(session, nav.IdleThreshold, idleWindow, &idleSince) {
			report.StopReason = "stable-bottom"
			return nil
		}

		if _, err := session.driver.ExecuteScript(`const selector = arguments[0];
		const target = selector ? document.querySelector(selector) : null;
		if (selector && !target) throw new Error("scroll container not found: " + selector);
		if (target) target.scrollBy(0, target.clientHeight || 800);
		else window.scrollBy(0, window.innerHeight || 800);`, []interface{}{nav.ScrollSelector}); err != nil {
			report.StopReason = "scroll-error"
			return fmt.Errorf("scroll page: %w", err)
		}
		report.Steps++
		if err := waitForContext(context.Background(), delay); err != nil {
			report.StopReason = "cancelled"
			return err
		}
		if session.Capture != nil {
			session.Capture.Drain(session)
		} else if session.Net != nil {
			session.Net.Drain(session.driver)
		}
	}
	report.StopReason = "max-steps"
	return nil
}

type pageMetrics struct {
	Y        float64
	Viewport float64
	Height   float64
	Content  string
}

func (m pageMetrics) AtBottom() bool {
	return m.Y+m.Viewport >= m.Height-2
}

func (s *service) pageMetrics(session *Session, selector string) (pageMetrics, error) {
	if session == nil || session.driver == nil {
		return pageMetrics{}, fmt.Errorf("webdriver session not open")
	}
	value, err := session.driver.ExecuteScript(`const selector = arguments[0];
	const target = selector ? document.querySelector(selector) : null;
	if (selector && !target) throw new Error("scroll container not found: " + selector);
	const root = target || document.scrollingElement || document.documentElement;
	const last = root && root.lastElementChild;
	return {
		y: target ? target.scrollTop : (window.pageYOffset || root.scrollTop || 0),
		viewport: target ? target.clientHeight : (window.innerHeight || root.clientHeight || 0),
		height: target ? target.scrollHeight : Math.max(document.body ? document.body.scrollHeight : 0, root ? root.scrollHeight : 0),
		content: last ? ((last.getAttribute && (last.getAttribute('data-id') || last.id)) || (last.textContent || '').slice(-256)) : ''
	};`, []interface{}{selector})
	if err != nil {
		return pageMetrics{}, fmt.Errorf("read page scroll metrics: %w", err)
	}
	values := toolbox.AsMap(value)
	if len(values) == 0 {
		return pageMetrics{}, fmt.Errorf("invalid page scroll metrics: %T", value)
	}
	return pageMetrics{
		Y:        toolbox.AsFloat(values["y"]),
		Viewport: toolbox.AsFloat(values["viewport"]),
		Height:   toolbox.AsFloat(values["height"]),
		Content:  toolbox.AsString(values["content"]),
	}, nil
}

func waitForContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *service) isNetworkIdle(session *Session, threshold int, window time.Duration, idleSince *time.Time) bool {
	if threshold < 0 {
		threshold = 0
	}
	if window <= 0 {
		window = 1500 * time.Millisecond
	}

	inflight := -1
	if session.Capture != nil {
		inflight = session.Capture.Summary().RequestsInFlight
	} else if session.Net != nil {
		inflight = session.Net.Inflight()
	}
	if inflight < 0 {
		return true // cannot observe; do not block
	}
	if networkIdle(inflight, threshold) {
		if idleSince != nil && idleSince.IsZero() {
			*idleSince = time.Now()
		}
		if idleSince == nil {
			return true
		}
		return time.Since(*idleSince) >= window
	}
	if idleSince != nil {
		*idleSince = time.Time{}
	}
	return false
}

func (s *service) scrollHeight(session *Session) float64 {
	if session == nil || session.driver == nil {
		return -1
	}
	v, err := session.driver.ExecuteScript("return (document.body && document.body.scrollHeight) ? document.body.scrollHeight : 0;", nil)
	if err != nil {
		return -1
	}
	switch actual := v.(type) {
	case float64:
		return actual
	case int:
		return float64(actual)
	case int64:
		return float64(actual)
	case uint64:
		return float64(actual)
	case string:
		return toolbox.AsFloat(actual)
	default:
		return toolbox.AsFloat(actual)
	}
}
