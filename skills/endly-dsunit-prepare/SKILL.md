---
name: endly-dsunit-prepare
description: "Prepare and reset Endly dsunit table fixtures from files or table-to-record data."
---

Use `dsunit:prepare` with a registered datastore and URL or Data as the live contract permits. JSON/CSV fixture directories map tables to records. An empty object in fixture data has special reset behavior; read the existing fixture convention before editing it. Preserve reset-before-sequence-before-conversion-before-population order. Related inserts should use the sequence skill to resolve symbolic foreign keys. DatasetResource Prefix/Postfix selection changes which files are loaded; verify the effective table set, especially when a case inherits aggregated data. Do not run unrelated cases concurrently against the same mutable fixture database.

Inspect `endly -s=dsunit -a=<action>` for the installed contract. Retrieve `endly-dsunit` for the full reference through MCP skills/get or endly_skill_get; its resource inventory includes references/service.md, readable with resources/read. Use endly-testing for test methodology.
