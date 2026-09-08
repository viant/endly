package webdriver

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tebeka/selenium"
	selog "github.com/tebeka/selenium/log"
	"github.com/viant/endly"
)

type runtimeElement struct {
	selenium.WebElement
	mu             sync.Mutex
	clicks         int
	interceptCount int
	text           string
}

func (e *runtimeElement) IsDisplayed() (bool, error)               { return true, nil }
func (e *runtimeElement) IsEnabled() (bool, error)                 { return true, nil }
func (e *runtimeElement) LocationInView() (*selenium.Point, error) { return &selenium.Point{}, nil }
func (e *runtimeElement) Location() (*selenium.Point, error)       { return &selenium.Point{}, nil }
func (e *runtimeElement) Size() (*selenium.Size, error)            { return &selenium.Size{}, nil }
func (e *runtimeElement) Text() (string, error)                    { return e.text, nil }
func (e *runtimeElement) Click() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.clicks++
	if e.clicks <= e.interceptCount {
		return &selenium.Error{LegacyCode: elementClickInterceptedException, Message: "element click intercepted"}
	}
	return nil
}

type runtimeDriver struct {
	selenium.WebDriver
	element       *runtimeElement
	findCalls     int
	findAllCalls  int
	metrics       []map[string]interface{}
	metricIndex   int
	scrollCalls   int
	jsClickCalls  int
	elementCount  int
	getErr        error
	stopCalls     int
	capture       *CaptureState
	responseOnGet *NetworkTransaction
}

func (d *runtimeDriver) FindElement(by, value string) (selenium.WebElement, error) {
	d.findCalls++
	if d.element == nil {
		return nil, nil
	}
	return d.element, nil
}
func (d *runtimeDriver) FindElements(by, value string) ([]selenium.WebElement, error) {
	d.findAllCalls++
	if d.elementCount == 0 && d.element != nil {
		return []selenium.WebElement{d.element}, nil
	}
	return make([]selenium.WebElement, d.elementCount), nil
}
func (d *runtimeDriver) SetPageLoadTimeout(time.Duration) error { return nil }
func (d *runtimeDriver) Get(string) error {
	if d.capture != nil && d.responseOnGet != nil {
		d.capture.mux.Lock()
		d.capture.finishLocked(d.responseOnGet.RequestID, d.responseOnGet)
		d.capture.mux.Unlock()
	}
	return d.getErr
}
func (d *runtimeDriver) Log(selog.Type) ([]selog.Message, error) { return nil, nil }
func (d *runtimeDriver) SessionID() string                       { return "fake" }
func (d *runtimeDriver) ExecuteScript(script string, args []interface{}) (interface{}, error) {
	if strings.Contains(script, "return {") {
		if len(d.metrics) == 0 {
			return map[string]interface{}{"y": 0, "viewport": 100, "height": 100, "content": ""}, nil
		}
		index := min(d.metricIndex, len(d.metrics)-1)
		d.metricIndex++
		return d.metrics[index], nil
	}
	if strings.Contains(script, "arguments[0].click") {
		d.jsClickCalls++
		return nil, nil
	}
	if strings.Contains(script, "window.stop") {
		d.stopCalls++
		return nil, nil
	}
	if strings.Contains(script, "scrollBy") {
		d.scrollCalls++
	}
	return nil, nil
}

func TestRunRetriesInterceptedClick(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	element := &runtimeElement{interceptCount: 1}
	driver := &runtimeDriver{element: element}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, driver: driver})
	request := &RunRequest{SessionID: "fake", Commands: []interface{}{`page.locator("#submit").click()`}, ActionTimeoutMs: 500, PollIntervalMs: 1}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	_, err := service.run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if element.clicks != 2 || driver.findAllCalls != 2 {
		t.Fatalf("clicks=%d findAllCalls=%d, wanted 2/2", element.clicks, driver.findAllCalls)
	}
}

func TestRunUsesOptInJavaScriptClickFallback(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	element := &runtimeElement{interceptCount: 100}
	driver := &runtimeDriver{element: element}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, driver: driver})
	request := &RunRequest{
		SessionID:            "fake",
		Commands:             []interface{}{`page.locator("#submit").click()`},
		ActionTimeoutMs:      500,
		PollIntervalMs:       1,
		AllowJavaScriptClick: true,
	}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	if _, err := service.run(ctx, request); err != nil {
		t.Fatal(err)
	}
	if driver.jsClickCalls != 1 {
		t.Fatalf("js clicks=%d, wanted 1", driver.jsClickCalls)
	}
}

