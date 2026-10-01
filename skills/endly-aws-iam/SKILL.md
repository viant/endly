---
name: endly-aws-iam
description: "Author and run Endly aws/iam service actions and diagnose their requests, responses, and workflow integration."
---

Use this skill for `aws/iam` actions in Endly YAML workflows.
Inspect the installed contract with `endly -s=aws/iam` and
`endly -s=aws/iam -a=<action>` before adding fields; these commands only print metadata.
Use `action: aws/iam:<action>` in a pipeline task. Expand state with `$variable`
or `${variable}`; use `init` for inputs and `post` to publish results for later tasks.

Source-declared actions: `getRoleInfo`, `getUserInfo`, `recreateRole`, `setupRole`.

Read [the service reference](references/service.md) for live contract discovery and maintained source locations.

Source: `service/system/cloud/aws/iam/service.go`. Treat the live contract as authoritative when documentation differs.
Run only the actions and environments covered by the user request. A skill documents execution; it does not grant additional authorization. For workflow selection and test design, retrieve the `endly-authoring` and `endly-testing` skills from the same catalog.
