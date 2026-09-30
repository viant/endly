---
name: endly-http-followup
description: Chain Endly HTTP requests by delegating IDs, headers, URLs, and payloads from one response into subsequent calls and validations.
---

Use http/runner:send and inspect its live contract with endly_service_info.
The named action publishes Responses; derive subsequent values from
`${create.Responses[0].JSONBody.id}` or `${create.Responses[0].Header.<Name>[0]}`
and use init/post to give them clear state names. Assert the first response before
using its data; do not silently reuse an old value when the first request failed.

Keep request templates, raw response bodies and structured JSON distinct. Use
LoadJSON/AsData only for the format actually returned. RequestUdf/ResponseUdf can
encode/decode protocol data, and DataSource selects the repeater's extraction input.
Use the existing UDF and extraction contract rather than substituting arbitrary
string concatenation for a required encoding or URL-escaping step.

A dependency may return a follow-up URL or list of callbacks. Delegate only the
response fields required for the next request; preserve correlation/session IDs,
method, body and encoding. Distinguish application follow-up traffic from HTTP
redirects: FollowRedirects controls redirect policy, not a response-driven workflow.
Assert each request's status/payload and the eventual persisted or logged outcome.
Do not expose authorization headers or decoded credentials in print diagnostics.

The runner reuses equivalent clients across send actions, keeps cookies explicit,
and attaches the operation context for cancellation. Use bounded HTTP defaults and
state overrides for legitimately slow requests rather than indefinite retries.
For case hydration, mocks, or delayed batch effects retrieve the corresponding
endly-dsunit-hydration, endly-http-mocks and endly-dsunit-batch-validation skills.

A delegated action can load `request: "@request-template @payload-data"`, letting
resource overlays supply case values to a shared request template. Preserve overlay
order, defaults and escaping. Publish its Responses under the action name, validate
against the selected expectation resource, then carry the same case/correlation ID
into subsequent service calls and collected batch expectations.

Dispatch HTTP requests with endly_http_runner_send and assertions with endly_validator_assert. Use a loaded workflow for response extraction, init/post and ordered follow-up composition; direct tools preserve the same session state.
