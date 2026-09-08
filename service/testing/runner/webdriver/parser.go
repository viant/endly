package webdriver

import (
	"fmt"
	"github.com/tebeka/selenium"
	smatcher "github.com/viant/endly/service/testing/runner/webdriver/matcher"
	"github.com/viant/parsly"
	"github.com/viant/parsly/matcher"
	"strconv"
	"strings"
	"unicode"
)

const (
	undefined int = iota
	eof
	illegal
	whitespaces
	id
	assign
	selector
	selectorBy
	params
	dot
	method
)

var whitespaceMatcher = parsly.NewToken(whitespaces, " ", matcher.NewWhiteSpace())
var idMatcher = parsly.NewToken(id, "IDENT", smatcher.NewIdentity())
var assignMatcher = parsly.NewToken(assign, "=", matcher.NewByte('='))
var dotMatcher = parsly.NewToken(dot, ".", matcher.NewByte('.'))
var selectorMatcher = parsly.NewToken(selector, "(...)", matcher.NewBlock('(', ')', '\\'))
var methodMatcher = parsly.NewToken(method, "Method", smatcher.NewLiteral())

// parser represents selenium command action parser
type parser struct{}

// Parse parses supplied expression. It returns criteria or parsing error.
func (p *parser) Parse(command string) (*Action, error) {
	if action, matched, err := parseExpectationCommand(command); matched {
		return action, err
	}
	if action, matched, err := parsePageCommand(command); matched {
		return action, err
	}
	return p.parseLegacy(command)
}

func parseExpectationCommand(command string) (*Action, bool, error) {
	expression := strings.TrimSpace(command)
	if !strings.HasPrefix(strings.ToLower(expression), "expect(") {
		return nil, false, nil
	}
	name, args, rest, err := consumeCall(expression)
	if err != nil {
		return nil, true, err
	}
	if !strings.EqualFold(name, "expect") || len(args) != 1 {
		return nil, true, fmt.Errorf("expect requires one locator")
	}
	selector := args[0]
	if strings.EqualFold(strings.TrimSpace(selector), "page") {
		if !strings.HasPrefix(rest, ".") {
			return nil, true, fmt.Errorf("expect(page) must be followed by a matcher")
		}
		matcher, matcherArgs, suffix, matcherErr := consumeCall(strings.TrimSpace(rest[1:]))
		if matcherErr != nil {
			return nil, true, matcherErr
		}
		if suffix != "" {
			return nil, true, fmt.Errorf("unexpected suffix %q", suffix)
		}
		return pageExpectationAction(matcher, matcherArgs)
	}
	if strings.HasPrefix(strings.ToLower(selector), "page.") {
		locatorName, locatorArgs, locatorRest, locatorErr := consumeCall(strings.TrimSpace(selector[len("page."):]))
		resolved, supported, resolveErr := pageSelector(locatorName, locatorArgs)
		if locatorErr != nil || resolveErr != nil || !supported || locatorRest != "" {
			return nil, true, fmt.Errorf("invalid expect locator %q", selector)
		}
		selector = resolved
	}
	if !strings.HasPrefix(rest, ".") {
		return nil, true, fmt.Errorf("expect locator must be followed by a matcher")
	}
	matcher, matcherArgs, suffix, err := consumeCall(strings.TrimSpace(rest[1:]))
	if err != nil {
		return nil, true, err
	}
	if suffix != "" {
		return nil, true, fmt.Errorf("unexpected suffix %q", suffix)
	}
	return expectationAction(selector, matcher, matcherArgs)
}

