# Endly MCP

Build `go build -o endly-mcp ./endly`. Run `./endly-mcp mcp` for stdio,
or `./endly-mcp mcp -transport=streamable -addr=127.0.0.1:4981` for HTTP `/mcp`.
HTTP binds only to loopback. Start in the workflow directory when its init uses
`$WorkingDirectory`. Workflow print output is redirected to stderr in stdio mode.

Every manager action has an `endly_<action>` tool, generated from the same request
contract as the REST control API: open/close/list sessions; load/list/unload workflows;
listTasks; runWorkflow/startWorkflow/runAction; get/list/stop operations;
inspectContext; set/get logging; getDebugState and debugCommand.

Open a session, load run.yaml, list tasks, then start a workflow. Empty tasks runs
all tasks (equivalent to `endly` in an e2e directory); tasks=test selects the test
task (equivalent to `endly -t=test`). Operations are asynchronous: use getOperation
to inspect status, result, errors and retained events. Shared state defaults to true,
so reruns reuse the session context. Debug runs pause at boundaries; inspect their
operation snapshot, then step/next/continue. A new run reruns actions; continue
resumes the existing paused operation.

`selectorMode: path` enables dotted paths and executes comma-separated selections
in supplied order, retaining ancestor lifecycle/state. Legacy mode is unchanged.
Repeated selections rerun their ancestors. A path is within the current workflow;
it does not cross a nested run action. `tagIds` selects exact generated template
TagIDs. listTasks returns a tree of paths and actions, with instances grouped by
TagID. The ordinary loader expands templates during workflow loading. Referenced
run workflows use Endly's existing resource/default fallback when loaded.

Skills are embedded with the binary and exposed via native skills/list, skills/get,
resources/list and resources/read. Tool-only clients can call endly_skill_list,
endly_skill_get (metadata plus entrypoint text) and endly_resource_read (references).
Native skills/list and its tool alias paginate: follow nextCursor. HTTP catalog:
GET /v1/endly/skills and GET /v1/endly/skills/<skill-name>.
The existing /v1/endly/sessions API shares the same runtime. It additionally exposes
GET /v1/endly/sessions/<id>/workflows/<alias>/tasks?path=<path>.

Regenerate service skills with `python3 tools/generate_skills.py` when maintained
service references change. General authoring, orchestration, testing methodology,
project patterns and dsunit function skills are maintained separately. A skill
provides instructions; reading one does not execute or authorize its actions.

Responses are concise by default. getOperation returns status/assertion counts;
detail=true includes full results and retained events. endly_operation_events
paginates by sequence (after, limit 1..200, optional types, detail). Event retention
keeps the newest events and counts evictions. List sessions/workflows/operations
with filter, offset and limit (default 20, max 100). endly_listInstances filters
and pages exact IDs in the root's nested run context and exposes skip expressions.
Use those IDs when running the root; IDs from a separately named workflow can differ.
Path mode also filters unmatched template groups, while keeping ordinary setup.
Assertion failures produce failed operations with passed/failed assertion counts.
Enable `-diagnostics` for `gops stack <pid>`; shutdown is bounded to ten seconds.

Public skills use generic examples. Add project-specific skills only by explicitly
passing repeatable `-skill-dir /path/to/local-skill` options; each directory contains
SKILL.md and its own resources. Local skills are sealed and served at startup,
and duplicate names fail registration. endly_service_info exposes live service
IDs/actions/request schemas without example payloads.
