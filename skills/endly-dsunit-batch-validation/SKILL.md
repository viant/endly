---
name: endly-dsunit-batch-validation
description: Validate Endly batches by use-case IDs across database expectations, asynchronous logs, and aggregate outputs with bounded waits.
---

Collect expected outcomes per case while executing the case, carrying a stable
TagId/correlation ID and the selected fixture IDs. Append expected records to a
batch collection rather than overwriting another case's expectations. Keep payload
expectations separate from status-only assertions; a successful request can still
produce incorrect persisted rows or logs.

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
