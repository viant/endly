# Amazon CloudWatch Logs service

This service is github.com/aws/aws-sdk-go/service/cloudwatchlogs.CloudWatchLogs proxy 

To check all supported method run
```bash
    endly -s="aws/logs"
```

To check method contract run endly -s="aws/logs" -a=methodName
```bash
    endly -s="aws/logs" -a='filterLogs'
```

Op top of that

fetchLogEvents

#### Usage:



##### Filter logs

```bash
endly -r=filterLogs.yaml
```


@filterLogs.yaml
```yaml
pipeline:
  task1:
    action: aws/logs:filterLogEvents
    credentials: aws
    loggroupname: "/aws/lambda/MsgLogFn"
    starttime: ${timestamp.5HoursAgoInUTC}    
```


##### Fetching logs with message based exclusion/inclusion (on client side)

```bash
endly -r=filterLogs.yaml
```

@filterLogs.yaml
```yaml
pipeline:
  task1:
    action: aws/logs:filterLogEvents
    credentials: aws
    loggroupname: "/aws/lambda/MsgLogFn"
    starttime: ${timestamp.5HoursAgoInUTC}    
```




#### Log based validation


```bash
endly -r=test
```

@test.yaml
```yaml
pipeline:
  fetchLogEvent:
    action: aws/logs:filterLogEventMessages
    logging: false
    credentials: $awsCredentials
    loggroupname: /aws/lambda/MsgLogFn
    starttime: $startTimestamp
    include:
      - '{'
  assert:
    action: validator:assert
    normalizeKVPairs: true
    actual: ${fetchLogEvent.Messages}
    expect:
      - '@indexBy@': messageAttributes.id.stringValue

      - body: /this is my 3 message/
        messageAttributes:
          id:
            stringValue: 3


      - body: /this is my 4 message/
        messageAttributes:
          id:
            stringValue: 4
```
