## Usage


## Endly

There are a few ways to execute workflow/action


1) Run inline workflow
    -  endly -r=[runs](runs.yaml)   
2) Run inline workflow
    -  endly [runs.yaml](runs.yaml)   
3) Run endly service action
    -  endly validator:assert actual=3 expect=4
    -  kubernetes:get secrets kind=secret
    
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
