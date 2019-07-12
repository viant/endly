# Deployment service
Deployment service checks if specified app has been installed with requested version,
if not is uses meta descriptor to install missing app

## Usage

### Maven

```endly maven.yaml```

[@maven.yaml](usage/maven.yaml)
```yaml
init:
  target:
    URL: ssh://127.0.0.1
    Credentials: localhost

pipeline:
  task1:
    action: deployment:deploy
    target: $target
    baseLocation: /tmp/
    appName: maven
    version: 3.5

```

Maven uses the following deployment [descriptor](https://github.com/viant/endly/blob/master/shared/meta/deployment/maven.json)

### Tomcat

### Gecko driver

### Selenium

### Go SDK

### Node SDK

### JDK SDK