func TestRunStopsInfiniteScrollAtGrowthLimit(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	driver := &runtimeDriver{metrics: []map[string]interface{}{
		{"y": 0, "viewport": 500, "height": 1000, "content": "a"},
		{"y": 0, "viewport": 500, "height": 1000, "content": "a"},
		{"y": 500, "viewport": 500, "height": 5000, "content": "b"},
	}}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, driver: driver})
	request := &RunRequest{
		SessionID: "fake",
		Commands:  []interface{}{`page.goto("https://example.test/feed")`},
		Navigation: &NavigationOptions{
			AutoScrollMs:      1000,
			ScrollDelayMs:     1,
			MaxScrollSteps:    100,
			MaxScrollGrowthPx: 1000,
		},
	}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	response, err := service.run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Navigations) != 1 || response.Navigations[0].StopReason != "growth-limit" {
		t.Fatalf("navigation=%#v", response.Navigations)
	}
	if driver.scrollCalls != 1 {
		t.Fatalf("scroll calls=%d, wanted 1", driver.scrollCalls)
	}
}

func TestRunStopsLoadingAfterNavigationTimeout(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	driver := &runtimeDriver{getErr: &selenium.Error{LegacyCode: 21, Message: "timeout"}}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, driver: driver})
	request := &RunRequest{SessionID: "fake", Commands: []interface{}{`page.goto("https://example.test/never-finishes")`}}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	response, err := service.run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Navigations) != 1 || !response.Navigations[0].TimedOut || !response.Navigations[0].LoadingStopped {
		t.Fatalf("navigation=%#v", response.Navigations)
	}
	if driver.stopCalls != 1 {
		t.Fatalf("stop calls=%d, wanted 1", driver.stopCalls)
	}
}

func TestRunHiddenExpectationPassesWhenElementIsAbsent(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	driver := &runtimeDriver{}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, driver: driver})
	request := &RunRequest{SessionID: "fake", Commands: []interface{}{`expect("#spinner").toBeHidden(50)`}}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	response, err := service.run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.LookupErrors) != 0 {
		t.Fatalf("lookup errors=%v", response.LookupErrors)
	}
}

func TestRunCountExpectation(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	driver := &runtimeDriver{elementCount: 3}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, driver: driver})
	request := &RunRequest{SessionID: "fake", Commands: []interface{}{`expect(page.locator("css selector:.row")).toHaveCount("3", 50)`}}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	response, err := service.run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Assertions) != 1 || !response.Assertions[0].Passed || response.Assertions[0].Actual != 3 {
		t.Fatalf("assertions=%#v", response.Assertions)
	}
}

func TestRunRejectsAmbiguousPlaywrightLocator(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	driver := &runtimeDriver{element: &runtimeElement{}, elementCount: 2}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, driver: driver})
	request := &RunRequest{SessionID: "fake", Commands: []interface{}{`page.locator("button").click()`}, ActionTimeoutMs: 50}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	_, err := service.run(ctx, request)
	if err == nil || !strings.Contains(err.Error(), "strict locator matched 2 elements") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunReturnsStructuredInlineAssertionFailure(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	driver := &runtimeDriver{element: &runtimeElement{text: "Pending"}}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, driver: driver})
	request := &RunRequest{SessionID: "fake", Commands: []interface{}{`expect("#status").toHaveText("Ready", 5)`}, PollIntervalMs: 1}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	response, err := service.run(ctx, request)
	if err == nil {
		t.Fatal("expected assertion failure")
	}
	if len(response.Assertions) != 1 || response.Assertions[0].Passed || response.Assertions[0].Actual != "Pending" {
		t.Fatalf("assertions=%#v", response.Assertions)
	}
	validations := response.Assertion()
	if len(validations) != 1 || validations[0].FailedCount != 1 {
		t.Fatalf("validations=%#v", validations)
	}
}

func TestRunWaitsForCapturedBrowserResponse(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	capture := newCaptureState(&CaptureStartRequest{})
	driver := &runtimeDriver{capture: capture, responseOnGet: &NetworkTransaction{RequestID: "one", URL: "https://example.test/api/orders/42", Status: 201}}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, Remote: "http://127.0.0.1:1", driver: driver, Capture: capture})
	request := &RunRequest{SessionID: "fake", Commands: []interface{}{`page.goto("https://example.test")`, `api = page.waitForResponse("/api/orders", 201, 50)`}}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	service := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	response, err := service.run(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Data["api"] == nil {
		t.Fatalf("response data=%#v", response.Data)
	}
}

func TestResponseURLMatcher(t *testing.T) {
	contains, err := responseURLMatcher("/api/orders")
	if err != nil || !contains("https://example.test/api/orders/1") {
		t.Fatalf("contains matcher err=%v", err)
	}
	regex, err := responseURLMatcher(`/api/(orders|users)/`)
	if err != nil || !regex("https://example.test/api/users") {
		t.Fatalf("regex matcher err=%v", err)
	}
}
