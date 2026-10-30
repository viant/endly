# Fixture aggregation and sequence allocation

Read this before authoring sequence-backed regression setup. The order is:
**schema/connection → static dictionaries → selected-table reset → mapping (if used) → sequence lookup → one conversion → prepare → case operations → final group expectations**.

For grouped regressions, lookup/conversion/prepare run once per selected datastore
over all collected case fixtures. Case init binds the published setup records; it
does not run database hydration again. Distinct tags preserve case isolation.

## Collect case data before setup

Endly expands template data while loading the regression workflow. A typical
module has a literal parent directory and an indexed child pattern:

```yaml
pipeline:
  setup:
    action: run
    request: '@database/setup'
  cases:
    tag: $pathMatch
    data:
      '[]app_dbsetup': '@dbsetup/app'
    subPath: 'cases/${index}_*'
    template:
      test:
        action: run
        init: {parentPath: '$path', tag: '$tag'}
        request: '@test'
```

The collected `data.app_dbsetup` is a table-to-record map available to setup.
Keys may accumulate from multiple modules at load time: `$IsDefined(data.X)` does
not prove that datastore X belongs to the selected run. Carry an explicit
selected-datastore list from the root and reset only those owned stores/tables.
Discover actual TagIDs with `endly_listInstances` in the root/nested context.
Do not substitute a folder number for the emitted TagID or claim a zero-case run.

## Keep allocation expressions unexpanded until conversion

Put sequence-bearing rows in case fixture files collected by `data`, or load a
standalone table map using `storage:download` with `udf: AsMap`. Do not put those
rows directly in ordinary workflow `init`: eager expansion can turn
`$Sequences.PARENT/case.parent` into `101/case.parent` before allocation runs.
That is a malformed identifier, not an allocated row or a database error.

Example fixture files under a case's `dbsetup/app` directory:

`PARENT.json`:
```json
[{"ID":"$Sequences.PARENT/${tag}.parent","CODE":"parent-$Sequences.PARENT"}]
```

`CHILD.json`:
```json
[{"ID":"$Sequences.CHILD/${tag}.child","PARENT_ID":"$Sequences.PARENT/${tag}.parent"}]
```

The bare `$Sequences.PARENT` in the same row's prefixed CODE uses that row's
allocated ID. Prefer this form over embedding a qualified allocation alias in
another string when working with older deployed Endly versions.

## Register, reset, query sequences, convert once

After registration, the selected mutable-table reset and any mapping setup:

```yaml
init:
  dsunitSequencer: {}
pipeline:
  loadSequences:
    action: dsunit:sequence
    datastore: app
    tables: $StringKeys(${data.app_dbsetup})
    post: {Sequences: '$Sequences'}
  populate:
    action: dsunit:prepare
    datastore: app
    data: $AsTableRecords('data.app_dbsetup/app')
```

Reset `dsunitSequencer` once at the beginning of a new allocation batch, not
between dependent tables. `AsTableRecords` is a UDF, not a dsunit action.
It first allocates IDs for all tables, then expands cross-table references.
The `/app` suffix publishes records under `app.TABLE.<tag>.<alias>`; a case can
read `${app.PARENT.${tag}.parent.ID}` and use that exact ID in requests, JWT claims
and expected database rows. `!tag` carries a row's case tag when data is aggregated.

A sequence is the **next value to assign**: the UDF uses the current integer and
then increments it. Do not blindly add one to a valid result. Check that every
requested integer auto-increment table has a returned, usable start value before
conversion. A missing or zero value must not proceed to MySQL: zero may trigger
MySQL's own allocation, colliding with the next explicitly assigned ID.

If the installed sequence provider returns an unusable value, diagnose it. For
MySQL, a native `dsunit:query` can read the actual allocator without changing it:

```sql
SELECT COALESCE(AUTO_INCREMENT, 1) AS n
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'PARENT'
```

Use the returned value for that table only, converted with `$AsInt`, before the
single `$AsTableRecords` call. Disable stale information-schema statistics on the
test connection where necessary (`information_schema_stats_expiry=0`). Do not
replace all sequences with hardcoded 1 or use guessed MAX-based counters.

## References and retries

A cross-table foreign key can reuse the parent allocation token because the
second pass resolves it after all IDs exist. A same-table foreign key is different:
an allocation token in another field of its own table can allocate again. Follow
a verified self-reference pattern (the current UDF has special handling for an
already allocated alias ending in `parent`) or resolve the relationship after
allocation. Assert both parent ID and stored FK; never assume they match.

`AsTableRecords` mutates source rows, and collection forms can be cached under a
`tableRecord` key. For a fresh allocation, reload original fixture expressions,
reset the batch sequencer state, and obtain fresh database sequence values. Do
not convert already expanded numeric rows and expect a second allocation.
Partial `prepare` failure can leave earlier tables populated. Inspect the failed
action and read back owned fixture tables before a scoped reset/retry. Do not
reset unrelated services or silently adopt another suite's rows.

## Authenticate with the fixture identities

Endly's `secret` service delegates signing and secret handling to SCY. Use
`secret:signJWT` with the allocated fixture user's identifiers and the owned test
signing key; use the exact claim names required by the target authority. The
Platform workflow passes the returned token through the ordinary HTTP request:

```yaml
Header:
  Authorization: ["Bearer ${signed.TokenString}"]
```

Do not assume a newer `bearerTokenSecret` HTTP field exists in the installed
runner: inspect `endly_service_info` first. For supported native JWT test flows,
use the normal header rather than diagnosing an omitted header as an IdP failure.
Keep credentials out of displayed diagnostics. Validate the actual identity/roles
returned by the application; fixture claims alone do not prove authorization.

## Verify actual hydration

Check published per-case IDs, stored primary/foreign-key relationships, and
application-visible data after required cache/index refresh. A conversion success
or HTTP 200 alone is insufficient. Native Endly validators should compare fixed
fixture expectations; never derive expected results from the response under test.
Static dictionaries and mutable case rows are separate prerequisites of focused
runs, even when infrastructure/build tasks are intentionally skipped.

Implementation: `service/testing/dsunit/udf.go` (`convertMultiTables`,
`allocateTableSequence`, `expandSequenceExpr`), its sequence tests, and
`service/testing/dsunit/sequences/e2e/run.yaml`. Inspect the installed contracts
through `endly_service_info` before changing action fields.
