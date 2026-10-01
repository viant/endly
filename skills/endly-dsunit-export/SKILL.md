---
name: endly-dsunit-export
description: "Export Endly dsunit fixture snapshots or schema DDL using freeze and dump."
---

Use `freeze` to capture setup or verification datasets from existing rows; use `dump` to export schema DDL. Confirm source datastore, SQL/table selection and DestURL with the live contracts. Keep exported data limited to the requested test dataset and review sensitive fields. A snapshot is evidence, not automatically an approved expectation: check it against the behavior contract before committing it. Do not overwrite an existing expectation to hide a failure.

Inspect `endly -s=dsunit -a=<action>` for the installed contract. Retrieve `endly-dsunit` for the full reference through MCP skills/get or endly_skill_get; its resource inventory includes references/service.md, readable with resources/read. Use endly-testing for test methodology.