func pageExpectationAction(matcher string, args []string) (*Action, bool, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, true, fmt.Errorf("%s expects a value and optional timeout", matcher)
	}
	timeoutMs := 10_000
	if len(args) == 2 {
		parsed, err := strconv.Atoi(args[1])
		if err != nil || parsed <= 0 {
			return nil, true, fmt.Errorf("invalid %s timeout %q", matcher, args[1])
		}
		timeoutMs = parsed
	}
	method := ""
	switch strings.ToLower(matcher) {
	case "tohaveurl":
		method = "CurrentURL"
	case "tohavetitle":
		method = "Title"
	default:
		return nil, true, fmt.Errorf("unsupported page expectation %q", matcher)
	}
	action := NewAction("expect", "", method)
	action.Calls[0].Wait = Wait{
		WaitTimeMs:  timeoutMs,
		Expectation: &CallExpectation{Matcher: "equal", Value: args[0]},
	}
	return action, true, nil
}

func expectationAction(selector, matcher string, args []string) (*Action, bool, error) {
	selector = normalizePageSelector(selector)
	if strings.EqualFold(matcher, "toHaveCount") {
		if len(args) < 1 || len(args) > 2 {
			return nil, true, fmt.Errorf("toHaveCount expects a count and optional timeout")
		}
		expected, err := strconv.Atoi(args[0])
		if err != nil || expected < 0 {
			return nil, true, fmt.Errorf("invalid toHaveCount value %q", args[0])
		}
		timeoutMs := 10_000
		if len(args) == 2 {
			timeoutMs, err = strconv.Atoi(args[1])
			if err != nil || timeoutMs <= 0 {
				return nil, true, fmt.Errorf("invalid toHaveCount timeout %q", args[1])
			}
		}
		by, value := WebSelector(selector).ByAndValue()
		action := NewAction("expect", "", "ElementCount", by, value)
		action.Calls[0].Wait = Wait{WaitTimeMs: timeoutMs, Expectation: &CallExpectation{Matcher: "equal", Value: expected}}
		return action, true, nil
	}
	action := NewAction("expect", selector, "")
	action.Strict = true
	action.Selector.Key = "expect"
	timeoutMs := 10_000
	parseTimeout := func(index int) error {
		if len(args) <= index {
			return nil
		}
		parsed, err := strconv.Atoi(args[index])
		if err != nil || parsed <= 0 {
			return fmt.Errorf("invalid %s timeout %q", matcher, args[index])
		}
		timeoutMs = parsed
		return nil
	}
	call := &MethodCall{Wait: Wait{WaitTimeMs: timeoutMs}}
	switch strings.ToLower(matcher) {
	case "tobevisible", "tobehidden":
		if len(args) > 1 {
			return nil, true, fmt.Errorf("%s accepts an optional timeout", matcher)
		}
		if err := parseTimeout(0); err != nil {
			return nil, true, err
		}
		call.Method = "IsDisplayed"
		visible := !strings.EqualFold(matcher, "toBeHidden")
		call.Expectation = &CallExpectation{Matcher: "equal", Value: visible}
		call.AllowMissing = !visible
	case "tobeenabled", "tobedisabled", "tobechecked", "tobeunchecked":
		if len(args) > 1 {
			return nil, true, fmt.Errorf("%s accepts an optional timeout", matcher)
		}
		if err := parseTimeout(0); err != nil {
			return nil, true, err
		}
		call.Method = "IsEnabled"
		expected := !strings.EqualFold(matcher, "toBeDisabled")
		if strings.EqualFold(matcher, "toBeChecked") || strings.EqualFold(matcher, "toBeUnchecked") {
			call.Method = "IsSelected"
			expected = !strings.EqualFold(matcher, "toBeUnchecked")
		}
		call.Expectation = &CallExpectation{Matcher: "equal", Value: expected}
	case "tohavetext", "tocontaintext", "tohavevalue":
		if len(args) < 1 || len(args) > 2 {
			return nil, true, fmt.Errorf("%s expects a value and optional timeout", matcher)
		}
		if err := parseTimeout(1); err != nil {
			return nil, true, err
		}
		call.Method = "Text"
		if strings.EqualFold(matcher, "toHaveValue") {
			call.Method = "GetAttribute"
			call.Parameters = []interface{}{"value"}
		}
		matchKind := "equal"
		if strings.EqualFold(matcher, "toContainText") {
			matchKind = "contains"
		}
		call.Expectation = &CallExpectation{Matcher: matchKind, Value: args[0]}
	case "tohaveattribute":
		if len(args) < 2 || len(args) > 3 {
			return nil, true, fmt.Errorf("toHaveAttribute expects name, value, and optional timeout")
		}
		if err := parseTimeout(2); err != nil {
			return nil, true, err
		}
		call.Method = "GetAttribute"
		call.Parameters = []interface{}{args[0]}
		call.Expectation = &CallExpectation{Matcher: "equal", Value: args[1]}
	default:
		return nil, true, fmt.Errorf("unsupported expectation %q", matcher)
	}
	call.WaitTimeMs = timeoutMs
	action.Calls = []*MethodCall{call}
	return action, true, nil
}

