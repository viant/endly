---
name: endly-orchestration
description: Orchestrate Endly services and nested workflows with ordered tasks, shared state, setup and cleanup, conditions, and asynchronous operation control through MCP.
---

Load the root YAML through `endly_loadWorkflow` in an opened MCP session before
execution. The ordinary Endly loader resolves nested `request: '@...'` resources
relative to their workflow source and expands template resources; nested run actions
load their referenced workflows when invoked. Keep the MCP process working directory
the same as the CLI workflow directory when workflows use `$WorkingDirectory`.
Loading discovers definitions and fixture assets; running executes service effects.

Use `endly_listTasks` to discover current task paths and instances. Load a referenced
regression workflow in the same session to inspect its expanded template instances
before running a root test task. Do not execute setup merely to discover test IDs.
Choose `tasks`, `tagIds`, `selectorMode`, and `params` on `endly_runWorkflow`.
A path selects within one workflow; it does not cross a nested run action boundary.
Path mode keeps ancestors and executes selections in the order supplied, including
repeated parents. Include prerequisite tasks explicitly when their state is needed.
Legacy CLI and MCP defaults remain unchanged.

Use a single session to retain state and loaded workflows across operations. Operations
in one session serialize; use separate sessions only when the external resources are
also isolated. `init`, action outputs, and `post` connect tasks. Preserve workflow
`catch`/`defer` tasks and conditional execution when narrowing a workflow. Do not
confuse queuing an operation with its completion: poll `endly_getOperation` and check
terminal status, errors, events, and test validations. `endly_stopOperation` cancels
a run; closing a session cancels operations and releases its state.

The existing REST control endpoints on the MCP HTTP server share these sessions.
Use service skills for each orchestrated action and endly-testing for e2e methodology.
