**Execution services**

The execution service is responsible for opening, managing terminal session, with the ability to send command and extract data.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| exec | open | open SSH session on the target resource. | [OpenSessionRequest](service_contract.go) | [OpenSessionResponse](service_contract.go) |
| exec | close | close SSH session | [CloseSessionRequest](service_contract.go) | [CloseSessionResponse](service_contract.go) |
| exec | run | execute basic commands | [RunRequest](service_contract.go) | [RunResponse](service_contract.go) |
| exec | extract | execute commands with ability to extract data, define error or success state | [ExtractRequest](service_contract.go) | [RunResponse](service_contract.go) |
