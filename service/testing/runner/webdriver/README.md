**webdriver Runner** 


webdriver runner opens a web session to run a various action on web driver or web elements.


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| webdriver | start | start standalone webdriver server | [ServerStartRequest](contract.go) | [ServerStartResponse](contract.go) |
| webdriver | stop | stop standalone webdriver server | [ServerStopRequest](contract.go) | [ServerStopResponse](contract.go) |
| webdriver | open | open a new browser with session id for further testing | [OpenSessionRequest](contract.go) | [OpenSessionResponse](contract.go) |
| webdriver | close | close browser session | [CloseSessionRequest](contract.go) | [CloseSessionResponse](contract.go) |
| webdriver | call-driver | call a method on web driver, i.e wb.GET(url)| [WebDriverCallRequest](contract.go) | [ServiceCallResponse](contract.go) |
| webdriver | call-element | call a method on a web element, i.e. we.Click() | [WebElementCallRequest](contract.go) | [WebElementCallResponse](contract.go) |
| webdriver | run | run set of action on a page | [RunRequest](contract.go) | [RunResponse](contract.go) |

call-driver and call-element actions's method and parameters are proxied to stand along webdriver server via [webdriver client](http://github.com/tebeka/webdriver)

See [webdriver selector](https://www.lambdatest.com/blog/complete-guide-for-using-xpath-in-selenium-with-examples/)
for more details on how to use xpath, css, id, name, class, tag, link, partial link, dom, and xpath selectors.


webdriver run request defines sequence of action/commands. In case a selector is not specified, call method's caller is a [WebDriver](https://github.com/tebeka/webdriver/blob/master/webdriver.go#L213), 
otherwise [WebElement](https://github.com/tebeka/webdriver/blob/master/webdriver.go#L370) defined by selector.
[Wait](./../../repeatable.go)  provides ability to wait either some time amount or for certain condition to take place, with regexp to extract data

Run request provide commands expression for easy webdriver interaction:

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

[@test.yaml](test/example_test.yaml)
 
```yaml
defaults:
  target:
     URL: ssh://127.0.0.1/
     credentials: localhost
pipeline:
  init:
    action: webdriver:start
  test:
    action: webdriver:run
    remotewebdriver:
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
 

    


