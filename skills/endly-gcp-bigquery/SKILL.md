---
name: endly-gcp-bigquery
description: "Author and run Endly gcp/bigquery service actions and diagnose their requests, responses, and workflow integration."
---

Use this skill for `gcp/bigquery` actions in Endly YAML workflows.
Inspect the installed contract with `endly -s=gcp/bigquery` and
`endly -s=gcp/bigquery -a=<action>` before adding fields; these commands only print metadata.
Use `action: gcp/bigquery:<action>` in a pipeline task. Expand state with `$variable`
or `${variable}`; use `init` for inputs and `post` to publish results for later tasks.

Source-declared actions: `copy`, `jobWait`, `load`, `patch`, `query`, `table`.

Read [the service reference](references/service.md) for live contract discovery and maintained source locations.

Source: `service/system/cloud/gcp/bigquery/service.go`. Treat the live contract as authoritative when documentation differs.
Run only the actions and environments covered by the user request. A skill documents execution; it does not grant additional authorization. For workflow selection and test design, retrieve the `endly-authoring` and `endly-testing` skills from the same catalog.
