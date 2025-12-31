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
| webdriver | capture-start | start capturing console+network (Chrome/Edge) | [CaptureStartRequest](contract.go) | [CaptureStartResponse](contract.go) |
| webdriver | capture-stop | stop capturing console+network | [CaptureStopRequest](contract.go) | [CaptureStopResponse](contract.go) |
| webdriver | capture-status | get capture counters | [CaptureStatusRequest](contract.go) | [CaptureStatusResponse](contract.go) |
| webdriver | capture-clear | clear capture buffers | [CaptureClearRequest](contract.go) | [CaptureClearResponse](contract.go) |
| webdriver | capture-export | export buffered capture data | [CaptureExportRequest](contract.go) | [CaptureExportResponse](contract.go) |

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

[@run.yaml](test/run.yaml)
 
```yaml
pipeline:
  init:
    action: webdriver:start
  test:
    action: webdriver:run
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
      - command: stdout = (.stdout).text
        exit: $stdout.Text:/Endly/
        waitTimeMs: 60000
        repeat: 10
      - close
    expect:
      stdout:
        Text: /Hello Endly!/

  defer:
    action: webdriver:stop

```
 

    

### Capture console + network (Chrome/Edge only)

Capture uses ChromeDriver "performance" logs (CDP events) and can optionally fetch response bodies via ChromeDriver CDP endpoints.
If `sinkURL` is provided, events are streamed as JSONL using `viant/afs` (for `file://` it appends by default).

[@capture.yaml](test/capture.yaml)

### Navigation guard for Get(url)

`webdriver:run` can set `navigation` options to avoid hanging on pages that never finish loading. On timeout it warns/continues and can optionally autoscroll for a short duration to load lazy content.
