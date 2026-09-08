package webdriver

import (
	"github.com/stretchr/testify/assert"
	"github.com/tebeka/selenium"
	"github.com/viant/assertly"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	var useCases = []struct {
		Description string
		Command     string
		Expected    *Action
		HasError    bool
	}{
		{
			Description: "WebDriver call",
			Command:     "get(http://127.0.0.1:8888/signup/)",
			Expected:    NewAction("", "", "Get", "http://127.0.0.1:8888/signup/"),
		},
		{
			Description: "WebDriver call empty",
			Command:     "get",
			Expected:    NewAction("", "", "Get"),
		},
		{
			Description: "WebDriver call assigments",
			Command:     "key1 = get(http://127.0.0.1:8888/signup/)",
			Expected:    NewAction("key1", "", "Get", "http://127.0.0.1:8888/signup/"),
		},

		{
			Description: "WebElement call",
			Command:     "(#email).clear",
			Expected:    NewAction("", "#email", "Clear"),
		},

		{
			Description: "xpath selector WebElement call",
			Command:     "(xpath://SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]).text",
			Expected:    NewAction("", "//SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]", "Text"),
		},

		{
			Description: "xpath selector WebElement call assigment",
			Command:     "key1 = (xpath://SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]).text",
			Expected:    NewAction("key1", "//SMALL[preceding-sibling::INPUT[@id='dateOfBirth']]", "Text"),
		},
	}

	parser := &parser{}
	for _, useCase := range useCases {
		//fmt.Println(useCase.Command)
		action, err := parser.Parse(useCase.Command)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		if !assert.Nil(t, err, useCase.Description) {
			continue
		}
		assertly.AssertValues(t, useCase.Expected, action, useCase.Description)

	}

}

func TestParser_ParsePageCommands(t *testing.T) {
	testCases := []struct {
		command    string
		key        string
		selectorBy string
		selector   string
		methods    []string
		parameters [][]interface{}
	}{
		{
			command:    `page.goto("https://example.test/path?a=1")`,
			methods:    []string{"Get"},
			parameters: [][]interface{}{{"https://example.test/path?a=1"}},
		},
		{
			command:    `page.locator("#email").fill("qa@example.test")`,
			selectorBy: "css selector",
			selector:   "#email",
			methods:    []string{"Clear", "SendKeys"},
			parameters: [][]interface{}{nil, {"qa@example.test"}},
		},
		{
			command:    `page.locator(".card > button.primary").click()`,
			selectorBy: "css selector",
			selector:   ".card > button.primary",
			methods:    []string{"Click"},
		},
		{
			command:    `page.locator("#search").press("Enter")`,
			selectorBy: "css selector",
			selector:   "#search",
			methods:    []string{"SendKeys"},
			parameters: [][]interface{}{{selenium.EnterKey}},
		},
		{
			command:    `status = page.locator("#status").text()`,
			key:        "status",
			selectorBy: "css selector",
			selector:   "#status",
			methods:    []string{"Text"},
		},
		{
			command:    `page.getByTestId("submit").click()`,
			selectorBy: "css selector",
			selector:   `[data-testid="submit"]`,
			methods:    []string{"Click"},
		},
		{
			command:    `page.getByText("Save changes").click()`,
			selectorBy: "xpath",
			selector:   `//*[normalize-space(.)='Save changes']`,
			methods:    []string{"Click"},
		},
		{
			command:    `page.getByLabel("Email").fill("qa@example.test")`,
			selectorBy: "xpath",
			selector:   `//*[@id = //label[normalize-space(.)='Email']/@for] | //label[normalize-space(.)='Email']//*[self::input or self::textarea or self::select]`,
			methods:    []string{"Clear", "SendKeys"},
			parameters: [][]interface{}{nil, {"qa@example.test"}},
		},
		{
			command:    `page.getByRole("button", "Save").click()`,
			selectorBy: "xpath",
			selector:   `(//*[@role='button'] | //button)[normalize-space(.)='Save' or @aria-label='Save']`,
			methods:    []string{"Click"},
		},
		{
			command:    `page.getByPlaceholder("Search").fill("endly")`,
			selectorBy: "xpath",
			selector:   `//*[@placeholder='Search']`,
			methods:    []string{"Clear", "SendKeys"},
			parameters: [][]interface{}{nil, {"endly"}},
		},
		{
			command: `tabs = page.tabs()`,
			key:     "tabs",
			methods: []string{"Tabs"},
		},
		{
			command:    `page.switchTab("checkout")`,
			methods:    []string{"SwitchTab"},
			parameters: [][]interface{}{{"checkout"}},
		},
		{
			command:    `page.newTab("https://example.test/checkout")`,
			methods:    []string{"NewTab", "Get"},
			parameters: [][]interface{}{{""}, {"https://example.test/checkout"}},
		},
		{
			command: `page.closeTab()`,
			methods: []string{"CloseTab"},
		},
		{
			command:    `page.frame("#payment-frame")`,
			methods:    []string{"SwitchFrameBySelector"},
			parameters: [][]interface{}{{"#payment-frame"}},
		},
		{
			command: `page.acceptDialog()`,
			methods: []string{"AcceptAlert"},
		},
		{
			command: `page.stopLoading()`,
			methods: []string{"StopLoading"},
		},
		{
			command:    `api = page.waitForResponse("/api/orders", 201, 5000)`,
			key:        "api",
			methods:    []string{"WaitForResponse"},
			parameters: [][]interface{}{{"/api/orders", 201, 5000}},
		},
	}

	parser := NewParser()
	for _, testCase := range testCases {
		t.Run(testCase.command, func(t *testing.T) {
			action, err := parser.Parse(testCase.command)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, testCase.key, action.Key)
			assert.Len(t, action.Calls, len(testCase.methods))
			if testCase.selector != "" {
				if assert.NotNil(t, action.Selector) {
					assert.Equal(t, testCase.selectorBy, action.Selector.By)
					assert.Equal(t, testCase.selector, action.Selector.Value)
				}
			} else {
				assert.Nil(t, action.Selector)
			}
			for index, method := range testCase.methods {
				assert.Equal(t, method, action.Calls[index].Method)
				if index < len(testCase.parameters) {
					assert.Equal(t, testCase.parameters[index], action.Calls[index].Parameters)
				}
			}
		})
	}
}

