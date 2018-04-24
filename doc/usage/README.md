## Usage


## Endly

There are a few ways to execute workflow/action

1) Run workflow with inline parameters
    - endly -w=workflowname param1=value params2=value2
2) Run single workflow with run request
    -  endly -r=[run](runs.yaml)   
2) Run inline workflow run request.
    -  endly -r=[run](runm.yaml)   



To check endly other options run the following:

```text
$ endly -h
```
         

## API integration
         
To integrate endly with unit test in golang, you can use one of the following  
  

**Silence mode**

With this method, you can run any endly service action directly (including workflow with *model.ProcessRequest) by providing endly supported request.


```go

        manager := endly.New()
        var context = manager.NewContext(nil)
        var target = url.NewResource("ssh://127.0.0.1/", "localhost")
        var runRequest = &docker.RunRequest{
           Target: target,
           Image:  "mysql:5.6",
           Ports: map[string]string{
               "3306": "3306",
           },
           Env: map[string]string{
               "MYSQL_ROOT_PASSWORD": "**mysql**",
           },
           Mount: map[string]string{
               "/tmp/my.cnf": "/etc/my.cnf",
           },
           Secrets: map[string]string{
               "**mysql**": mySQLcredentialFile,
           },
       }
                                   
        var runResponse = &docker.RunResponse{}
        err := endly.Run(context, runRequest, runResponse) //(use 'nil' as last parameters to ignore actual response)
        if err != nil {
           log.Fatal(err)
        }
		
```         

**CLI mode**

In this method, a workflow runs with command runner similarly to 'endly' command line.

```go

    runner := cli.New()
	cli.OnError = func(code int) {}//to supres os.Exit(1) in case of error
	err := runner.Run(&workflow.RunRequest{
			URL: "action",
			Tasks:       "run",
			Params: map[string]interface{}{
				"service": "logger",
				"action":  "print",
				"request": &endly.PrintRequest{Message: "hello"},
			},
	}, nil)
    if err != nil {
    	log.Fatal(err)
    }

```         
