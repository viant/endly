---
name: endly-dsunit-register
description: "Register and initialize Endly dsunit datastores and run schema or SQL setup."
---

Use `register` for a named connection, `create` for datastore creation, and `init` for composed schema/setup work. `script` runs scripts and `sql` runs SQL; inspect each live contract before choosing. Match driver, DSN and credential alias to the selected test environment. Registration precedes sequence, prepare, query, and expect. Recreate/schema scripts can destroy existing data; use the target documented by the test request. Inspect `endly -s=dsunit -a=init` for Admin, Scripts, Recreate and table configuration. Preserve SQL quoting settings from existing setup.

Inspect `endly -s=dsunit -a=<action>` for the installed contract. Retrieve `endly-dsunit` for the full reference through MCP skills/get or endly_skill_get; its resource inventory includes references/service.md, readable with resources/read. Use endly-testing for test methodology.
