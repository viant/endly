##Log validation service
 


### Usage:
 
   1) register log listener, to dynamically detect any log changes (log shrinking/rotation is supported), any new log records are queued to be validated.
   2) run log validation. Log validation verifies actual and expected log records, shifting record from actual logs pending queue.
   3) reset - optionally reset log queues, to discard pending validation logs.

### Supported actions:



| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| validator/log | listen | start listening for log file changes on specified location  |  [ListenRequest](contract.go) | [ListenResponse](contract.go)  |
| validator/log | reset | discard logs detected by listener | [ResetRequest](contract.go) | [ResetResponse](contract.go)  |
| validator/log | assert | perform validation on provided expected log records against actual log file records. | [AssertRequest](contract.go) | [AssertResponse](contract.go)  |


### Validation strategies:

A log validation verifies produced by a logger with a user provides a desired log records in the asset request.Expect[logTypeIndex].Records
Any arbitrary data structure can represent records.


Once a log/validator listener detects data produce by a logger, it places it to the pending validation queue, 
then later when assert request takes place,  validator takes (and removes) records from pending validation queue 
to match and validate with expected records.


This process may use either _position_ or _index based_ matching method.
In the first strategy,  a matcher takes the first record from the pending validation queue (FIFO) for each expected record.
The latter strategy  requires an indexing expression (provided in listen request IndexRegExpr i.e. \"UUID\":\"([^\"]+)\" ) which is used for both
indexing pending logs and desired logs. If the validator is unable to match record with indexing expression, it falls back to the position based one.

Validator also supports data transformation on the fly just before validation with [UDF](../../doc/udf)

Actual validation is delegated to [assertly](http://github.com/viant/assertly/)

### Examples

#### Standalone testing workflow example:**

 
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


_with request delegation:_


```bash
endly -r=test
```


[@test.yaml](test/test.yaml)
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
    request: '@exec.yaml'
  listen:
    action: validator/log:listen
    request: '@listen.yaml'
  validate:
    action: validator/log:assert
    request: '@validate.json'
```

[@exec.yaml](test/exec.yaml)
```yaml
commands:
  - mkdir -p $logLocation
  - "> ${logLocation}/myevents.log"
  - echo '{"EventID":111, "EventType":"event1", "X":11111111}' >> ${logLocation}/myevents.log
  - echo '{"EventID":222, "EventType":"event2", "X":11141111}' >> ${logLocation}/myevents.log
  - echo '{"EventID":333, "EventType":"event1","X":22222222}' >>  ${logLocation}/myevents.log
```

[@listen.yaml](test/listen.yaml)
```yaml
frequencyMs: 500
source:
  URL: $logLocation
types:
  - format: json
    inclusion: event1
    mask: '*.log'
    name: event1
```


[@validate.json](test/validate.json)
```json
{
  "Expect": [
    {
      "type": "event1",
      "records": [
        {
          "EventID": 111,
          "X": 11111111
        },
        {
          "EventID": 333,
          "X": 22222222
        }
      ]
    }
  ]
}
```


#### Workflow with csv UDF example

```bash
endly -r=csv
```

[csv.yaml](csv.yaml)
```yaml
init:
  i: 0
  j: 0
  logLocation: /tmp/logs
  target:
    url:  ssh://127.0.0.1/
    credentials: ${env.HOME}/.secret/localhost.json
defaults:
  target: $target
pipeline:
  init:
    make-dir:
      action: exec:run
      commands:
      - mkdir -p $logLocation
      - "> ${logLocation}/events.csv"
    register-udf:
      action: udf:register
      udfs:
        - id: UserCsvReader
          provider: CsvReader
          params:
            - id,type,timestamp,user
    listen:
      action: validator/log:listen
      frequencyMs: 500
      source:
        URL: $logLocation
      types:
        - format: json
          inclusion: event1
          mask: '*.csv'
          name: event1
          UDF: UserCsvReader
          debug: true
  test:
    multiAction: true
    produce:
      async: true
      repeat: 6
      sleepTimeMs: 400
      action: exec:run
      commands:
        - echo '$i++,event1,${timestamp.now},user $j++' >> ${logLocation}/events.csv

    validate:
      action: validator/log:assert
      logTypes:
        - event1
      description: E-logger event log validation
      expect:
      - type: event1
        records:
          - id: 0
            user: user 0
          - id: 1
            user: user 1
          - id: 2
            user: user 2
          - id: 3
            user: user 3
          - id: 4
            user: user 4
          - id: 5
            user: user 5

      logWaitRetryCount: 10
      logWaitTimeMs: 2000
```


**As part of workflow**

   [See more](https://github.com/viant/endly/tree/master/example/rt/elogger)



