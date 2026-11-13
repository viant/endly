---
name: endly-dsunit-sequences
description: "Allocate Endly dsunit sequence-backed fixture IDs and resolve AsTableRecords references across tables."
---

Query `dsunit:sequence` after datastore registration and any resets that affect
IDs. For a grouped regression, aggregate all selected cases first, then perform
one sequence lookup, one conversion and one prepare per selected datastore,
outside the case template. Publish the response with `post: {Sequences: $Sequences}`.
Call `$AsTableRecords('data.sample_dbsetup/sample')`; the prefix publishes
`sample.TABLE.tag.label` models for case init and supplies the prepare records.
Case-owned data means distinct identities, not repeated hydration per case.

Allocation tokens use `$Sequences.TABLE/${tag}.label`; cross-table foreign keys
can use the same symbolic token. Same-table references need the verified
self-reference convention; another allocation token can allocate another ID.
Conversion allocates IDs across tables before expanding references. `!tag` sets
a per-record case tag. Reset `dsunitSequencer` once per new batch, not between
cases or dependent tables. The UDF mutates rows and caches some forms: reload
original symbolic fixtures for a fresh allocation. It is not a dsunit action.
Inspect `service/testing/dsunit/udf.go` and sequence/data tests for unusual formats.

Inspect `endly -s=dsunit -a=<action>` for the installed contract. Retrieve `endly-dsunit` for the full reference through MCP skills/get or endly_skill_get; its resource inventory includes references/service.md, readable with resources/read. Use endly-testing for test methodology.

For the complete setup/aggregation pattern, retrieve `endly-dsunit` and read its `references/fixtures-and-sequences.md`. It covers eager init expansion, scoped datastore selection, starting-value validation, two-pass references and fresh retries.