func (p *parser) parseLegacy(command string) (*Action, error) {
	result := &Action{
		PathKind: PathKindSimple,
		Calls:    []*MethodCall{{}},
	}
	cursor := parsly.NewCursor("", []byte(command), 0)
	var call = result.Calls[0]
	var webSelector WebSelector
	var callParams = ""

	expectTokens := []*parsly.Token{selectorMatcher, methodMatcher, idMatcher}

outer:
	for {

		match := cursor.MatchAfterOptional(whitespaceMatcher, expectTokens...)

		switch match.Token.Code {

		case selector:
			matched := match.Text(cursor)
			webSelector = WebSelector(matched[1 : len(matched)-1])
			match = cursor.MatchAfterOptional(whitespaceMatcher, dotMatcher)
			if match.Token.Code != dot {
				return nil, cursor.NewError(dotMatcher)
			}
			match = cursor.MatchAfterOptional(whitespaceMatcher, methodMatcher, idMatcher)
			switch match.Token.Code {
			case method:
				matched = match.Text(cursor)
				index := strings.Index(matched, "(")
				call.Method = matched[:index]
				callParams = strings.Trim(matched[index+1:len(matched)-1], `'`)
				if len(callParams) > 0 {
					call.Parameters = []interface{}{callParams}
				}
			case id:
				call.Method = match.Text(cursor)
			default:
				return nil, cursor.NewError(methodMatcher, idMatcher)
			}
		case method:
			matched := match.Text(cursor)
			index := strings.Index(matched, "(")
			call.Method = matched[:index]
			callParams = strings.Trim(matched[index+1:len(matched)-1], `'`)
			if len(callParams) > 0 {
				call.Parameters = []interface{}{callParams}
			}

		case id:
			matched := match.Text(cursor)
			match = cursor.MatchAfterOptional(whitespaceMatcher, assignMatcher)
			if match.Token.Code == assign {
				result.Key = matched
			} else {
				call.Method = matched
			}
		case parsly.EOF:
			break outer
		default:
			return nil, cursor.NewError(expectTokens...)
		}
	}

	if len(webSelector) > 0 {
		result.Selector = &WebElementSelector{}
		result.Selector.By, result.Selector.Value = webSelector.ByAndValue()
		result.Selector.Key = result.Key
	}

	method := call.Method
	if len(method) > 1 {
		call.Method = strings.ToUpper(string(method[0])) + string(method[1:])
	}
	return result, nil
}

