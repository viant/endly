# SQL fixture actions

`dbunit` is a thin Endly adapter over `github.com/viant/dbunit`. It coexists with
legacy `dsunit`; existing NoSQL workflows and the global `AsTableRecords` UDF
continue to use `dsunit`.

Actions: `register`, `init`, `script`, `sql`, `prepare`, `query`, `expect`, `dump`,
`export`, `compare`, `hydrate`, `sequence`, `map`.
Use `endly -s=dbunit -a=prepare` or MCP `endly_service_info` to inspect the installed
request schema. Native MCP tools are named `endly_dbunit_<action>` and use the same
Endly manager sessions, action policies and cancellation as `endly_runAction`.

```yaml
init:
  customerName: Alice
pipeline:
  connection:
    action: dbunit:register
    database: app
    driver: sqlite
    dsn: /tmp/orders/app.db
  schema:
    action: dbunit:script
    database: app
    statements:
      - CREATE TABLE customers (id INTEGER PRIMARY KEY, name TEXT)
  prepare:
    action: dbunit:prepare
    database: app
    datasets:
      - table: [customers]
        operation: replace
        source:
          kind: maps
          data:
            - id: 1
              name: $customerName
  query:
    action: dbunit:query
    database: app
    sql: SELECT id, name FROM customers
    capture: customers
  verify:
    action: dbunit:expect
    database: app
    expectations:
      - table: [customers]
        scope: full
        expected:
          kind: maps
          data:
            - id: 1
              name: $customerName
  extract:
    action: dbunit:dump
    database: app
    tables: [customers]
    destination: /tmp/orders/schema.sql
```

Create the destination directory before running. `init` combines connection
registration with ordered `sources` or `statements`. `script` also accepts a
previously returned `artifact`; `dump` returns SQL plus metadata and completeness
information. `sql` executes an ordered list of DML `{sql, args}` statements through
core `Engine.Exec`. Its default `transactionMode: prepareAtomic` commits all
statements together. Explicit `transactionMode: nonAtomic` preserves earlier
successful statements if a later statement fails. Caller-owned transaction and
case rollback modes require the Go APIs. SQL text is sent as authored;
bound `args` expand once in the core. Statements must start with INSERT, UPDATE,
DELETE, MERGE or REPLACE; use `script` for DDL.

`prepare` accepts the same `transactionMode` choices. Native loading is disabled
by default. A caller can explicitly certify a registered driver/loader workload
with `register.nativeLoad: {transactional: true, format: csv, minRows: 1000}`.
Use `transactional: false` with `prepare.transactionMode: nonAtomic` for a
certified nontransactional loader. `load.strategy: bulkRequired` still fails if
the runtime has not registered the corresponding SQLX loader. This option does
not certify a vendor automatically. Go embedders may pass one optional
`core.NativeLoadOptions` value to `BindDatabase`; settings are copied into the
session connection binding and survive `init` and later actions. Preparation
datasets and table expectations select the action's `database`; a different
child database is rejected to preserve its dialect and capability binding.
Explicit query comparison may use separate named session pools.

Explicit table identifiers are arrays of components: `[customers]`
or `[catalog, customers]`; `[table.with.dot]` retains a literal dot in its name.

Fixture sources are discriminated objects: `{kind: file, path: prepare/users.json}`,
`{kind: maps, data: [...]}`, or an explicit URL/content source supported by the
core resolver. File sources require an absolute `root`, or `fixtures` naming a
session-bound resolver. These two selectors are mutually exclusive. Paths never
derive from a test name or caller. Endly `@request` assets remain request assets.

`export` accepts a bounded `query` and optional `format: json` or `ndjson`, returning
fixture content and optionally writing an absolute `destination`. `compare`
accepts explicit `expected` and `actual` query requests and matching options;
it returns the same assertion response as `expect`. Their query argument arrays
are expanded once by core dbunit.

An embedding Go program imports the Endly adapter as `dbunitservice` and binds
assets and borrowed pools explicitly:

```go
//go:embed testdata/orders
var assets embed.FS

// ctx is an Endly session context, db is caller-owned.
err := dbunitservice.BindDatabase(ctx, "app", db)
err = dbunitservice.BindFixtures(ctx, "orders", fixture.Embed(&assets, "testdata/orders"))
```

YAML then selects `fixtures: orders` and `source: {kind: file, path: users.json}`.
Resolver registrations and database names are isolated per session. The adapter
closes only pools opened through `register`, once on session shutdown. Borrowed
pools remain caller-owned. Replacement requires `replace: true` and waits for
outstanding actions. SQLite aliases `sqlite3` and `moderncsqlite` select the pure
Go `sqlite` driver. Other SQL drivers and the corresponding SQLX
`metadata/product` packages must be registered by the chosen Endly runtime. Credential aliases use the existing secret registry;
`credentials` can populate explicit `{username}` / `{password}` DSN placeholders.

Endly expands normal parameters once. The `datasets`, `expectations`, query
`args`, SQL `statements` and local `state` fields remain raw until the adapter snapshots state
and core dbunit expands fixture rows and query arguments. File contents bypass Endly request expansion naturally.
Use templates in raw inline rows or files; do not supply rows already expanded
by another layer. Direct Go requests use concrete routing/SQL parameters and
raw fixture and query argument templates. Stateful fixture UDFs are invoked
once by the core.

