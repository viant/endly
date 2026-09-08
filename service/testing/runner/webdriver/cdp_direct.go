package webdriver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tebeka/selenium"
	selog "github.com/tebeka/selenium/log"
)

type directCDPTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type directCDPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type directCDPEnvelope struct {
	ID     int64           `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *directCDPError `json:"error,omitempty"`
}

type directCDPConnection struct {
	conn        *websocket.Conn
	sequence    atomic.Int64
	writeMu     sync.Mutex
	mu          sync.Mutex
	pending     map[int64]chan *directCDPEnvelope
	performance []selog.Message
	browser     []selog.Message
	dialogText  string
	closed      chan struct{}
	closeOnce   sync.Once
}

func newDirectCDPConnection(websocketURL string) (*directCDPConnection, error) {
	conn, _, err := websocket.DefaultDialer.Dial(websocketURL, nil)
	if err != nil {
		return nil, err
	}
	result := &directCDPConnection{
		conn:    conn,
		pending: make(map[int64]chan *directCDPEnvelope),
		closed:  make(chan struct{}),
	}
	go result.readLoop()
	for _, method := range []string{"Page.enable", "Runtime.enable", "Network.enable"} {
		_, _ = result.command(context.Background(), method, map[string]interface{}{})
	}
	return result, nil
}

func (c *directCDPConnection) command(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}
	id := c.sequence.Add(1)
	response := make(chan *directCDPEnvelope, 1)
	c.mu.Lock()
	c.pending[id] = response
	c.mu.Unlock()
	c.writeMu.Lock()
	err := c.conn.WriteJSON(map[string]interface{}{"id": id, "method": method, "params": params})
	c.writeMu.Unlock()
	if err != nil {
		c.removePending(id)
		return nil, err
	}
	select {
	case envelope := <-response:
		if envelope == nil {
			return nil, errors.New("CDP connection closed")
		}
		if envelope.Error != nil {
			return nil, fmt.Errorf("CDP %s failed (%d): %s", method, envelope.Error.Code, envelope.Error.Message)
		}
		return envelope.Result, nil
	case <-ctx.Done():
		c.removePending(id)
		return nil, ctx.Err()
	case <-c.closed:
		c.removePending(id)
		return nil, errors.New("CDP connection closed")
	}
}

func (c *directCDPConnection) removePending(id int64) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *directCDPConnection) readLoop() {
	defer c.close()
	for {
		envelope := &directCDPEnvelope{}
		if err := c.conn.ReadJSON(envelope); err != nil {
			return
		}
		if envelope.ID != 0 {
			c.mu.Lock()
			response := c.pending[envelope.ID]
			delete(c.pending, envelope.ID)
			c.mu.Unlock()
			if response != nil {
				response <- envelope
			}
			continue
		}
		if envelope.Method != "" {
			c.appendEvent(envelope)
		}
	}
}

func (c *directCDPConnection) appendEvent(envelope *directCDPEnvelope) {
	now := time.Now()
	inner, _ := json.Marshal(map[string]interface{}{
		"message": map[string]interface{}{"method": envelope.Method, "params": json.RawMessage(envelope.Params)},
	})
	c.mu.Lock()
	if envelope.Method != "Runtime.consoleAPICalled" {
		c.performance = append(c.performance, selog.Message{Timestamp: now, Level: selog.Info, Message: string(inner)})
		if len(c.performance) > 10_000 {
			c.performance = append([]selog.Message(nil), c.performance[len(c.performance)-10_000:]...)
		}
	}
	if envelope.Method == "Runtime.consoleAPICalled" {
		var params struct {
			Type string `json:"type"`
			Args []struct {
				Value       interface{} `json:"value"`
				Description string      `json:"description"`
			} `json:"args"`
		}
		if json.Unmarshal(envelope.Params, &params) == nil {
			parts := make([]string, 0, len(params.Args))
			for _, argument := range params.Args {
				if argument.Value != nil {
					parts = append(parts, fmt.Sprint(argument.Value))
				} else if argument.Description != "" {
					parts = append(parts, argument.Description)
				}
			}
			c.browser = append(c.browser, selog.Message{Timestamp: now, Level: selog.Info, Message: strings.Join(parts, " ")})
			if len(c.browser) > 10_000 {
				c.browser = append([]selog.Message(nil), c.browser[len(c.browser)-10_000:]...)
			}
		}
	}
	if envelope.Method == "Page.javascriptDialogOpening" {
		var params struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(envelope.Params, &params) == nil {
			c.dialogText = params.Message
		}
	}
	c.mu.Unlock()
}

func (c *directCDPConnection) logs(kind selog.Type) []selog.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result []selog.Message
	switch kind {
	case selog.Browser:
		result = append(result, c.browser...)
		c.browser = nil
	case selog.Performance:
		result = append(result, c.performance...)
		c.performance = nil
	}
	return result
}

func (c *directCDPConnection) close() {
	c.closeOnce.Do(func() {
		_ = c.conn.Close()
		close(c.closed)
		c.mu.Lock()
		for id, response := range c.pending {
			delete(c.pending, id)
			response <- nil
		}
		c.mu.Unlock()
	})
}

type directCDPDriver struct {
	selenium.WebDriver
	address          string
	client           *http.Client
	mu               sync.Mutex
	targets          map[string]*directCDPTarget
	connections      map[string]*directCDPConnection
	current          string
	pageLoadTimeout  time.Duration
	pageLoadStrategy string
	frameRoot        string
	navigationStops  atomic.Int64
}

func newDirectCDPDriver(address string) (*directCDPDriver, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, errors.New("debuggerAddress was empty")
	}
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	address = strings.TrimRight(address, "/")
	result := &directCDPDriver{
		address:         address,
		client:          &http.Client{Timeout: 5 * time.Second},
		targets:         make(map[string]*directCDPTarget),
		connections:     make(map[string]*directCDPConnection),
		pageLoadTimeout: 45 * time.Second,
	}
	if err := result.refreshTargets(); err != nil {
		return nil, err
	}
	if result.current == "" {
		return nil, errors.New("no controllable page targets were found")
	}
	if _, err := result.currentConnection(); err != nil {
		return nil, err
	}
	return result, nil
}

func (d *directCDPDriver) refreshTargets() error {
	request, err := http.NewRequest(http.MethodGet, d.address+"/json/list", nil)
	if err != nil {
		return err
	}
	response, err := d.client.Do(request)
	if err != nil {
		return fmt.Errorf("list Chrome debug targets: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("list Chrome debug targets: %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var targets []*directCDPTarget
	if err := json.NewDecoder(response.Body).Decode(&targets); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.targets = make(map[string]*directCDPTarget)
	for _, target := range targets {
		if target == nil || target.Type != "page" || target.WebSocketDebuggerURL == "" {
			continue
		}
		d.targets[target.ID] = target
		if d.current == "" {
			d.current = target.ID
		}
	}
	if _, ok := d.targets[d.current]; !ok {
		d.current = ""
		for id := range d.targets {
			d.current = id
			break
		}
	}
	return nil
}

func (d *directCDPDriver) currentConnection() (*directCDPConnection, error) {
	d.mu.Lock()
	target := d.targets[d.current]
	connection := d.connections[d.current]
	d.mu.Unlock()
	if target == nil {
		return nil, errors.New("current Chrome target was not found")
	}
	if connection != nil {
		return connection, nil
	}
	created, err := newDirectCDPConnection(target.WebSocketDebuggerURL)
	if err != nil {
		return nil, err
	}
	d.mu.Lock()
	if existing := d.connections[target.ID]; existing != nil {
		d.mu.Unlock()
		created.close()
		return existing, nil
	}
	d.connections[target.ID] = created
	d.mu.Unlock()
	return created, nil
}

func (d *directCDPDriver) evaluate(script string, args []interface{}) (interface{}, error) {
	connection, err := d.currentConnection()
	if err != nil {
		return nil, err
	}
	argumentExpressions := make([]string, len(args))
	for index, argument := range args {
		switch actual := argument.(type) {
		case *directCDPElement:
			argumentExpressions[index] = actual.nodeExpression()
		default:
			encoded, encodeErr := json.Marshal(argument)
			if encodeErr != nil {
				return nil, encodeErr
			}
			argumentExpressions[index] = string(encoded)
		}
	}
	expression := `(function(){const __args=[` + strings.Join(argumentExpressions, ",") + `];return (function(){` + script + `}).apply(null,__args);})()`
	raw, err := connection.command(context.Background(), "Runtime.evaluate", map[string]interface{}{
		"expression": expression, "returnByValue": true, "awaitPromise": true,
	})
	if err != nil {
		return nil, err
	}
	var result struct {
		Result struct {
			Type        string      `json:"type"`
			Subtype     string      `json:"subtype"`
			Value       interface{} `json:"value"`
			Description string      `json:"description"`
		} `json:"result"`
		Exception *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.Exception != nil {
		return nil, errors.New(result.Exception.Text)
	}
	if result.Result.Subtype == "null" || result.Result.Type == "undefined" {
		return nil, nil
	}
	return result.Result.Value, nil
}

func (d *directCDPDriver) objectID(expression string) (string, error) {
	connection, err := d.currentConnection()
	if err != nil {
		return "", err
	}
	raw, err := connection.command(context.Background(), "Runtime.evaluate", map[string]interface{}{
		"expression": expression, "returnByValue": false,
	})
	if err != nil {
		return "", err
	}
	var result struct {
		Result struct {
			ObjectID string `json:"objectId"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.Result.ObjectID == "" {
		return "", errors.New("CDP element object ID was empty")
	}
	return result.Result.ObjectID, nil
}

