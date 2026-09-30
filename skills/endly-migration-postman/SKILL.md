---
name: endly-migration-postman
description: "Author and run Endly migration/postman service actions and diagnose their requests, responses, and workflow integration."
---

Use this skill for `migration/postman` actions in Endly YAML workflows.
Inspect the installed contract with `endly -s=migration/postman` and
`endly -s=migration/postman -a=<action>` before adding fields; these commands only print metadata.
Use `action: migration/postman:<action>` in a pipeline task. Expand state with `$variable`
or `${variable}`; use `init` for inputs and `post` to publish results for later tasks.

Source-declared actions: `postman`.

Source: `service/migration/postman/service.go`. Treat the live contract as authoritative when documentation differs.
Run only the actions and environments covered by the user request. A skill documents execution; it does not grant additional authorization. For workflow selection and test design, retrieve the `endly-authoring` and `endly-testing` skills from the same catalog.