Successful actions publish their result at an explicit `capture` path or
`dbunit.<database>`. Preparation publishes after commit. No pools, engines,
resolvers, transactions or credentials enter workflow state. Expectation
mismatches return the core assertion error and a report implementing Endly's
`Assertion()` contract; CLI and manager read completed counts without rerunning
predicates. Failed reports remain on the action response/event.

SQLite is the local integration acceptance dialect. Other vendor schema
fidelity and BigQuery nested loading require their respective core certification before this adapter can advertise support. Legacy fixture translation is not automatic. Hydration and mapping use the
explicit contracts below.


## Generation, sequences and mapped fixtures

`hydrate` accepts ordered `tables` with explicit `source`, `generate`,
`references` and `captures` specifications. It calls the core hydration engine;
there is no new global UDF or competing `AsTableRecords` registration.
`sequence` resolves named local fixture counters; it does not allocate database
sequence/identity values or reserve IDs in another transaction. Every request
owns its counters. A seed query is a snapshot of exactly one integer, so prepare
committed fixtures or continue from a prior result when IDs must advance.

```yaml
generate:
  action: dbunit:hydrate
  database: app
  capture: generated
  sequences:
    - name: parentIDs
      seed: {database: app, sql: 'SELECT COALESCE(MAX(id), 0)+1 FROM parents'}
    - name: childIDs
      seed: {database: other, sql: 'SELECT COALESCE(MAX(id), 0)+1 FROM children'}
  tables:
    - name: parents
      table: [parents]
      tagField: ['!tag']
      source:
        kind: maps
        data: [{'!tag': first, name: alpha}, {'!tag': second, name: beta}]
      generate:
        - {field: [id], sequence: parentIDs}
      captures:
        - {field: [id], path: created.parentIDs, append: true}
    - name: children
      table: [children]
      source:
        kind: maps
        data: [{parentTag: first}, {parentTag: second}]
      generate:
        - {field: [id], sequence: childIDs}
      references:
        - {field: [parent_id], table: parents, tag: '$row.parentTag', from: [id]}
```

Both `app` and `other` must already be registered in the session. Each seed's
`database` selects its binding explicitly. Tagged references resolve each
child's parent after generation, including forward references; they never use
a shared last-parent variable. `tagField` explicitly removes the chosen source
field. The current row tag is available as `$tag` during template expansion.
Slash-based legacy sequence expressions are replaced by explicit references.

`Before` steps run once per table entry, including an empty entry. `Generate`
steps run in declaration order per row and retain existing fields (including
null/zero) unless `overwrite: true`. `uuid: true` generates once for each field
that needs a value. `postIncrement` changes private aliases once after a
nonempty entry. Private aliases are published only through `stateCaptures`;
row captures use declared `captures` paths. All capture destinations are staged
and validated before any workflow-state root changes. Existing sibling state
is preserved; captured rows replace the declared destination exactly.

New action captures contain ordinary addressable maps with exact scalar values.
To continue explicitly in a later `hydrate` or `sequence` request, use
`sequences: [{name: parentIDs, startFrom: generated.sequences.parentIDs.next}]`.
`start` and `step` supply concrete counter configuration; `startFrom` is a
literal state path. Repeating a request starts fresh; there is no implicit
cache, reset, process-global sequence, or reuse of previously generated UUIDs.

`map` projects a logical source into explicit physical tables. `mapping.tables`
contains columns, exact paths, keys and optional associations. Required fields
fail by default; `onMissingRequired: skip` is explicit. Defaults apply only to
absent fields; explicit null/zero values remain present. Keys deduplicate equal
rows and reject conflicting values. Associations add parent dependencies; output
order is deterministic and cycles are rejected. Mapping defaults are literal,
so they are never evaluated as another expansion pass.

```yaml
project:
  action: dbunit:map
  database: app
  capture: projected
  sourceFrom: generated.tables[0].source
  mapping:
    name: parent-fixture
    tables:
      - name: parents
        table: [parents]
        keys: [id]
        columns: [{name: id}, {name: name}]
prepare:
  action: dbunit:prepare
  database: app
  datasetsFrom: projected.datasets
```

`sourceFrom` and `datasetsFrom` are literal field/index paths, without `$`.
They select captured containers without invoking functions or expanding rows.
`sourceFrom` is mutually exclusive with `source`; `datasetsFrom` is mutually
exclusive with `datasets`. Hydrated and mapped sources carry `expanded: true`,
which core dbunit preserves through mapping/preparation. Strings returned by a
UDF, such as `$NotAgain()`, remain literal after that first evaluation.
For direct Go composition, use `fixture.Resolved(rows)` for already concrete
records. Automatic workflow response publication also preserves completed data;
explicit `init`/`post` variables retain their normal expression behavior.

Use separate map/prepare actions for tables destined for different databases.
The complete two-database, multiple-parent workflow (including explicit sequence
continuation and exact-once UDF assertions) is exercised by
`hydration_workflow_test.go` beside this README.
