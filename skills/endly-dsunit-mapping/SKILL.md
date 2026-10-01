---
name: endly-dsunit-mapping
description: "Configure Endly dsunit dataset mappings and relationships for fixture loading and assertions."
---

Use `dsunit:mapping` with the live MappingRequest contract. Inspect existing mapping files and the full service reference to preserve logical dataset/table names, key relations and query behavior. Register mappings before dependent fixture or expectation operations. Validate a representative prepared and expected dataset after changing mapping rules; do not assume a renamed fixture still maps to the same table.

Inspect `endly -s=dsunit -a=<action>` for the installed contract. Retrieve `endly-dsunit` for the full reference through MCP skills/get or endly_skill_get; its resource inventory includes references/service.md, readable with resources/read. Use endly-testing for test methodology.

A mapping's Name identifies the logical dataset and Table the physical target.
Columns can rename source fields with FromColumn and specify DefaultValue,
Required, and Unique. Associations expand related rows into additional tables.
Preserve parent/child keys and uniqueness across associations. MappingResponse.Tables
is the physical table inventory for subsequent dsunit:sequence lookup; publish
its values under the sequence namespace expected by the fixture before conversion.
For the full setup/hydration dependency chain retrieve endly-dsunit-hydration.

Dispatch mapping directly with endly_dsunit_mapping; its request is the native MappingRequest. Observe MappingResponse through the resulting operation.
