package static

import (
	"bytes"
	"github.com/viant/toolbox/storage"
	"log"
)

func init() {
	var memStorage = storage.NewMemoryService();
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
,deployment,deploy,Deploy tomcat server into target host,#req/tomcat_deploy.json,,,,
[]Tasks,,Name,Description,Actions,,,,
,,start,This task will start tomcat,%Start,,,,
[]Start,Service,Action,Description,Request,RunCriteria,Variables,Post,
,sdk,set,Set jdk,#req/set_jdk.json,,,,
,process,status,Check tomcat process,#req/tomcat_check.json,,,"[{""name"":""tomcatPid"", ""from"":""Pid"", ""value"":""0""}]",
,exec,extractable-command,Stop tomcat server,#req/tomcat_stop.json,$tomcatPid:!0,,,
,exec,extractable-command,Start tomcat server,#req/tomcat_start.json,,,,
[]Tasks,,Name,Description,Actions,,,,
,,stop,This task will stop tomcat,%Stop,,,,
[]Stop,Service,Action,Description,Request,RunCriteria,,Post,SleepTimeMs
,sdk,set,Set jdk,#req/set_jdk.json,,,,
,process,status,Check tomcat process,#req/tomcat_check.json,,,"[{""name"":""tomcatPid"", ""from"":""Pid"", ""value"":""0""}]",
,exec,extractable-command,Stop tomcat server,#req/tomcat_stop.json,$tomcatPid:!0,,,500
,process,status,Check tomcat process,#req/tomcat_check.json,,,"[{""name"":""tomcatPid"", ""from"":""Pid"", ""value"":""0""}]",
,process,stop,Kill tomcat process,#req/tomcat_kill.json,$tomcatPid:!0,,,`)))
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
,,vc_maven_build,Workflow check outs/pulls latest change from vc to maven build it,%Tasks,#vc_maven_build_var.json
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
,,vc_maven_module_build,Workflow check outs/pulls latest change from vc to maven build it,%Tasks,#vc_maven_module_build_var.json
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
,,dockerized_memcached,This workflow manages memcached service,%Tasks,,#dockerized_memcached_init.json
[]Tasks,,Name,Description,Actions,,
,,start,This task will start memcached docker service,%Start,,
[]Start,Service,Action,Description,Request,RunCriteria,Post
,daemon,status,Check if system docker service is running,#req/docker_system.json,,"[{""name"":""dockerSystemStatus"", ""from"":""State""}]"
,daemon,start,Start docker service,#req/docker_system.json,$dockerSystemStatus:!running,
,docker,run,Start docker memcached service,#req/docker_memcached.json,,
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
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/req/docker_system.json", bytes.NewReader([]byte(`{
  "Service": "docker",
  "Target": "$dockerTarget"
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/req/docker_system.json %v", err)
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
		err := memStorage.Upload("mem://github.com/viant/endly/workflow/dockerized_aerospike.csv", bytes.NewReader([]byte(`Workflow,,Name,Description,Tasks,,Init,
,,dockerized_aerospike,This workflow managed aeropsike docker service,%Tasks,,#dockerized_aerospike_init.json,
[]Tasks,,Name,Description,Actions,RunCriteria,,
,,start,Start aerospike contaner,%Start,,,
[]Start,Service,Action,Description,Request,RunCriteria,,Post
,daemon,status,Check if system docker service is running,#req/docker_system.json,,,"[{""name"":""dockerSystemStatus"", ""from"":""State""}]"
,daemon,start,Start docker service,#req/docker_system.json,$dockerSystemStatus:!running,,
,transfer,copy,This action will copy template config file to the temp folder,#req/docker_config.json,$configURL:/!configURL/,,
,docker,run,Start docker aerospike service,#req/docker_aerospike.json,,,
[]Tasks,,Name,Description,Actions,,,
,,stop,Stop aerospike container,%Stop,,,
[]Stop,Service,Action,Description,Request,,,
,docker,container-stop,Start docker aerospike service,#req/docker.json,,,`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/dockerized_aerospike.csv %v", err)
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
,,dockerized_mysql,This workflow manages mysql as docker service,%Tasks,,#dockerized_mysql_init.json
[]Tasks,,Name,Description,Actions,,
,,start,"This task will stop system mysql if it is running, it only run if  $param.stopSystemMysql is true",%Start,,
[]Start,Service,Action,Description,Request,RunCriteria,Post
,daemon,status,Check if system mysql service is running,#req/mysql_system.json,$params.stopSystemMysql:true,"[{""name"":""mysqlSystemStatus"", ""from"":""State""}]"
,daemon,stop,Stop system mysql service,#req/mysql_system.json,$mysqlSystemStatus:running,
,daemon,status,Check if system docker service is running,#req/docker_system.json,,"[{""name"":""dockerSystemStatus"", ""from"":""State""}]"
,daemon,start,Start docker service,#req/docker_system.json,$dockerSystemStatus:!running,
,transfer,copy,This action will copy template config file to the temp folder,#req/docker_config.json,,
,docker,run,Start docker mysql service,#req/docker_mysql.json,,
[]Tasks,,Name,Description,Actions,,
,,stop,This task will stop docker mysql,%Stop,,
[]Stop,Service,Action,Description,Request,,
,docker,container-stop,Stop docker mysql service,#req/docker.json,,
[]Tasks,,Name,Description,Actions,,
,,export,Export mysql schema from docker,%Export,,
[]Export,Service,Action,Description,Request,,
,docker,container-command,Export all databases,#req/docker_mysql_export.json,,
[]Tasks,,Name,Description,Actions,,
,,import,Export mysql schema from docker,%Import,,
[]Import,Service,Action,Description,Request,,
,docker,container-command,Import mysql dump,#req/docker_mysql_import.json,,`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/workflow/dockerized_mysql.csv %v", err)
		}
	}
}
