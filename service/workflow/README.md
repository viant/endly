
<a name="Workfowservice"></a>
## Workflow service


**Workflow Service**

Workflow service provide capability to run task, action from any defined workflow.

Nested `run` actions inherit the caller's parameters when `params` is omitted.
An explicit `params` map supplies the child parameters, including an empty map.
Inline pipeline requests resolve action outputs when their actions execute;
rerunning a retained workflow uses the current invocation's outputs.

### Conditional work from prior outputs

`when` can evaluate nested fields and indexed collections from a completed action.
For example, if `probe` returned a structured application observation, a condition
can use `$probe.StructuredContent.apps[0].state:running`. These are Endly output
paths; they do not require an XML XPath document or a separate orchestration layer.
An explicit `post` can retain the observation for later steps:

```yaml
post:
  currentApp: $StructuredContent
```

Use positive evidence for each branch. A readiness workflow can inspect once,
open only when `$currentApp.apps[0].state:stopped`, re-inspect only after the current
open action has reported success, and run dependent work when
`$currentApp.apps[0].state:running && $currentApp.apps[0].ready:true`.
The response shape here is illustrative: the selected service must actually
provide these observations. A running process alone does not prove that the
intended document or inbox is usable.

Missing fields, an empty collection, unknown state, and an unfinished operation
must not be interpreted as a stopped application or permission to retry input.
Poll an existing operation by its identifier rather than repeating its opening,
typing, export, or submission action. Add a final current-run result assertion and run with
`FailOnAssertion` enabled on the workflow run request, so skipped dependent work
cannot be mistaken for successful completion.

Shared contexts can retain results from a previous invocation when a step is
skipped. Reset invocation-local status variables in an unconditional first step,
then set them in the relevant action's `post`. For example, initialize
`currentLaunchStatus = notStarted`, publish
`currentLaunchStatus = $StructuredContent.executionStatus` from the open action,
and gate re-inspection with `$currentLaunchStatus:succeeded`. Similarly reset a
current-read completion marker and include it in the final assertion. Do not let
an old `$read` result satisfy completion when this invocation skipped the read.
Keep retained receipts/history separate from these current-run markers.

[The conditional readiness regression](conditional_readiness_test.go) exercises
this pattern through the workflow engine with synthetic service responses. It
checks that an already-ready app needs one inspection and no launch, a stopped
app opens once and is independently rechecked, and uncertain states fail the
readiness gate without launching again. It does not qualify a desktop executor
or a mailbox-reading adapter.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| workflow | run | run workflow  with specified tasks and parameters | [RunRequest](service_workflow_contract.go) | [RunResponse](service_workflow_contract.go) |
| workflow | goto | switch current execution to the specified task on current workflow | [GotoRequest](service_workflow_goto.go) | [GotoResponse](service_workflow_contract.go) 
| workflow | switch | run matched  case action or task  | [SwitchRequest](service_workflow_contract.go) | [SwitchResponse](service_workflow_contract.go) |
| workflow | exit | terminate execution of active workflow (caller) | n/a | n/a |
| workflow | fail | fail  workflow | [FailRequest](service_workflow_contract.go) | n/a  |


**Predefined workflows**

<a name="predefined_workflows">	</a>
**Predefined workflows**

Predefined set provides commonly used workflow for services, app to build, deploy and testing. 

[Workflows](../shared/workflow)


