package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/viant/endly"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/mcp/client"
	"github.com/viant/scy/auth/authorizer"
	"github.com/viant/scy/cred/secret"
	"golang.org/x/oauth2"
)

const ServiceID = "mcp/runner"

const defaultTimeout = 30 * time.Second

type service struct {
	*endly.AbstractService
	authorizeOAuth func(context.Context, *authorizer.Command) (*oauth2.Token, error)
}

func New() endly.Service {
	s := &service{AbstractService: endly.NewAbstractService(ServiceID), authorizeOAuth: authorizer.New().Authorize}
	s.AbstractService.Service = s
	s.registerRoutes()
	return s
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action:           "listTools",
		RequestInfo:      &endly.ActionInfo{Description: "List tools on an external MCP server"},
		RequestProvider:  func() interface{} { return &ListToolsRequest{} },
		ResponseProvider: func() interface{} { return &ListToolsResponse{} },
		Handler: func(ctx *endly.Context, value interface{}) (interface{}, error) {
			return s.listTools(ctx, value.(*ListToolsRequest))
		},
	})
	s.Register(&endly.Route{
		Action:           "call",
		RequestInfo:      &endly.ActionInfo{Description: "Call one external MCP tool once"},
		RequestProvider:  func() interface{} { return &CallRequest{} },
		ResponseProvider: func() interface{} { return &CallResponse{} },
		Handler: func(ctx *endly.Context, value interface{}) (interface{}, error) {
			return s.call(ctx, value.(*CallRequest))
		},
	})
}

func (s *service) operationContext(ctx *endly.Context, request *Request) (context.Context, context.CancelFunc, string, error) {
	timeout := defaultTimeout
	if request.TimeoutMs > 0 {
		timeout = time.Duration(request.TimeoutMs) * time.Millisecond
	}
	opCtx, cancel := context.WithTimeout(ctx.Background(), timeout)
	if request.OAuth != nil {
		command := &authorizer.Command{
			OAuthConfig: authorizer.OAuthConfig{ConfigURL: request.OAuth.ConfigURL},
			SecretsURL:  request.OAuth.SecretsURL, AuthFlow: "OOB",
			Scopes: request.OAuth.Scopes, UsePKCE: request.OAuth.UsePKCE,
		}
		value, err := s.authorizeOAuth(opCtx, command)
		if err != nil || value == nil || strings.TrimSpace(value.AccessToken) == "" {
			ctxErr := opCtx.Err()
			cancel()
			if ctxErr != nil {
				return nil, nil, "", fmt.Errorf("MCP OAuth authorization failed: %w", ctxErr)
			}
			return nil, nil, "", fmt.Errorf("MCP OAuth authorization failed")
		}
		return opCtx, cancel, strings.TrimSpace(value.AccessToken), nil
	}
	if request.BearerTokenSecret == "" {
		return opCtx, cancel, "", nil
	}
	if ctx.Secrets == nil {
		cancel()
		return nil, nil, "", fmt.Errorf("MCP bearer token secret lookup is unavailable")
	}
	value, err := ctx.Secrets.Lookup(opCtx, secret.Resource(request.BearerTokenSecret))
	if err != nil {
		cancel()
		return nil, nil, "", fmt.Errorf("MCP bearer token secret lookup failed")
	}
	if value == nil || strings.TrimSpace(value.String()) == "" {
		cancel()
		return nil, nil, "", fmt.Errorf("MCP bearer token secret is empty")
	}
	token := strings.TrimSpace(value.String())
	return opCtx, cancel, token, nil
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (t bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	if t.token != "" {
		clone.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(clone)
}

func routeHeaders(_ context.Context, body []byte, header http.Header) error {
	header.Set("Accept", "application/json, text/event-stream")
	var request struct {
		ID     json.RawMessage            `json:"id"`
		Method string                     `json:"method"`
		Params map[string]json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		return err
	}
	if request.Method == "" || len(request.ID) == 0 || string(request.ID) == "null" {
		return nil
	}
	header.Set(schema.HeaderMethod, request.Method)
	if request.Method == schema.MethodToolsCall {
		var name string
		if err := json.Unmarshal(request.Params["name"], &name); err != nil {
			return err
		}
		header.Set(schema.HeaderName, name)
	}
	return nil
}

func (s *service) withClient(ctx *endly.Context, request *Request, action func(context.Context, *client.Client) error) error {
	opCtx, cancel, token, err := s.operationContext(ctx, request)
	if err != nil {
		return err
	}
	defer cancel()
	// Redirects are rejected so bearer credentials cannot cross origins. A fresh
	// stateless transport and client are used for each operation. No reconnect
	// callback is installed, so a failed tool call is never replayed.
	httpClient := &http.Client{
		Transport:     bearerTransport{base: http.DefaultTransport, token: token},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	transport, err := streamable.New(opCtx, request.URL,
		streamable.WithHTTPClient(httpClient),
		streamable.WithProtocolVersion(schema.LatestProtocolVersion),
		streamable.WithStateless(), streamable.WithRunTimeout(0),
		streamable.WithRequestHeaderProvider(routeHeaders))
	if err != nil {
		return safeError("MCP connection failed", opCtx, err)
	}
	cli := client.New("Endly", "1.0", transport, client.WithProtocolVersion(schema.LatestProtocolVersion))
	defer cli.Close()
	if _, err := cli.Initialize(opCtx); err != nil {
		return safeError("MCP connection failed", opCtx, err)
	}
	return action(opCtx, cli)
}

func safeError(prefix string, ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("%s: %w", prefix, ctxErr)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	// Provider error bodies and URLs can contain credentials. Keep the error
	// category while omitting all remote text from workflow logs.
	return fmt.Errorf("%s", prefix)
}

func (s *service) listTools(ctx *endly.Context, request *ListToolsRequest) (*ListToolsResponse, error) {
	var response *ListToolsResponse
	err := s.withClient(ctx, &request.Request, func(opCtx context.Context, cli *client.Client) error {
		result, err := cli.ListTools(opCtx, request.Cursor)
		if err != nil {
			return safeError("MCP listTools failed", opCtx, err)
		}
		response = &ListToolsResponse{Tools: result.Tools, NextCursor: result.NextCursor}
		return nil
	})
	return response, err
}

func (s *service) call(ctx *endly.Context, request *CallRequest) (*CallResponse, error) {
	var response *CallResponse
	err := s.withClient(ctx, &request.Request, func(opCtx context.Context, cli *client.Client) error {
		result, err := cli.CallTool(opCtx, &schema.CallToolRequestParams{Name: request.Name, Arguments: request.Arguments})
		if err != nil {
			return safeError("MCP call failed", opCtx, err)
		}
		response = &CallResponse{Result: result, StructuredContent: result.StructuredContent, Content: result.Content, IsError: result.IsError != nil && *result.IsError}
		if response.IsError && !request.AllowToolError {
			return fmt.Errorf("MCP tool %q returned isError", request.Name)
		}
		return nil
	})
	return response, err
}
