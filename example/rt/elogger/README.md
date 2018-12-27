# Logger - simple http request event logger

This application was build to provide end to end testing example of runtime application.

The end to end test provide uses cases testing log both with index and position based methodology.

Prerequisites:

Enable ssh logic you your use on your machine (on osx System Preference / Sharing / Remote Login )
 
Download [endly](https://github.com/viant/endly/releases/)

Provide a username and password to login to your box.
```text
endly -c=localhost
```
Verify that secret file were created
```text
cat ~/.secret/localhost.json
```


#### Run E-logger app workflow

Run the following command:

```text
git clone https://github.com/viant/endly
cd endly/example/rt/elogger/e2e/
```


## run test with inline workflow[run](e2e/run.yaml)
```text
endly -r=run
```

## To check workflow tasks list
```text
endly -t='?'
```