// parsePageCommand accepts a compact, Playwright-inspired DSL while retaining
// the original Endly command syntax. Supported examples include:
//
//	page.goto("https://example.test")
//	page.locator("#email").fill("qa@example.test")
//	status = page.locator("#status").text()
//	page.getByTestId("submit").click()
func parsePageCommand(command string) (*Action, bool, error) {
	expression := strings.TrimSpace(command)
	key := ""
	if index := topLevelAssignment(expression); index >= 0 {
		key = strings.TrimSpace(expression[:index])
		if !isIdentifier(key) {
			return nil, true, fmt.Errorf("invalid result key %q", key)
		}
		expression = strings.TrimSpace(expression[index+1:])
	}
	if !strings.HasPrefix(strings.ToLower(expression), "page.") {
		return nil, false, nil
	}
	expression = strings.TrimSpace(expression[len("page."):])
	name, args, rest, err := consumeCall(expression)
	if err != nil {
		return nil, true, err
	}

	selector, isSelector, selectorErr := pageSelector(name, args)
	if selectorErr != nil {
		return nil, true, selectorErr
	}
	if !isSelector {
		if rest != "" {
			return nil, true, fmt.Errorf("unexpected suffix %q", rest)
		}
		return pageAction(key, name, args)
	}

	if !strings.HasPrefix(rest, ".") {
		return nil, true, fmt.Errorf("page.%s must be followed by an action", name)
	}
	methodName, methodArgs, rest, err := consumeCall(strings.TrimSpace(rest[1:]))
	if err != nil {
		return nil, true, err
	}
	if strings.TrimSpace(rest) != "" {
		return nil, true, fmt.Errorf("unexpected suffix %q", rest)
	}
	return locatorAction(key, selector, methodName, methodArgs)
}

func pageSelector(name string, args []string) (string, bool, error) {
	switch strings.ToLower(name) {
	case "locator":
		if len(args) != 1 {
			return "", true, fmt.Errorf("page.locator expects one selector")
		}
		return normalizePageSelector(args[0]), true, nil
	case "getbytestid":
		if len(args) != 1 {
			return "", true, fmt.Errorf("page.getByTestId expects one value")
		}
		return "css selector:[data-testid=" + strconv.Quote(args[0]) + "]", true, nil
	case "getbytext":
		if len(args) != 1 {
			return "", true, fmt.Errorf("page.getByText expects one value")
		}
		return "xpath://*[normalize-space(.)=" + xpathLiteral(args[0]) + "]", true, nil
	case "getbylabel":
		if len(args) != 1 {
			return "", true, fmt.Errorf("page.getByLabel expects one value")
		}
		label := xpathLiteral(args[0])
		return "xpath://*[@id = //label[normalize-space(.)=" + label + "]/@for] | //label[normalize-space(.)=" + label + "]//*[self::input or self::textarea or self::select]", true, nil
	case "getbyplaceholder":
		if len(args) != 1 {
			return "", true, fmt.Errorf("page.getByPlaceholder expects one value")
		}
		return "xpath://*[@placeholder=" + xpathLiteral(args[0]) + "]", true, nil
	case "getbyalttext":
		if len(args) != 1 {
			return "", true, fmt.Errorf("page.getByAltText expects one value")
		}
		return "xpath://*[@alt=" + xpathLiteral(args[0]) + "]", true, nil
	case "getbytitle":
		if len(args) != 1 {
			return "", true, fmt.Errorf("page.getByTitle expects one value")
		}
		return "xpath://*[@title=" + xpathLiteral(args[0]) + "]", true, nil
	case "getbyrole":
		if len(args) < 1 || len(args) > 2 {
			return "", true, fmt.Errorf("page.getByRole expects a role and optional accessible name")
		}
		role := strings.ToLower(args[0])
		base := "//*[@role=" + xpathLiteral(role) + "]"
		if tag := nativeRoleTag(role); tag != "" {
			base = "(" + base + " | //" + tag + ")"
		}
		if len(args) == 2 {
			name := xpathLiteral(args[1])
			base += "[normalize-space(.)=" + name + " or @aria-label=" + name + "]"
		}
		return "xpath:" + base, true, nil
	default:
		return "", false, nil
	}
}

func normalizePageSelector(selector string) string {
	selector = strings.TrimSpace(selector)
	lower := strings.ToLower(selector)
	for candidate := range selectors {
		if strings.HasPrefix(lower, strings.ToLower(candidate)+":") {
			return selector
		}
	}
	if strings.HasPrefix(selector, "//") || strings.HasPrefix(selector, "(") {
		return "xpath:" + selector
	}
	return selenium.ByCSSSelector + ":" + selector
}

