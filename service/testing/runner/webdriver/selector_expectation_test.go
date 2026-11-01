package webdriver

import (
	"github.com/tebeka/selenium"
	"github.com/viant/endly"
	"testing"
)

type switchingSelectionDriver struct {
	selenium.WebDriver
	lookups   int
	old, next *runtimeElement
}

func (d *switchingSelectionDriver) FindElement(string, string) (selenium.WebElement, error) {
	d.lookups++
	if d.lookups == 1 {
		return d.old, nil
	}
	return d.next, nil
}
func TestElementExpectationResolvesChangingSelector(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	driver := &switchingSelectionDriver{old: &runtimeElement{text: "First tab"}, next: &runtimeElement{text: "Second tab"}}
	putSession(ctx, "changing-selector", &Session{SessionID: "changing-selector", Browser: ChromeBrowser, driver: driver})
	s := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	_, err := s.callWebElement(ctx, &WebElementCallRequest{SessionID: "changing-selector", Selector: &WebElementSelector{By: "css selector", Value: "[aria-selected='true']"}, Call: &MethodCall{Method: "Text", Wait: Wait{WaitTimeMs: 200, PollIntervalMs: 1, Expectation: &CallExpectation{Matcher: "contains", Value: "Second"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if driver.lookups != 2 {
		t.Fatalf("lookups=%d; want fresh selector for each assertion attempt", driver.lookups)
	}
}
