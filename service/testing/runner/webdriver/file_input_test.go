package webdriver

import (
	"testing"

	"github.com/tebeka/selenium"
	"github.com/viant/endly"
)

type uploadElement struct {
	selenium.WebElement
	kind     string
	enabled  bool
	selected string
}

func (e *uploadElement) TagName() (string, error) { return "input", nil }
func (e *uploadElement) GetAttribute(name string) (string, error) {
	if name == "type" {
		return e.kind, nil
	}
	return "", nil
}
func (e *uploadElement) IsEnabled() (bool, error) { return e.enabled, nil }
func (e *uploadElement) IsDisplayed() (bool, error) {
	panic("file selection must not require a visible native input")
}
func (e *uploadElement) SendKeys(value string) error { e.selected = value; return nil }

type uploadDriver struct {
	selenium.WebDriver
	element *uploadElement
}

func (d *uploadDriver) FindElements(by, value string) ([]selenium.WebElement, error) {
	return []selenium.WebElement{d.element}, nil
}

func TestRunSelectsHiddenFileInput(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	element := &uploadElement{kind: "file", enabled: true}
	putSession(ctx, "fake", &Session{SessionID: "fake", Browser: ChromeBrowser, driver: &uploadDriver{element: element}})
	request := &RunRequest{SessionID: "fake", Commands: []interface{}{`page.locator("#upload").setInputFiles("/tmp/test.xlsx")`}, ActionTimeoutMs: 100}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	svc := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	if _, err := svc.run(ctx, request); err != nil {
		t.Fatal(err)
	}
	if element.selected != "/tmp/test.xlsx" {
		t.Fatalf("selected %q", element.selected)
	}
}

func TestSetInputFilesRejectsInvalidTargets(t *testing.T) {
	for _, test := range []struct {
		kind    string
		enabled bool
		path    string
	}{{"text", true, "/tmp/a.xlsx"}, {"file", false, "/tmp/a.xlsx"}, {"file", true, ""}} {
		element := &uploadElement{kind: test.kind, enabled: test.enabled}
		if err := (&service{}).SetInputFiles(element, test.path); err == nil {
			t.Fatalf("accepted invalid target: %+v", test)
		}
		if element.selected != "" {
			t.Fatal("sent a path to invalid target")
		}
	}
}