func nativeRoleTag(role string) string {
	switch role {
	case "button":
		return "button"
	case "link":
		return "a"
	case "heading":
		return "h1 | //h2 | //h3 | //h4 | //h5 | //h6"
	case "textbox":
		return "input | //textarea"
	case "checkbox":
		return "input[@type='checkbox']"
	case "radio":
		return "input[@type='radio']"
	case "option":
		return "option"
	default:
		return ""
	}
}

func pageAction(key, method string, args []string) (*Action, bool, error) {
	switch strings.ToLower(method) {
	case "goto":
		if len(args) != 1 {
			return nil, true, fmt.Errorf("page.goto expects one URL")
		}
		return NewAction(key, "", "Get", args[0]), true, nil
	case "reload":
		return noArgumentPageAction(key, method, args, "Refresh")
	case "back", "goback":
		return noArgumentPageAction(key, method, args, "Back")
	case "forward", "goforward":
		return noArgumentPageAction(key, method, args, "Forward")
	case "tabs":
		return noArgumentPageAction(key, method, args, "Tabs")
	case "currenttab":
		return noArgumentPageAction(key, method, args, "CurrentWindowHandle")
	case "url":
		return noArgumentPageAction(key, method, args, "CurrentURL")
	case "title":
		return noArgumentPageAction(key, method, args, "Title")
	case "closetab":
		return noArgumentPageAction(key, method, args, "CloseTab")
	case "acceptdialog":
		return noArgumentPageAction(key, method, args, "AcceptAlert")
	case "dismissdialog":
		return noArgumentPageAction(key, method, args, "DismissAlert")
	case "dialogtext":
		return noArgumentPageAction(key, method, args, "AlertText")
	case "setdialogtext":
		if len(args) != 1 {
			return nil, true, fmt.Errorf("page.setDialogText expects one value")
		}
		return NewAction(key, "", "SetAlertText", args[0]), true, nil
	case "frame":
		if len(args) != 1 {
			return nil, true, fmt.Errorf("page.frame expects one selector")
		}
		return NewAction(key, "", "SwitchFrameBySelector", args[0]), true, nil
	case "mainframe":
		return noArgumentPageAction(key, method, args, "MainFrame")
	case "stoploading":
		return noArgumentPageAction(key, method, args, "StopLoading")
	case "waitforresponse":
		if len(args) < 1 || len(args) > 3 {
			return nil, true, fmt.Errorf("page.waitForResponse expects URL pattern, optional status, and optional timeout")
		}
		status := 0
		timeoutMs := 10_000
		var err error
		if len(args) >= 2 && strings.TrimSpace(args[1]) != "" {
			status, err = strconv.Atoi(args[1])
			if err != nil || status < 0 {
				return nil, true, fmt.Errorf("invalid response status %q", args[1])
			}
		}
		if len(args) == 3 {
			timeoutMs, err = strconv.Atoi(args[2])
			if err != nil || timeoutMs <= 0 {
				return nil, true, fmt.Errorf("invalid response timeout %q", args[2])
			}
		}
		return NewAction(key, "", "WaitForResponse", args[0], status, timeoutMs), true, nil
	case "switchtab":
		if len(args) != 1 {
			return nil, true, fmt.Errorf("page.switchTab expects a handle, URL, or title fragment")
		}
		return NewAction(key, "", "SwitchTab", args[0]), true, nil
	case "newtab":
		if len(args) > 1 {
			return nil, true, fmt.Errorf("page.newTab accepts an optional URL")
		}
		action := NewAction(key, "", "NewTab", "")
		if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
			action.Calls = append(action.Calls, &MethodCall{Method: "Get", Parameters: []interface{}{args[0]}})
		}
		return action, true, nil
	case "click", "fill", "type", "press", "text", "textcontent", "inputvalue", "attribute", "getattribute", "clear", "submit", "waitforvisible", "check", "uncheck", "selectoption", "hover", "setinputfiles":
		if len(args) == 0 {
			return nil, true, fmt.Errorf("page.%s expects a selector", method)
		}
		return locatorAction(key, args[0], method, args[1:])
	default:
		return nil, true, fmt.Errorf("unsupported page action %q", method)
	}
}

