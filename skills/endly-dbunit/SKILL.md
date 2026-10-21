---
name: endly-dbunit
description: Author and diagnose Endly SQL fixture preparation, validation, query and schema workflows using the dbunit service; preserve dsunit for legacy and NoSQL workflows.
---

Use `action: dbunit:<action>` for SQL fixtures over named session connections.
Inspect `endly -s=dbunit -a=<action>` or MCP `endly_service_info` for the installed
contract before adding fields. Actions are `register`, `init`, `script`, `sql`,
`prepare`, `expect`, `query`, `dump`, `export`, `compare`, `hydrate`, `sequence`,
and `map`.

Read [the service reference](references/service.md) for fixture sources,
expansion boundaries, captures and a runnable SQLite workflow shape.

Use an explicit absolute fixture root or a named resolver bound by the embedding
program. Table identifiers are arrays of components, e.g. `[customers]`.
Inline fixture data stays raw until core dbunit expands it once. Put application
or HTTP actions between committed preparation and expectations. Read completed
assertion reports instead of reevaluating predicates.

Existing `dsunit` actions and `AsTableRecords` retain their original behavior;
there is no transparent migration or global hydration UDF override. Driver
availability does not imply certified schema/load support.

For generated IDs, multiple parents, cross-database seeds, or logical fixtures,
read [generation and composition](references/fixtures.md). Keep sequences local
and continuation explicit; route each child's tag reference to its intended
parent. Use `sourceFrom`/`datasetsFrom` to pass resolved outputs without another
expansion pass.
