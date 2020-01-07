# Execution service

The execution service is responsible for opening, managing terminal session, with the ability to send command and extract data.
Service keeps tracking the session current directory commands, env variable setting, and will only issue SSH command if there is actual change. 


- [Usage](#usage)
- [Contract](#contract)
- [Code Level Integration](#code-level-integration)



### Command execution

Command executor uses SSH service, to run all commands. 


Each command is defined with the following:

   - When: optional run match criteria i.e.  $stdout:/password/"
   - Command: command to be send to STDIN
   - Extract: optional data scraping rules
   - Errors: optional matching fragments resulting in error (blacklist) 
   - Success: optional matching fragment defining success (whitelist) 
   - TimeoutMs: optional command timeout 

Each command has access to previous commands output with:
    - current workflow state
    - $stdout all run request commands output upto execution point 
    - $cmd[ index ].stdout individual command output  

 
Each command is sent to SSH session STDIN with optional response timeout and custom terminators to detect command result.
Besides the custom terminators, shell prompt is also used to detect command ended.
If stdout produces no output or no output is match, run command wait specified timeout (default 20sec).

```go
    sshSession.Run(command, listener, timeoutMs, terminators...)
````

Optionally after each step, command execution code is check with ```echo $?``` if checkError flag is set (false by default). 


[@run.yaml](usage/run.yaml)
```yaml
pipeline:
  testMe:
    action: exec:extract
    systemPaths:
      - /opt/sdk/go/bin
    commands:
      - command: go version
        extract:
          - Key: Version
            RegExpr: go(\d\.\d\d)
      - command: echo 'YOUR GO VERSION is $Version'

      - command: passwd tester
        terminators:
          - Old Password
        timeoutMs: 10000

      - command: changme
        success:
          - New Password
        timeoutMs: 10000

      - command: testerPass@1
        success:
          - Retype New Password
        timeoutMs: 10000

      - command: testerPass@1
        success:
          - Retype New Password

      - command: echo 'Done'
```
    

## Usage


### Credentials

####  Default credentials 

By default SSH 'exec' service uses ssh://127.0.0.1:22 and ${env.USER} with private key auth.
The private key has to exists in either locations:
 - ${env.HOME}/.secret/id_rsa
 - ${env.HOME}/.ssh/id_rsa


```bash
endly default_cred.yaml
```

[@default_cred.yaml](usage/default_cred.yaml)

```yaml
pipeline:
  task1:
    action: exec:run
    commands:
      - hostname
      - echo 'welcome ${os.user} on $TrimSpace($cmd[0].stdout)'

```

You can also set default credentials with the following

```yaml
pipeline:
  task1:
    action: exec:setTarget
    url: ssh://myCloudMatchine/
    credentials: myCloudCredentials
```

In that case you can skip defining target in all service using SSH exec service.




#### Custom credentials 
When default method is not available you can generate encrypted credentials for your user useing the 
following [instruction](https://github.com/viant/endly/tree/master/doc/secrets#ssh-credentials)

To connect to host: myhost.com with myuser/mypassword
1. Create credentials file with myuser credentials
    ```bash
    endly -c=myuser-myhost
    ```
2. Define workflow with _target_ attribute
    ```bash
    endly custom_cred.yaml
    ```
[@custom_cred.yaml](usage/custom_cred.yaml)

```yaml
pipeline:
  task1:
    action: exec:run
    target:
      URL: ssh://myhost:com/
      credentials: myuser-myhost
    commands:
      - hostname
      - echo 'welcome ${os.user} on $TrimSpace($cmd[0].stdout)'
```

### Conditional command execution  

Conditional execution uses the following syntax:
```COND ? WHEN_TRUE```


```bash
endly cond.yaml
```
[@cond.yaml](usage/cond.yaml)
```yaml

pipeline:
  myConTask:
    action: exec:run
    commands:
      - ls -al /tmp/myapp
      - ${cmd[0].stdout}:/No such file or directory/?  mkdir -p /tmp/myapp
      - ls -al /tmp/myapp
      - ${cmd[2].stdout}:/No such file or directory/? echo 'failed to create app folder'


  debugInfo:
    action: print
    message: $AsJSON($myConTask)


  nextStep:
    when: ${myConTask.Output}:!/failed/
    action: print
    message: Created app folder, moving to next step ...

```


```bash
endly cond_external_arg.yaml p=123
```

[@cond_external_arg.yaml](usage/cond_external_arg.yaml)
```yaml
pipeline:
  myConTask:
    action: exec:run
    commands:
      - $p = 123 ? echo 'p was $p'
      - echo 'done'
  myDebugInfo:
    action: print
    message: $myConTask.Output

```

### Scraping data

```bash
endly scrape.yaml 
```

[@scrape.yaml](usage/scrape.yaml)
```yaml
pipeline:
  extract:
    action: exec:run
    commands:
      - whoami
      - cat /etc/hosts
    extract:
      - key: aliases
        regExpr: (?sm)\s+127.0.0.1(.+)

  info:
    action: print
    message: "Extracted: ${extract.Data.aliases}"

```






### Super user mode

To run command as super user set _superUser_ to true

```bash
endly super.yaml 
```

[@super.yaml](usage/super.yaml)
```
init:
  target:
    URL: ssh://127.0.0.1/
    credentials: localhost

pipeline:
  myConTask:
    action: exec:run
    target: $target
    superUser: true
    commands:
      - whoami
      - mkdir /tmp/app2
      - chown ${os.user} /tmp/app2
      - ls -al /tmp/app2

```

### Auto sudo mode

To run command as super user only when there is permission needed, set _autoSudo_ flag


```bash
endly sudo.yaml 
```

[@auto_sudo.yaml](usage/auto_sudo.yaml)
```
init:
  target:
    URL: ssh://127.0.0.1/
    credentials: localhost

pipeline:
  myConTask:
    action: exec:run
    target: $target
    autoSudo: true
    commands:
      - whoami
      - mkdir /opt/myapp
      - chown ${os.user} /opt/myapp
      - ls -al /opt/myapp
```


### Controlling error

#### Commands error 

By default any command error exit code is ignored, to enable command exit code check set checkError attribute.

```endly check_errors.yaml```

[@check_errors.yaml](usage/check_errors.yaml)
```yaml
pipeline:
  build:
    action: exec:run
    checkError: true
    commands:
      - export GO111MODULE=on
      - unset GOPATH
      - cd $appPath
      - go build
```

#### Custom error detection

In some scenario, when a command returns success (0) code, you may still terminate command execution based on command output.

```endly custom_error.yaml```

[@custom_error.yaml](usage/custom_error.yaml)
```yaml
pipeline:
  task1:
    action: exec:run
    errors:
      - myError
    commands:
      - echo 'starting .. '
      - echo ' myError'
      - echo 'done.'
```

### Handling secrets

For security reason credentials should never be store in plain form,  neither reveal on the terminal or any logs files.

Imaging that you have to build an app that uses private git repository.

1. Encrypt private git repo credentials
```endly -c='myuser-git'``` 
2. Check created credentials ```ls -al ~/.secret/myuser-git.json```
3. Define workflow with _secrets_ section.
    * [@handle_secrets.yaml](usage/handle_secrets.yaml)
    ```yaml
    pipeline:
    
      build:
        action: exec:run
        checkError: true
        terminators:
          - Password
          - Username
        secrets:
          gitSecrets: myuser-git
        commands:
          - export GIT_TERMINAL_PROMPT=1
          - export GO111MODULE=on
          - unset GOPATH
          - cd ${appPath}/
          - go build
          - '${cmd[3].stdout}:/Username/? $gitSecrets.Username'
          - '${output}:/Password/? $gitSecrets.Password'
    ```






### Session variables:
- ${os.user}
- ${cmd[x].stdout}
- $stdout
- $secrets



## Contract

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
| exec | open | open SSH session on the target resource. | [OpenSessionRequest](contract.go) | [OpenSessionResponse](contract.go) |
| exec | close | close SSH session | [CloseSessionRequest](contract.go) | [CloseSessionResponse](contract.go) |
| exec | run | execute basic commands | [RunRequest](contract.go) | [RunResponse](contract.go) |
| exec | extract | execute commands with ability to extract data, define error or success state | [ExtractRequest](contract.go) | [RunResponse](contract.go) |



## Code level integration

<a name="session"></a>

### SSH Session

In order to run any SSH command, service needs to open a session, it uses target.Credentials and [secret service](https://github.com/viant/toolbox/tree/master/secret) to connect the target host.

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

RunReuest provide a simple way to execute SSH command with conditional execution, it uses [util.StdErrors](https://github.com/viant/endly/blob/master/util/stdoututil.go#L16) as stdout errors.
ExtractRequest has ability to fine tune SSH command execution with extraction data ability. Error matching in ExtractRequest does use any default value.

<a name="run"></a>
### Run Commands

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
### Run Command to Extract Data


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
### Exec SSH Unit Testing

This module provide  SSH session recording ability to later replay it during unit testing without actual SSH involvement.  

**Recording SSH session**

To record actual SSH session use  exec.OpenRecorderContext helper method, the last parameters specify location where conversation is recorded, actual dump takes place when context is closed (defer context.Clode()).
If you use **sudo**. any **secret or credentials** make sure that you rename it to *** before checking in any code so you can use  `var credentials, err = util.GetDummyCredential()`


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
	var credentials, err = util.GetDummyCredential()
	if err != nil {
		log.Fatal(err)
	}
	target := url.NewResource("ssh://127.0.0.1", credentials)
	context, err := exec.NewSSHReplayContext(manager, target, "test/session/transient")
	if err != nil {
		log.Fatal(err)
	}
	response, err := manager.Run(context, exec.NewOpenSessionRequest(target, []string{"/usr/local/bin"}, map[string]string{"M2_HOME": "/users/test/.m2/"}, false, "/"))
    if err != nil {
        log.Fatal(err)
    }
```