func noArgumentPageAction(key, source string, args []string, target string) (*Action, bool, error) {
	if len(args) != 0 {
		return nil, true, fmt.Errorf("page.%s does not accept arguments", source)
	}
	return NewAction(key, "", target), true, nil
}

func locatorAction(key, selector, method string, args []string) (*Action, bool, error) {
	if strings.TrimSpace(selector) == "" {
		return nil, true, fmt.Errorf("selector was empty")
	}
	selector = normalizePageSelector(selector)
	action := NewAction(key, selector, "")
	action.Strict = true
	if action.Selector != nil && action.Selector.Key == "" {
		action.Selector.Key = action.Selector.Value
	}
	newCall := func(name string, params ...interface{}) *MethodCall {
		return &MethodCall{Method: name, Parameters: params}
	}
	switch strings.ToLower(method) {
	case "click":
		action.Calls = []*MethodCall{newCall("Click")}
	case "fill":
		if len(args) != 1 {
			return nil, true, fmt.Errorf("fill expects one value")
		}
		action.Calls = []*MethodCall{newCall("Clear"), newCall("SendKeys", args[0])}
	case "type", "press":
		if len(args) != 1 {
			return nil, true, fmt.Errorf("%s expects one value", method)
		}
		value := args[0]
		if strings.EqualFold(method, "press") {
			value = webdriverKey(value)
		}
		action.Calls = []*MethodCall{newCall("SendKeys", value)}
	case "text", "textcontent":
		action.Calls = []*MethodCall{newCall("Text")}
	case "inputvalue":
		action.Calls = []*MethodCall{newCall("GetAttribute", "value")}
	case "attribute", "getattribute":
		if len(args) != 1 {
			return nil, true, fmt.Errorf("%s expects one attribute name", method)
		}
		action.Calls = []*MethodCall{newCall("GetAttribute", args[0])}
	case "clear":
		action.Calls = []*MethodCall{newCall("Clear")}
	case "submit":
		action.Calls = []*MethodCall{newCall("Submit")}
	case "check":
		action.Calls = []*MethodCall{newCall("Check")}
	case "uncheck":
		action.Calls = []*MethodCall{newCall("Uncheck")}
	case "selectoption":
		if len(args) != 1 {
			return nil, true, fmt.Errorf("selectOption expects one value or visible label")
		}
		action.Calls = []*MethodCall{newCall("SelectOption", args[0])}
	case "hover":
		action.Calls = []*MethodCall{newCall("MoveTo", 0, 0)}
	case "setinputfiles":
		if len(args) != 1 {
			return nil, true, fmt.Errorf("setInputFiles expects one path")
		}
		action.Calls = []*MethodCall{newCall("SendKeys", args[0])}
	case "waitforvisible":
		timeoutMs := 10_000
		if len(args) > 1 {
			return nil, true, fmt.Errorf("waitForVisible accepts an optional timeout in milliseconds")
		}
		if len(args) == 1 {
			parsed, err := strconv.Atoi(args[0])
			if err != nil || parsed <= 0 {
				return nil, true, fmt.Errorf("invalid waitForVisible timeout %q", args[0])
			}
			timeoutMs = parsed
		}
		if action.Key == "" {
			action.Key = "visible"
			action.Selector.Key = "visible"
		}
		call := newCall("IsDisplayed")
		call.WaitTimeMs = timeoutMs
		call.Exit = "$" + action.Selector.Key + " = true"
		action.Calls = []*MethodCall{call}
	default:
		return nil, true, fmt.Errorf("unsupported locator action %q", method)
	}
	return action, true, nil
}

