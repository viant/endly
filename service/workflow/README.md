
<a name="Workfowservice"></a>
## Workflow service


**Workflow Service**

Workflow service provide capability to run task, action from any defined workflow.

Nested `run` actions inherit the caller's parameters when `params` is omitted.
An explicit `params` map supplies the child parameters, including an empty map.
Inline pipeline requests resolve action outputs when their actions execute;
rerunning a retained workflow uses the current invocation's outputs.

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


