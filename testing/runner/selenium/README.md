**Selenium Runner** 


Selenium runner opens a web session to run a various action on web driver or web elements.


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| selenium | start | start standalone selenium server | [ServerStartRequest](service_contract.go) | [ServerStartResponse](service_contract.go) |
| selenium | stop | stop standalone selenium server | [ServerStopRequest](service_contract.go) | [ServerStopResponse](service_contract.go) |
| selenium | open | open a new browser with session id for further testing | [OpenSessionRequest](service_contract.go) | [OpenSessionResponse](service_contract.go) |
| selenium | close | close browser session | [CloseSessionRequest](service_contract.go) | [CloseSessionResponse](service_contract.go) |
| selenium | call-driver | call a method on web driver, i.e wb.GET(url)| [WebDriverCallRequest](service_contract.go) | [ServiceCallResponse](service_contract.go) |
| selenium | call-element | call a method on a web element, i.e. we.Click() | [WebElementCallRequest](service_contract.go) | [WebElementCallResponse](service_contract.go) |
| selenium | run | run set of action on a page | [RunRequest](service_contract.go) | [RunResponse](service_contract.go) |

call-driver and call-element actions's method and parameters are proxied to stand along selenium server via [selenium client](http://github.com/tebeka/selenium)



Selenium run request defines sequence of action/commands. In case a selector is not specified, call method's caller is a [WebDriver](https://github.com/tebeka/selenium/blob/master/selenium.go#L213), 
otherwise [WebElement](https://github.com/tebeka/selenium/blob/master/selenium.go#L370) defined by selector.
[Wait](./../../repeatable.go)  provides ability to wait either some time amount or for certain condition to take place, with regexp to extract data

Run request provide commands expression for easy selenium interaction:

Command syntax:
```text
  [RESULT_KEY=] [(WEB_ELEMENT_SELECTOR).]METHOD_NAME(PARAMETERS)
  
  i.e:
  (#name).sendKeys('dummy 123')
  (xpath://SELECT[@id='typeId']/option[text()='type1']).click()
  get(http://127.0.0.1:8080/form.html)
  
```  


Time wait
```text
    - command: CurrentURL = CurrentURL()
    exit: $CurrentURL:/dummy/
    sleepTimeMs: 1000
    repeat: 10

```

 
 
 
### Inline pipeline tasks

```bash
endly -r=test
```

[@test.yaml](test/test.yaml)
 
```yaml
defaults:
  target:
     URL: ssh://127.0.0.1/
     credentials: localhost
pipeline:
  init:
    action: selenium:start
    version: 3.4.0
    port: 8085
    sdk: jdk
    sdkVersion: 1.8
  test:
    action: selenium:run
    browser: firefox
    remoteSelenium:
      URL: http://127.0.0.1:8085
    commands:
      - get(http://play.golang.org/?simple=1)
      - (#code).clear
      - (#code).sendKeys(package main

          import "fmt"

          func main() {
              fmt.Println("Hello Endly!")
          }
        )
      - (#run).click
      - command: output = (#output).text
        exit: $output.Text:/Endly/
        sleepTimeMs: 1000
        repeat: 10
      - close
    expect:
      output:
        Text: /Hello Endly!/

```
 

    


