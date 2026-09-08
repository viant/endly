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

`webdriver:run` can set `navigation` options to avoid hanging on pages that never finish loading. Navigation and lazy-content scrolling have independent hard limits. Every navigation returns a report with the elapsed time, final document height, scroll count, timeout state, and a stop reason such as `stable-bottom`, `growth-limit`, `max-steps`, or `time-budget`.

```yaml
navigation:
  timeoutMs: 15000
  continueOnTimeout: true
  stopLoadingOnTimeout: true
  autoScrollMs: 5000
  scrollSelector: "#results-scroll-container"
  scrollDelayMs: 200
  stableWindowMs: 800
  maxScrollSteps: 15
  maxScrollGrowthPx: 20000
  maxScrollHeightPx: 60000
  idleThreshold: 0
  idleWindowMs: 500
  returnToTop: true
```

`stopLoadingOnTimeout` defaults to true and calls `window.stop()` before continuing, preventing an unresolved resource or streaming page from occupying the browser indefinitely. `autoScrollMs: 0` disables scrolling. When enabled, scrolling runs after both successful and timed-out navigation, so normally-loaded lazy pages are handled too. Omit `scrollSelector` to scroll the document. A configured selector targets an inner CSS scroll container. Stability includes a last-content signature so virtualized lists are not considered stable merely because their document height is constant. `autoScrollMs`, `maxScrollSteps`, and `maxScrollGrowthPx` ensure an infinite feed cannot scroll forever.

### Playwright-inspired commands

The original Endly commands remain supported. The following aliases offer a smaller browser-oriented DSL:

```yaml
commands:
  - page.goto("https://example.test/login")
  - page.locator("#email").fill("qa@example.test")
  - page.locator("#password").fill("$testPassword")
  - page.getByTestId("submit").click()
  - createResponse = page.waitForResponse("/api/orders", 201, 10000)
  - page.locator("#status").waitForVisible(10000)
  - status = page.locator("#status").text()
  - expect(page.locator("#status")).toContainText("Signed in", 10000)
  - expect("#email").toHaveValue("qa@example.test")
  - expect(page).toHaveURL("/dashboard/", 10000)
  - expect(page).toHaveTitle("Example")
  - title = page.title()
expect:
  status: /Signed in/
  title: Example
```

For the default local `localhost:4444` session, `webdriver:run` now starts ChromeDriver automatically when needed and registers idempotent context cleanup. Set `autoStart: false` to require explicit lifecycle tasks, or keep `webdriver:start`/`webdriver:stop` when driver deployment settings must be customized.

Every locator action auto-waits for its element and retries transient stale, intercepted, hidden, and not-interactable states. Defaults can be adjusted per run:

```yaml
actionTimeoutMs: 10000
pollIntervalMs: 100
allowJavaScriptClick: false
# strictSelectors: false # optional global compatibility override
```

`allowJavaScriptClick` is deliberately opt-in. When enabled, Endly uses a DOM click only after a native click fails because of an overlay or interactability problem; stale elements are always re-resolved instead.

Playwright-style locators are strict by default: an action fails when its selector matches multiple elements. Legacy Endly commands retain their existing first-match behavior. Set `strictSelectors` explicitly only when a run needs to override either default.

For faster Chrome tests, use `pageLoadStrategy: eager` and optionally block nonessential resources with CDP URL patterns:

```yaml
pageLoadStrategy: eager
blockedURLs:
  - "*://*/analytics/*"
  - "*.woff2"
  - "*.mp4"
```

Blocking is explicit and disabled by default because images, fonts, analytics, or media may be part of the behavior under test. `blockedURLs` currently requires Chrome.

Supported page commands include `goto`, `reload`, `back`, `forward`, `stopLoading`, `title`, `url`, `tabs`, `switchTab`, `newTab`, `closeTab`, `frame`, `mainFrame`, `acceptDialog`, `dismissDialog`, `dialogText`, and `setDialogText`.

`page.waitForResponse(URLPattern, status, timeoutMs)` starts bounded Chrome network capture before the run's first action and waits on observed browser traffic rather than an arbitrary sleep. The URL pattern may be a substring or `/regular expression/`; status `0` accepts any status. The captured transaction—including timing, headers, status, and any configured captured body—is stored under the assigned result key.

Locators include `locator`, `getByTestId`, `getByText`, `getByLabel`, `getByPlaceholder`, `getByAltText`, `getByTitle`, and `getByRole`. Locator actions include `click`, `fill`, `type`, `press`, `text`, `inputValue`, `attribute`, `clear`, `submit`, `check`, `uncheck`, `selectOption`, `hover`, `setInputFiles`, and `waitForVisible`.

