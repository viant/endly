
<a name="Workfowservice"></a>
## Workflow service


**Workflow Service**

Workflow service provide capability to run task, action from any defined workflow.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| workflow | run | run workflow  or inline pipelines with specified tasks and parameters | [RunRequest](service_workflow_contract.go) | [RunResponse](service_workflow_contract.go) |
| workflow | goto | switch current execution to the specified task on current workflow | [GotoRequest](service_workflow_goto.go) | [GotoResponse](service_workflow_contract.go) 
| workflow | switch | run matched  case action or task  | [SwitchRequest](service_workflow_contract.go) | [SwitchResponse](service_workflow_contract.go) |
| workflow | exit | terminate execution of active workflow (caller) | n/a | n/a |
| workflow | fail | fail  workflow | [FailRequest](service_workflow_contract.go) | n/a  |


**Predefined workflows**

<a name="predefined_workflows">	</a>
**Predefined workflows**

Predefined set provides commonly used workflow for services, app to build, deploy and testing. 

[Workflows](../shared/workflow)


