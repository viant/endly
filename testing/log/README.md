**Log validation service** 

To get log validation, 
   1) register log listener, to dynamically detect any log changes (log shrinking/rotation is supported), any new log records are queued to be validated.
   2) run log validation. Log validation verifies actual and expected log records, shifting record from actual logs pending queue.
   3) reset - optionally reset log queues, to discard pending validation logs.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| validator/log | listen | start listening for log file changes on specified location  |  [ListenRequest](service_contract.go) | [ListenResponse](service_contract.go)  |
| validator/log | reset | discard logs detected by listener | [ResetRequest](service_contract.go) | [ResetResponse](service_contract.go)  |
| validator/log | assert | perform validation on provided expected log records against actual log file records. | [AssertRequest](service_contract.go) | [AssertResponse](service_contract.go)  |


