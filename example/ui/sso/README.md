# SSO application - a simple ui application to register and sigin users.

This application was build to provide end to end testing example of ui application.

Is uses aerospike to store user profiles.


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


#### Run reporter webservice workflow

Run the following command:

```text
git clone https://github.com/viant/endly
cd endly/example/ui/sso/endly/
```

## run test with [manager](endly/manager.csv) workflow:
```text
endly -w=manager
```

## run test with inline pipeline tasks [run](endly/run.yaml) request
```text
endly -r=run
```

## To check manager workflow tasks list
```text
endly -w=manager -t='?'
 
```


#Troubleshooting

to check you aerospike just run

docker exec -it db1 aql
SELECT * FROM db1.users;


  
