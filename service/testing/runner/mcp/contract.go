package mcp

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/viant/mcp-protocol/schema"
)

// Request configures one outbound MCP operation. BearerTokenSecret is a Scy
// resource reference (or a workflow credentialMap alias), never a token value.
type Request struct {
	URL               string        `json:"url" yaml:"url"`
	BearerTokenSecret string        `json:"bearerTokenSecret,omitempty" yaml:"bearerTokenSecret,omitempty"`
	OAuth             *OAuthRequest `json:"oauth,omitempty" yaml:"oauth,omitempty"`
	TimeoutMs         int           `json:"timeoutMs,omitempty" yaml:"timeoutMs,omitempty"`
	ProtocolVersion   string        `json:"protocolVersion,omitempty" yaml:"protocolVersion,omitempty"`
}

// OAuthRequest identifies Scy encrypted resources. No credential literals are
// accepted in the workflow request.
type OAuthRequest struct {
	ConfigURL  string   `json:"configURL" yaml:"configURL"`
	SecretsURL string   `json:"secretsURL" yaml:"secretsURL"`
	AuthFlow   string   `json:"authFlow,omitempty" yaml:"authFlow,omitempty"`
	Scopes     []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	UsePKCE    bool     `json:"usePKCE,omitempty" yaml:"usePKCE,omitempty"`
}

func (r *Request) Validate() error {
	u, err := url.Parse(r.URL)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return fmt.Errorf("URL must be an HTTP(S) endpoint without user information")
	}
	if r.TimeoutMs < 0 {
		return fmt.Errorf("TimeoutMs must be nonnegative")
	}
	if r.TimeoutMs > 24*60*60*1000 {
		return fmt.Errorf("TimeoutMs must not exceed one day")
	}
	if r.OAuth != nil {
		if r.BearerTokenSecret != "" {
			return fmt.Errorf("OAuth and BearerTokenSecret are mutually exclusive")
		}
		if r.OAuth.ConfigURL == "" || r.OAuth.SecretsURL == "" || (r.OAuth.AuthFlow != "" && r.OAuth.AuthFlow != "OOB") {
			return fmt.Errorf("OAuth requires ConfigURL and SecretsURL with AuthFlow OOB")
		}
		if strings.HasPrefix(r.OAuth.ConfigURL, "inlined://") || strings.HasPrefix(r.OAuth.SecretsURL, "inlined://") {
			return fmt.Errorf("OAuth credentials must use Scy resource references")
		}
	}
	if r.BearerTokenSecret != "" || r.OAuth != nil {
		if u.Scheme == "http" {
			host := u.Hostname()
			if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
				if !strings.EqualFold(host, "localhost") {
					return fmt.Errorf("authenticated MCP endpoints require HTTPS or loopback HTTP")
				}
			}
		}
	}
	if r.ProtocolVersion != "" && r.ProtocolVersion != schema.LatestProtocolVersion {
		return fmt.Errorf("only stateless MCP protocol %s is supported", schema.LatestProtocolVersion)
	}
	return nil
}

type ListToolsRequest struct {
	Request
	Cursor *string `json:"cursor,omitempty" yaml:"cursor,omitempty"`
}

func (r *ListToolsRequest) Validate() error { return r.Request.Validate() }

type ListToolsResponse struct {
	Tools      []schema.Tool
	NextCursor *string
}

type CallRequest struct {
	Request
	Name      string                 `json:"name" yaml:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty" yaml:"arguments,omitempty"`
	// AllowToolError returns an MCP isError result as a successful Endly action
	// so a workflow can inspect the complete diagnostic content.
	AllowToolError bool `json:"allowToolError,omitempty" yaml:"allowToolError,omitempty"`
}

func (r *CallRequest) Validate() error {
	if err := r.Request.Validate(); err != nil {
		return err
	}
	if r.Name == "" {
		return fmt.Errorf("Name is required")
	}
	return nil
}

// CallResponse preserves the full MCP result, including content and isError.
// StructuredContent is the decoded JSON object returned by tools that provide it.
type CallResponse struct {
	Result            *schema.CallToolResult
	StructuredContent interface{}
	Content           []schema.CallToolResultContentElem
	IsError           bool
}
