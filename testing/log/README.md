##Log validation service
 


### Usage:
 
   1) register log listener, to dynamically detect any log changes (log shrinking/rotation is supported), any new log records are queued to be validated.
   2) run log validation. Log validation verifies actual and expected log records, shifting record from actual logs pending queue.
   3) reset - optionally reset log queues, to discard pending validation logs.

### Supported actions:



| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| validator/log | listen | start listening for log file changes on specified location  |  [ListenRequest](service_contract.go) | [ListenResponse](service_contract.go)  |
| validator/log | reset | discard logs detected by listener | [ResetRequest](service_contract.go) | [ResetResponse](service_contract.go)  |
| validator/log | assert | perform validation on provided expected log records against actual log file records. | [AssertRequest](service_contract.go) | [AssertResponse](service_contract.go)  |



### Validation strategies:

A log validation verifies produced by a logger with desired log records.

Once log listener detects data produce by a logger it places it to the pending validation queue, 
then later when assert request takes place,  validator takes (and removes) records from pending validation 
corresponding to expected records. This process may use either _position_ or _index based_ matching.

The first strategy,  a matcher takes the older record from the pending validation qeueue (FIFO) for each expected record.
The later strategy  requires an indexing expression (provided in listen request IndexRegExpr i.e. \"UUID\":\"([^\"]+)\" ) which is used for both
indexing pending logs and desired logs. If expected record can not be matched with indexing expression it falls back to the first strategy.



### Example

Standalone testing workflow example:

 
```bash
endly -r=run
```


[@run.yaml](test/run.yaml)

```yaml
init:
  logLocation: /tmp/logs
  target:
    url:  ssh://127.0.0.1/
    credentials: ${env.HOME}/.secret/localhost.json
defaults:
  target: $target
pipeline:
  init:
    action: exec:run
    commands:
      - mkdir -p $logLocation
      - "> ${logLocation}/myevents.log"
      - echo '{"EventID":111, "EventType":"event1", "X":11111111}' >> ${logLocation}/myevents.log
      - echo '{"EventID":222, "EventType":"event2", "X":11141111}' >> ${logLocation}/myevents.log
      - echo '{"EventID":333, "EventType":"event1","X":22222222}' >>  ${logLocation}/myevents.log
  listen:
    action: validator/log:listen
    frequencyMs: 500
    source:
      URL: $logLocation
    types:
      - format: json
        inclusion: event1
        mask: '*.log'
        name: event1
  validate:
    action: validator/log:assert
    logTypes:
      - event1
    description: E-logger event log validation
    expect:
      - type: event1
        records:
         - EventID: 111
           X: 11111111
         - EventID: 333
           X: 22222222
    logWaitRetryCount: 2
    logWaitTimeMs: 5000
```


**As part of workflow**

   [See more](https://github.com/viant/endly/tree/master/example/rt/elogger)


