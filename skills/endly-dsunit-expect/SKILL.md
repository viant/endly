---
name: endly-dsunit-expect
description: "Validate Endly dsunit persisted rows, query results, datastore comparisons, and database schema."
---

Choose `expect` for fixture-based row validation, `query` with Expect for a SQL assertion, `compare` for comparison between query-backed datasets, and `checkSchema` for schema assertions. Inspect actual request contracts for datastore/dataset and matching options. Assert the behavior under test, including absence and row counts where required. QueryResponse, ExpectResponse, CompareResponse and CheckSchemaResponse publish assertion validations; an action returning successfully alone does not prove expectations matched. Use deterministic expected values and inspect failure details rather than replacing expected data with observed output. Read the full service reference for assertly directives and dataset checking policy.

Inspect `endly -s=dsunit -a=<action>` for the installed contract. Retrieve `endly-dsunit` for the full reference through MCP skills/get or endly_skill_get; its resource inventory includes references/service.md, readable with resources/read. Use endly-testing for test methodology.
