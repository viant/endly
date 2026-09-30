---
name: endly-dsunit-sequences
description: "Allocate Endly dsunit sequence-backed fixture IDs and resolve AsTableRecords references across tables."
---

Query `dsunit:sequence` after datastore registration and any resets that affect IDs. Supply fixture table names and publish the response with `post: {Sequences: $Sequences}`. Aggregate case fixtures before conversion. Call `$AsTableRecords('data.ci_ads_dbsetup/ci_ads')`: the prefix publishes records under ci_ads.TABLE.tag.label; the returned map is input to prepare. Allocation tokens use `$Sequences.TABLE/${tag}.label`, and foreign keys use the same symbolic token. Conversion first allocates sequence IDs across tables, then expands references. `!tag` can set a per-record tag. The UDF mutates source records and caches some input forms: do not repeatedly convert a fixture expecting fresh IDs. This is an Endly UDF, not a dsunit action. Inspect `service/testing/dsunit/udf.go` and its sequence/data tests for unusual formats.

Inspect `endly -s=dsunit -a=<action>` for the installed contract. Retrieve `endly-dsunit` for the full reference through MCP skills/get or endly_skill_get; its resource inventory includes references/service.md, readable with resources/read. Use endly-testing for test methodology.
