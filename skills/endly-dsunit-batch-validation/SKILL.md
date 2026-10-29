---
name: endly-dsunit-batch-validation
description: Validate Endly batches by use-case IDs across database expectations, asynchronous logs, and aggregate outputs with bounded waits.
---

Collect expected outcomes per case while executing the case, carrying a stable
TagId/correlation ID and the selected fixture IDs. Append expected records to a
batch collection rather than overwriting another case's expectations. Keep payload
expectations separate from status-only assertions; a successful request can still
produce incorrect persisted rows or logs.

## Grouped database test flow

For grouped integration/E2E tests, use this order:

1. Define and hydrate dedicated sequence-backed fixtures for each case before its
   operations. Keep each actor, authorization scenario and mutable record identity
   independent. Shared schema and static dictionaries can remain suite setup.
2. Execute the selected cases and accumulate their expected persisted outcomes.
   Resolve each case's IDs and values in its own context before advancing; do not
   leave references that will resolve against the next case's tag or actor.
3. After the group operations complete, run `dsunit:expect` over the accumulated
   records, once per participating datastore. Keep every original expected value,
   matching/check policy, null/omitted distinction, false and zero.
4. Treat operation/UI success as provisional until the final database batch passes.
   Preserve operation failures and batch failures separately. Never count skipped,
   unexecuted or unverified cases as passed.

Use existing template `data` aggregation, native append (`->`), and
`$AsTableRecords` where appropriate. A typical runtime accumulation is:

```yaml
init:
  expected:
    records: []
pipeline:
  # Run inside each case after its dedicated hydration and tested operation.
  accumulate:
    action: nop
    init:
      '->expected.records': {ID: '$caseRecord.ID', VALUE: '$expectedValue'}
  # Run outside the case template, after all selected group operations.
  verifyGroup:
    action: dsunit:expect
    datastore: example_store
    checkPolicy: 1
    data: $AsTableRecords('expected')
```

The snippet illustrates placement; it is not a standalone fixture setup. Preserve
existing project policies rather than copying the example's policy blindly.
Retain table directives once when combining physical-table expectations. Keep
case-specific `@fromQuery@` datasets under distinct logical aliases inside the
same final batch; do not overwrite one case's SQL or combine unrelated scopes.
Do not add a new collector service or external verification runner when these
existing Endly primitives cover the pattern. Business schemas, fixture data,
credentials and private endpoints belong in the application repository; public
Endly skills and regression examples must remain domain-neutral.

Use bounded readiness/retry checks only for real asynchronous completion. The
batching pattern does not require fixed sleeps, inflated timeouts, or weaker
performance limits. Measure speed improvements rather than inferring them from
fewer actions alone.

Start validator/log:listen and reset its observation buffer before the batch.
Configure format, file mask and correlation extraction so records map to the
correct use case. After request generation, validate the accumulated expectations
with validator/log:assert using bounded LogWaitTimeMs/LogWaitRetryCount. Preserve
per-case descriptions and IDs in failures. Avoid turning a long arbitrary sleep
into evidence of asynchronous completion; use the harness's retry/readiness checks.

For persisted/aggregate outcomes, use dsunit:expect or query with Expect after
writes have settled. Inspect the live checkPolicy and matching options before
changing them; never weaken a matching policy just to accept unrelated rows.
Validate both expected presence and unwanted extras when the contract requires it.
Preserve mappings, dictionary prerequisites and per-case hydration throughout.

Focused reruns must narrow both executed cases and accumulated expectations;
otherwise a batch validator can report errors for cases that were not executed.
Use actual discovered TagIDs and inspect assertion counts. Operation status is
failed when assertions fail; zero assertions or a skipped case is not a pass.
Retrieve operation_events pages filtered to assertion/failure types and inspect
only the state paths needed to diagnose the mismatched case.

When the listener can fall back to FIFO before a correlation ID is indexed, use
a case-specific inclusion rule or queue so it cannot consume an unrelated earlier
record. Preserve that rule in a focused run; a longer wait alone does not repair
ambiguous matching.

Dispatch database checks with endly_dsunit_expect/query/compare and correlated log checks with endly_validator_log_listen/reset/assert. Each action returns an operation; check status and assertion counts before continuing.
