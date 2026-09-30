---
name: endly-dsunit
description: "Author and run Endly dsunit service actions and diagnose their requests, responses, and workflow integration."
---

Use this skill for `dsunit` actions in Endly YAML workflows.
Inspect the installed contract with `endly -s=dsunit` and
`endly -s=dsunit -a=<action>` before adding fields; these commands only print metadata.
Use `action: dsunit:<action>` in a pipeline task. Expand state with `$variable`
or `${variable}`; use `init` for inputs and `post` to publish results for later tasks.

Source-declared actions: `checkSchema`, `compare`, `create`, `dump`, `expect`, `freeze`, `init`, `mapping`, `prepare`, `query`, `register`, `script`, `sequence`, `sql`.

Read [the service reference](references/service.md) for live contract discovery and maintained source locations.

Function skills: `endly-dsunit-register`, `endly-dsunit-sequences`, `endly-dsunit-prepare`, `endly-dsunit-expect`, `endly-dsunit-query`, `endly-dsunit-export`, `endly-dsunit-mapping`, `endly-dsunit-hydration`, `endly-dsunit-batch-validation`. Retrieve only those relevant to the task.

Source: `service/testing/dsunit/service.go`. Treat the live contract as authoritative when documentation differs.
Run only the actions and environments covered by the user request. A skill documents execution; it does not grant additional authorization. For workflow selection and test design, retrieve the `endly-authoring` and `endly-testing` skills from the same catalog.