func (d *directCDPDriver) Status() (*selenium.Status, error) {
	return &selenium.Status{Ready: true, Message: "Endly direct CDP"}, nil
}
func (d *directCDPDriver) SessionID() string { d.mu.Lock(); defer d.mu.Unlock(); return d.current }
func (d *directCDPDriver) SessionId() string { return d.SessionID() }
func (d *directCDPDriver) Capabilities() (selenium.Capabilities, error) {
	return selenium.Capabilities{"browserName": ChromeBrowser, "endly:backend": "cdp"}, nil
}
func (d *directCDPDriver) SetPageLoadTimeout(timeout time.Duration) error {
	d.mu.Lock()
	d.pageLoadTimeout = timeout
	d.mu.Unlock()
	return nil
}
func (d *directCDPDriver) SetAsyncScriptTimeout(time.Duration) error  { return nil }
func (d *directCDPDriver) SetImplicitWaitTimeout(time.Duration) error { return nil }
func (d *directCDPDriver) CurrentWindowHandle() (string, error)       { return d.SessionID(), nil }

func (d *directCDPDriver) WindowHandles() ([]string, error) {
	if err := d.refreshTargets(); err != nil {
		return nil, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	result := make([]string, 0, len(d.targets))
	for id := range d.targets {
		result = append(result, id)
	}
	return result, nil
}

func (d *directCDPDriver) SwitchWindow(name string) error {
	if err := d.refreshTargets(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.targets[name]; !ok {
		return fmt.Errorf("Chrome target %q was not found", name)
	}
	d.current = name
	d.frameRoot = ""
	return nil
}

func (d *directCDPDriver) CurrentURL() (string, error) {
	value, err := d.evaluate("return window.location.href;", nil)
	return fmt.Sprint(value), err
}
func (d *directCDPDriver) Title() (string, error) {
	value, err := d.evaluate("return document.title;", nil)
	return fmt.Sprint(value), err
}
func (d *directCDPDriver) PageSource() (string, error) {
	value, err := d.evaluate("return document.documentElement.outerHTML;", nil)
	return fmt.Sprint(value), err
}

func (d *directCDPDriver) Get(URL string) error {
	connection, err := d.currentConnection()
	if err != nil {
		return err
	}
	if _, err := connection.command(context.Background(), "Page.navigate", map[string]interface{}{"url": URL}); err != nil {
		return err
	}
	d.mu.Lock()
	timeout := d.pageLoadTimeout
	strategy := strings.ToLower(d.pageLoadStrategy)
	d.mu.Unlock()
	if strategy == "none" {
		return nil
	}
	deadline := time.Now().Add(timeout)
	stopSequence := d.navigationStops.Load()
	for time.Now().Before(deadline) {
		if d.navigationStops.Load() != stopSequence {
			return context.Canceled
		}
		state, stateErr := d.evaluate("return document.readyState;", nil)
		if stateErr == nil {
			readyState := fmt.Sprint(state)
			if readyState == "complete" || strategy == "eager" && readyState == "interactive" {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return &selenium.Error{Err: "timeout", Message: fmt.Sprintf("page load exceeded %s", timeout), LegacyCode: 21}
}

func (d *directCDPDriver) StopLoading() error {
	connection, err := d.currentConnection()
	if err != nil {
		return err
	}
	_, err = connection.command(context.Background(), "Page.stopLoading", map[string]interface{}{})
	if err == nil {
		d.navigationStops.Add(1)
	}
	return err
}

func (d *directCDPDriver) CDPCommand(method string, params map[string]interface{}) (json.RawMessage, error) {
	connection, err := d.currentConnection()
	if err != nil {
		return nil, err
	}
	return connection.command(context.Background(), method, params)
}

func (d *directCDPDriver) Back() error {
	_, err := d.evaluate("history.back(); return true;", nil)
	return err
}
func (d *directCDPDriver) Forward() error {
	_, err := d.evaluate("history.forward(); return true;", nil)
	return err
}
func (d *directCDPDriver) Refresh() error {
	connection, err := d.currentConnection()
	if err != nil {
		return err
	}
	_, err = connection.command(context.Background(), "Page.reload", map[string]interface{}{})
	return err
}

func (d *directCDPDriver) ExecuteScript(script string, args []interface{}) (interface{}, error) {
	return d.evaluate(script, args)
}

func (d *directCDPDriver) Screenshot() ([]byte, error) {
	connection, err := d.currentConnection()
	if err != nil {
		return nil, err
	}
	raw, err := connection.command(context.Background(), "Page.captureScreenshot", map[string]interface{}{"format": "png"})
	if err != nil {
		return nil, err
	}
	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(result.Data)
}

func (d *directCDPDriver) Log(kind selog.Type) ([]selog.Message, error) {
	connection, err := d.currentConnection()
	if err != nil {
		return nil, err
	}
	return connection.logs(kind), nil
}

func (d *directCDPDriver) AcceptAlert() error {
	connection, err := d.currentConnection()
	if err != nil {
		return err
	}
	_, err = connection.command(context.Background(), "Page.handleJavaScriptDialog", map[string]interface{}{"accept": true})
	return err
}
func (d *directCDPDriver) DismissAlert() error {
	connection, err := d.currentConnection()
	if err != nil {
		return err
	}
	_, err = connection.command(context.Background(), "Page.handleJavaScriptDialog", map[string]interface{}{"accept": false})
	return err
}
func (d *directCDPDriver) AlertText() (string, error) {
	connection, err := d.currentConnection()
	if err != nil {
		return "", err
	}
	connection.mu.Lock()
	defer connection.mu.Unlock()
	return connection.dialogText, nil
}
func (d *directCDPDriver) SetAlertText(text string) error {
	connection, err := d.currentConnection()
	if err != nil {
		return err
	}
	_, err = connection.command(context.Background(), "Page.handleJavaScriptDialog", map[string]interface{}{"accept": true, "promptText": text})
	return err
}

func (d *directCDPDriver) GetCookies() ([]selenium.Cookie, error) {
	connection, err := d.currentConnection()
	if err != nil {
		return nil, err
	}
	raw, err := connection.command(context.Background(), "Network.getAllCookies", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var result struct {
		Cookies []struct {
			Name, Value, Domain, Path, SameSite string
			Secure, HTTPOnly                    bool
			Expires                             float64
		} `json:"cookies"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	cookies := make([]selenium.Cookie, len(result.Cookies))
	for index, item := range result.Cookies {
		cookies[index] = selenium.Cookie{Name: item.Name, Value: item.Value, Domain: item.Domain, Path: item.Path, Secure: item.Secure, HTTPOnly: item.HTTPOnly, Expiry: uint(item.Expires), SameSite: selenium.SameSite(item.SameSite)}
	}
	return cookies, nil
}
func (d *directCDPDriver) GetCookie(name string) (selenium.Cookie, error) {
	cookies, err := d.GetCookies()
	if err != nil {
		return selenium.Cookie{}, err
	}
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie, nil
		}
	}
	return selenium.Cookie{}, fmt.Errorf("cookie %q was not found", name)
}
func (d *directCDPDriver) AddCookie(cookie *selenium.Cookie) error {
	if cookie == nil {
		return errors.New("cookie was nil")
	}
	connection, err := d.currentConnection()
	if err != nil {
		return err
	}
	params := map[string]interface{}{"name": cookie.Name, "value": cookie.Value, "secure": cookie.Secure, "httpOnly": cookie.HTTPOnly}
	if cookie.Domain != "" {
		params["domain"] = cookie.Domain
	} else if currentURL, currentErr := d.CurrentURL(); currentErr == nil {
		params["url"] = currentURL
	} else {
		return currentErr
	}
	if cookie.Path != "" {
		params["path"] = cookie.Path
	}
	if cookie.Expiry > 0 {
		params["expires"] = cookie.Expiry
	}
	if cookie.SameSite != "" {
		params["sameSite"] = cookie.SameSite
	}
	_, err = connection.command(context.Background(), "Network.setCookie", params)
	return err
}
func (d *directCDPDriver) DeleteAllCookies() error {
	connection, err := d.currentConnection()
	if err != nil {
		return err
	}
	_, err = connection.command(context.Background(), "Network.clearBrowserCookies", map[string]interface{}{})
	return err
}
func (d *directCDPDriver) DeleteCookie(name string) error {
	connection, err := d.currentConnection()
	if err != nil {
		return err
	}
	currentURL, urlErr := d.CurrentURL()
	if urlErr != nil {
		return urlErr
	}
	_, err = connection.command(context.Background(), "Network.deleteCookies", map[string]interface{}{"name": name, "url": currentURL})
	return err
}

func (d *directCDPDriver) Close() error {
	id := d.SessionID()
	if id == "" {
		return nil
	}
	response, err := d.client.Get(d.address + "/json/close/" + url.PathEscape(id))
	if response != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return err
	}
	d.mu.Lock()
	if connection := d.connections[id]; connection != nil {
		connection.close()
		delete(d.connections, id)
	}
	delete(d.targets, id)
	d.current = ""
	d.frameRoot = ""
	d.mu.Unlock()
	return d.refreshTargets()
}

func (d *directCDPDriver) OpenTab(URL string) (string, error) {
	if strings.TrimSpace(URL) == "" {
		URL = "about:blank"
	}
	request, err := http.NewRequest(http.MethodPut, d.address+"/json/new?"+url.QueryEscape(URL), nil)
	if err != nil {
		return "", err
	}
	response, err := d.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("create Chrome target: %s", response.Status)
	}
	target := &directCDPTarget{}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return "", err
	}
	if err := d.refreshTargets(); err != nil {
		return "", err
	}
	if err := d.SwitchWindow(target.ID); err != nil {
		return "", err
	}
	return target.ID, nil
}

func (d *directCDPDriver) Quit() error {
	d.mu.Lock()
	connections := make([]*directCDPConnection, 0, len(d.connections))
	for _, connection := range d.connections {
		connections = append(connections, connection)
	}
	d.connections = make(map[string]*directCDPConnection)
	d.mu.Unlock()
	for _, connection := range connections {
		connection.close()
	}
	return nil
}

func (d *directCDPDriver) FindElements(by, value string) ([]selenium.WebElement, error) {
	d.mu.Lock()
	root := d.frameRoot
	d.mu.Unlock()
	if root == "" {
		root = "document"
	}
	nodes := directCDPNodesExpression(by, value, root)
	countValue, err := d.evaluate("return ("+nodes+").length;", nil)
	if err != nil {
		return nil, err
	}
	count := int(toFloat64(countValue))
	result := make([]selenium.WebElement, count)
	for index := 0; index < count; index++ {
		result[index] = &directCDPElement{driver: d, root: root, by: by, value: value, index: index}
	}
	return result, nil
}

func (d *directCDPDriver) SwitchFrame(frame interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if frame == nil {
		d.frameRoot = ""
		return nil
	}
	element, ok := frame.(*directCDPElement)
	if !ok {
		return fmt.Errorf("direct CDP frame expected *directCDPElement, had %T", frame)
	}
	d.frameRoot = "(" + element.nodeExpression() + ").contentDocument"
	return nil
}

func (d *directCDPDriver) FindElement(by, value string) (selenium.WebElement, error) {
	elements, err := d.FindElements(by, value)
	if err != nil {
		return nil, err
	}
	if len(elements) == 0 {
		return nil, &selenium.Error{Err: "no such element", Message: value, LegacyCode: 7}
	}
	return elements[0], nil
}

func toFloat64(value interface{}) float64 {
	switch actual := value.(type) {
	case float64:
		return actual
	case float32:
		return float64(actual)
	case int:
		return float64(actual)
	case json.Number:
		result, _ := actual.Float64()
		return result
	default:
		return 0
	}
}

func directCDPNodesExpression(by, value, root string) string {
	encoded, _ := json.Marshal(value)
	selector := string(encoded)
	switch by {
	case selenium.ByXPATH:
		return `(()=>{const r=[];const x=document.evaluate(` + selector + `,` + root + `,null,XPathResult.ORDERED_NODE_ITERATOR_TYPE,null);let n;while(n=x.iterateNext())r.push(n);return r;})()`
	case selenium.ByID:
		return `Array.from(` + root + `.querySelectorAll('[id]')).filter(e=>e.id===` + selector + `)`
	case selenium.ByName:
		return `Array.from(` + root + `.querySelectorAll('[name]')).filter(e=>e.name===` + selector + `)`
	case selenium.ByClassName:
		return `Array.from(` + root + `.getElementsByClassName(` + selector + `))`
	case selenium.ByTagName:
		return `Array.from(` + root + `.getElementsByTagName(` + selector + `))`
	case selenium.ByLinkText:
		return `Array.from(` + root + `.querySelectorAll('a')).filter(e=>(e.textContent||'').trim()===` + selector + `)`
	case selenium.ByPartialLinkText:
		return `Array.from(` + root + `.querySelectorAll('a')).filter(e=>(e.textContent||'').includes(` + selector + `))`
	default:
		return `Array.from(` + root + `.querySelectorAll(` + selector + `))`
	}
}

type directCDPElement struct {
	selenium.WebElement
	driver *directCDPDriver
	parent *directCDPElement
	root   string
	by     string
	value  string
	index  int
}

func (e *directCDPElement) nodeExpression() string {
	root := "document"
	if e.root != "" {
		root = e.root
	}
	if e.parent != nil {
		root = e.parent.nodeExpression()
	}
	return `(` + directCDPNodesExpression(e.by, e.value, root) + `)[` + fmt.Sprint(e.index) + `]`
}

func (e *directCDPElement) evaluate(body string, args ...interface{}) (interface{}, error) {
	encodedArgs := make([]string, len(args))
	for index, argument := range args {
		data, _ := json.Marshal(argument)
		encodedArgs[index] = string(data)
	}
	expression := e.nodeExpression()
	script := `const __element=` + expression + `;if(!__element)throw new Error('element is no longer available');const __args=[` + strings.Join(encodedArgs, ",") + `];return (function(){` + body + `}).apply(__element,__args);`
	return e.driver.evaluate(script, nil)
}

func (e *directCDPElement) Click() error {
	x, y, err := e.center()
	if err != nil {
		return err
	}
	for _, event := range []map[string]interface{}{
		{"type": "mouseMoved", "x": x, "y": y},
		{"type": "mousePressed", "x": x, "y": y, "button": "left", "clickCount": 1},
		{"type": "mouseReleased", "x": x, "y": y, "button": "left", "clickCount": 1},
	} {
		if _, err := e.driver.CDPCommand("Input.dispatchMouseEvent", event); err != nil {
			return err
		}
	}
	return nil
}
func (e *directCDPElement) Clear() error {
	_, err := e.evaluate(`this.focus();this.value='';this.dispatchEvent(new Event('input',{bubbles:true}));this.dispatchEvent(new Event('change',{bubbles:true}));return true;`)
	return err
}
func (e *directCDPElement) SendKeys(keys string) error {
	inputType, typeErr := e.evaluate(`return (this.type||'').toLowerCase();`)
	if typeErr == nil && fmt.Sprint(inputType) == "file" {
		objectID, err := e.driver.objectID(e.nodeExpression())
		if err != nil {
			return err
		}
		files := strings.Split(strings.ReplaceAll(keys, "\r\n", "\n"), "\n")
		_, err = e.driver.CDPCommand("DOM.setFileInputFiles", map[string]interface{}{"files": files, "objectId": objectID})
		return err
	}
	if _, err := e.evaluate(`this.focus();return true;`); err != nil {
		return err
	}
	names := map[string]string{
		selenium.BackspaceKey: "Backspace", selenium.TabKey: "Tab", selenium.EnterKey: "Enter",
		selenium.EscapeKey: "Escape", selenium.SpaceKey: " ", selenium.PageUpKey: "PageUp",
		selenium.PageDownKey: "PageDown", selenium.EndKey: "End", selenium.HomeKey: "Home",
		selenium.LeftArrowKey: "ArrowLeft", selenium.UpArrowKey: "ArrowUp",
		selenium.RightArrowKey: "ArrowRight", selenium.DownArrowKey: "ArrowDown", selenium.DeleteKey: "Delete",
	}
	if key := names[keys]; key != "" {
		for _, kind := range []string{"keyDown", "keyUp"} {
			if _, err := e.driver.CDPCommand("Input.dispatchKeyEvent", map[string]interface{}{"type": kind, "key": key}); err != nil {
				return err
			}
		}
		return nil
	}
	_, err := e.driver.CDPCommand("Input.insertText", map[string]interface{}{"text": keys})
	return err
}
func (e *directCDPElement) Submit() error {
	_, err := e.evaluate(`const f=this.form||this.closest('form');if(!f)throw new Error('form not found');if(f.requestSubmit)f.requestSubmit();else f.submit();return true;`)
	return err
}
func (e *directCDPElement) MoveTo(xOffset, yOffset int) error {
	x, y, err := e.center()
	if err != nil {
		return err
	}
	_, err = e.driver.CDPCommand("Input.dispatchMouseEvent", map[string]interface{}{"type": "mouseMoved", "x": x + float64(xOffset), "y": y + float64(yOffset)})
	return err
}

func (e *directCDPElement) center() (float64, float64, error) {
	value, err := e.evaluate(`this.scrollIntoView({block:'center',inline:'center'});const r=this.getBoundingClientRect();return {x:r.left+r.width/2,y:r.top+r.height/2};`)
	if err != nil {
		return 0, 0, err
	}
	values, _ := value.(map[string]interface{})
	return toFloat64(values["x"]), toFloat64(values["y"]), nil
}
func (e *directCDPElement) Text() (string, error) {
	value, err := e.evaluate(`return this.innerText!==undefined?this.innerText:this.textContent;`)
	return fmt.Sprint(value), err
}
func (e *directCDPElement) TagName() (string, error) {
	value, err := e.evaluate(`return this.tagName.toLowerCase();`)
	return fmt.Sprint(value), err
}
func (e *directCDPElement) IsSelected() (bool, error) {
	value, err := e.evaluate(`return Boolean(this.checked||this.selected);`)
	result, _ := value.(bool)
	return result, err
}
func (e *directCDPElement) IsEnabled() (bool, error) {
	value, err := e.evaluate(`return !this.disabled;`)
	result, _ := value.(bool)
	return result, err
}
func (e *directCDPElement) IsDisplayed() (bool, error) {
	value, err := e.evaluate(`const s=getComputedStyle(this),r=this.getBoundingClientRect();return s.display!=='none'&&s.visibility!=='hidden'&&Number(s.opacity)!==0&&r.width>0&&r.height>0;`)
	result, _ := value.(bool)
	return result, err
}
func (e *directCDPElement) GetAttribute(name string) (string, error) {
	value, err := e.evaluate(`return this.getAttribute(arguments[0]);`, name)
	if value == nil {
		return "", nil
	}
	return fmt.Sprint(value), err
}
func (e *directCDPElement) GetProperty(name string) (string, error) {
	value, err := e.evaluate(`return this[arguments[0]];`, name)
	if value == nil {
		return "", nil
	}
	return fmt.Sprint(value), err
}
func (e *directCDPElement) CSSProperty(name string) (string, error) {
	value, err := e.evaluate(`return getComputedStyle(this).getPropertyValue(arguments[0]);`, name)
	return fmt.Sprint(value), err
}
func (e *directCDPElement) Location() (*selenium.Point, error) {
	value, err := e.evaluate(`const r=this.getBoundingClientRect();return {x:r.x,y:r.y};`)
	values, _ := value.(map[string]interface{})
	return &selenium.Point{X: int(toFloat64(values["x"])), Y: int(toFloat64(values["y"]))}, err
}
func (e *directCDPElement) LocationInView() (*selenium.Point, error) {
	if _, err := e.evaluate(`this.scrollIntoView({block:'center',inline:'center'});return true;`); err != nil {
		return nil, err
	}
	return e.Location()
}
func (e *directCDPElement) Size() (*selenium.Size, error) {
	value, err := e.evaluate(`const r=this.getBoundingClientRect();return {width:r.width,height:r.height};`)
	values, _ := value.(map[string]interface{})
	return &selenium.Size{Width: int(toFloat64(values["width"])), Height: int(toFloat64(values["height"]))}, err
}
func (e *directCDPElement) FindElements(by, value string) ([]selenium.WebElement, error) {
	nodes := directCDPNodesExpression(by, value, e.nodeExpression())
	countValue, err := e.driver.evaluate("return ("+nodes+").length;", nil)
	if err != nil {
		return nil, err
	}
	count := int(toFloat64(countValue))
	result := make([]selenium.WebElement, count)
	for i := 0; i < count; i++ {
		result[i] = &directCDPElement{driver: e.driver, parent: e, by: by, value: value, index: i}
	}
	return result, nil
}
func (e *directCDPElement) FindElement(by, value string) (selenium.WebElement, error) {
	values, err := e.FindElements(by, value)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, &selenium.Error{Err: "no such element", Message: value, LegacyCode: 7}
	}
	return values[0], nil
}
func (e *directCDPElement) Screenshot(scroll bool) ([]byte, error) {
	if scroll {
		_, _ = e.evaluate(`this.scrollIntoView({block:'center',inline:'center'});return true;`)
	}
	return e.driver.Screenshot()
}
