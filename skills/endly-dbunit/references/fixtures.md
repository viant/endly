# Generation and mapped fixture composition

Read the Generation, sequences and mapped fixtures section of
`service/testing/dbunit/README.md` in the matching Endly checkout for the current
contracts and two-database parent/child example.

`hydrate` uses ordered tables/generation, explicit tag fields/references and
captures. Each `FixtureSequence.seed.database` selects a registered binding;
seeds are snapshots, not concurrent ID reservations. Continue a local counter
with a literal `startFrom: prior.sequences.ids.next` path. Repeated calls start
fresh, and UUIDs are not cached implicitly.

For several parents, retain a tag per parent and use a per-row reference such
as `{field: [parent_id], table: parents, tag: '$row.parentTag', from: [id]}`.
Avoid a shared last-parent alias. Publish private Before/PostIncrement aliases
only through explicit `stateCaptures`.

`map` accepts an inline/file `source` or `sourceFrom: generated.tables[0].source`.
Its result has `datasets`; prepare with `datasetsFrom: projected.datasets`.
These selectors are literal field/index paths, without `$`, and preserve resolved
source markers. Defaults are literal and absent-only; conflicting duplicate keys
fail. For tables targeting different databases use separate map/prepare actions.

The new actions do not register or replace the legacy global `AsTableRecords`.
