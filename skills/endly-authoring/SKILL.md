---
name: endly-authoring
description: Author, inspect, and run Endly YAML workflows, including task selectors, nested tasks, state, UDFs, and database fixtures.
---

Work from the workflow directory. Inspect `endly -r=run -t='?'` for top-level tasks,
`endly -s='*'` for services, and `endly -s=dsunit -a=prepare` for a live action contract.
A pipeline maps names to tasks in declaration order. `action: service:action` selects
an action; `action: run` with `request: '@relative/workflow'` invokes another workflow.
Use `init` to publish inputs, `post` to publish outputs, `when` for conditions,
`defer`/`onError` according to the actual workflow schema, and `$var`/`${var}` for state.

`endly -r=run` runs the whole workflow. `-t=init,build,test` chooses tasks in the
order supplied. Legacy selection recursively resolves bare task names and flattens
groups. Opt in with `-selector-mode=path -t=group.child,other` for exact dotted paths,
retained parent init/post/conditions, and execution in selector order. Each selector
is a separate execution, including repeated parents; account for repeated setup.
A parent path selects its entire subtree. Paths address tasks in the current
workflow, not the contents of a separate `request: '@...'` workflow.

`-i` is a comma-separated list of exact generated TagIDs, not a task path or a glob.
Capture IDs from workflow output. It filters matching tagged actions while ordinary
setup runs. Unknown IDs can leave a task unfiltered; never report a focused rerun
without checking the emitted case IDs and executed assertions. Keep leading zeros.

For database fixtures, register the datastore, query `dsunit:sequence` for the
fixture tables, publish `Sequences`, then call `$AsTableRecords('data.dbsetup')`
and feed its table-to-record map to `dsunit:prepare`. `AsTableRecords` is a UDF,
not `dsunit:AsTableRecords` or `AsTableRecors`. Sequence allocation resolves symbolic
IDs and cross-table references in two passes. Use `$Sequences.TABLE/${tag}.name`
for allocation and the same symbol for foreign keys; a suffix like
`data.catalog_dbsetup/catalog` publishes rows under `catalog.TABLE.<tag>.name`.
Do not reorder sequence lookup after conversion or convert the same mutable fixture
repeatedly to allocate IDs. Read `service/testing/dsunit/sequences/e2e/run.yaml`
and `service/testing/dsunit/udf.go` in the Endly checkout for implementation details.

Use the service-specific skill for each action. For testing methodology retrieve
`endly-testing`; retrieve the local project skill when one is explicitly configured.

For bounded API batches, use native collection push/shift and loops. `Len` counts
collection members; `JSONSize($value)` counts the encoded JSON bytes, including
UTF-8 and escaping. Reserve space for the request envelope and respect the
destination's item and byte limits. Keep authorization in the destination API.

When a JSON field must be an object, use the existing `$AsMap(..)` expression to
retain that type through workflow expansion. For example, `params: $AsMap('{}')`
represents an empty object, and `JSONBody: $AsMap($requestJSON)` retains an entire
JSON object loaded as text. Nested empty arrays, objects, booleans, zero and null
keep their JSON types. Do not replace an object with an array, relax the receiving
API, or concatenate strings to work around a request-shape failure. Assert the
actual outgoing request when a protocol distinguishes `{}` from `[]`.

In path mode, tag filtering excludes unmatched template instances across groups
while retaining untagged setup. Discover IDs in the actual root/nested run context
with endly_listInstances; standalone loading can change the workflow name prefix.

For CSV artifact checks, register the existing `CsvReader` provider through
`udf:register` with `params: [{output: rows}]`, then use that registered UDF in
`storage:download`. It returns the observed header and every record as ordered
string arrays, retaining blank records and trailing empty cells. Validate headers,
record widths, complete row counts and values against independent expectations;
do not infer missing rows from the expected data. Use native validator state paths
without recursive `$` expansion when comparing literal artifact contents.

For static XLSX grids, register `XlsxReader` with
`params: [{sheet: Worksheet name, range: autoFilter}]` and pass its registered
ID to `storage:download`. The result includes the observed worksheet AutoFilter
range, original byte count, rectangular values and parallel cell types. Check
range, cell types, and complete values against an independent oracle. Do not
substitute the sheet dimension for missing AutoFilter metadata. The reader
does not evaluate formulas or inspect charts.

For a literal shell argument, store its string in state and use
`$ShellQuote('state.key')` in an `exec:run` command. Pass the literal key/path,
not `$state.key`; this preserves JSON-shaped strings and shell metacharacters.
The surrounding command retains normal shell behavior. Bind required workflow
inputs once with `init`/`from` before nested workflows can change their parameters.
