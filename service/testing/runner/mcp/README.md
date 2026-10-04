# MCP runner

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
