# Deployment service
Deployment service checks if specified app has been installed with requested version,
if not is uses meta descriptor to install missing app

- [Usage](#usage)
    - [Maven](#maven)
    - [Tomcat](#tomcat)
    
    
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

```endly tomcat.yaml```
[@tomcat.yaml](usage/tomcat.yaml)

```yaml
init:
  target:
    URL: ssh://127.0.0.1
    Credentials: localhost

pipeline:
  deployTomcat:
    action: deployment:deploy
    target: $target
    baseLocation: /use/local/
    appName: tomcat
    version: 7.0  
```

### Gecko driver


```endly geckodriver.yaml```
[@geckodriver.yaml](usage/geckodriver.yaml)
```yaml
init:
  target:
    URL: ssh://127.0.0.1
    Credentials: localhost

pipeline:
  task1:
    action: deployment:deploy
    target: $target
    appName: geckodriver
```

### Selenium


```endly selenium.yaml```
[@selenium.yaml](usage/selenium.yaml)
```yaml
init:
  target:
    URL: ssh://127.0.0.1
    Credentials: localhost

pipeline:
  task1:
    action: deployment:deploy
    target: $target
    appName: selenium-server-standalone
    version: 3.4
```

### Go SDK

```endly gosdk.yaml```
[@gosdk.yaml](usage/gosdk.yaml)

```yaml
init:
  target:
    URL: ssh://127.0.0.1
    Credentials: localhost

pipeline:
  task1:
    action: deployment:deploy
    target: $target
    baseLocation: /usr/local
    appName: go
    version: 1.12

```

### Node SDK



### JDK SDK


