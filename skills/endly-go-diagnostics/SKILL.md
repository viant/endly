---
name: endly-go-diagnostics
description: Diagnose stalled Endly Go orchestration with gops stacks, operation events, and targeted state inspection before fixing the underlying code.
---

Start the MCP process with `endly mcp -diagnostics` to enable gops. Identify that
exact Endly PID with gops/process inspection; do not signal unrelated processes.
Use `gops stack <pid>` to save goroutine stacks, then inspect the operation's
executeWorkflow/executeAction stack and the goroutine it is waiting for. If the
state has not progressed, capture a second stack after independent work to distinguish
a stable block from normal I/O. `gops stats` and `gops memstats` can distinguish
resource pressure from a mutex/channel wait. Avoid CPU/heap profiling unless needed.

Correlate the frames with MCP operation events and paused state. A dsunit.prepare
channel send can mean its dataset worker is waiting on database or BigQuery I/O;
inspect the worker before calling it a deadlock. A pending step/next command can
mean the action is still executing: poll debug status and pausedAt before issuing
continue. Never treat an RPC timeout as the operation's terminal status.

Fix the smallest demonstrated cause and add a regression test for observable
behavior. Preserve the asynchronous operation/session model and external test
fixtures. Use cancellation through MCP first, and ensure shutdown uses a bounded
context. Retain concise stack findings and code locations without dumping secrets
or large raw logs into the conversation.
