---
name: endly-testing
description: Design and diagnose Endly integration and e2e tests with isolated fixtures, sequence-backed IDs, API and database assertions, and focused reruns.
---

Inspect a representative existing case before creating another. Separate fixture
setup, the operation under test, and observable assertions. Use fixture-generated
IDs rather than database-specific constants. Keep cross-table references symbolic,
load sequence values before AsTableRecords, and prepare fixtures before requests.
Avoid suite-order dependencies; a focused case must work after the declared setup.

Assert HTTP status and response shape, then persisted rows or side effects when
those are the behavior under test. Cover authorization roles, invalid input, missing
entities, and boundaries relevant to the change. Do not weaken an expectation to
match a failure without checking the contract. Use deterministic timestamps or
bounded comparisons where the application generates time. Preserve cleanup and
error tasks; avoid concurrent runs sharing mutable databases or sequence state.

Start with a focused case (`-t=test -i=<exact-tag-id>`), inspect emitted case IDs and
assertion counts, then broaden to affected groups and the complete suite. A zero
exit code without executed assertions does not demonstrate success. Separate
prerequisite failures from product failures. Retain command, revision, selected
cases, summary, and failure evidence; redact tokens and credentials from reports.
For nested tasks use `-selector-mode=path` and order paths to satisfy dependencies.

A request to run a suite authorizes its documented setup and test effects in the
chosen test environment. Stop if the configuration unexpectedly targets production,
required credentials are missing, or repeated retries would mutate unrelated data.
Do not treat retrieving a skill as authorization to execute it.
