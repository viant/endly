# Reporter application - a simple report builder web service.

This application was build to provide end to end testing example of web service application.

Is uses mysql to register a pivot report definition, and to run reports.


Prerequisites:

Enable ssh logic you your use on your machine (on osx System Preference / Sharing / Remote Login )
 
Install [docker](https://docs.docker.com/engine/installation/) service

Download [endly](https://github.com/viant/endly/releases/)

Provide a username and password to login to your box.
```text
endly -c=localhost
```

Verify that secret file were created
```text
cat ~/.secret/localhost.json
```

Create 'mysql' secret credentials, provide  **root** as username and non empty password for docker mysqladmin
```text
endly -c=mysql
```



#### Run reporter webservice workflow

Run the following command:

```text
git clone https://github.com/viant/endly
cd endly/example/ws/reporter/e2e/
```

## run test with [run](e2e/run.csv) workflow:


## run test with inline workflow[run](endly/run.yaml) request
```text
endly -r=run
```

## To check run workflow tasks list
```text
endly  r=run -t='?'
```




#Troubleshooting

to check you aerospike just run



docker exec -it mydb1 mysql
show tables;


  