func webdriverKey(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "enter":
		return selenium.EnterKey
	case "escape", "esc":
		return selenium.EscapeKey
	case "tab":
		return selenium.TabKey
	case "space":
		return selenium.SpaceKey
	case "backspace":
		return selenium.BackspaceKey
	case "delete":
		return selenium.DeleteKey
	case "arrowleft", "left":
		return selenium.LeftArrowKey
	case "arrowright", "right":
		return selenium.RightArrowKey
	case "arrowup", "up":
		return selenium.UpArrowKey
	case "arrowdown", "down":
		return selenium.DownArrowKey
	case "home":
		return selenium.HomeKey
	case "end":
		return selenium.EndKey
	case "pageup":
		return selenium.PageUpKey
	case "pagedown":
		return selenium.PageDownKey
	default:
		return value
	}
}

func consumeCall(expression string) (string, []string, string, error) {
	expression = strings.TrimSpace(expression)
	open := strings.IndexByte(expression, '(')
	if open <= 0 {
		return "", nil, "", fmt.Errorf("expected function call in %q", expression)
	}
	name := strings.TrimSpace(expression[:open])
	if !isIdentifier(name) {
		return "", nil, "", fmt.Errorf("invalid action name %q", name)
	}
	quote := rune(0)
	escaped := false
	depth := 0
	for index, char := range expression[open:] {
		absolute := open + index
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		switch char {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				args, err := splitArguments(expression[open+1 : absolute])
				return name, args, strings.TrimSpace(expression[absolute+1:]), err
			}
		}
	}
	return "", nil, "", fmt.Errorf("unterminated call in %q", expression)
}

func splitArguments(source string) ([]string, error) {
	if strings.TrimSpace(source) == "" {
		return nil, nil
	}
	result := make([]string, 0, 2)
	start := 0
	quote := rune(0)
	escaped := false
	depth := 0
	for index, char := range source {
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		switch char {
		case '(':
			depth++
		case ')':
			depth--
		}
		if char == ',' && depth == 0 {
			value, err := normalizeArgument(source[start:index])
			if err != nil {
				return nil, err
			}
			result = append(result, value)
			start = index + 1
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted argument")
	}
	value, err := normalizeArgument(source[start:])
	if err != nil {
		return nil, err
	}
	return append(result, value), nil
}

func normalizeArgument(source string) (string, error) {
	value := strings.TrimSpace(source)
	if len(value) < 2 || (value[0] != '\'' && value[0] != '"') {
		return value, nil
	}
	if value[len(value)-1] != value[0] {
		return "", fmt.Errorf("unterminated quoted argument %q", value)
	}
	if value[0] == '"' {
		return strconv.Unquote(value)
	}
	value = strings.ReplaceAll(value[1:len(value)-1], `\'`, `'`)
	value = strings.ReplaceAll(value, `\\`, `\`)
	return value, nil
}

func topLevelAssignment(source string) int {
	quote := rune(0)
	escaped := false
	depth := 0
	for index, char := range source {
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		switch char {
		case '(':
			depth++
		case ')':
			depth--
		case '=':
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		if !(unicode.IsLetter(char) || char == '_' || (index > 0 && unicode.IsDigit(char))) {
			return false
		}
	}
	return true
}

func xpathLiteral(value string) string {
	if !strings.Contains(value, "'") {
		return "'" + value + "'"
	}
	if !strings.Contains(value, `"`) {
		return `"` + value + `"`
	}
	parts := strings.Split(value, "'")
	quoted := make([]string, 0, len(parts)*2-1)
	for index, part := range parts {
		if index > 0 {
			quoted = append(quoted, `"'"`)
		}
		quoted = append(quoted, "'"+part+"'")
	}
	return "concat(" + strings.Join(quoted, ",") + ")"
}

// NewParser creates a new criteria parser
func NewParser() *parser {
	return &parser{}
}

type illegalTokenParsingError struct {
	Index    int
	Expected string
	error    string
}

func (e illegalTokenParsingError) Error() string {
	return e.error
}

func newIllegalTokenParsingError(index, token int, expected string) error {
	return &illegalTokenParsingError{Index: index, Expected: expected, error: fmt.Sprintf("illegal token:%v at %v, expected %v", token, index, expected)}
}
