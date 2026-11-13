---
name: endly-storage
description: "Author and run Endly storage service actions and diagnose their requests, responses, and workflow integration."
---

Use this skill for `storage` actions in Endly YAML workflows.
Inspect the installed contract with `endly -s=storage` and
`endly -s=storage -a=<action>` before adding fields; these commands only print metadata.
Use `action: storage:<action>` in a pipeline task. Expand state with `$variable`
or `${variable}`; use `init` for inputs and `post` to publish results for later tasks.

Source-declared actions: `copy`, `create`, `download`, `exists`, `generate`, `list`, `remove`, `upload`.

When a downloaded YAML, JSON or Markdown file is needed as text, declare
`udf: AsString`. The default payload representation encodes non-ASCII content as
`base64:...`; valid UTF-8 punctuation can therefore change an otherwise working
ASCII-only workflow. Do not pass that encoded payload to an API expecting text.

```yaml
sourceDocument:
  action: storage:download
  source: {URL: '${workPath}/document.yaml'}
  destKey: sourceText
  udf: AsString
```

Keep binary downloads explicit (`Base64Encode`, for example, when an API expects
an archive). `IsJSON` takes a file/resource name, not an in-memory JSON payload.
Validate payload structure with native validators before writes.

For large authored text fixtures, `expand: true` alone is insufficient: the
default copy matcher skips substitution for files of 128 KiB or larger. Use the
existing explicit matcher when those files intentionally contain state tokens:

```yaml
fixture:
  action: storage:copy
  source: {URL: '${projectPath}/fixtures/facts.json'}
  dest: {URL: '${workPath}/facts.json'}
  expand: true
  expandIf: {suffix: '.json'}
```

Verify the copied fixture contains resolved IDs before hydration. Limit expansion
to the intended authored files; do not expand arbitrary downloaded data or binary
archives. This is the native matcher override, not a reason to add a separate
template script.

Read [the service reference](references/service.md) for live contract discovery and maintained source locations.

Source: `service/system/storage/service.go`. Treat the live contract as authoritative when documentation differs.
Run only the actions and environments covered by the user request. A skill documents execution; it does not grant additional authorization. For workflow selection and test design, retrieve the `endly-authoring` and `endly-testing` skills from the same catalog.
