---
name: endly-http-mocks
description: Orchestrate Endly HTTP mock servers and recorded conversations with deterministic matching, per-case responses, and cleanup.
---

Inspect http/endpoint listen/append/shutdown through endly_service_info. Start the
mock before the application sends requests. IndexKeys should include the fields
that distinguish conversations (URL, Method, and a correlation header when needed).
Avoid indexing on unstable values unless the fixture deliberately matches them.
Recorded trips or a request/response template supply the dependency's responses.
Append only the conversations for the selected case; keep request ordering and
rotation rules explicit when one key has multiple responses.

Point the application at the local mock endpoint and isolate the port or request
key across concurrent sessions. Reusing a listening port can return an existing
endpoint rather than replacing its configuration; do not assume a second listen
resets old trips. Shut down the mock in the workflow's cleanup/deferred task.

Validate the application's response and the downstream request observed by the
mock when both are part of the behavior. A request that never reaches the mock,
an unexpected key, and a wrong response payload are different failures. Preserve
correlation IDs so batch log assertions identify the originating use case.
For response-driven follow-up traffic, retrieve endly-http-followup.
