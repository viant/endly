---
name: endly-android
description: "Author and run Endly android service actions and diagnose their requests, responses, and workflow integration."
---

Use this skill for `android` actions in Endly YAML workflows.
Inspect the installed contract with `endly -s=android` and
`endly -s=android -a=<action>` before adding fields; these commands only print metadata.
Use `action: android:<action>` in a pipeline task. Expand state with `$variable`
or `${variable}`; use `init` for inputs and `post` to publish results for later tasks.

Source-declared actions: `artifact`, `attach`, `build`, `capture-start`, `capture-stop`, `cleanup`, `close`, `device-register`, `device-release`, `device-start`, `device-stop`, `doctor`, `install`, `launch`, `open`, `repl`, `run`, `server-start`, `server-stop`, `terminate`, `test`, `uninstall`.

Read [the service reference](references/service.md) for live contract discovery and maintained source locations.

Source: `service/testing/runner/android/service.go`. Treat the live contract as authoritative when documentation differs.
Run only the actions and environments covered by the user request. A skill documents execution; it does not grant additional authorization. For workflow selection and test design, retrieve the `endly-authoring` and `endly-testing` skills from the same catalog.
