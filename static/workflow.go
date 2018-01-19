package static

import (
	"bytes"
	"github.com/viant/toolbox/storage"
	"log"
)

func init() {
	var memStorage = storage.NewMemoryService()
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/tomcat_init.json", bytes.NewReader([]byte(`[
  {
    "Name": "targetHost",
    "From": "params.targetHost",
    "Required":true
  },
  {
    "Name": "targetHostCredential",
    "From": "params.targetHostCredential",
    "Required":true
  },
  {
    "Name": "tomcatVersion",
    "From": "params.tomcatVersion",
    "Required":true
  },
  {
    "Name": "catalinaOpts",
    "From": "params.catalinaOpts",
    "Value": "-Xms512m -Xmx1g -XX:MaxPermSize=256m"
  },
  {
    "Name": "tomcatPort",
    "From": "params.tomcatPort",
    "Value": "8080"
  },
  {
    "Name": "jpdaAddress",
    "From": "params.jpdaAddress",
    "Value": "5000"
  },
  {
    "Name": "tomcatShutdownPort",
    "From": "params.tomcatShutdownPort",
    "Value": "8005"
  },
  {
    "Name": "tomcatAJPConnectorPort",
    "From": "params.tomcatAJPConnectorPort",
    "Value": "8009"
  },
  {
    "Name": "tomcatRedirectPort",
    "From": "params.tomcatRedirectPort",
    "Value": "8443"
  },
  {
    "Name": "jdkVersion",
    "From": "params.jdkVersion",
    "Value": "1.7"
  },
  {
    "Name": "appDirectory",
    "Value": "$params.appDirectory",
    "Required":true
  },
  {
    "Name": "forceDeploy",
    "From": "params.forceDeploy",
    "Value": false
  },
  {
    "Name": "configUrl",
    "From": "params.configUrl",
    "Required":true
  },
  {
    "Name": "configURLCredential",
    "From": "params.configURLCredential",
    "Required":true
  },
  {
    "Name": "tomcat",
    "Value": "apache-tomcat-${tomcatVersion}"
  }
]`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/tomcat_init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/tomcat.csv", bytes.NewReader([]byte(`Workflow,,Name,Description,Tasks,,Init,,
,,tomcat,Tomcat server action manager,%Tasks,,#tomcat_init.json,,
[]Tasks,,Name,Description,Actions,,,,
,,install,This task will install tomcat,%Install,,,,
[]Install,Service,Action,Description,Request,,,,
,deployment,deploy,Deploy tomcat server into target host,#req/tomcat_deploy,,,,
[]Tasks,,Name,Description,Actions,,,,
,,start,This task will start tomcat,%Start,,,,
[]Start,Service,Action,Description,Request,RunCriteria,Variables,Post,
,sdk,set,Set jdk,#req/set_jdk,,,,
,process,status,Check tomcat process,#req/tomcat_check,,,"[{""name"":""tomcatPid"", ""from"":""Pid"", ""value"":""0""}]",
,exec,extractable-command,Stop tomcat server,#req/tomcat_stop,$tomcatPid:!0,,,
,exec,extractable-command,Start tomcat server,#req/tomcat_start,,,,
[]Tasks,,Name,Description,Actions,,,,
,,stop,This task will stop tomcat,%Stop,,,,
[]Stop,Service,Action,Description,Request,RunCriteria,,Post,SleepTimeMs
,sdk,set,Set jdk,#req/set_jdk,,,,
,process,status,Check tomcat process,#req/tomcat_check,,,"[{""name"":""tomcatPid"", ""from"":""Pid"", ""value"":""0""}]",
,exec,extractable-command,Stop tomcat server,#req/tomcat_stop,$tomcatPid:!0,,,500
,process,status,Check tomcat process,#req/tomcat_check,,,"[{""name"":""tomcatPid"", ""from"":""Pid"", ""value"":""0""}]",
,process,stop,Kill tomcat process,#req/tomcat_kill,$tomcatPid:!0,,,`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/tomcat.csv %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/vc_maven_build_var.json", bytes.NewReader([]byte(`[
  {
    "Name": "jdkVersion",
    "From": "$params.jdkVersion",
    "Required": true
  },
  {
    "Name":"mavenVersion",
    "From":"params.mavenVersion",
    "Required": true
  },
  {
    "Name": "buildGoal",
    "From": "params.buildGoal",
    "Value": "install"
  },
  {
    "Name": "buildArgs",
    "From": "params.buildArgs",
    "Value": ""
  },
  {
    "Name": "originType",
    "From": "params.originType",
    "Required": true
  },
  {
    "Name": "originUrl",
    "From": "params.originUrl",
    "Required": true
  },
  {
    "Name": "originCredential",
    "From": "params.originCredential",
    "Required": true
  },
  {
    "Name": "buildTarget",
    "Value": {
      "URL": "$params.targetUrl",
      "Credential": "$params.targetHostCredential"
    }
  },
  {
    "Name": "origin",
    "Value": {
      "Type": "$originType",
      "URL": "$originUrl",
      "Credential": "$originCredential"
    }
  },
  {
    "Name": "checkoutRequest",
    "Value": {
      "Origin": "$origin",
      "Target": "$buildTarget"
    }
  },
  {
    "Name": "buildRequest",
    "Value": {
      "BuildSpec": {
        "Name": "maven",
        "Version":"$mavenVersion",
        "Goal": "build",
        "BuildGoal": "$buildGoal",
        "Args": "$buildArgs",
        "Sdk": "jdk",
        "SdkVersion": "$jdkVersion"
      },
      "Target": "$buildTarget"
    }
  }
]`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/vc_maven_build_var.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/vc_maven_build.csv", bytes.NewReader([]byte(`Workflow,,Name,Description,Tasks,Init
,,vc_maven_build,Workflow check outs/pulls latest change from vc to maven build it,%Tasks,#vc_maven_build_var
[]Tasks,Step,Name,Description,Actions,
,1,checkout,This task wil get the latest code,%Checkout,
[]Checkout,Service,Action,Description,Request,
,version/control,checkout,Check out the latest code,$checkoutRequest,
[]Tasks,Step,Name,Description,Actions,
,1,build,This task builds checked out code with maven,%Build,
[]Build,Service,Action,Description,Request,
,build,build,Build maven artifact,$buildRequest,`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/vc_maven_build.csv %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/dockerized_mysql_init.json", bytes.NewReader([]byte(`[
  {
    "Name": "serviceInstanceName",
    "From": "params.serviceInstanceName",
    "Required": true
  },
  {
    "Name": "targetHost",
    "From": "params.targetHost",
    "Required": true
  },
  {
    "Name": "targetHostCredential",
    "From": "params.targetHostCredential",
    "Required": true
  },
  {
    "Name": "configURL",
    "From": "params.configURL"
  },
  {
    "Name": "configURLCredential",
    "From": "params.configURLCredential"
  },
  {
    "Name": "mysqlCredential",
    "Value": "$params.mysqlCredential",
    "Required": true
  },
  {
    "Name": "mysqlVersion",
    "Value": "$params.mysqlVersion"
  },
  {
    "Name": "exportFile",
    "Value": "$params.exportFile"
  },
  {
    "Name": "importFile",
    "Value": "$params.importFile"
  },
  {
    "Name": "mysqlVersion",
    "Value": "$params.mysqlVersion",
    "Required": true
  },
  {
    "Name": "dockerTarget",
    "Value": {
      "Name": "$serviceInstanceName",
      "URL": "scp://${targetHost}/",
      "Credential": "$targetHostCredential"
    }
  },
  {
    "Name": "configFile",
    "Value": "/tmp/my${serviceInstanceName}.cnf"
  }
]`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/dockerized_mysql_init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/vc_maven_module_build.csv", bytes.NewReader([]byte(`Workflow,,Name,Description,Tasks,Init
,,vc_maven_module_build,Workflow check outs/pulls latest change from vc to maven build it,%Tasks,#vc_maven_module_build_var
[]Tasks,Step,Name,Description,Actions,
,1,checkout,This task wil get the latest code and build it,%Checkout,
[]Checkout,Service,Action,Description,Request,
,version/control,checkout,Check out the latest code,$checkoutRequest,
,transfer,copy,Copy module pom file,$transferPomRequest,
[]Tasks,Step,Name,Description,Actions,
,1,build,This task wil build the checked out code,%Build,
[]Build,Service,Action,Description,Request,
,build,build,Build maven artifact,$buildRequest,`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/vc_maven_module_build.csv %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/dockerized_memcached.csv", bytes.NewReader([]byte(`Workflow,,Name,Description,Tasks,,Init
,,dockerized_memcached,This workflow manages memcached service,%Tasks,,#dockerized_memcached_init
[]Tasks,,Name,Description,Actions,,
,,start,This task will start memcached docker service,%Start,,
[]Start,Service,Action,Description,Request,RunCriteria,Post
,docker,run,Start docker memcached service,#req/docker_memcached,,
[]Tasks,,Name,Description,Actions,,
,,stop,This task will stop docker mysql,%Stop,,
[]Stop,Service,Action,Description,Request,,
,docker,container-stop,Stop docker memcached service,#req/docker.json,,`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/dockerized_memcached.csv %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/docker_memcached.json", bytes.NewReader([]byte(`{
  "SysPath": [
    "/usr/local/bin"
  ],
  "Target": "$dockerTarget",
  "Image": "library/memcached:alpine",
  "MappedPort": {
    "11211": "11211"
  },
  "Params": {
    "-m": "$maxMemory"
  }
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/docker_memcached.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/docker_aerospike.json", bytes.NewReader([]byte(`{
  "SysPath": [
    "/usr/local/bin"
  ],
  "Target": "$dockerTarget",
  "Image": "aerospike/aerospike-server:latest",
  "Mount": {
    "$configFile": "/etc/aerospike/aerospike.conf"
  },
  "MappedPort": {
    "3000": "3000",
    "3001": "3001",
    "3002": "3002",
    "3004": "3004",
    "8081": "8081"
  }
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/docker_aerospike.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/tomcat_stop.json", bytes.NewReader([]byte(`{
  "Target": {
    "URL": "ssh://${targetHost}${appDirectory}/",
    "Credential": "$targetHostCredential"
  },
  "ExtractableCommand": {
    "Options": {
      "Directory": "$appDirectory"
    },
    "Executions": [
      {
        "Command": "tomcat/bin/shutdown.sh"
      }
    ]
  }
}
`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/tomcat_stop.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/docker_mysql_import.json", bytes.NewReader([]byte(`{
  "SysPath": [
    "/usr/local/bin"
  ],
  "Target": "$dockerTarget",
  "Credentials": {
    "***mysql***": "$mysqlCredential"
  },
  "Interactive":true,
  "Command":"mysql  -uroot -p***mysql*** < $importFile"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/docker_mysql_import.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/ec2_status.json", bytes.NewReader([]byte(`{
  "Credential": "$awsCredential",
  "Method": "DescribeInstances",
  "Input": {
    "InstanceIds": [
      "$ec2InstanceId"
    ]
  }
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/ec2_status.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/docker_system.json", bytes.NewReader([]byte(`{
  "Service": "docker",
  "Target": "$dockerTarget"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/docker_system.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/ec2_start.json", bytes.NewReader([]byte(`{
  "Credential": "$awsCredential",
  "Method": "StartInstances",
  "Input": {
    "InstanceIds": [
      "$ec2InstanceId"
    ]
  }
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/ec2_start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/mysql_system.json", bytes.NewReader([]byte(`{
  "Service": "mysql",
  "Target": {
    "URL":"$targetHost",
    "Credential":"$targetHostCredential"
  }

}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/mysql_system.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/docker_mysql_export.json", bytes.NewReader([]byte(`{
  "SysPath": [
    "/usr/local/bin"
  ],
  "Target": "$dockerTarget",
  "Credentials": {
    "***mysql***": "$mysqlCredential"
  },
  "Interactive":true,
  "AllocateTerminal":true,
  "Command":"mysqldump  -uroot -p***mysql*** --all-databases --routines | grep -v 'Warning' > $exportFile"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/docker_mysql_export.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/docker_mysql.json", bytes.NewReader([]byte(`{
  "Env": {
    "MYSQL_ROOT_PASSWORD": "***mysql***"
  },
  "SysPath": [
    "/usr/local/bin"
  ],
  "Target": "$dockerTarget",
  "Credentials": {
    "***mysql***":"$mysqlCredential"
  },
  "Image": "mysql:$mysqlVersion",
  "Mount": {
    "$configFile": "/etc/my.cnf"
  },
  "MappedPort": {
    "3306": "3306"
  }
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/docker_mysql.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/docker_config.json", bytes.NewReader([]byte(`{
  "Transfers": [
    {
      "Source": {
        "URL": "$configURL",
        "Credential": "$configURLCredential"
      },
      "Target": {
        "URL": "scp://${targetHost}${configFile}",
        "Credential":"$targetHostCredential"
      },
      "Expand": true
    }
  ]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/docker_config.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/docker.json", bytes.NewReader([]byte(`{
  "SysPath": [
    "/usr/local/bin"
  ],
  "Target": "$dockerTarget"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/docker.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/ec2_stop.json", bytes.NewReader([]byte(`{
  "Credential": "$awsCredential",
  "Method": "StopInstances",
  "Input": {
    "InstanceIds": [
      "$ec2InstanceId"
    ]
  }
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/ec2_stop.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/tomcat_deploy.json", bytes.NewReader([]byte(`{
  "Force": "$forceDeploy",
  "Sdk": "jdk",
  "SdkVersion": "$jdkVersion",
  "AppName": "tomcat",
  "Version": "$tomcatVersion",
  "Target": {
    "URL": "scp://${targetHost}/${appDirectory}/",
    "Credential": "$targetHostCredential"
  }
}
`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/tomcat_deploy.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/tomcat_kill.json", bytes.NewReader([]byte(`{
  "Target": {
    "URL": "ssh://${targetHost}${appDirectory}/",
    "Credential": "$targetHostCredential"
  },
  "Pid": "$tomcatPid"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/tomcat_kill.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/tomcat_check.json", bytes.NewReader([]byte(`{
  "Target": {
    "URL": "ssh://${targetHost}${appDirectory}/",
    "Credential": "$targetHostCredential"
  },
  "Command": "catalina | grep ${appDirectory}/tomcat"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/tomcat_check.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/tomcat_start.json", bytes.NewReader([]byte(`{
  "Target": {
    "URL": "ssh://${targetHost}${appDirectory}/",
    "Credential": "$targetHostCredential"
  },
  "ExtractableCommand": {
    "Options": {
      "Env": {
        "CATALINA_OPTS": "$catalinaOpts",
        "JPDA_ADDRESS": "$jpdaAddress"
      },
      "Directory": "$appDirectory"
    },
    "Executions": [
      {
        "Command": "tomcat/bin/catalina.sh jpda start",
        "Success": [
          "Tomcat started."
        ],
        "Extraction": [
          {
            "Key": "Version",
            "RegExpr": "Server number: (\\d+\\.\\d+\\.\\d+)"
          }
        ]
      }
    ]
  }
}
`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/tomcat_start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/dockerized_aerospike.csv", bytes.NewReader([]byte(`Workflow,,Name,Description,Tasks,,Init
,,dockerized_aerospike,This workflow managed aeropsike docker service,%Tasks,,#dockerized_aerospike_init
[]Tasks,,Name,Description,Actions,RunCriteria,
,,start,Start aerospike contaner,%Start,,
[]Start,Service,Action,Description,Request,RunCriteria,
,transfer,copy,This action will copy template config file to the temp folder,#req/docker_config,$configURL:/!configURL/,
,docker,run,Start docker aerospike service,#req/docker_aerospike,,
[]Tasks,,Name,Description,Actions,,
,,stop,Stop aerospike container,%Stop,,
[]Stop,Service,Action,Description,Request,,
,docker,container-stop,Start docker aerospike service,#req/docker,,`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/dockerized_aerospike.csv %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/ec2.csv", bytes.NewReader([]byte(`Workflow,,,Name,Description,Tasks,Init,,
,,,ec2,Start/Stop ec2 instances,%Tasks,#ec2_init,,
[]Tasks,,,Name,Description,Actions,,,
,,,start,start ec2 instance,%Start,,,
[]Start,,Service,Action,Description,Request,Post,var.Name,var.From
,,aws/ec2,call,Check instance status,#req/ec2_status,[$arg0]|$var,instanceState,Reservations[0].Instances[0].State.Name
[]Start,,Service,Action,Description,Request.SourceKey,Request.Cases,,
,,workflow,switch-action,switch/case for instance status,instanceState,%StartCases,,
[]StartCases,Value,Service,Action,Description,Request,,,
,stopped,aws/ec2,call,Start Ec2 instance,#req/ec2_start,,,
,running,workflow,exit,exit workflow,{},,,
[]Start,,Service,Action,Description,Request.Task,SleepTimeMs,,
,,workflow,run-task,goto start task,start,5000,,
[]Tasks,,,Name,Description,Actions,,,
,,,stop,stop ec2 instance,%Stop,,,
[]Stop,,Service,Action,Description,Request,Post,var.Name,var.From
,,aws/ec2,call,Check instance status,#req/ec2_status,[$arg0]|$var,instanceState,Reservations[0].Instances[0].State.Name
[]Stop,,Service,Action,Description,Request.SourceKey,Request.Cases,,
,,workflow,switch-action,switch/case for instance status,instanceState,%StopCases,,
[]StopCases,Value,Service,Action,Description,Request,,,
,running,aws/ec2,call,Start Ec2 instance,#req/ec2_stop,,,
,stopped,workflow,exit,exit workflow,{},,,
[]Stop,,Service,Action,Description,Request.Task,SleepTimeMs,,
,,workflow,run-task,goto stop task,stop,5000,,`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/ec2.csv %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/000_main/0004_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/000_main/0004_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/000_main/0003_State.Init.json", bytes.NewReader([]byte(`{
	"state": {
		"AsBool": "func()",
		"AsFloat": "func()",
		"AsInt": "func()",
		"AsMap": "func()",
		"AsTableRecords": "func()",
		"Cat": "func()",
		"FormatTime": "func()",
		"HasResource": "func()",
		"Length": "func()",
		"LoadNeatly": "func()",
		"Markdown": "func()",
		"Md5": "func()",
		"Unzip": "func()",
		"UnzipText": "func()",
		"WorkingDirectory": "func()",
		"Zip": "func()",
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"data": {},
		"date": "2018-01-18",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf",
		"env": "func()",
		"ownerURL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
		"params": {
			"awsCredential": "/Users/awitas/.secret/aws.json",
			"ec2InstanceId": "i-0ef8d9260eaf47fdf"
		},
		"rand": 8300262019636288784,
		"time": "2018-01-18 11:12:01",
		"timestamp": "func()",
		"tmpDir": "func()",
		"ts": "201801181112000",
		"uuid": "func()"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/000_main/0003_State.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/000_main/0001_Workflow.Loaded.json", bytes.NewReader([]byte(`{
	"workflow": {
		"Source": {
			"URL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
			"Credential": "",
			"ParsedURL": {
				"Scheme": "file",
				"Opaque": "",
				"User": {},
				"Host": "",
				"Path": "/Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
				"RawPath": "",
				"ForceQuery": false,
				"RawQuery": "",
				"Fragment": ""
			},
			"Cache": "",
			"CacheExpiryMs": 0,
			"Name": "",
			"Type": ""
		},
		"Data": null,
		"Name": "ec2",
		"Description": "Start/Stop ec2 instances",
		"Init": [
			{
				"Name": "awsCredential",
				"Value": null,
				"From": "params.awsCredential",
				"Persist": false,
				"Required": true
			},
			{
				"Name": "ec2InstanceId",
				"Value": null,
				"From": "params.ec2InstanceId",
				"Persist": false,
				"Required": true
			}
		],
		"Post": null,
		"Tasks": [
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "start",
				"Description": "start ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StartInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "stopped"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "running"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "start"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			}
		],
		"OnErrorTask": "",
		"SleepTimeMs": 0
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/000_main/0001_Workflow.Loaded.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/000_main/0002_Workflow.Init.json", bytes.NewReader([]byte(`{
	"sources": {
		"params.awsCredential": "/Users/awitas/.secret/aws.json",
		"params.ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"values": {
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"variables": [
		{
			"Name": "awsCredential",
			"Value": null,
			"From": "params.awsCredential",
			"Persist": false,
			"Required": true
		},
		{
			"Name": "ec2InstanceId",
			"Value": null,
			"From": "params.ec2InstanceId",
			"Persist": false,
			"Required": true
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/000_main/0002_Workflow.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:12:04.193869-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0014_WorkflowTask.End.json", bytes.NewReader([]byte(`{}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0014_WorkflowTask.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0010_WorkflowExitRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Source": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0010_WorkflowExitRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0004_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0004_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0011_WorkflowExitRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0011_WorkflowExitRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:12:01.576661-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0012_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0012_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:12:04.193164-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0015_WorkflowRunRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Data": null,
			"SessionID": "0b9ff31a-fce8-11e7-ac49-784f438e6f38"
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0015_WorkflowRunRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0013_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0013_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/0b9ff31a-fce8-11e7-ac49-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/000_main/0004_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/000_main/0004_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/000_main/0003_State.Init.json", bytes.NewReader([]byte(`{
	"state": {
		"AsBool": "func()",
		"AsFloat": "func()",
		"AsInt": "func()",
		"AsMap": "func()",
		"AsTableRecords": "func()",
		"Cat": "func()",
		"FormatTime": "func()",
		"HasResource": "func()",
		"Length": "func()",
		"LoadNeatly": "func()",
		"Markdown": "func()",
		"Md5": "func()",
		"Unzip": "func()",
		"UnzipText": "func()",
		"WorkingDirectory": "func()",
		"Zip": "func()",
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"data": {},
		"date": "2018-01-18",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf",
		"env": "func()",
		"ownerURL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
		"params": {
			"awsCredential": "/Users/awitas/.secret/aws.json",
			"ec2InstanceId": "i-0ef8d9260eaf47fdf"
		},
		"rand": 9104118478177583598,
		"time": "2018-01-18 10:58:41",
		"timestamp": "func()",
		"tmpDir": "func()",
		"ts": "201801181058000",
		"uuid": "func()"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/000_main/0003_State.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/000_main/0001_Workflow.Loaded.json", bytes.NewReader([]byte(`{
	"workflow": {
		"Source": {
			"URL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
			"Credential": "",
			"ParsedURL": {
				"Scheme": "file",
				"Opaque": "",
				"User": {},
				"Host": "",
				"Path": "/Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
				"RawPath": "",
				"ForceQuery": false,
				"RawQuery": "",
				"Fragment": ""
			},
			"Cache": "",
			"CacheExpiryMs": 0,
			"Name": "",
			"Type": ""
		},
		"Data": null,
		"Name": "ec2",
		"Description": "Start/Stop ec2 instances",
		"Init": [
			{
				"Name": "awsCredential",
				"Value": null,
				"From": "params.awsCredential",
				"Persist": false,
				"Required": true
			},
			{
				"Name": "ec2InstanceId",
				"Value": null,
				"From": "params.ec2InstanceId",
				"Persist": false,
				"Required": true
			}
		],
		"Post": null,
		"Tasks": [
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "start",
				"Description": "start ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StartInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "stopped"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "running"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "start"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			}
		],
		"OnErrorTask": "",
		"SleepTimeMs": 0
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/000_main/0001_Workflow.Loaded.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/000_main/0002_Workflow.Init.json", bytes.NewReader([]byte(`{
	"sources": {
		"params.awsCredential": "/Users/awitas/.secret/aws.json",
		"params.ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"values": {
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"variables": [
		{
			"Name": "awsCredential",
			"Value": null,
			"From": "params.awsCredential",
			"Persist": false,
			"Required": true
		},
		{
			"Name": "ec2InstanceId",
			"Value": null,
			"From": "params.ec2InstanceId",
			"Persist": false,
			"Required": true
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/000_main/0002_Workflow.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:58:42.848638-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0020_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0020_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T22:58:42.849448-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T22:58:48.76412-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0004_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0004_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:58:47.855773-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:58:48.763366-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "start"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:58:41.106837-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0029_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0029_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:58:48.76274-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:58:42.847997-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0013_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0013_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2e81f240-fce6-11e7-b0a8-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/000_main/0004_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/000_main/0004_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/000_main/0003_State.Init.json", bytes.NewReader([]byte(`{
	"state": {
		"AsBool": "func()",
		"AsFloat": "func()",
		"AsInt": "func()",
		"AsMap": "func()",
		"AsTableRecords": "func()",
		"Cat": "func()",
		"FormatTime": "func()",
		"HasResource": "func()",
		"Length": "func()",
		"LoadNeatly": "func()",
		"Markdown": "func()",
		"Md5": "func()",
		"Unzip": "func()",
		"UnzipText": "func()",
		"WorkingDirectory": "func()",
		"Zip": "func()",
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"data": {},
		"date": "2018-01-18",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf",
		"env": "func()",
		"ownerURL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
		"params": {
			"awsCredential": "/Users/awitas/.secret/aws.json",
			"ec2InstanceId": "i-0ef8d9260eaf47fdf"
		},
		"rand": 7233138902852820666,
		"time": "2018-01-18 10:59:44",
		"timestamp": "func()",
		"tmpDir": "func()",
		"ts": "201801181059000",
		"uuid": "func()"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/000_main/0003_State.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/000_main/0001_Workflow.Loaded.json", bytes.NewReader([]byte(`{
	"workflow": {
		"Source": {
			"URL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
			"Credential": "",
			"ParsedURL": {
				"Scheme": "file",
				"Opaque": "",
				"User": {},
				"Host": "",
				"Path": "/Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
				"RawPath": "",
				"ForceQuery": false,
				"RawQuery": "",
				"Fragment": ""
			},
			"Cache": "",
			"CacheExpiryMs": 0,
			"Name": "",
			"Type": ""
		},
		"Data": null,
		"Name": "ec2",
		"Description": "Start/Stop ec2 instances",
		"Init": [
			{
				"Name": "awsCredential",
				"Value": null,
				"From": "params.awsCredential",
				"Persist": false,
				"Required": true
			},
			{
				"Name": "ec2InstanceId",
				"Value": null,
				"From": "params.ec2InstanceId",
				"Persist": false,
				"Required": true
			}
		],
		"Post": null,
		"Tasks": [
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "start",
				"Description": "start ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StartInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "stopped"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "running"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "start"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			}
		],
		"OnErrorTask": "",
		"SleepTimeMs": 0
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/000_main/0001_Workflow.Loaded.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/000_main/0002_Workflow.Init.json", bytes.NewReader([]byte(`{
	"sources": {
		"params.awsCredential": "/Users/awitas/.secret/aws.json",
		"params.ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"values": {
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"variables": [
		{
			"Name": "awsCredential",
			"Value": null,
			"From": "params.awsCredential",
			"Persist": false,
			"Required": true
		},
		{
			"Name": "ec2InstanceId",
			"Value": null,
			"From": "params.ec2InstanceId",
			"Persist": false,
			"Required": true
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/000_main/0002_Workflow.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:59:54.801048-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T22:59:54.801849-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0004_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0004_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:59:44.819941-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:59:54.800346-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0013_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0013_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/547bc750-fce6-11e7-bec8-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/000_main/0004_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/000_main/0004_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/000_main/0003_State.Init.json", bytes.NewReader([]byte(`{
	"state": {
		"AsBool": "func()",
		"AsFloat": "func()",
		"AsInt": "func()",
		"AsMap": "func()",
		"AsTableRecords": "func()",
		"Cat": "func()",
		"FormatTime": "func()",
		"HasResource": "func()",
		"Length": "func()",
		"LoadNeatly": "func()",
		"Markdown": "func()",
		"Md5": "func()",
		"Unzip": "func()",
		"UnzipText": "func()",
		"WorkingDirectory": "func()",
		"Zip": "func()",
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"data": {},
		"date": "2018-01-18",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf",
		"env": "func()",
		"ownerURL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
		"params": {
			"awsCredential": "/Users/awitas/.secret/aws.json",
			"ec2InstanceId": "i-0ef8d9260eaf47fdf"
		},
		"rand": 1011605122787359877,
		"time": "2018-01-18 11:05:28",
		"timestamp": "func()",
		"tmpDir": "func()",
		"ts": "201801181105000",
		"uuid": "func()"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/000_main/0003_State.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/000_main/0001_Workflow.Loaded.json", bytes.NewReader([]byte(`{
	"workflow": {
		"Source": {
			"URL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
			"Credential": "",
			"ParsedURL": {
				"Scheme": "file",
				"Opaque": "",
				"User": {},
				"Host": "",
				"Path": "/Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
				"RawPath": "",
				"ForceQuery": false,
				"RawQuery": "",
				"Fragment": ""
			},
			"Cache": "",
			"CacheExpiryMs": 0,
			"Name": "",
			"Type": ""
		},
		"Data": null,
		"Name": "ec2",
		"Description": "Start/Stop ec2 instances",
		"Init": [
			{
				"Name": "awsCredential",
				"Value": null,
				"From": "params.awsCredential",
				"Persist": false,
				"Required": true
			},
			{
				"Name": "ec2InstanceId",
				"Value": null,
				"From": "params.ec2InstanceId",
				"Persist": false,
				"Required": true
			}
		],
		"Post": null,
		"Tasks": [
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "start",
				"Description": "start ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StartInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "stopped"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "running"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "start"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			}
		],
		"OnErrorTask": "",
		"SleepTimeMs": 0
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/000_main/0001_Workflow.Loaded.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/000_main/0002_Workflow.Init.json", bytes.NewReader([]byte(`{
	"sources": {
		"params.awsCredential": "/Users/awitas/.secret/aws.json",
		"params.ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"values": {
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"variables": [
		{
			"Name": "awsCredential",
			"Value": null,
			"From": "params.awsCredential",
			"Persist": false,
			"Required": true
		},
		{
			"Name": "ec2InstanceId",
			"Value": null,
			"From": "params.ec2InstanceId",
			"Persist": false,
			"Required": true
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/000_main/0002_Workflow.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:05:30.856129-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0020_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0020_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:05:30.856984-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:05:36.214525-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0004_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0004_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:05:35.867982-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:05:36.213776-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "start"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:05:28.762696-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0029_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0029_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:05:36.213203-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:05:30.855521-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0013_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0013_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/217d52d2-fce7-11e7-96d7-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/6770c714-fce8-11e7-be6c-784f438e6f38/000_main/0001_ErrorEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"Error": "failed to load workflow: file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv, Unresolved references: StartCasesalue"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/6770c714-fce8-11e7-be6c-784f438e6f38/000_main/0001_ErrorEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/6770c714-fce8-11e7-be6c-784f438e6f38/000_main/0002_WorkflowRunRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Data": null,
			"SessionID": "6770c714-fce8-11e7-be6c-784f438e6f38"
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/6770c714-fce8-11e7-be6c-784f438e6f38/000_main/0002_WorkflowRunRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/000_main/0004_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/000_main/0004_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/000_main/0003_State.Init.json", bytes.NewReader([]byte(`{
	"state": {
		"AsBool": "func()",
		"AsFloat": "func()",
		"AsInt": "func()",
		"AsMap": "func()",
		"AsTableRecords": "func()",
		"Cat": "func()",
		"FormatTime": "func()",
		"HasResource": "func()",
		"Length": "func()",
		"LoadNeatly": "func()",
		"Markdown": "func()",
		"Md5": "func()",
		"Unzip": "func()",
		"UnzipText": "func()",
		"WorkingDirectory": "func()",
		"Zip": "func()",
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"data": {},
		"date": "2018-01-18",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf",
		"env": "func()",
		"ownerURL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
		"params": {
			"awsCredential": "/Users/awitas/.secret/aws.json",
			"ec2InstanceId": "i-0ef8d9260eaf47fdf"
		},
		"rand": 7125226505125067515,
		"time": "2018-01-18 10:57:34",
		"timestamp": "func()",
		"tmpDir": "func()",
		"ts": "201801181057000",
		"uuid": "func()"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/000_main/0003_State.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/000_main/0001_Workflow.Loaded.json", bytes.NewReader([]byte(`{
	"workflow": {
		"Source": {
			"URL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
			"Credential": "",
			"ParsedURL": {
				"Scheme": "file",
				"Opaque": "",
				"User": {},
				"Host": "",
				"Path": "/Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
				"RawPath": "",
				"ForceQuery": false,
				"RawQuery": "",
				"Fragment": ""
			},
			"Cache": "",
			"CacheExpiryMs": 0,
			"Name": "",
			"Type": ""
		},
		"Data": null,
		"Name": "ec2",
		"Description": "Start/Stop ec2 instances",
		"Init": [
			{
				"Name": "awsCredential",
				"Value": null,
				"From": "params.awsCredential",
				"Persist": false,
				"Required": true
			},
			{
				"Name": "ec2InstanceId",
				"Value": null,
				"From": "params.ec2InstanceId",
				"Persist": false,
				"Required": true
			}
		],
		"Post": null,
		"Tasks": [
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "start",
				"Description": "start ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StartInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "stopped"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "running"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "start"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			}
		],
		"OnErrorTask": "",
		"SleepTimeMs": 0
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/000_main/0001_Workflow.Loaded.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/000_main/0002_Workflow.Init.json", bytes.NewReader([]byte(`{
	"sources": {
		"params.awsCredential": "/Users/awitas/.secret/aws.json",
		"params.ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"values": {
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"variables": [
		{
			"Name": "awsCredential",
			"Value": null,
			"From": "params.awsCredential",
			"Persist": false,
			"Required": true
		},
		{
			"Name": "ec2InstanceId",
			"Value": null,
			"From": "params.ec2InstanceId",
			"Persist": false,
			"Required": true
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/000_main/0002_Workflow.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:57:35.821675-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0020_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0020_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T22:57:35.822456-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T22:57:41.095748-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0004_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0004_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:57:40.827141-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:57:41.094907-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "start"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:57:34.150979-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0029_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0029_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:57:41.094312-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T22:57:35.821062-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0013_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0013_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/06994c4c-fce6-11e7-9784-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/000_main/0004_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/000_main/0004_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/000_main/0003_State.Init.json", bytes.NewReader([]byte(`{
	"state": {
		"AsBool": "func()",
		"AsFloat": "func()",
		"AsInt": "func()",
		"AsMap": "func()",
		"AsTableRecords": "func()",
		"Cat": "func()",
		"FormatTime": "func()",
		"HasResource": "func()",
		"Length": "func()",
		"LoadNeatly": "func()",
		"Markdown": "func()",
		"Md5": "func()",
		"Unzip": "func()",
		"UnzipText": "func()",
		"WorkingDirectory": "func()",
		"Zip": "func()",
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"data": {},
		"date": "2018-01-18",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf",
		"env": "func()",
		"ownerURL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
		"params": {
			"awsCredential": "/Users/awitas/.secret/aws.json",
			"ec2InstanceId": "i-0ef8d9260eaf47fdf"
		},
		"rand": 4011312098707539423,
		"time": "2018-01-18 11:12:48",
		"timestamp": "func()",
		"tmpDir": "func()",
		"ts": "201801181112000",
		"uuid": "func()"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/000_main/0003_State.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/000_main/0001_Workflow.Loaded.json", bytes.NewReader([]byte(`{
	"workflow": {
		"Source": {
			"URL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
			"Credential": "",
			"ParsedURL": {
				"Scheme": "file",
				"Opaque": "",
				"User": {},
				"Host": "",
				"Path": "/Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
				"RawPath": "",
				"ForceQuery": false,
				"RawQuery": "",
				"Fragment": ""
			},
			"Cache": "",
			"CacheExpiryMs": 0,
			"Name": "",
			"Type": ""
		},
		"Data": null,
		"Name": "ec2",
		"Description": "Start/Stop ec2 instances",
		"Init": [
			{
				"Name": "awsCredential",
				"Value": null,
				"From": "params.awsCredential",
				"Persist": false,
				"Required": true
			},
			{
				"Name": "ec2InstanceId",
				"Value": null,
				"From": "params.ec2InstanceId",
				"Persist": false,
				"Required": true
			}
		],
		"Post": null,
		"Tasks": [
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "start",
				"Description": "start ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StartInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "stopped"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "running"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "start"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			}
		],
		"OnErrorTask": "",
		"SleepTimeMs": 0
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/000_main/0001_Workflow.Loaded.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/000_main/0002_Workflow.Init.json", bytes.NewReader([]byte(`{
	"sources": {
		"params.awsCredential": "/Users/awitas/.secret/aws.json",
		"params.ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"values": {
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"variables": [
		{
			"Name": "awsCredential",
			"Value": null,
			"From": "params.awsCredential",
			"Persist": false,
			"Required": true
		},
		{
			"Name": "ec2InstanceId",
			"Value": null,
			"From": "params.ec2InstanceId",
			"Persist": false,
			"Required": true
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/000_main/0002_Workflow.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:12:50.075731-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0014_WorkflowTask.End.json", bytes.NewReader([]byte(`{}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0014_WorkflowTask.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0010_WorkflowExitRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Source": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0010_WorkflowExitRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0004_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0004_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0011_WorkflowExitRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0011_WorkflowExitRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:12:48.467265-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0012_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0012_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:12:50.075114-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0015_WorkflowRunRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Data": null,
			"SessionID": "2792e384-fce8-11e7-9b5a-784f438e6f38"
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0015_WorkflowRunRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0013_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0013_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/2792e384-fce8-11e7-9b5a-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/000_main/0004_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "stop"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/000_main/0004_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/000_main/0003_State.Init.json", bytes.NewReader([]byte(`{
	"state": {
		"AsBool": "func()",
		"AsFloat": "func()",
		"AsInt": "func()",
		"AsMap": "func()",
		"AsTableRecords": "func()",
		"Cat": "func()",
		"FormatTime": "func()",
		"HasResource": "func()",
		"Length": "func()",
		"LoadNeatly": "func()",
		"Markdown": "func()",
		"Md5": "func()",
		"Unzip": "func()",
		"UnzipText": "func()",
		"WorkingDirectory": "func()",
		"Zip": "func()",
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"data": {},
		"date": "2018-01-18",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf",
		"env": "func()",
		"ownerURL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
		"params": {
			"awsCredential": "/Users/awitas/.secret/aws.json",
			"ec2InstanceId": "i-0ef8d9260eaf47fdf"
		},
		"rand": 26968773272940302,
		"time": "2018-01-18 11:15:11",
		"timestamp": "func()",
		"tmpDir": "func()",
		"ts": "201801181115000",
		"uuid": "func()"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/000_main/0003_State.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/000_main/0001_Workflow.Loaded.json", bytes.NewReader([]byte(`{
	"workflow": {
		"Source": {
			"URL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
			"Credential": "",
			"ParsedURL": {
				"Scheme": "file",
				"Opaque": "",
				"User": {},
				"Host": "",
				"Path": "/Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
				"RawPath": "",
				"ForceQuery": false,
				"RawQuery": "",
				"Fragment": ""
			},
			"Cache": "",
			"CacheExpiryMs": 0,
			"Name": "",
			"Type": ""
		},
		"Data": null,
		"Name": "ec2",
		"Description": "Start/Stop ec2 instances",
		"Init": [
			{
				"Name": "awsCredential",
				"Value": null,
				"From": "params.awsCredential",
				"Persist": false,
				"Required": true
			},
			{
				"Name": "ec2InstanceId",
				"Value": null,
				"From": "params.ec2InstanceId",
				"Persist": false,
				"Required": true
			}
		],
		"Post": null,
		"Tasks": [
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "start",
				"Description": "start ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StartInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "stopped"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "running"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "start"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			},
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "stop",
				"Description": "stop ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Stop",
						"TagIndex": "",
						"TagID": "ec2Stop",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StopInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StopCases",
									"TagId": "ec2StopCases",
									"Value": "running"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StopCases",
									"TagId": "ec2StopCases",
									"Value": "stopped"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Stop",
						"TagIndex": "",
						"TagID": "ec2Stop",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "stop"
						},
						"Tag": "Stop",
						"TagIndex": "",
						"TagID": "ec2Stop",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			}
		],
		"OnErrorTask": "",
		"SleepTimeMs": 0
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/000_main/0001_Workflow.Loaded.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/000_main/0002_Workflow.Init.json", bytes.NewReader([]byte(`{
	"sources": {
		"params.awsCredential": "/Users/awitas/.secret/aws.json",
		"params.ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"values": {
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"variables": [
		{
			"Name": "awsCredential",
			"Value": null,
			"From": "params.awsCredential",
			"Persist": false,
			"Required": true
		},
		{
			"Name": "ec2InstanceId",
			"Value": null,
			"From": "params.ec2InstanceId",
			"Persist": false,
			"Required": true
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/000_main/0002_Workflow.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0067_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StopInstances"
				},
				"Value": "running"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "stopped"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0067_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0066_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:33.488049-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StopInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "running"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "stopped"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0066_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0083_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0083_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0008_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:12.505715-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "/Users/awitas/.secret/aws.json",
			"Input": {
				"InstanceIds": [
					"i-0ef8d9260eaf47fdf"
				]
			},
			"Method": "StopInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0008_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0037_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 64,
							"Name": "stopping"
						},
						"StateReason": {
							"Code": "Client.UserInitiatedShutdown",
							"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
						},
						"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0037_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0034_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0034_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0018_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "stop"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0018_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0079_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "",
						"PublicIpAddress": null,
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 80,
							"Name": "stopped"
						},
						"StateReason": {
							"Code": "Client.UserInitiatedShutdown",
							"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
						},
						"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0079_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0030_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0030_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0092_WorkflowTask.End.json", bytes.NewReader([]byte(`{}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0092_WorkflowTask.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0091_WorkflowRunTaskRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0091_WorkflowRunTaskRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0080_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:38.720777-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StopInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "running"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "stopped"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0080_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0033_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:23.023964-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0033_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0087_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0087_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0063_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 64,
								"Name": "stopping"
							},
							"StateReason": {
								"Code": "Client.UserInitiatedShutdown",
								"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
							},
							"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0063_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0098_WorkflowTask.End.json", bytes.NewReader([]byte(`{}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0098_WorkflowTask.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0031_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "stop"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0031_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0071_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0071_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0047_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:28.193109-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0047_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0070_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:15:33.488848-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "stop"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0070_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0041_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "",
		"Response": null,
		"Service": ""
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0041_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0090_WorkflowTask.End.json", bytes.NewReader([]byte(`{}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0090_WorkflowTask.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0081_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StopInstances"
				},
				"Value": "running"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "stopped"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0081_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0062_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0062_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0064_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "stopping"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0064_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0005_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0005_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0007_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StopInstances"
				},
				"Value": "running"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "stopped"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0007_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0028_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:15:18.020381-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "stop"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0028_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0004_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0004_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0096_WorkflowTask.End.json", bytes.NewReader([]byte(`{}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0096_WorkflowTask.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0040_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "",
			"Action": "",
			"Response": null
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0040_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0032_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "stop"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0032_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0075_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:38.490848-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0075_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0042_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:15:23.187512-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "stop"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0042_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0017_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "stop"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0017_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0088_WorkflowTask.End.json", bytes.NewReader([]byte(`{}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0088_WorkflowTask.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0024_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:18.019462-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StopInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "running"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "stopped"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0024_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0084_WorkflowExitRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Source": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0084_WorkflowExitRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0094_WorkflowTask.End.json", bytes.NewReader([]byte(`{}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0094_WorkflowTask.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0077_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": null,
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": null,
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "",
							"PublicIpAddress": null,
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 80,
								"Name": "stopped"
							},
							"StateReason": {
								"Code": "Client.UserInitiatedShutdown",
								"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
							},
							"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0077_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0050_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "stopping"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0050_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0002_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0002_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0049_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 64,
								"Name": "stopping"
							},
							"StateReason": {
								"Code": "Client.UserInitiatedShutdown",
								"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
							},
							"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0049_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0099_WorkflowRunRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Data": null,
			"SessionID": "7cf20490-fce8-11e7-92e7-784f438e6f38"
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0099_WorkflowRunRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0044_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0044_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0097_WorkflowRunTaskRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0097_WorkflowRunTaskRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0072_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0072_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0026_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "",
			"Action": "",
			"Response": null
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0026_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0001_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:11.698734-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0001_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0029_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0029_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0054_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "",
			"Action": "",
			"Response": null
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0054_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0051_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 64,
							"Name": "stopping"
						},
						"StateReason": {
							"Code": "Client.UserInitiatedShutdown",
							"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
						},
						"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0051_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0089_WorkflowRunTaskRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0089_WorkflowRunTaskRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0059_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "stop"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0059_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0019_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:17.764979-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0019_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0045_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "stop"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0045_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0025_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StopInstances"
				},
				"Value": "running"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "stopped"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0025_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0023_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 64,
							"Name": "stopping"
						},
						"StateReason": {
							"Code": "Client.UserInitiatedShutdown",
							"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
						},
						"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0023_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0073_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "stop"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0073_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0035_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 64,
								"Name": "stopping"
							},
							"StateReason": {
								"Code": "Client.UserInitiatedShutdown",
								"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
							},
							"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0035_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0074_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "stop"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0074_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0093_WorkflowRunTaskRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0093_WorkflowRunTaskRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0011_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"StoppingInstances": [
			{
				"CurrentState": {
					"Code": 64,
					"Name": "stopping"
				},
				"InstanceId": "i-0ef8d9260eaf47fdf",
				"PreviousState": {
					"Code": 16,
					"Name": "running"
				}
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0011_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0055_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "",
		"Response": null,
		"Service": ""
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0055_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0022_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "stopping"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0022_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0085_WorkflowExitRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0085_WorkflowExitRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0069_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "",
		"Response": null,
		"Service": ""
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0069_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0036_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "stopping"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0036_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0052_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:28.346166-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StopInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "running"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "stopped"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0052_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0016_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0016_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0027_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "",
		"Response": null,
		"Service": ""
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0027_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0048_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0048_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0012_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "aws/ec2",
			"Action": "call",
			"Response": {
				"StoppingInstances": [
					{
						"CurrentState": {
							"Code": 64,
							"Name": "stopping"
						},
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"PreviousState": {
							"Code": 16,
							"Name": "running"
						}
					}
				]
			}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0012_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0058_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0058_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0057_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0057_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0038_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:23.186476-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StopInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "running"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "stopped"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0038_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0061_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:33.351374-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0061_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0056_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:15:28.347066-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "stop"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0056_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0082_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:38.721328-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0082_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0006_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:15:12.505125-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StopInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "running"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StopCases",
					"TagId": "ec2StopCases",
					"Value": "stopped"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0006_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0043_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0043_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0095_WorkflowRunTaskRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0095_WorkflowRunTaskRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0009_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "StopInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0009_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0068_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "",
			"Action": "",
			"Response": null
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0068_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0065_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 64,
							"Name": "stopping"
						},
						"StateReason": {
							"Code": "Client.UserInitiatedShutdown",
							"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
						},
						"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0065_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0046_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "stop"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0046_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0076_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0076_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0039_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StopInstances"
				},
				"Value": "running"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "stopped"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0039_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0060_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "stop"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0060_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0021_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 64,
								"Name": "stopping"
							},
							"StateReason": {
								"Code": "Client.UserInitiatedShutdown",
								"Message": "Client.UserInitiatedShutdown: User initiated shutdown"
							},
							"StateTransitionReason": "User initiated (2018-01-19 07:15:12 GMT)",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0021_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0014_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Stop",
		"TagIndex": "",
		"TagID": "ec2Stop",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:15:12.760046-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "stop"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0014_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0078_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "stopped"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0078_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0020_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0020_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0013_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "call",
		"Response": {
			"StoppingInstances": [
				{
					"CurrentState": {
						"Code": 64,
						"Name": "stopping"
					},
					"InstanceId": "i-0ef8d9260eaf47fdf",
					"PreviousState": {
						"Code": 16,
						"Name": "running"
					}
				}
			]
		},
		"Service": "aws/ec2"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0013_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0010_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"StoppingInstances": [
				{
					"CurrentState": {
						"Code": 64,
						"Name": "stopping"
					},
					"InstanceId": "i-0ef8d9260eaf47fdf",
					"PreviousState": {
						"Code": 16,
						"Name": "running"
					}
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0010_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0086_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0086_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0003_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0003_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0015_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0015_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0053_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StopInstances"
				},
				"Value": "running"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "stopped"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/7cf20490-fce8-11e7-92e7-784f438e6f38/001_ec2Stop/0053_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/000_main/0004_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/000_main/0004_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/000_main/0003_State.Init.json", bytes.NewReader([]byte(`{
	"state": {
		"AsBool": "func()",
		"AsFloat": "func()",
		"AsInt": "func()",
		"AsMap": "func()",
		"AsTableRecords": "func()",
		"Cat": "func()",
		"FormatTime": "func()",
		"HasResource": "func()",
		"Length": "func()",
		"LoadNeatly": "func()",
		"Markdown": "func()",
		"Md5": "func()",
		"Unzip": "func()",
		"UnzipText": "func()",
		"WorkingDirectory": "func()",
		"Zip": "func()",
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"data": {},
		"date": "2018-01-18",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf",
		"env": "func()",
		"ownerURL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
		"params": {
			"awsCredential": "/Users/awitas/.secret/aws.json",
			"ec2InstanceId": "i-0ef8d9260eaf47fdf"
		},
		"rand": 1625060046875282832,
		"time": "2018-01-18 11:02:27",
		"timestamp": "func()",
		"tmpDir": "func()",
		"ts": "201801181102000",
		"uuid": "func()"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/000_main/0003_State.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/000_main/0001_Workflow.Loaded.json", bytes.NewReader([]byte(`{
	"workflow": {
		"Source": {
			"URL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
			"Credential": "",
			"ParsedURL": {
				"Scheme": "file",
				"Opaque": "",
				"User": {},
				"Host": "",
				"Path": "/Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
				"RawPath": "",
				"ForceQuery": false,
				"RawQuery": "",
				"Fragment": ""
			},
			"Cache": "",
			"CacheExpiryMs": 0,
			"Name": "",
			"Type": ""
		},
		"Data": null,
		"Name": "ec2",
		"Description": "Start/Stop ec2 instances",
		"Init": [
			{
				"Name": "awsCredential",
				"Value": null,
				"From": "params.awsCredential",
				"Persist": false,
				"Required": true
			},
			{
				"Name": "ec2InstanceId",
				"Value": null,
				"From": "params.ec2InstanceId",
				"Persist": false,
				"Required": true
			}
		],
		"Post": null,
		"Tasks": [
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "start",
				"Description": "start ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StartInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "stopped"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "running"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "start"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			}
		],
		"OnErrorTask": "",
		"SleepTimeMs": 0
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/000_main/0001_Workflow.Loaded.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/000_main/0002_Workflow.Init.json", bytes.NewReader([]byte(`{
	"sources": {
		"params.awsCredential": "/Users/awitas/.secret/aws.json",
		"params.ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"values": {
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"variables": [
		{
			"Name": "awsCredential",
			"Value": null,
			"From": "params.awsCredential",
			"Persist": false,
			"Required": true
		},
		{
			"Name": "ec2InstanceId",
			"Value": null,
			"From": "params.ec2InstanceId",
			"Persist": false,
			"Required": true
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/000_main/0002_Workflow.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:02:29.706961-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0020_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0020_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:02:29.707897-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:02:35.112491-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0004_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0004_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:02:34.710002-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:02:35.111711-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "start"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:02:27.254448-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0029_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0029_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:02:35.111026-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:02:29.706252-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0013_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0013_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/b54d536e-fce6-11e7-a7f8-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/000_main/0004_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/000_main/0004_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/000_main/0003_State.Init.json", bytes.NewReader([]byte(`{
	"state": {
		"AsBool": "func()",
		"AsFloat": "func()",
		"AsInt": "func()",
		"AsMap": "func()",
		"AsTableRecords": "func()",
		"Cat": "func()",
		"FormatTime": "func()",
		"HasResource": "func()",
		"Length": "func()",
		"LoadNeatly": "func()",
		"Markdown": "func()",
		"Md5": "func()",
		"Unzip": "func()",
		"UnzipText": "func()",
		"WorkingDirectory": "func()",
		"Zip": "func()",
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"data": {},
		"date": "2018-01-18",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf",
		"env": "func()",
		"ownerURL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
		"params": {
			"awsCredential": "/Users/awitas/.secret/aws.json",
			"ec2InstanceId": "i-0ef8d9260eaf47fdf"
		},
		"rand": 6746951229542070475,
		"time": "2018-01-18 11:01:05",
		"timestamp": "func()",
		"tmpDir": "func()",
		"ts": "201801181101000",
		"uuid": "func()"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/000_main/0003_State.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/000_main/0001_Workflow.Loaded.json", bytes.NewReader([]byte(`{
	"workflow": {
		"Source": {
			"URL": "file:///Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
			"Credential": "",
			"ParsedURL": {
				"Scheme": "file",
				"Opaque": "",
				"User": {},
				"Host": "",
				"Path": "/Users/awitas/go/src/github.com/viant/endly/workflow/ec2.csv",
				"RawPath": "",
				"ForceQuery": false,
				"RawQuery": "",
				"Fragment": ""
			},
			"Cache": "",
			"CacheExpiryMs": 0,
			"Name": "",
			"Type": ""
		},
		"Data": null,
		"Name": "ec2",
		"Description": "Start/Stop ec2 instances",
		"Init": [
			{
				"Name": "awsCredential",
				"Value": null,
				"From": "params.awsCredential",
				"Persist": false,
				"Required": true
			},
			{
				"Name": "ec2InstanceId",
				"Value": null,
				"From": "params.ec2InstanceId",
				"Persist": false,
				"Required": true
			}
		],
		"Post": null,
		"Tasks": [
			{
				"RunCriteria": "",
				"Seq": 0,
				"Name": "start",
				"Description": "start ec2 instance",
				"Actions": [
					{
						"Service": "aws/ec2",
						"Action": "call",
						"Request": {
							"Credential": "$awsCredential",
							"Input": {
								"InstanceIds": [
									"$ec2InstanceId"
								]
							},
							"Method": "DescribeInstances"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "Check instance status",
						"Init": null,
						"Post": [
							{
								"Name": "instanceState",
								"Value": null,
								"From": "Reservations[0].Instances[0].State.Name",
								"Persist": false,
								"Required": false
							}
						],
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "switch-action",
						"Request": {
							"Cases": [
								{
									"Action": "call",
									"Description": "Start Ec2 instance",
									"Request": {
										"Credential": "$awsCredential",
										"Input": {
											"InstanceIds": [
												"$ec2InstanceId"
											]
										},
										"Method": "StartInstances"
									},
									"Service": "aws/ec2",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "stopped"
								},
								{
									"Action": "exit",
									"Description": "exit workflow",
									"Request": {},
									"Service": "workflow",
									"Tag": "StartCases",
									"TagId": "ec2StartCases",
									"Value": "running"
								}
							],
							"SourceKey": "instanceState"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "switch/case for instance status",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 0,
						"Async": false
					},
					{
						"Service": "workflow",
						"Action": "run-task",
						"Request": {
							"Task": "start"
						},
						"Tag": "Start",
						"TagIndex": "",
						"TagID": "ec2Start",
						"TagDescription": "",
						"RunCriteria": "",
						"SkipCriteria": "",
						"Name": "",
						"Description": "goto start task",
						"Init": null,
						"Post": null,
						"SleepTimeMs": 5000,
						"Async": false
					}
				],
				"Init": null,
				"Post": null,
				"TimeSpentMs": 0
			}
		],
		"OnErrorTask": "",
		"SleepTimeMs": 0
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/000_main/0001_Workflow.Loaded.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/000_main/0002_Workflow.Init.json", bytes.NewReader([]byte(`{
	"sources": {
		"params.awsCredential": "/Users/awitas/.secret/aws.json",
		"params.ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"values": {
		"awsCredential": "/Users/awitas/.secret/aws.json",
		"ec2InstanceId": "i-0ef8d9260eaf47fdf"
	},
	"variables": [
		{
			"Name": "awsCredential",
			"Value": null,
			"From": "params.awsCredential",
			"Persist": false,
			"Required": true
		},
		{
			"Name": "ec2InstanceId",
			"Value": null,
			"From": "params.ec2InstanceId",
			"Persist": false,
			"Required": true
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/000_main/0002_Workflow.Init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0019_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:01:19.83852-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0008_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0020_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0020_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:01:19.839453-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0012_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0005_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0007_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "run-task",
		"Description": "goto start task",
		"Error": "",
		"StartTime": "2018-01-18T23:01:25.109034-08:00",
		"Ineligible": false,
		"Request": {
			"Task": "start"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0028_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0004_Action.Post.json", bytes.NewReader([]byte(`{
	"sources": {
		"Reservations[0].Instances[0].State.Name": null
	},
	"values": {
		"instanceState": "running"
	},
	"variables": [
		{
			"Name": "instanceState",
			"Value": null,
			"From": "Reservations[0].Instances[0].State.Name",
			"Persist": false,
			"Required": false
		}
	]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0004_Action.Post.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0018_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0009_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:01:24.845413-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0017_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0010_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "exit",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:01:25.108214-08:00",
		"Ineligible": false,
		"Request": {},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": true
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0024_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Credential": "/Users/awitas/.secret/aws.json",
		"Method": "DescribeInstances",
		"Input": {
			"InstanceIds": [
				"i-0ef8d9260eaf47fdf"
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0002_EC2CallRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"Task": "start"
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0015_WorkflowRunTaskRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"Service": "workflow",
			"Action": "exit",
			"Response": {}
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0026_WorkflowSwitchActionRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "aws/ec2",
		"Action": "call",
		"Description": "Check instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:01:05.874593-08:00",
		"Ineligible": false,
		"Request": {
			"Credential": "$awsCredential",
			"Input": {
				"InstanceIds": [
					"$ec2InstanceId"
				]
			},
			"Method": "DescribeInstances"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0001_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0029_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0029_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:01:25.107637-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0022_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json", bytes.NewReader([]byte(`{
	"request": {
		"SourceKey": "instanceState",
		"Cases": [
			{
				"Service": "aws/ec2",
				"Action": "call",
				"Request": {
					"Credential": "/Users/awitas/.secret/aws.json",
					"Input": {
						"InstanceIds": [
							"i-0ef8d9260eaf47fdf"
						]
					},
					"Method": "StartInstances"
				},
				"Value": "stopped"
			},
			{
				"Service": "workflow",
				"Action": "exit",
				"Request": {},
				"Value": "running"
			}
		],
		"Default": null
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0023_WorkflowSwitchActionRequest.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0011_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Action": "exit",
		"Response": {},
		"Service": "workflow"
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0027_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json", bytes.NewReader([]byte(`{
	"ID": "start"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0016_WorkflowTask.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json", bytes.NewReader([]byte(`{
	"activity": {
		"Tag": "Start",
		"TagIndex": "",
		"TagID": "ec2Start",
		"TagDescription": "",
		"Workflow": "ec2",
		"Service": "workflow",
		"Action": "switch-action",
		"Description": "switch/case for instance status",
		"Error": "",
		"StartTime": "2018-01-18T23:01:19.837905-08:00",
		"Ineligible": false,
		"Request": {
			"Cases": [
				{
					"Action": "call",
					"Description": "Start Ec2 instance",
					"Request": {
						"Credential": "$awsCredential",
						"Input": {
							"InstanceIds": [
								"$ec2InstanceId"
							]
						},
						"Method": "StartInstances"
					},
					"Service": "aws/ec2",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "stopped"
				},
				{
					"Action": "exit",
					"Description": "exit workflow",
					"Request": {},
					"Service": "workflow",
					"Tag": "StartCases",
					"TagId": "ec2StartCases",
					"Value": "running"
				}
			],
			"SourceKey": "instanceState"
		},
		"Response": {},
		"ServiceResponse": null,
		"ExitWorkflow": false,
		"Exit": false
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0006_ServiceAction.Start.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0013_SleepEventType.json", bytes.NewReader([]byte(`{
	"value": {
		"SleepTimeMs": 5000
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0013_SleepEventType.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {
		"NextToken": null,
		"Reservations": [
			{
				"Groups": null,
				"Instances": [
					{
						"AmiLaunchIndex": 0,
						"Architecture": "x86_64",
						"BlockDeviceMappings": [
							{
								"DeviceName": "/dev/sda1",
								"Ebs": {
									"AttachTime": {},
									"DeleteOnTermination": true,
									"Status": "attached",
									"VolumeId": "vol-02a896754c8662379"
								}
							}
						],
						"ClientToken": "",
						"EbsOptimized": false,
						"ElasticGpuAssociations": null,
						"EnaSupport": true,
						"Hypervisor": "xen",
						"IamInstanceProfile": {},
						"ImageId": "ami-82f4dae7",
						"InstanceId": "i-0ef8d9260eaf47fdf",
						"InstanceLifecycle": null,
						"InstanceType": "t2.micro",
						"KernelId": null,
						"KeyName": "aw",
						"LaunchTime": {},
						"Monitoring": {
							"State": "disabled"
						},
						"NetworkInterfaces": [
							{
								"Association": {
									"IpOwnerId": "amazon",
									"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
									"PublicIp": "18.220.19.199"
								},
								"Attachment": {
									"AttachTime": {},
									"AttachmentId": "eni-attach-8038ca6d",
									"DeleteOnTermination": true,
									"DeviceIndex": 0,
									"Status": "attached"
								},
								"Description": "",
								"Groups": [
									{
										"GroupId": "sg-506d8f3b",
										"GroupName": "launch-wizard-1"
									}
								],
								"Ipv6Addresses": null,
								"MacAddress": "06:8c:c8:e6:42:74",
								"NetworkInterfaceId": "eni-23a89b76",
								"OwnerId": "514827850472",
								"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
								"PrivateIpAddress": "172.31.23.14",
								"PrivateIpAddresses": [
									{
										"Association": {
											"IpOwnerId": "amazon",
											"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
											"PublicIp": "18.220.19.199"
										},
										"Primary": true,
										"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
										"PrivateIpAddress": "172.31.23.14"
									}
								],
								"SourceDestCheck": true,
								"Status": "in-use",
								"SubnetId": "subnet-ecaabf97",
								"VpcId": "vpc-96dc17fe"
							}
						],
						"Placement": {
							"Affinity": null,
							"AvailabilityZone": "us-east-2b",
							"GroupName": "",
							"HostId": null,
							"SpreadDomain": null,
							"Tenancy": "default"
						},
						"Platform": null,
						"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
						"PrivateIpAddress": "172.31.23.14",
						"ProductCodes": null,
						"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
						"PublicIpAddress": "18.220.19.199",
						"RamdiskId": null,
						"RootDeviceName": "/dev/sda1",
						"RootDeviceType": "ebs",
						"SecurityGroups": [
							{
								"GroupId": "sg-506d8f3b",
								"GroupName": "launch-wizard-1"
							}
						],
						"SourceDestCheck": true,
						"SpotInstanceRequestId": null,
						"SriovNetSupport": null,
						"State": {
							"Code": 16,
							"Name": "running"
						},
						"StateReason": {},
						"StateTransitionReason": "",
						"SubnetId": "subnet-ecaabf97",
						"Tags": [
							{
								"Key": "Name",
								"Value": "aw_test"
							}
						],
						"VirtualizationType": "hvm",
						"VpcId": "vpc-96dc17fe"
					}
				],
				"OwnerId": "514827850472",
				"RequesterId": null,
				"ReservationId": "r-077a2b5729332a117"
			}
		]
	},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0021_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json", bytes.NewReader([]byte(`{
	"response": {
		"Status": "ok",
		"Error": "",
		"Response": {
			"NextToken": null,
			"Reservations": [
				{
					"Groups": null,
					"Instances": [
						{
							"AmiLaunchIndex": 0,
							"Architecture": "x86_64",
							"BlockDeviceMappings": [
								{
									"DeviceName": "/dev/sda1",
									"Ebs": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"DeleteOnTermination": true,
										"Status": "attached",
										"VolumeId": "vol-02a896754c8662379"
									}
								}
							],
							"ClientToken": "",
							"EbsOptimized": false,
							"ElasticGpuAssociations": null,
							"EnaSupport": true,
							"Hypervisor": "xen",
							"IamInstanceProfile": null,
							"ImageId": "ami-82f4dae7",
							"InstanceId": "i-0ef8d9260eaf47fdf",
							"InstanceLifecycle": null,
							"InstanceType": "t2.micro",
							"KernelId": null,
							"KeyName": "aw",
							"LaunchTime": "2018-01-19T06:23:01Z",
							"Monitoring": {
								"State": "disabled"
							},
							"NetworkInterfaces": [
								{
									"Association": {
										"IpOwnerId": "amazon",
										"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
										"PublicIp": "18.220.19.199"
									},
									"Attachment": {
										"AttachTime": "2018-01-09T17:43:33Z",
										"AttachmentId": "eni-attach-8038ca6d",
										"DeleteOnTermination": true,
										"DeviceIndex": 0,
										"Status": "attached"
									},
									"Description": "",
									"Groups": [
										{
											"GroupId": "sg-506d8f3b",
											"GroupName": "launch-wizard-1"
										}
									],
									"Ipv6Addresses": null,
									"MacAddress": "06:8c:c8:e6:42:74",
									"NetworkInterfaceId": "eni-23a89b76",
									"OwnerId": "514827850472",
									"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
									"PrivateIpAddress": "172.31.23.14",
									"PrivateIpAddresses": [
										{
											"Association": {
												"IpOwnerId": "amazon",
												"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
												"PublicIp": "18.220.19.199"
											},
											"Primary": true,
											"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
											"PrivateIpAddress": "172.31.23.14"
										}
									],
									"SourceDestCheck": true,
									"Status": "in-use",
									"SubnetId": "subnet-ecaabf97",
									"VpcId": "vpc-96dc17fe"
								}
							],
							"Placement": {
								"Affinity": null,
								"AvailabilityZone": "us-east-2b",
								"GroupName": "",
								"HostId": null,
								"SpreadDomain": null,
								"Tenancy": "default"
							},
							"Platform": null,
							"PrivateDnsName": "ip-172-31-23-14.us-east-2.compute.internal",
							"PrivateIpAddress": "172.31.23.14",
							"ProductCodes": null,
							"PublicDnsName": "ec2-18-220-19-199.us-east-2.compute.amazonaws.com",
							"PublicIpAddress": "18.220.19.199",
							"RamdiskId": null,
							"RootDeviceName": "/dev/sda1",
							"RootDeviceType": "ebs",
							"SecurityGroups": [
								{
									"GroupId": "sg-506d8f3b",
									"GroupName": "launch-wizard-1"
								}
							],
							"SourceDestCheck": true,
							"SpotInstanceRequestId": null,
							"SriovNetSupport": null,
							"State": {
								"Code": 16,
								"Name": "running"
							},
							"StateReason": null,
							"StateTransitionReason": "",
							"SubnetId": "subnet-ecaabf97",
							"Tags": [
								{
									"Key": "Name",
									"Value": "aw_test"
								}
							],
							"VirtualizationType": "hvm",
							"VpcId": "vpc-96dc17fe"
						}
					],
					"OwnerId": "514827850472",
					"RequesterId": null,
					"ReservationId": "r-077a2b5729332a117"
				}
			]
		}
	}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0003_EC2CallRequest.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0025_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json", bytes.NewReader([]byte(`{
	"response": {},
	"value": {}
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/logs/84cbbec4-fce6-11e7-9beb-784f438e6f38/001_ec2Start/0014_ServiceAction.End.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/dockerized_memcached_init.json", bytes.NewReader([]byte(`[
  {
    "Name": "serviceInstanceName",
    "From": "params.serviceInstanceName",
    "Required": true
  },
  {
    "Name": "targetHost",
    "From": "params.targetHost",
    "Required": true
  },
  {
    "Name": "targetHostCredential",
    "From": "params.targetHostCredential",
    "Required": true
  },
  {
    "Name": "configURL",
    "From": "params.configURL",
    "Required": true
  },
  {
    "Name": "configURLCredential",
    "From": "params.configURLCredential",
    "Required": true
  },
  {
    "Name": "dockerTarget",
    "Value": {
      "Name": "$serviceInstanceName",
      "URL": "scp://${targetHost}/",
      "Credential": "$targetHostCredential"
    }
  },
  {
    "Name": "maxMemory",
    "From": "params.maxMemory",
    "Value": {
      "Target": "512m"
    }
  }

]`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/dockerized_memcached_init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/ec2_init.json", bytes.NewReader([]byte(`[
  {
    "Name":"awsCredential",
    "From":"params.awsCredential",
    "Required":true
  },
  {
    "Name":"ec2InstanceId",
    "From":"params.ec2InstanceId",
    "Required":true
  }
]`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/ec2_init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/dockerized_aerospike_init.json", bytes.NewReader([]byte(`[
  {
    "Name": "serviceInstanceName",
    "From": "params.serviceInstanceName",
    "Required": true
  },
  {
    "Name": "targetHost",
    "From": "params.targetHost",
    "Required": true
  },
  {
    "Name": "targetHostCredential",
    "From": "params.targetHostCredential",
    "Required": true
  },
  {
    "Name": "configURL",
    "From": "params.configURL",
    "Required": true
  },
  {
    "Name": "configURLCredential",
    "From": "params.configURLCredential",
    "Required": true
  },
  {
    "Name": "dockerTarget",
    "Value": {
      "Name": "$serviceInstanceName",
      "URL": "scp://${targetHost}/",
      "Credential": "$targetHostCredential"
    }
  },


  {
    "Name": "configFile",
    "Value": "/tmp/aerospike${serviceInstanceName}.conf"
  }
]`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/dockerized_aerospike_init.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/vc_maven_module_build_var.json", bytes.NewReader([]byte(`[
  {
    "Name": "jdkVersion",
    "From": "params.jdkVersion",
    "Required": true
  },
  {
    "Name":"mavenVersion",
    "From":"params.mavenVersion",
    "Required": true
  },
  {
    "Name": "buildGoal",
    "From": "params.buildGoal",
    "Value": "install"
  },
  {
    "Name": "buildArgs",
    "From": "params.buildArgs",
    "Required": true
  },
  {
    "Name": "originType",
    "From": "params.originType",
    "Required": true
  },
  {
    "Name": "originUrl",
    "From": "params.originUrl",
    "Required": true
  },
  {
    "Name": "originCredential",
    "From": "params.originCredential",
    "Required": true
  },
  {
    "Name": "targetUrl",
    "From": "params.targetUrl",
    "Required": true
  },
  {
    "Name": "targetHostCredential",
    "From": "params.targetHostCredential",
    "Required": true
  },
  {
    "Name": "modules",
    "From": "params.modules",
    "Required": true
  },
  {
    "Name": "parentPomUrl",
    "From": "params.parentPomUrl",
    "Required": true
  },
  {
    "Name": "origin",
    "Value": {
      "Type": "$originType",
      "URL": "$originUrl",
      "Credential": "$originCredential"
    }
  },
  {
    "Name": "buildTarget",
    "Value": {
      "URL": "$targetUrl",
      "Credential": "$targetHostCredential"
    }
  },

  {
    "Name": "transferPomRequest",
    "Value": {
      "Transfers": [
        {
          "Source": {
            "URL": "$parentPomUrl",
            "Credential": "$originCredential"
          },
          "Target": {
            "URL": "${targetUrl}",
            "Credential": "$targetHostCredential"
          }
        }
      ]
    }
  },

  {
    "Name": "checkoutRequest",
    "Value": {
      "Origin": "$origin",
      "Modules": "$modules",
      "Target": "$buildTarget"
    }
  },
  {
    "Name": "buildRequest",
    "Value": {
      "BuildSpec": {
        "Name": "maven",
        "Version": "$mavenVersion",
        "Goal": "build",
        "BuildGoal": "$buildGoal",
        "Args": "$buildArgs",
        "Sdk": "jdk",
        "SdkVersion": "$jdkVersion"
      },
      "Target": "$buildTarget"
    }
  }
]`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/vc_maven_module_build_var.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/dockerized_mysql.csv", bytes.NewReader([]byte(`Workflow,,Name,Description,Tasks,,Init
,,dockerized_mysql,This workflow manages mysql as docker service,%Tasks,,#dockerized_mysql_init
[]Tasks,,Name,Description,Actions,,
,,start,"This task will stop system mysql if it is running, it only run if  $param.stopSystemMysql is true",%Start,,
[]Start,Service,Action,Description,Request,RunCriteria,Post
,daemon,status,Check if system mysql service is running,#req/mysql_system,$params.stopSystemMysql:true,"[{""name"":""mysqlSystemStatus"", ""from"":""State""}]"
,daemon,stop,Stop system mysql service,#req/mysql_system,$mysqlSystemStatus:running,
,transfer,copy,This action will copy template config file to the temp folder,#req/docker_config,,
,docker,run,Start docker mysql service,#req/docker_mysql,,
[]Tasks,,Name,Description,Actions,,
,,stop,This task will stop docker mysql,%Stop,,
[]Stop,Service,Action,Description,Request,,
,docker,container-stop,Stop docker mysql service,#req/docker,,
[]Tasks,,Name,Description,Actions,,
,,export,Export mysql schema from docker,%Export,,
[]Export,Service,Action,Description,Request,,
,docker,container-command,Export all databases,#req/docker_mysql_export,,
[]Tasks,,Name,Description,Actions,,
,,import,Export mysql schema from docker,%Import,,
[]Import,Service,Action,Description,Request,,
,docker,container-command,Import mysql dump,#req/docker_mysql_import,,`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/dockerized_mysql.csv %v", err)
		}
	}
}
