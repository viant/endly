---
name: endly-dsunit-query
description: "Query Endly dsunit datastores and inspect SQL results for test diagnostics."
---

Use `dsunit:query` with Datastore and SQL, and inspect its live QueryRequest/QueryResponse contract before setting Params or Expect. Read returned rows and validation details. Do not enable IgnoreError when diagnosing a failed test: it suppresses the action error. A query action can execute mutating SQL, as the platform setup's SET/DELETE/ALTER statements show; select read-only SQL for state diagnosis unless mutation is part of the requested setup. Use session state for fixture IDs and compare persisted rows with API results.

Inspect `endly -s=dsunit -a=<action>` for the installed contract. Retrieve `endly-dsunit` for the full reference through MCP skills/get or endly_skill_get; its resource inventory includes references/service.md, readable with resources/read. Use endly-testing for test methodology.
