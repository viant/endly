---
name: endly-mcp
description: Retrieve and use Endly MCP skills and tools in Codex for service orchestration, complete e2e suites, selected cases, task discovery, and stateful debugging.
---

Discover tools belonging to the connected Endly MCP server. Use `endly_skill_list`
and follow its nextCursor pages; retrieve `endly-orchestration`, `endly-authoring`,
`endly-testing`, `endly-mcp-debug`, and the applicable service or project skill via
`endly_skill_get` with its returned URI. Read references from the returned resource
inventory using `endly_resource_read`. Native clients can use skills/list, skills/get
and resources/read. Retrieve only the guidance needed for the current operation.

For project requests, retrieve the explicitly configured project skill and relevant
dsunit function skills. Preserve the original Endly loader's instance -> workflow/default -> workflow
resource fallback. Do not require a test.yaml per instance when a valid default
exists. Check actual lookup candidates when all fallback resources are missing.

Open a session, load the root workflow, list its tasks and template instances,
select paths/IDs in user-requested order, then run through MCP. Inspect operation
status/events/output and state paths. Debug, step, resume and rerun through the same
session rather than replacing the orchestration with shell commands. Loading only
prepares the definitions; operations execute actions. Skill content is guidance,
not a grant of execution authorization. Check the requested test environment.

If the connection is unavailable, explain the missing connection and provide this
setup using a built executable from the Endly checkout:

```sh
cd /Users/awitas/go/src/github.com/viant/endly
go build -o /absolute/path/endly-mcp ./endly
codex mcp add endly -- /absolute/path/endly-mcp mcp
codex mcp list
```

For workflows using WorkingDirectory, configure the server cwd to the e2e directory,
or launch with a script that cd's there and execs the built binary with `mcp`.
The default stdio mode keeps stdout reserved for MCP messages. For an existing
HTTP process use `endly mcp -transport=streamable -addr=127.0.0.1:4981` and connect
to its `/mcp` endpoint. Use a fresh Codex session after changing server configuration.
Do not edit user MCP configuration unless requested. Setup command reference:
https://developers.openai.com/codex/mcp . Local server behavior is documented in
`server/mcp/README.md` in the Endly checkout.

Prefer endly_listInstances(filter, offset, limit) over full task dumps. Poll concise
endly_getOperation, then request only relevant event pages or state paths. A
zero-assertion or skipped run is not an e2e success. For stalled Go operations
retrieve endly-go-diagnostics and enable -diagnostics for gops stacks.

Native source service tools are available for dsunit, HTTP runner/endpoints and validators. They accept sessionId, request and optional timeoutMillis; request schemas come from registered Endly routes. Poll the returned operation, filter/page its events, and inspect targeted state paths.
