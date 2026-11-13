---
name: endly-validator
description: "Author and run Endly validator service actions and diagnose their requests, responses, and workflow integration."
---

Use this skill for `validator` actions in Endly YAML workflows.
Inspect the installed contract with `endly -s=validator` and
`endly -s=validator -a=<action>` before adding fields; these commands only print metadata.
Use `action: validator:<action>` in a pipeline task. Use `init` for inputs and
`post` to publish results for later tasks.

For an existing captured object or collection, use its literal state path, for
example `actual: captured.Transactions`. The validator resolves that path
directly. `actual: $captured.Transactions` instead expands and copies the graph
before validation, potentially reinterpreting captured `$` expressions as code.
Keep `$` expressions for authored computations such as collection counts.
Preserve positive count/`each` gates; a state reference does not establish that
the expected requests occurred. Do not apply this shortcut to `expect`, whose
authored variables still need expansion. With `ignore`/`omitEmpty`, inspect the
service's filtering order before changing an existing request to a state path.

For an independently specified decimal precision, reuse Assertly's existing
`@numericPrecisionPoint@` directive instead of adding a numeric export parser.
Keep the captured values unchanged. For example:

```yaml
init:
  - name: observed
    value: {value: 0.2436781609195402}
pipeline:
  compare:
    action: validator:assert
    normalizeKVPairs: true
    actual: observed
    expect:
      '@numericPrecisionPoint@': 6
      value: 0.24367816091954023
```

The directive rounds numeric comparisons to the declared decimal places; it is
not an absolute-error bound. `normalizeKVPairs` preserves this authored YAML map
as a directive-bearing expectation. Keep row order, count, null and cell-type
checks separate. Do not lower precision to hide a mismatch or apply it to JSON
strings: `json:0.24` is a literal string, not a numeric value.

Source-declared actions: `assert`.

Read [the service reference](references/service.md) for live contract discovery and maintained source locations.

Source: `service/testing/validator/service.go`. Treat the live contract as authoritative when documentation differs.
Run only the actions and environments covered by the user request. A skill documents execution; it does not grant additional authorization. For workflow selection and test design, retrieve the `endly-authoring` and `endly-testing` skills from the same catalog.
