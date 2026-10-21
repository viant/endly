# dbunit contract discovery

Read `service/testing/dbunit/README.md` in the matching Endly checkout for the
maintained workflow example and lifecycle/expansion contract. DTOs compose the
core requests in `service/testing/dbunit/contract.go`; live CLI/MCP schemas are
authoritative for the installed build.

MCP native tools are `endly_dbunit_register`, `endly_dbunit_init`,
`endly_dbunit_script`, `endly_dbunit_sql`, `endly_dbunit_prepare`,
`endly_dbunit_expect`, `endly_dbunit_query`, `endly_dbunit_dump`,
`endly_dbunit_export`, `endly_dbunit_compare`, `endly_dbunit_hydrate`,
`endly_dbunit_sequence`, and `endly_dbunit_map`.
Each takes an existing `sessionId` and a typed action `request`, and returns an
operation ID. Inspect status and events for assertion counts and failures.

`prepare.datasets[].source` and `expect.expectations[].expected` are source
objects, with `kind`, `path`/`data`/`url` as appropriate. File sources require
`root` (absolute) or `fixtures` (an existing session resolver), never inferred
paths. For inline data use `{kind: maps, data: [...]}`. `script.sources` is an
ordered list of source objects; `script.statements` is an explicit statement
array. `dump.destination` is an absolute writable SQL path separate from the
fixture resolver. `capture` selects a workflow output path; defaults are
`dbunit.<database>`.

`register.nativeLoad` is an explicit caller capability assertion; it is disabled
by default and requires the corresponding runtime SQLX loader. `prepare` and
`sql` accept `transactionMode: prepareAtomic` (default) or `nonAtomic`.
Nontransactional native loaders require the explicit `nonAtomic` mode.
