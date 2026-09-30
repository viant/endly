---
name: endly-gcp-cloudfunctions
description: "Author and run Endly gcp/cloudfunctions service actions and diagnose their requests, responses, and workflow integration."
---

Use this skill for `gcp/cloudfunctions` actions in Endly YAML workflows.
Inspect the installed contract with `endly -s=gcp/cloudfunctions` and
`endly -s=gcp/cloudfunctions -a=<action>` before adding fields; these commands only print metadata.
Use `action: gcp/cloudfunctions:<action>` in a pipeline task. Expand state with `$variable`
or `${variable}`; use `init` for inputs and `post` to publish results for later tasks.

Source-declared actions: `call`, `delete`, `deploy`, `get`, `list`.

Read [the service reference](references/service.md) for live contract discovery and maintained source locations.

Source: `service/system/cloud/gcp/cloudfunctions/service.go`. Treat the live contract as authoritative when documentation differs.
Run only the actions and environments covered by the user request. A skill documents execution; it does not grant additional authorization. For workflow selection and test design, retrieve the `endly-authoring` and `endly-testing` skills from the same catalog.