Like Playwright, `page.locator(...)` treats its value as CSS unless it starts with `xpath:` or `//`. This avoids misclassifying compound CSS selectors containing spaces or attribute expressions.

Retrying locator expectations include `toHaveText`, `toContainText`, `toHaveValue`, `toHaveAttribute`, `toHaveCount`, `toBeVisible`, `toBeHidden`, `toBeEnabled`, `toBeDisabled`, `toBeChecked`, and `toBeUnchecked`. Page expectations include `toHaveURL` and `toHaveTitle`. Their final argument may be a timeout in milliseconds. Text comparisons normalize browser whitespace, and `/regular expressions/` are accepted by equality checks. `toBeHidden` succeeds when the element is either absent or not displayed.

Inline `expect(...)` commands fail the action with expected/actual diagnostics and append a structured entry to `RunResponse.assertions` containing selector, method, matcher, expected value, actual value, outcome, error, and elapsed time. Both inline assertions and final `expect:` map validation are exposed through `RunResponse.Assertion`, so failures are visible to CLI and xUnit reporting instead of being nested silently in the response.

### Failure evidence

Failure artifacts are opt-in because screenshots and page HTML may contain sensitive test data. When configured, action errors and final validation failures capture a screenshot, bounded page source, current URL/title, navigation reports, and optional redacted console/network history:

```yaml
failureArtifacts:
  directory: file:///tmp/endly/browser-failures
  screenshot: true
  pageSource: true
  maxSourceBytes: 2000000
  includeCapture: true
```

The resulting `RunResponse.failures` entries contain the generated PNG, HTML, and JSON metadata URLs plus any capture errors. `includeCapture` uses the existing capture buffer, so start capture before the run when console or network evidence is required.

### Attach to an existing Chrome

Endly can control tabs in a Chrome instance that was started with an authorized remote-debugging endpoint. A normally launched Chrome cannot be adopted after startup; start a dedicated browser profile with a remote-debugging port, and keep that port bound to a trusted interface.

```yaml
pipeline:
  start-driver:
    action: webdriver:start
    pageLoadStrategy: eager
  attach-browser:
    action: webdriver:open
    browser: chrome
    debuggerAddress: 127.0.0.1:9222
    directCDP: true
  inspect-tabs:
    action: webdriver:run
    commands:
      - tabs = page.tabs()
      - page.switchTab("checkout")
      - currentURL = page.url()
    expect:
      currentURL: /checkout/
```

With `directCDP: true`, Endly connects directly to the Chrome debugging endpoint: ChromeDriver and Selenium Server are not required. Direct CDP supports navigation, tabs, locators, form actions, frames, dialogs, cookies, screenshots, console/network capture, assertions, and browser HTTP waits. Omit it to use ChromeDriver attachment for maximum legacy WebDriver-method compatibility.

`page.switchTab` accepts an exact window handle or a case-insensitive URL/title fragment. Attaching preserves the existing browser profile and tabs, but it is still browser automation and does not attempt to conceal or spoof that fact. For applications you own, use a test environment or allowlisted test identity rather than anti-detection workarounds.

### Live control and activity recording

Start the interactive planner with `endly -w=8082`. It binds to `127.0.0.1` by default and provides:

- cancellable one-step live execution of any `page`, `locator`, or `expect` command; cancellation also sends Chrome `Page.stopLoading` outside the WebDriver command queue so a blocked navigation can be interrupted;
- current tabs, URL, title, capture status, and recorded activity state;
- attachable in-page recording for an existing controlled tab;
- live conversion of clicks, debounced input/change events, checkbox/radio changes, selects, submit, Enter/Escape presses, and navigation into the new DSL;
- automatic plan append with pause/resume recording controls.

Password values and password `value` attributes are never sent back; recorded password fills use the `$PASSWORD` placeholder. Target and holder HTML are bounded, only the latest 1,000 recorded activities are retained, and the recorder is reinjected after navigation-triggering user actions. Activity is delivered over an authenticated local endpoint with a browser-console fallback, then deduplicated, so recording can continue when a page's content-security policy blocks cross-origin requests.

The planner generates a random recorder token, requires it on cross-origin activity events, restricts its WebSocket to same-host/loopback origins, and no longer exposes live browser control on every network interface.

### Live-browser verification

The default suite remains hermetic. To run the opt-in Chrome integration flow against an already-running driver:

```bash
ENDLY_WEBDRIVER_REMOTE=http://127.0.0.1:4444/wd/hub \
go test ./service/testing/runner/webdriver -run TestLiveBrowserFlow -v
```

Alternatively, set only `ENDLY_CHROME_DEBUGGER_ADDRESS=127.0.0.1:9222` to run the same integration flow using direct CDP without ChromeDriver. This test navigates the selected browser tab and should only be run against a disposable test profile.
