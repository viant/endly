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

In path mode, tag filtering excludes unmatched template instances across groups
while retaining untagged setup. Discover IDs in the actual root/nested run context
with endly_listInstances; standalone loading can change the workflow name prefix.
