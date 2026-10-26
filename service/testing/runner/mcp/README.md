# MCP runner

`readCapture` reads an existing WebDriver JSONL network capture; it makes no
remote call. It returns `Transactions` with `method`, `args`, HTTP `status`, the
complete JSON-RPC `response`, `requestId`, and `durationMs`. Responses are paired
by JSON-RPC ID within each network transaction, so reused IDs after navigation
cannot pair with an earlier page's response. JSON batches, base64 bodies, and
SSE messages are supported. Missing, truncated, malformed, ambiguous, or empty
tool-call evidence fails with an event/stage diagnostic that omits body contents
and headers. `Methods` optionally filters returned tool names; all captured tool
calls are still checked. `MaxTransactions` defaults to 10000 and fails rather than
silently dropping excess calls. Use explicit counts and `validator:assert` with
`each: true` so an empty filtered collection cannot become a passing assertion.

```yaml
pipeline:
  captured:
    action: mcp/runner:readCapture
    source: {URL: '${runDir}/browser.jsonl'}
    methods: [namespaces/preview]
  check:
    action: validator:assert
    each: true
    actual: $captured.Transactions
    expect:
      method: namespaces/preview
      status: 200
```

`mcp/runner` calls an external MCP server over stateless Streamable HTTP with the
2026-07-28 protocol. It opens a fresh connection for each Endly action. Tool
calls are sent once; transport failures are returned without replaying a call.

The `listTools` action returns `Tools` and `NextCursor`. The `call` action takes
`name` and structured `arguments`, and returns `Result` (the full MCP result),
`StructuredContent`, `Content`, and `IsError`. A tool result with `isError: true`
fails the Endly action while retaining the result in the service response. Set
`allowToolError: true` on a `call` action when a workflow needs to inspect and
publish an expected tool error; transport and protocol errors still fail.

```yaml
pipeline:
  mcp-tool:
    action: mcp/runner:call
    init:
      url: https://example.com/mcp
      bearerTokenSecret: example-mcp-token
      timeoutMs: 30000
      name: describe
      arguments:
        entity: sample
```

`bearerTokenSecret` is a Scy resource reference or a root workflow
`credentialMap` alias. The secret value is loaded at runtime and never included
in the workflow response. Authenticated HTTP requires HTTPS, except for
loopback HTTP. HTTP redirects are rejected when calling MCP endpoints.

For Scy OOB OAuth, set `oauth` instead of `bearerTokenSecret`:

```yaml
oauth:
  configURL: /path/to/oauth-config.json|blowfish://default
  secretsURL: /path/to/basic-secret.json|blowfish://default
  authFlow: OOB
  scopes: [read]
  usePKCE: true
```

The OAuth config and Basic credentials must be stored as Scy resources. The
runner passes these references to `scy/auth/authorizer`, receives an access
token, and uses it only for the current MCP operation. OOB authorization may
require interactive input. `timeoutMs` bounds authorization and the MCP call
together. OAuth and bearer token references are mutually exclusive. No OAuth
provider is contacted by the unit tests; they inject a local authorizer and
mock HTTP server.

For negative invocation tests, `allowProtocolError: true` captures a JSON-RPC
error as `IsError: true` and `ProtocolErrorCode`. It preserves only the numeric
code; remote error messages/data are omitted because they may contain credentials.
Transport failures still fail the action. The default remains false.

Native resources and skills use `listResources`, `readResource`, `listSkills`
and `getSkill`. Listing accepts an optional `cursor`; reads accept `uri`.
They share the same endpoint, SCY authentication and timeout options as tools.
Read operations optionally capture redacted protocol errors with
`allowProtocolError: true`. Skills listing never prefetches skill content.

```yaml
pipeline:
  catalog:
    action: mcp/runner:listSkills
    url: $namespaceMCP
    bearerTokenSecret: $userToken
  skill:
    action: mcp/runner:getSkill
    url: $namespaceMCP
    bearerTokenSecret: $userToken
    uri: $catalog.Skills[0].Uri
  definition:
    action: mcp/runner:readResource
    url: $namespaceMCP
    bearerTokenSecret: $userToken
    uri: mcp://ns/sales
```

For exact JSON shape preservation, `argumentsJSON` accepts a JSON object string
instead of `arguments`. This is useful when workflow map conversion would turn
an empty array into an empty object. The runner parses it immediately before the
MCP call, preserving empty arrays, objects, nulls and large integer tokens. Do not
set both options. Invalid JSON errors omit the supplied content; authentication
still uses the normal secret/OAuth fields.

```yaml
action: mcp/runner:call
url: http://127.0.0.1:8080/mcp
name: update_record
argumentsJSON: '{"recordId":123,"labels":[]}'
```
