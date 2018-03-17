# Execution service

The execution service is responsible for opening, managing terminal session, with the ability to send command and extract data.
Service keeps tracking the session current directory commands, env variable setting, and will only issue SSH command if there is actual change. 


- [Endly Workflow Integration](#endly)
- [SSH Session](#session)
- [Run Commands](#run)
- [Run Command to Extract Data](#extract)
- [SSH Unit Testing](#testing)


<a name="endly"></a>
## Endly service action integration

Run the following command for exec service operation details:

```bash

endly -s=exec

endly -s=exec -a=run
endly -s=exec -a=extract
endly -s=exec -a=open
endly -s=exec -a=close

```


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| exec | open | open SSH session on the target resource. | [OpenSessionRequest](service_contract.go) | [OpenSessionResponse](service_contract.go) |
| exec | close | close SSH session | [CloseSessionRequest](service_contract.go) | [CloseSessionResponse](service_contract.go) |
| exec | run | execute basic commands | [RunRequest](service_contract.go) | [RunResponse](service_contract.go) |
| exec | extract | execute commands with ability to extract data, define error or success state | [ExtractRequest](service_contract.go) | [RunResponse](service_contract.go) |


**RunRequest example**


@run.json

```json
{
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  },
  "SuperUser":true,
  "Commands":["mkdir /tmp/app1"]
}
```

@run.yaml

```yaml
target:
  url:  ssh://127.0.0.1/
  credential: ${env.HOME}/.secret/localhost.json
commands:
  whoami
  ${cmd[0].stdout}:/root/?  mkdir -p /tmp/app
  ${cmd[0].stdout}:!/root/? mkdir ~/app
  echo cmd[0].stdout
  
```

**ExtractRequest example**


```json
{
	"Target": {
	  "URL": "ssh://127.0.0.1/",
	  "Credential": "${env.HOME}/.secret/localhost.json"
	},
	"SystemPaths": ["/opt/sdk/go/bin"],
	"Commands": [
	  {
		"Command": "go version",
		"Extraction": [
		  {
			"RegExpr": "go(\\d\\.\\d)",
			"Key": "Version"
		  }
		]
	  }
	]
}
```



<a name="session"></a>
## SSH Session

In order to run any SSH command, service needs to open a session, it uses target.Credential and [secret service](https://github.com/viant/toolbox/tree/master/secret) to connect the target host.

Opening session is an optional step, run or extract request will open session automatically.

By default session is open in non transient mode, which means once context.Close is called, session will be will be terminated. Otherwise caller is responsible for closing it.


```go
    
        manager := endly.New()
        context := manager.NewContext(toolbox.NewContext())
        target := url.NewResource("ssh://127.0.0.1", "~/.secret/localhost.json")
        defer context.Close() // session closes as part of context.Close
        response, err := manager.Run(context, exec.NewOpenSessionRequest(target, []string{"/usr/local/bin"}, map[string]string{"M2_HOME":"/users/test/.m2/"},false, "/"))
        if err != nil {
            log.Fatal(err)
        }
        openResponse := response.(*exec.OpenSessionResponse)
        sessions :=context.TerminalSessions()
        assert.True(t,sessions.Has(openResponse.SessionID))
        log.Print(openResponse.SessionID)


``` 


**Run vs Extract:**

RunReuest provide a simple way to excute SSH command with conditional execution, it uses [util.StdErrors](https://github.com/viant/endly/blob/master/util/stdoututil.go#L16) as stdout errors.
ExtractRequest has ability to fine tune SSH command execution with extraction data ability. Error matching in ExtractRequest does use any default value.

<a name="run"></a>
## Run Commands

Command in RunRequest can represents one of the following:

1) Simple command: i.e echo $HOME   
2) Conditional command: [criteria ?] command
    i.e. $stdout:/root/? echo 'hello root'",  
       

```go

    manager := endly.New()
    context := manager.NewContext(toolbox.NewContext())
    var target= url.NewResource("ssh://127.0.0.1", "localhost")
    var runRequest = exec.NewRunRequest(target, true, "whoami", "$stdout:/root/? echo 'hello root'")
    var runResponse = &exec.RunResponse{}
    err := endly.Run(context, runRequest, runResponse)

```    

<a name="extract"></a>
## Run Command to Extract Data


```go

    extractRequest := exec.NewExtractRequest(target,
		exec.DefaultOptions(),
		exec.NewExtractCommand(fmt.Sprintf("svn info"), "", nil, nil,
			endly.NewDataExtraction("origin", "^URL:[\\t\\s]+([^\\s]+)", false),
			endly.NewDataExtraction("revision", "Revision:\\s+([^\\s]+)", false)))
    manager := endly.New()
    context := manager.NewContext(toolbox.NewContext())
    var runResponse := &exec.RunResponse{}
    err := endly.Run(context, extractRequest, runResponse)
    if err != nil {
        log.Fatal(err)
    }
  			
```


<a name="testing"></a>
## Exec SSH Unit Testing

This module provide  SSH session recording ability to later replay it during unit testing without actual SSH involvement.  

**Recroding SSH session**

To record actual SSH session use  exec.OpenRecorderContext helper method, the last parameters specify location where conversation is recorded, actual dump takes place when context is closed (defer context.Clode()).
If you use **sudo**. any **secret or credentials** make sure that you rename it to *** before checking in any code so you can use  `var credential, err = util.GetDummyCredential()`


```go
	manager := endly.New()
	target := url.NewResource("ssh://127.0.0.1", "~/.secret/localhost.json")
	context, err :=  exec.NewSSHRecodingContext(manager, target, "test/session/context")
	if err != nil {
		log.Fatal(err)
	}
	defer context.Close()

```


**Replaying SSH session**

In order to replay previously recoded SSH session use `exec.GetReplayService` helper method to create
a test SSHService, use location of stored SSH conversation  as parameter, then create context with `exec.OpenTestContext` 


```go
	manager := endly.New()
	var credential, err = util.GetDummyCredential()
	if err != nil {
		log.Fatal(err)
	}
	target := url.NewResource("ssh://127.0.0.1", credential)
	context, err := exec.NewSSHReplayContext(manager, target, "test/session/transient")
	if err != nil {
		log.Fatal(err)
	}
	response, err := manager.Run(context, exec.NewOpenSessionRequest(target, []string{"/usr/local/bin"}, map[string]string{"M2_HOME": "/users/test/.m2/"}, false, "/"))
    if err != nil {
        log.Fatal(err)
    }
```