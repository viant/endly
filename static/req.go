package static

import (
	"bytes"
	"github.com/viant/toolbox/storage"
	"log"
)

func init() {
	var memStorage = storage.NewMemoryService();
	{
		err := memStorage.Upload("mem://github.com/viant/endly/req/mysql.json", bytes.NewReader([]byte(`{
  "Name": "dockerized_mysql",
  "Tasks":"$tasks",
  "Params": {
    "stopSystemMysql": true,
    "targetHost":"$targetHost",
    "targetHostCredential": "$targetHostCredential",
    "configURL": "config/my.cnf",
    "configURLCredential": "$configURLCredential",
    "mysqlCredential":"$mysqlCredential",
    "serviceInstanceName": "$instance",
    "mysqlVersion":"$mysqlVersion",
    "exportFile":"$exportFile",
    "importFile":"$importFile"
  }
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/req/mysql.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/req/set_go.json", bytes.NewReader([]byte(`{
  "Target": {
    "URL": "ssh://${targetHost}/",
    "Credential": "$targetHostCredential"
  },
  "Sdk": "go",
  "Version": "$goVersion",
  "Env":{
    "GOPATH":"$GOPATH"
  }
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/req/set_go.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/req/README.txt", bytes.NewReader([]byte(`This directory stores workflow run request that can be used as reference in any workflow if the local req with the same name does not exist.`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/req/README.txt %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/req/set_jdk.json", bytes.NewReader([]byte(`{
  "Target": {
    "URL": "ssh://${targetHost}/",
    "Credential": "$targetHostCredential"
  },
  "Sdk": "jdk",
  "Version": "$jdkVersion"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/req/set_jdk.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/req/memcached.json", bytes.NewReader([]byte(`{
  "Name": "dockerized_memcached",
  "Tasks":"$tasks",
  "Params": {
    "url": "scp://${targetHost}/",
    "targetHost":"$targetHost",
    "targetHostCredential": "$targetHostCredential",
    "configURL": "config/aerospike.conf",
    "configURLCredential": "$configURLCredential",
    "serviceInstanceName": "$instance",
    "maxMemory":"$maxMemory"
  }
}
`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/req/memcached.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/req/aerospike.json", bytes.NewReader([]byte(`{
  "Name": "dockerized_aerospike",
  "Tasks":"$tasks",
  "Params": {
    "targetHost":"$targetHost",
    "targetHostCredential": "$targetHostCredential",
    "configURL": "config/aerospike.conf",
    "configURLCredential": "$configURLCredential",
    "serviceInstanceName": "$instance"
  }
}
`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/req/aerospike.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/req/tomcat.json", bytes.NewReader([]byte(`{
  "Name": "tomcat",
  "Tasks":"$tasks",
  "Params": {
    "app": "$app",
    "catalinaOpts": "$catalinaOpts",
    "jdkVersion": "$jdkVersion",
    "targetHost": "$targetHost",
    "targetHostCredential": "$targetHostCredential",
    "appDirectory": "${appRootDirectory}/${app}",
    "configUrl": "config/tomcat-server.xml",
    "configURLCredential":"$configURLCredential",
    "tomcatPort": "$port",
    "tomcatShutdownPort": "$killPort",
    "tomcatAJPConnectorPort": "$connectorPort",
    "tomcatRedirectPort": "$redirectPort",
    "tomcatVersion":"$tomcatVersion",
    "forceDeploy":"$forceDeploy",
    "jpdaAddress":"$jpdaAddress"
  }
}

`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/req/tomcat.json %v", err)
		}
	}
}
