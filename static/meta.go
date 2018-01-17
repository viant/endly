package static

import (
	"bytes"
	"github.com/viant/toolbox/storage"
	"log"
)

func init() {
	var memStorage = storage.NewMemoryService();
	{
		err := memStorage.Upload("mem://github.com/viant/endly/meta/deployment/maven.json", bytes.NewReader([]byte(`{
  "Name": "maven",
  "Versioning": "MajorVersion.MinorVersion.ReleaseVersion",
  "Targets": [
    {
      "MinReleaseVersion": {
        "3.5": "2"
      },
      "Deployment": {
        "Pre": {
          "SuperUser": true,
          "Commands": [
            "mkdir -p /opt/build/",
            "chmod a+rw /opt/build/"
          ]
        },
        "Transfer": {
          "Source": {
            "URL": "http://mirrors.gigenet.com/apache/maven/maven-${artifact.MajorVersion}/${artifact.Version}/binaries/apache-maven-${artifact.Version}-bin.tar.gz"
          },
          "Target": {
            "Name": "apache-maven",
            "URL": "scp://${buildSpec.host}/opt/build/",
            "Credential": "$buildSpec.credential"
          }
        },
        "VersionCheck": {
          "Options": {
            "SystemPaths": [
              "/opt/build/maven/bin"
            ]
          },
          "Executions": [
            {
              "Command": "mvn -version",
              "Extraction": [
                {
                  "Key": "Version",
                  "RegExpr": "Apache Maven (\\d+\\.\\d+\\.\\d+)"
                }
              ]
            }
          ]
        },
        "Command": {
          "Options": {
            "Directory": "/opt/build/"
          },
          "Executions": [
            {
              "Command": "tar xvzf apache-maven-${artifact.Version}-bin.tar.gz",
              "Error": [
                "Error"
              ]
            },
            {
              "Command": "/bin/bash -c '[[ -e /opt/build/maven ]] && rm -rf /opt/build/maven'"
            },
            {
              "Command": "mv apache-maven-${artifact.Version} maven",
              "Error": [
                "No"
              ]
            }
          ]
        }
      }
    }
  ]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/meta/deployment/maven.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/meta/deployment/geckodriver.json", bytes.NewReader([]byte(`{
    "Name": "geckodriver",
  "Targets": [
    {
      "OsTarget": {
        "System": "darwin"
      },
      "Deployment": {
        "Pre": {
          "SuperUser": true,
          "Commands": [
            "mkdir -p /opt/selenium/",
            "chmod a+rw /opt/selenium/"
          ]
        },
        "Transfer": {
          "Source": {
            "URL": "https://github.com/mozilla/geckodriver/releases/download/v0.19.1/geckodriver-v0.19.1-macos.tar.gz"
          },
          "Target": {
            "URL": "scp://${targetHost}/opt/selenium/geckodriver-v0.19.1-macos.tar.gz",
            "Credential": "${targetHostCredential}"
          }
        },
        "Command": {
          "Options": {
            "Directory": "/opt/selenium"
          },
          "Executions": [
            {
              "Command": "tar xvzf geckodriver-v0.19.1-macos.tar.gz",
              "Error": [
                "Error"
              ]
            }
          ]
        }
      }
    },
    {
      "OsTarget": {
        "System": "linux"
      },
      "Deployment": {
        "Pre": {
          "SuperUser": true,
          "Commands": [
            "mkdir -p /opt/selenium/",
            "chmod a+rw /opt/selenium/"
          ]
        },
        "Transfer": {
          "Source": {
            "URL": "https://github.com/mozilla/geckodriver/releases/download/v0.19.1/geckodriver-v0.19.1-linux64.tar.gz"
          },
          "Target": {
            "URL": "scp://${targetHost}/opt/selenium/",
            "Credential": "${targetHostCredential}"
          }
        },
        "Command": {
          "Options": {
            "Directory": "/opt/selenium"
          },
          "Executions": [
            {
              "Command": "tar xvzf geckodriver-v0.19.1-macos.tar.gz",
              "Error": [
                "Error"
              ]
            }
          ]
        }
      }
    }
  ]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/meta/deployment/geckodriver.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/meta/deployment/go.json", bytes.NewReader([]byte(`{
  "Name": "go",
  "Versioning": "MajorVersion.MinorVersion.ReleaseVersion",
  "Targets": [
    {
      "MinReleaseVersion": {
        "1.8": "5",
        "1.2": "2"
      },
      "Deployment": {
        "Pre": {
          "SuperUser": true,
          "Commands": [
            "mkdir -p /opt/sdk/",
            "chmod a+rw /opt/sdk/"
          ]
        },
        "Transfer": {
          "Source": {
            "URL": "https://redirector.gvt1.com/edgedl/go/go${artifact.Version}.${os.System}-${os.Architecture}.tar.gz"
          },
          "Target": {
            "URL": "scp://${targetHost}/opt/sdk/go_${artifact.Version}.tar.gz",
            "Credential": "${targetHostCredential}"
          }
        },
        "VersionCheck": {
          "Options": {
            "SystemPaths": [
              "/opt/sdk/go/bin"
            ]
          },
          "Executions": [
            {
              "Command": "go version",
              "Extraction": [
                {
                  "Key": "Version",
                  "RegExpr": "go(\\d\\.\\d)"
                }
              ]
            }
          ]
        },
        "Command": {
          "Options": {
            "Directory": "/opt/sdk",
            "TimeoutMs": 120000
          },
          "Executions": [
            {
              "Command": "/bin/bash -c '[[ -e /opt/sdk/go ]] && rm -rf /opt/sdk/go'"
            },
            {
              "Command": "tar xvzf go_${artifact.Version}.tar.gz",
              "Error": [
                "Error"
              ]
            }
          ]
        }
      }
    }
  ]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/meta/deployment/go.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/meta/deployment/jdk.json", bytes.NewReader([]byte(`{
  "Name": "java",
  "Targets": [
    {
      "Version": "1.7",
      "OsTarget": {
        "System": "linux"
      },
      "Deployment": {
        "Pre": {
          "SuperUser": true,
          "Commands": [

            "mkdir -p /opt/sdk/jdk",
            "chmod a+rw /opt/sdk/jdk",
            "mkdir -p /usr/lib/jvm",
            "chmod a+rw /usr/lib/jvm"
          ]
        },
        "Transfer": {
          "Source": {
            "URL": "sdk/jdk-7u80-linux-x64.tar.gz"
          },
          "Target": {
            "URL": "scp://${targetHost}/opt/sdk/jdk/jdk-7u80-linux-x64.tar.gz",
            "Credential": "${targetHostCredential}"
          }
        },
        "VersionCheck": {
          "Options": {
            "SystemPaths": [
              "/usr/lib/jvm/java-7-oracle/bin"
            ]
          },
          "Executions": [
            {
              "Command": "java -version",
              "Extraction": [
                {
                  "Key": "Version",
                  "RegExpr": "build (\\d\\.\\d).+"
                }
              ]
            }
          ]
        },
        "Command": {
          "Options": {
            "Directory": "/opt/sdk/jdk"
          },
          "Executions": [
            {
              "Command": "tar xvzf jdk-7u80-linux-x64.tar.gz",
              "Error": [
                "Error"
              ]
            },
            {
              "Command": "/bin/bash -c '[[ -e /usr/lib/jvm/java-7-oracle ]] && rm -rf /usr/lib/jvm/java-7-oracle'"
            },
            {
              "Command": "mkdir -p /usr/lib/jvm/java-7-oracle"
            },
            {
              "Command": "cp -rf /opt/sdk/jdk/jdk1.7.0_80/* /usr/lib/jvm/java-7-oracle/",
              "Error": [
                "No"
              ]
            }
          ]
        }
      }
    }
  ]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/meta/deployment/jdk.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/meta/deployment/tomcat.json", bytes.NewReader([]byte(`{
  "Name": "tomcat",
  "Versioning": "MajorVersion.MinorVersion.ReleaseVersion",
  "Targets": [
    {
      "MinReleaseVersion": {
        "7.0": "82"
      },
      "Deployment": {
        "Pre": {
          "SuperUser": true,
          "Commands": [
            "rm -rf $appDirectory",
            "mkdir -p $appDirectory",
            "chmod  -R a+rw $appDirectory"
          ]
        },
        "Transfer": {
          "Source": {
            "URL": "http://mirror.metrocast.net/apache/tomcat/tomcat-${artifact.MajorVersion}/v${artifact.Version}/bin/apache-tomcat-${artifact.Version}.tar.gz"
          },
          "Target": {
            "Name": "tomcat",
            "Version": "$tomcatVersion",
            "URL": "scp://${targetHost}/${appDirectory}/apache-tomcat-${artifact.Version}.tar.gz",
            "Credential": "$targetHostCredential"
          }
        },
        "Command": {
          "Options": {
            "Directory": "$appDirectory"
          },
          "Executions": [
            {
              "Command": "tar xvzf apache-tomcat-${artifact.Version}.tar.gz",
              "Error": [
                "Error"
              ]
            },
            {
              "Command": "mv apache-tomcat-${artifact.Version} tomcat",
              "Error": [
                "No"
              ]
            }
          ]
        },
        "VersionCheck": {
          "Executions": [
            {
              "Command": "sh tomcat/bin/version.sh",
              "Extraction": [
                {
                  "Key": "Version",
                  "RegExpr": "Apache Tomcat/(\\d+\\.\\d+\\.\\d+)"
                }
              ]
            }
          ]
        },
        "Post": {
          "Commands": [
            "mkdir -p $appDirectory/tomcat/logs",
            "mkdir -p $appDirectory/tomcat/conf",
            "chmod  -R a+rw $appDirectory"
          ],
          "Transfers": [
            {
              "Source": {
                "URL": "$configUrl",
                "Credential": "$configURLCredential"
              },
              "Target": {
                "URL": "scp://${targetHost}${appDirectory}/tomcat/conf/server.xml",
                "Credential": "$targetHostCredential"
              },
              "Expand": true
            }
          ]
        }
      }
    }
  ]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/meta/deployment/tomcat.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/meta/deployment/selenium-server-standalone.json", bytes.NewReader([]byte(`{
  "Name": "selenium-server-standalone",
  "Versioning":"MajorVersion.MinorVersion.ReleaseVersion",
  "Targets": [
    {

      "MinReleaseVersion": {
        "3.4": "0"
      },
      "Deployment": {

        "Pre": {
          "SuperUser": true,
          "Commands": [
            "mkdir -p /opt/selenium/",
            "chmod a+rw /opt/selenium/"
          ]
        },

        "Transfer": {
          "Source": {
            "URL": "http://selenium-release.storage.googleapis.com/${artifact.MajorVersion}.${artifact.MinorVersion}/selenium-server-standalone-${artifact.Version}.jar"
          },
          "Target": {
            "URL": "scp://${targetHost}/opt/selenium/selenium-server-standalone.jar",
            "Credential": "${targetHostCredential}"
          }
        }
      }
    }
  ]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/meta/deployment/selenium-server-standalone.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/meta/build/maven.json", bytes.NewReader([]byte(`{
  "Name": "maven",
  "Dependencies": [
    {
      "Name": "maven",
      "Version": "${buildSpec.version}"
    }
  ],
  "Goals": [
    {
      "Name": "build",
      "Command": {
        "Options": {
          "Directory": "$buildSpec.path",
          "TimeoutMs": 720000
        },
        "Executions": [
          {
            "Command": "cd $buildSpec.path"
          },
          {
            "Command": "mvn clean $buildSpec.args",
            "Errors": [
              "Error",
              "command not found"
            ]
          },
          {
            "Command": "mvn clean $buildSpec.args",
            "Errors": [
              "Error",
              "command not found"
            ]
          },
          {
            "Command": "mvn $buildSpec.goal $buildSpec.args",
            "Success": [
              "BUILD SUCCESS"
            ],
            "Extraction": [
              {
                "Key": "Artifact",
                "RegExpr": "Building jar:[^\/]+(.+)"
              }
            ]
          }
        ]
      }
    }
  ]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/meta/build/maven.json %v", err)
		}
	}
	{
		err := memStorage.Upload("mem://github.com/viant/endly/meta/build/go.json", bytes.NewReader([]byte(`{
  "Name": "go",
  "Dependencies": [
    {
      "Name": "$buildSpec.sdk",
      "Version": "$buildSpec.sdkVersion"
    }
  ],
  "Goals": [
    {
      "Name": "build",
      "Command": {
        "Options": {
          "Directory": "$buildSpec.path",
          "TimeoutMs": 120000,
          "Env": {
            "GIT_TERMINAL_PROMPT": "1"
          },
          "Terminators": [
            "Error",
            "command not found",
            "imported and not used",
            "package ",
            "Password",
            "in single-value context",
            "cannot use "
          ]
        },
        "Executions": [
          {
            "Command": "cd $buildSpec.path"
          },
          {
            "Command": "go clean",
            "Errors": [
              "Error",
              "command not found"
            ]
          },
          {
            "Command": "go get -u .",
            "Errors": [
              "Error",
              "command not found"
            ]
          },
          {
            "MatchOutput": "Password",
            "Command": "**git**",
            "Error": [
              "Password"
            ]
          },
          {
            "Command": "go ${buildSpec.goal} ${buildSpec.args}",
            "Error": [
              "failed",
              "error",
              "imported and not used",
              "in single-value context",
              "package ",
              "cannot use "
            ]
          }
        ]
      }
    }
  ]
}`)))
		if err != nil {
			log.Printf("failed to upload: mem://github.com/viant/endly/meta/build/go.json %v", err)
		}
	}
}
