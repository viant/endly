package mobile

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type AppiumClient struct {
	Endpoint string
	HTTP     *http.Client
}

type AppiumSession struct {
	ID           string
	Capabilities map[string]interface{}
	client       *AppiumClient
}

func NewAppiumClient(endpoint string, client *http.Client) (*AppiumClient, error) {
	parsed, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid Appium endpoint %q", endpoint)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported Appium endpoint scheme %q", parsed.Scheme)
	}
	if client == nil {
		client = &http.Client{Timeout: 180 * time.Second}
	}
	return &AppiumClient{Endpoint: strings.TrimRight(endpoint, "/"), HTTP: client}, nil
}

func (c *AppiumClient) Status(ctx context.Context) (map[string]interface{}, error) {
	var value map[string]interface{}
	if err := c.do(ctx, http.MethodGet, "/status", nil, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (c *AppiumClient) NewSession(ctx context.Context, capabilities map[string]interface{}) (*AppiumSession, error) {
	payload := map[string]interface{}{
		"capabilities": map[string]interface{}{"alwaysMatch": capabilities},
	}
	var value struct {
		SessionID    string                 `json:"sessionId"`
		Capabilities map[string]interface{} `json:"capabilities"`
	}
	if err := c.do(ctx, http.MethodPost, "/session", payload, &value); err != nil {
		return nil, err
	}
	if value.SessionID == "" {
		return nil, fmt.Errorf("Appium new session response did not contain sessionId")
	}
	return &AppiumSession{ID: value.SessionID, Capabilities: value.Capabilities, client: c}, nil
}

func (s *AppiumSession) Close(ctx context.Context) error {
	return s.client.do(ctx, http.MethodDelete, s.sessionPath(""), nil, nil)
}

func (s *AppiumSession) FindElement(ctx context.Context, using, value string) (string, error) {
	var response map[string]interface{}
	if err := s.client.do(ctx, http.MethodPost, s.sessionPath("element"), map[string]interface{}{"using": using, "value": value}, &response); err != nil {
		return "", err
	}
	for _, key := range []string{"element-6066-11e4-a52e-4f735466cecf", "ELEMENT"} {
		if id, ok := response[key].(string); ok && id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("Appium find element response did not contain an element ID")
}

func (s *AppiumSession) Click(ctx context.Context, elementID string) error {
	return s.client.do(ctx, http.MethodPost, s.elementPath(elementID, "click"), map[string]interface{}{}, nil)
}

func (s *AppiumSession) Clear(ctx context.Context, elementID string) error {
	return s.client.do(ctx, http.MethodPost, s.elementPath(elementID, "clear"), map[string]interface{}{}, nil)
}

func (s *AppiumSession) SendKeys(ctx context.Context, elementID, text string) error {
	return s.client.do(ctx, http.MethodPost, s.elementPath(elementID, "value"), map[string]interface{}{"text": text, "value": []string{text}}, nil)
}

func (s *AppiumSession) Text(ctx context.Context, elementID string) (string, error) {
	var value string
	err := s.client.do(ctx, http.MethodGet, s.elementPath(elementID, "text"), nil, &value)
	return value, err
}

func (s *AppiumSession) Attribute(ctx context.Context, elementID, name string) (interface{}, error) {
	var value interface{}
	err := s.client.do(ctx, http.MethodGet, s.elementPath(elementID, "attribute/"+url.PathEscape(name)), nil, &value)
	return value, err
}

func (s *AppiumSession) ElementBool(ctx context.Context, elementID, property string) (bool, error) {
	if property != "displayed" && property != "enabled" && property != "selected" {
		return false, fmt.Errorf("unsupported element boolean property %q", property)
	}
	var value bool
	err := s.client.do(ctx, http.MethodGet, s.elementPath(elementID, property), nil, &value)
	return value, err
}

func (s *AppiumSession) PageSource(ctx context.Context) (string, error) {
	var value string
	err := s.client.do(ctx, http.MethodGet, s.sessionPath("source"), nil, &value)
	return value, err
}

func (s *AppiumSession) Screenshot(ctx context.Context) ([]byte, error) {
	var value string
	if err := s.client.do(ctx, http.MethodGet, s.sessionPath("screenshot"), nil, &value); err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode Appium screenshot: %w", err)
	}
	return data, nil
}

func (s *AppiumSession) Execute(ctx context.Context, script string, args ...interface{}) (interface{}, error) {
	var value interface{}
	err := s.client.do(ctx, http.MethodPost, s.sessionPath("execute/sync"), map[string]interface{}{"script": script, "args": args}, &value)
	return value, err
}

func (s *AppiumSession) sessionPath(suffix string) string {
	base := "/session/" + url.PathEscape(s.ID)
	if suffix == "" {
		return base
	}
	return base + "/" + strings.TrimLeft(suffix, "/")
}

func (s *AppiumSession) elementPath(elementID, suffix string) string {
	return s.sessionPath("element/" + url.PathEscape(elementID) + "/" + strings.TrimLeft(suffix, "/"))
}

func (c *AppiumClient) do(ctx context.Context, method, suffix string, payload, target interface{}) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode Appium request: %w", err)
		}
		body = bytes.NewReader(data)
	}
	requestURL, err := joinEndpoint(c.Endpoint, suffix)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return fmt.Errorf("create Appium request: %w", err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return fmt.Errorf("Appium %s %s: %w", method, suffix, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("read Appium response: %w", err)
	}
	envelope := struct {
		Value json.RawMessage `json:"value"`
	}{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &envelope); err != nil {
			return fmt.Errorf("decode Appium response: %w", err)
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := strings.TrimSpace(string(envelope.Value))
		var appiumError struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(envelope.Value, &appiumError) == nil && appiumError.Message != "" {
			message = appiumError.Error + ": " + appiumError.Message
		}
		return fmt.Errorf("Appium %s %s returned %s: %s", method, suffix, response.Status, message)
	}
	if target != nil && len(envelope.Value) > 0 && string(envelope.Value) != "null" {
		if err := json.Unmarshal(envelope.Value, target); err != nil {
			return fmt.Errorf("decode Appium value: %w", err)
		}
	}
	return nil
}

func joinEndpoint(endpoint, suffix string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	parsed.Path = path.Join(parsed.Path, suffix)
	if strings.HasSuffix(suffix, "/") {
		parsed.Path += "/"
	}
	return parsed.String(), nil
}