func TestParser_ParsePageCommandErrors(t *testing.T) {
	parser := NewParser()
	for _, command := range []string{
		`page.locator("#email").fill()`,
		`page.goto()`,
		`page.unknown()`,
		`page.locator("#email")`,
	} {
		_, err := parser.Parse(command)
		assert.Error(t, err, command)
	}
}

func TestParser_ParseExpectCommands(t *testing.T) {
	parser := NewParser()
	testCases := []struct {
		command    string
		method     string
		matcher    string
		expected   interface{}
		timeoutMs  int
		parameters []interface{}
	}{
		{`expect(page.locator("#status")).toHaveText("Ready", 2500)`, "Text", "equal", "Ready", 2500, nil},
		{`expect("#status").toContainText("Ready")`, "Text", "contains", "Ready", 10000, nil},
		{`expect("#submit").toBeVisible()`, "IsDisplayed", "equal", true, 10000, nil},
		{`expect("#submit").toBeEnabled(1200)`, "IsEnabled", "equal", true, 1200, nil},
		{`expect("#spinner").toBeHidden(1200)`, "IsDisplayed", "equal", false, 1200, nil},
		{`expect("#submit").toBeDisabled()`, "IsEnabled", "equal", false, 10000, nil},
		{`expect("#email").toHaveValue("qa@example.test")`, "GetAttribute", "equal", "qa@example.test", 10000, []interface{}{"value"}},
		{`expect("#status").toHaveAttribute("data-state", "ready")`, "GetAttribute", "equal", "ready", 10000, []interface{}{"data-state"}},
		{`expect(page.getByTestId("terms")).toBeChecked()`, "IsSelected", "equal", true, 10000, nil},
		{`expect(page.getByTestId("terms")).toBeUnchecked()`, "IsSelected", "equal", false, 10000, nil},
		{`expect(page.getByRole("checkbox", "Terms")).toBeChecked()`, "IsSelected", "equal", true, 10000, nil},
		{`expect(page).toHaveURL("/checkout/", 2500)`, "CurrentURL", "equal", "/checkout/", 2500, nil},
		{`expect(page).toHaveTitle("Checkout")`, "Title", "equal", "Checkout", 10000, nil},
		{`expect(page.locator("css selector:.row")).toHaveCount("3", 1500)`, "ElementCount", "equal", 3, 1500, []interface{}{"css selector", ".row"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.command, func(t *testing.T) {
			action, err := parser.Parse(testCase.command)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Len(t, action.Calls, 1) {
				return
			}
			call := action.Calls[0]
			assert.Equal(t, testCase.method, call.Method)
			assert.Equal(t, testCase.timeoutMs, call.WaitTimeMs)
			assert.Equal(t, testCase.parameters, call.Parameters)
			if assert.NotNil(t, call.Expectation) {
				assert.Equal(t, testCase.matcher, call.Expectation.Matcher)
				assert.Equal(t, testCase.expected, call.Expectation.Value)
			}
		})
	}
}

func TestRunRequest_InitRetainsCommandExpectation(t *testing.T) {
	request := &RunRequest{Commands: []interface{}{map[string]interface{}{
		"command":        `expect("#status").toHaveText("Ready")`,
		"pollIntervalMs": 25,
	}}}
	if err := request.Init(); err != nil {
		t.Fatal(err)
	}
	if !assert.Len(t, request.Actions, 1) || !assert.Len(t, request.Actions[0].Calls, 1) {
		return
	}
	call := request.Actions[0].Calls[0]
	assert.NotNil(t, call.Expectation)
	assert.Equal(t, 25, call.PollIntervalMs)
	assert.Equal(t, 10_000, call.WaitTimeMs)
}
