---
name: endly-mcp-debug
description: Discover, run, rerun, pause, step, resume, and diagnose Endly workflows and e2e instances through stateful MCP tools.
---

1. `endly_open` creates a session; retain its `sessionId`.
2. `endly_loadWorkflow` takes sessionId and URL (or inline Content plus source URL).
   Load from the e2e directory so WorkingDirectory-based init matches CLI behavior.
3. `endly_listTasks` returns ordered task paths; use endly_listInstances for filtered,
   paged IDs from the actual nested run context (sessionId, workflow, path, filter,
   offset, limit). Read skipExpression before choosing a proof case.
   Discover instances from the loaded root to retain its nested run context.
4. `endly_runWorkflow` takes sessionId, workflow alias, tasks, selectorMode and tagIds.
   Empty tasks means all tasks. Use exact discovered IDs and include needed setup.
5. `endly_getOperation` returns concise status and assertion counts. Use detail=true
   for full output, or endly_operation_events with after/limit/types/detail for pages;
   `endly_listOperations` lists previous runs. Keep operationId for diagnosis.
6. `endly_inspectContext` reads a path in session state. Supply operationId to read a
   paused debug snapshot; Full requests the available full state. Secrets are redacted
   by the same control runtime as the web API. Inspect fixture rows, action outputs
   and selected IDs rather than extracting credentials.

For a diagnostic rerun, use `debug: true` on runWorkflow. It pauses at a boundary.
Poll `endly_getDebugState`, inspect the current step and state, then use
`endly_debugCommand` with command `step`, `next`, `continue`, `pause` or `stop`.
Breakpoints use the existing DebugCommandRequest/Step contract exposed in tools/list.
Resume with `continue` on the same operation; rerun by starting a new operation in
the same session. Resume and rerun have different effects: reruns execute setup again.
Enable/disable event logging with `endly_setLogging`; read it with endly_getLogging.

Check terminal operation status and assertion events. A loaded workflow or accepted
operation is not proof of test success. Avoid dumping unbounded full suite events;
inspect the relevant operation and state path. Close the session when finished.
Retrieve skills with native skills/list and skills/get, or endly_skill_list and
endly_skill_get. Read files in the returned resource inventory with resources/read
or endly_resource_read. Skill retrieval itself does not execute or authorize actions.

List sessions/workflows/operations with filter/offset/limit; default pages contain
20 items. Use instance pages of 20 (max 100) and event pages of 50 (max 200). Full
payloads are explicit. For Go stalls retrieve endly-go-diagnostics and use gops.
