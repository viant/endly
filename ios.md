# iOS runner service and DSL proposal

> Status: accepted design contract. The `ios` service implements the Simulator/build/deploy/session/test/capture/REPL lifecycle plus strict collections, W3C gestures, typed commands, hybrid contexts, richer element/device assertions, location/orientation/alerts/permissions/appearance/biometrics, archive/export signing inputs, process-shared leases, segmented video, automatic screenshot/source/log/video failure evidence, portable session descriptors, cross-process attach/reconnect, terminal completion, arrow-key editing, persistent history, and explicit managed/prebuilt/preinstalled/external WDA modes. The core paths, reconnect, intentional DSL and device-side XCTest failures with `.xcresult` normalization, checkpointed video, two concurrent isolated Simulator/Appium/WDA lanes, managed/external/prebuilt WDA, and leak-free cleanup are verified on real iOS 27 Simulators through managed Appium 3.7.0/XCUITest 12.11.1; preinstalled WDA is compatibility-rejected on iOS 27 because the current stack cannot keep its direct-launched XCTest runner active. Archive/export signing currently has deterministic protocol tests. Non-Darwin Endly builds register an action-compatible iOS stub. Physical-device deployment and remote macOS workers/cloud farms remain tracked below.

## Decision

Add a new Endly service with ID `ios` under `service/testing/runner/ios`.

It is independent from both `webdriver` and `android`: it owns its routes, public contracts, session registry, destination leases, Appium processes, WebDriverAgent lifecycle, iOS runner, artifacts, and tests. It is an in-process Endly service registered in the normal Endly binary, not a separately deployed daemon. A distributed runner protocol would require a separate proposal.

The service covers:

- Xcode build, archive/export, and build-for-testing automation;
- Simulator and physical-device allocation;
- signing-aware application and WebDriverAgent deployment;
- native, hybrid, and mobile-Safari E2E automation;
- app-owned XCTest/XCUITest execution and `.xcresult` processing;
- screenshots, hierarchy, unified logs, video, crash, WDA, and result evidence;
- local macOS, remote macOS worker, and external Appium/WDA execution;
- safe parallel CI execution.

## Backend and repository fit

Use Appium 3's official XCUITest driver for DSL UI automation. It translates W3C/Appium commands to WebDriverAgent (WDA) and manages WDA where configured. See the official [XCUITest overview](https://appium.github.io/appium-xcuitest-driver/latest/overview/), [capabilities](https://appium.github.io/appium-xcuitest-driver/latest/reference/capabilities/), [locator strategies](https://appium.github.io/appium-xcuitest-driver/latest/reference/locator-strategies/), and [execute methods](https://appium.github.io/appium-xcuitest-driver/latest/reference/execute-methods/).

Use Apple's tools directly:

- `xcodebuild` for resolve/build/build-for-testing/archive/export;
- `xcodebuild test`, `test-without-building`, or an `.xctestrun` file only through `ios:test`;
- `xcrun simctl` for Simulator lifecycle and supported Simulator features;
- `xcrun devicectl` for supported physical-device lifecycle;
- `xcresulttool` for structured result extraction.

Apple lists `xcodebuild`, `simctl`, and `devicectl` in the [Xcode command-line tool reference](https://developer.apple.com/documentation/xcode/xcode-command-line-tool-reference) and describes command-line tests in [Running tests and interpreting results](https://developer.apple.com/documentation/xcode/running-tests-and-interpreting-results).

Retain Endly's typed route, runtime state expansion, auto-wait, AFS artifact, validator, `Assertion()`, xUnit, tags, and workflow `defer` patterns. Do not reuse webdriver's DOM inference, browser navigation, CDP, or arbitrary reflected Selenium calls. Do not extend Endly's deprecated generic build service.

```text
service/testing/runner/internal/mobile/ # protocol-neutral, non-public helpers only
  ast.go parser.go protocol.go retry.go assertion.go redaction.go artifact.go
service/testing/runner/ios/
  init.go contract.go service.go session.go resource_manager.go
  server.go simulator.go device.go build.go deploy.go signing.go
  wda.go xctest.go xcresult.go capture.go selector.go event.go
  test/
```

`ios` must not import Android contracts or share Android sessions. `service/bootstrap/bootstrap.go` must blank-import the package.

```go
const ServiceID = "ios"

func init() {
    endly.Registry.Register(func() endly.Service { return New() })
}
```

## Execution topology

All Apple build, Simulator, device, and managed-WDA actions require a macOS target with full Xcode installed.

| Mode | Xcode/device commands | Appium/WDA | Client connectivity |
| --- | --- | --- | --- |
| `local` | local macOS Endly host | managed or external | direct loopback/reachable URL |
| `remoteWorker` | Endly macOS `Target` over SSH | managed on worker | Endly-owned authenticated SSH forward to worker loopback |
| `external` | selected worker or provider | externally managed | explicitly reachable protected URL |

Rules:

- `HostPath` always names a path on the resolved macOS target; `ArtifactURL` is an AFS URL.
- Inputs are staged to an owned target workspace. Outputs are checksummed and uploaded before cleanup.
- Managed Appium binds worker loopback. The returned endpoint is coordinator-reachable through an owned tunnel when required.
- An external Appium/WDA endpoint is health-checked but never stopped or reconfigured by Endly.
- Target identity is retained in every destination, server, WDA, session, capture, and artifact handle.
- Non-macOS Endly coordinators can orchestrate remote macOS workers but cannot locally build/sign/manage Apple destinations.

## V1 route surface

| Action | Required input | Output |
| --- | --- | --- |
| `ios:doctor` | macOS target and requested features | checks and compatibility matrix |
| `ios:provision` | target, pinned Appium manifest, explicit mutation policy | installed non-Xcode tool manifest |
| `ios:server-start` | fenced destination lease and managed/external options | `ServerHandle` |
| `ios:server-stop` | owned server handle | stop/cleanup report |
| `ios:simulator-start` | target and exact/base Simulator profile | fenced `DestinationLease` |
| `ios:simulator-stop` | fenced owned Simulator lease | release/cleanup report |
| `ios:device-lease` | target and exact physical-device UDID | fenced `DestinationLease` |
| `ios:device-release` | fenced physical-device lease | release/cleanup report |
| `ios:build` | target, project/workspace, scheme, destination/build mode | `[]Artifact` and reports |
| `ios:install` | fenced destination, selected app artifact, bundle/state | install metadata |
| `ios:uninstall` | fenced destination and exact bundle ID | removal status |
| `ios:test` | fenced destination and build/test products | normalized XCTest result |
| `ios:capture-start` | fenced destination and options | `CaptureHandle` |
| `ios:capture-stop` | capture handle | artifact manifest |
| `ios:open` | fenced destination, server, WDA mode, app identity | `SessionHandle` |
| `ios:attach` | descriptor or endpoint/backend session ID | reconnected `SessionHandle` |
| `ios:close` | session handle | close/cleanup report |
| `ios:cleanup` | acquired session/capture/destination/server handles | aggregate LIFO cleanup report |
| `ios:run` | session handle, commands, expectations | data, validations, steps, failures |
| `ios:repl` | optional session ID, terminal and inspector options | interactive history/data/artifacts summary |
| `ios:artifact` | session or fenced destination, artifact kinds | artifact manifest |

Build, test, destination, Appium, WDA, capture, and automation session are distinct lifecycles.

## Frozen V1 data model

The Go field names below define YAML field names through Endly conversion, and all examples use the same nesting.

```go
type TargetRef struct {
    Resource *location.Resource
    Topology string // local, remoteWorker, external
}

type LeaseRef struct {
    ID     string
    Fence  uint64
    UDID   string
    Target TargetRef
}

type PortSet struct {
    Appium int
    WDA    int
    MJPEG  int
}

type DestinationLease struct {
    LeaseRef
    Name            string
    Runtime         string
    PlatformVersion string
    RealDevice      bool
    OwnedClone      bool
    Ports           PortSet
    Workspace       string // HostPath
    ArtifactRoot    string // AFS URL reserved for the lane
    ExpiresAt       time.Time
}

type ServerHandle struct {
    ID        string
    Fence     uint64
    Target    TargetRef
    Endpoint  string
    Ownership string // managed or external
    PID       int
    BasePath  string
    Log       Artifact
}

type SessionHandle struct {
    ID               string
    BackendSessionID string
    Destination      LeaseRef
    Server           ServerHandle
    WDAMode          string
}

type CaptureHandle struct {
    ID          string
    Fence       uint64
    Destination LeaseRef
    Started     time.Time
}

type Artifact struct {
    ID              string
    Kind            string // simulatorApp, deviceApp, ipa, xcarchive, xctestrun, xcresult...
    Scheme          string
    Configuration   string
    SDK             string
    Architecture    string
    BundleID        string
    Version         string
    Build           string
    SigningIdentity string
    TeamID          string
    SHA256          string
    HostPath        string
    ArtifactURL     string
    Sensitive       bool
}

type SigningProfile struct {
    Mode                   string // simulator, automatic, manual
    TeamID                 string
    SigningIdentity        string
    ProvisioningProfiles   map[string]string
    KeychainPath           string // HostPath
    KeychainPasswordSecret string
    ExportOptions          *location.Resource
}

type BuildResponse struct {
    Artifacts  []Artifact
    Selected   map[string]Artifact // populated only for a unique compatible kind
    Reports    []Artifact
    DurationMs int
}

type ServerStartResponse struct { Server ServerHandle }
type DestinationStartResponse struct { Lease DestinationLease }
type CaptureStartResponse struct { Capture CaptureHandle }
type OpenResponse struct { Session SessionHandle }
```

Every mutating destination action requires lease ID, fence, UDID, and matching target. The service validates the fence immediately before install, uninstall, privacy/location/appearance mutation, launch, WDA/session creation, capture, shutdown, deletion, or release. Expiry alone cannot authorize a stale owner.

Important route contracts:

```go
type ServerStartRequest struct {
    Destination      LeaseRef // target and reserved ports come from the atomic destination lease
    Mode             string // managed or external
    ServerURL        string // external only
    AppiumExecutable string
    AppiumHome       string
    Address          string // managed default/required loopback
    BasePath         string
    Driver           string // fixed to xcuitest in V1
    LogURL           string
    AllowInsecure    []string
    StartupTimeoutMs int
}

type BuildRequest struct {
    Target              TargetRef
    ProjectPath         string // HostPath; mutually exclusive with WorkspacePath
    WorkspacePath       string // HostPath
    Scheme              string
    Configuration       string
    SDK                 string
    Destination         *LeaseRef // optional for generic build/archive
    DerivedDataPath     string // HostPath
    Mode                string // resolve, build, buildForTesting, archive, export
    ArchivePath         string // HostPath
    ExportPath          string // HostPath
    Signing             *SigningProfile
    BuildSettings       map[string]string
    TimeoutMs           int
    ArtifactDir         string // AFS URL
}

type TestRequest struct {
    Target          TargetRef
    Destination     LeaseRef
    Mode            string // scheme, withoutBuilding, xctestrun, logic
    ProjectPath     string
    WorkspacePath   string
    Scheme          string
    TestPlan        string
    Artifacts       []Artifact // xctestrun/build products as needed
    OnlyTesting     []string
    SkipTesting     []string
    RetryCount      int
    ResultBundleURL string
    TimeoutMs       int
}

type InstallRequest struct {
    Destination LeaseRef
    Artifact    Artifact // exactly one simulatorApp, deviceApp, or supported ipa
    BundleID    string
    State       string // freshInstall or preserve; eraseSimulator belongs to simulator-start
    TimeoutMs   int
}

type OpenRequest struct {
    SessionID         string
    Destination       LeaseRef
    Server            ServerHandle
    App               *Artifact
    BundleID          string
    Reset             string // none, appData where supported, reinstall
    AutoLaunch        *bool
    WDAMode           string // managed, prebuilt, preinstalled, external
    WDAArtifact       *Artifact
    WebDriverAgentURL string
    WDASigning        *SigningProfile
    DerivedDataPath   string // HostPath
    Capabilities      map[string]interface{}
}

type CaptureStartRequest struct {
    Destination LeaseRef
    BundleID    string
    UnifiedLog  bool
    Video       bool
    ArtifactDir string
    MaxBytes    int64
}

type CaptureStopRequest struct {
    Capture CaptureHandle
}

type CleanupRequest struct {
    Session     *SessionHandle
    Capture     *CaptureHandle
    Destination *DestinationLease
    Server      *ServerHandle
}

type RunRequest struct {
    Session          SessionHandle
    Commands         []interface{}
    ActionTimeoutMs  int
    PollIntervalMs   int
    Expect           interface{}
    FailureArtifacts *FailureArtifactOptions
}

type DoctorRequest struct {
    Target            TargetRef
    RequestedFeatures []string
}

type ProvisionRequest struct {
    Target         TargetRef
    Manifest       *location.Resource
    AllowDownloads bool
    ArtifactDir    string
}

type ServerStopRequest struct { Server ServerHandle }

type SimulatorStartRequest struct {
    Target        TargetRef
    BaseName      string
    CloneName     string
    DeviceType    string
    Runtime       string
    Erase         bool
    Locale        string
    Appearance    string
    BootTimeoutMs int
    LeaseTTLms    int
    ArtifactRoot  string
}

type SimulatorStopRequest struct { Lease DestinationLease }

type DeviceLeaseRequest struct {
    Target       TargetRef
    UDID         string
    LeaseTTLms   int
    ArtifactRoot string
}

type DeviceReleaseRequest struct { Lease DestinationLease }

type UninstallRequest struct {
    Destination LeaseRef
    BundleID    string
}

type CloseRequest struct {
    Session      SessionHandle
    TerminateApp bool
}

type ArtifactRequest struct {
    Session     *SessionHandle
    Destination *LeaseRef
    Kinds       []string
    Directory   string
    MaxBytes    int64
}

type FailureArtifactOptions struct {
    Directory   string
    Screenshot  bool
    PageSource  bool
    UnifiedLog  bool
    Video       bool
    CrashLogs   bool
    Diagnostics bool
    MaxBytes    int64
}

type TestCase struct {
    ID, Suite, Status, Failure, Source, Stdout string
    DurationMs int
    Attempt    int
    Flaky      bool
    Artifacts  []Artifact
}

type TestResponse struct {
    Cases       []TestCase
    Validations []*assertly.Validation
    Artifacts   []Artifact
}

type RunResponse struct {
    Data        map[string]interface{}
    Steps       []StepResult
    Validations []*assertly.Validation
    Failures    []Failure
    Warnings    []string
}

func (r *RunResponse) Assertion() []*assertly.Validation { return r.Validations }

type StepResult struct {
    Index, Attempts, DurationMs int
    Command, Status, Error      string
    Expected, Actual            interface{}
}

type Failure struct {
    Kind, Message string
    Step          int
    Artifacts     []Artifact
}

type Check struct { Name, Status, Version, Detail, Remediation string }
type DoctorResponse struct { Checks []Check; Features map[string]bool }
type ProvisionResponse struct { Manifest Artifact; Checks []Check }
type MutationResponse struct { Changed bool; Artifacts []Artifact; Cleanup *CleanupReport }
type ArtifactResponse struct { Artifacts []Artifact }
type CleanupReport struct { Operations []StepResult; Artifacts []Artifact; Errors []string }
```

Every contract gets `Init()` and `Validate()` tests. Validation rejects unknown modes, ambiguous products, destination/product incompatibility, stale fences, unsafe paths, capability aliases/duplicates, unsupported features, signing mismatches, and port collisions before mutation.

Raw capabilities are canonicalized to W3C/Appium keys. Duplicate normalized keys and unsupported namespaces fail. Endly-owned platform, destination, port, WDA lifecycle, provisioning, and security capabilities cannot be overridden.

## Appium server and WDA ownership

Managed Appium:

- uses pinned executable and isolated `APPIUM_HOME`;
- verifies the XCUITest driver version;
- binds `127.0.0.1`, never enables `--relaxed-security`, and allowlists any insecure feature;
- allocates ports with the destination lease tuple;
- launches a process group, probes `/status`, captures bounded logs, and records PID/start token;
- stops only when PID/start token and fencing still match.

External mode is never stopped by Endly. Appium must remain on loopback or a protected trusted network; see [Appium server security](https://appium.io/docs/en/3.3/guides/security/).

WDA has four explicit modes:

| Mode | Ownership and requirements |
| --- | --- |
| `managed` | XCUITest driver builds/starts and reuses WDA for the compatible leased destination |
| `prebuilt` | supplied WDA product is compatibility/signing validated before driver launch |
| `preinstalled` | existing/supplied WDA is validated against OS/driver constraints |
| `external` | attach to `WebDriverAgentURL`; Endly does not manage WDA |

The driver documents current signing, DerivedData, reuse, and prebuilt/preinstalled capabilities in its [capability reference](https://appium.github.io/appium-xcuitest-driver/latest/reference/capabilities/). Current preinstalled constraints are documented in [Run Preinstalled WDA](https://appium.github.io/appium-xcuitest-driver/latest/guides/run-preinstalled-wda/). `ios:doctor` resolves support; the runner never guesses.

Cache keys include Xcode build, driver/WDA version, platform, architecture, destination kind/OS, deployment target, and signing team. Parallel lanes use unique WDA local/MJPEG ports and compatible isolated DerivedData paths.

## Resource management and cleanup

Endly's current `Context.Deffer` invokes callbacks FIFO and discards errors. The iOS service registers one callback for one platform `ResourceManager`. A destination lease atomically reserves the destination, ports, workspace, result paths, and artifact root before Appium starts. The manager maintains an idempotent LIFO stack:

```text
session -> capture -> WDA/managed Appium/tunnel -> destination -> workspace -> temporary keychain
```

`CloseAll` continues after failures, aggregates a `CleanupReport`, uploads it when possible, stores it in iOS service state, and publishes an event. The `ios:cleanup` route accepts all acquired handles and runs the same LIFO logic as one reportable deferred action. Multiple deferred child actions are unsafe because an early failure can suppress later cleanup. Automatic context cleanup is a safety net.

No cleanup may stop external Appium/WDA, delete an unowned Simulator, erase a physical device, or operate after a fence mismatch. Temporary keychains have restrictive permissions and bounded lifetime.

## DSL grammar and typed commands

Android and iOS may share a private parser, but compile into distinct platform AST adapters. No arbitrary method reflection is allowed.

```ebnf
command     = [ identifier, "=" ], invocation | expectation ;
invocation  = namespace, ".", call, { ".", call } ;
expectation = "expect", "(", invocation, ")", ".", call ;
namespace   = "app" | "device" ;
call        = identifier, "(", [ argument, { ",", argument } ], ")" ;
argument    = string | number | boolean | null ;
identifier  = letter, { letter | digit | "_" } ;
```

Strings use single/double quotes and backslash escaping. Variables expand after parsing immediately before execution. Endly `/pattern/` regex values are accepted only by matchers. Zero matches wait, one acts, and multiple fail unless `.first()`, `.nth(index)`, or a count matcher makes selection explicit. `fill` clears/replaces; `type` appends. Attribute JSON types are retained. Cancellation propagates through polling and HTTP.

Typed form is also supported:

```yaml
commands:
  - locator: {strategy: accessibilityId, value: email}
    action: fill
    args: [qa@example.test]
  - expect:
      locator: {strategy: accessibilityId, value: order-confirmed}
      matcher: visible
      timeoutMs: 15000
```

## Locator and feature policy

`getByTestId` uses Appium accessibility ID, but the application policy must assign unique accessibility identifiers. XCUITest driver `id`, `name`, and accessibility-ID strategies resolve through element name semantics and may fall back to labels, so Endly must detect multi-match collisions rather than claim identifier-only behavior.

Explicit locators are `getByAccessibilityId`, `getByName`, `getByLabel`, `getByType`, `getByPredicate`, `getByClassChain`, `getByText`, `getByXPath`, and `locator(strategy,value)`. Predicate/class chain are preferred advanced queries; text is localization-sensitive; XPath is last resort.

Portable actions include tap, double tap, long press, fill/type/clear, swipe/scroll/drag, state reads, and waits. iOS extensions include app lifecycle, alerts, orientation, location, contexts, appearance, permissions, deep link, pasteboard, and biometrics.

`ios:doctor` returns a per-destination feature matrix used to preflight every `ios:run`. It accounts for Xcode, iOS, XCUITest-driver, WDA, destination kind, and optional dependencies. Examples include:

- deep-link execution requires the compatible Xcode/iOS/driver combination documented by the driver;
- current preinstalled WDA has minimum OS/driver requirements;
- permission operations differ between `simctl`, optional `applesimutils`, WDA, and system UI;
- pasteboard and biometric simulation are Simulator-only;
- physical-device transports differ from Simulator transports.

Unsupported commands fail preflight before any journey step. The current compatibility source is the official [execute-method reference](https://appium.github.io/appium-xcuitest-driver/latest/reference/execute-methods/) and capability/WDA documentation, not hard-coded assumptions in examples.

## Assertions and error model

`RunResponse` contains `Data`, `Steps`, `Validations`, `Failures`, and `Warnings`. `Assertion()` returns inline and final-map validations. XCTest cases are separately normalized and also exposed as tagged validations. Expected/final actual, locator, timing by Endly wait versus XCTest quiescence, attempts, and last backend error are retained.

Assertion/XCTest failures and flaky passes are test results. Host, destination, signing, Appium, WDA, transport, installation, app crash, protocol, and artifact failures are classified infrastructure/product errors as appropriate. A retrying pass stays marked flaky with all attempts.

## Build, archive, export, and products

`ios:build` owns only `resolve`, `build`, `buildForTesting`, `archive`, and `export`. All `test`, `test-without-building`, and `.xctestrun` execution belongs to `ios:test`.

The service sets `DEVELOPER_DIR` per process rather than changing global selection, validates the scheme/destination, uses argv-safe build settings, and records the resolved invocation. `archive` and `export` are distinct internal invocations even when requested sequentially by the workflow.

Return `[]Artifact`, never a singular app/IPA. Each product records kind, scheme/configuration/SDK/architecture, bundle/version/build, signing/team summary, checksum, HostPath, and ArtifactURL. Consumers require exactly one compatible kind or fail ambiguity.

Signing modes are Simulator-disabled, automatic, or manual. Secret values never enter retained argv/logs. Archive/export results include `.xcarchive`, exported products, `.xctestrun`, logs, and metadata. `.xcresult` is produced by `ios:test`, not `ios:build` unless Xcode emits a build result bundle explicitly requested as a build report.

## Provisioning and application deployment

Xcode and Simulator runtimes must be preinstalled and licensed in the macOS worker image. Endly never redistributes or silently installs Xcode.

`ios:doctor` is read-only: it reports macOS/Xcode/SDK/runtimes/destinations, `simctl`/`devicectl`, disk, signing identity/profile metadata without private material, Node/Appium/driver/WDA compatibility, ports, and feature matrix.

`ios:provision` is explicitly mutating but limited to a pinned isolated Node/Appium/XCUITest workspace and optional helper tools. It cannot select global Xcode, install runtimes, accept agreements, import certificates, unlock keychains, enroll/register devices, or enable automatic provisioning unless a separate explicit signing/device operation and secret policy authorizes it.

Deployment rules:

- Simulator accepts exactly one compatible `simulatorApp`.
- Physical-device mode accepts a development/ad-hoc `deviceApp` or supported IPA compatible with the installed Xcode/devicectl transport.
- For IPA, Endly safely extracts and validates exactly one `Payload/*.app` when the selected transport requires an app bundle; it rejects path traversal and ambiguous payloads.
- App Store-encrypted IPAs are not accepted for direct lab installation.
- Bundle ID, architecture, platform, minimum OS, embedded profile, application identifier/team entitlement, signing identity, and requested UDID eligibility are validated before install.
- `freshInstall` removes only the exact requested bundle; `preserve` uses supported update behavior. Simulator erase is only a fenced `simulator-start` option.
- Physical devices are never erased.

## XCTest and `.xcresult`

`ios:test` supports scheme/test plan, test-without-building products, an explicit `.xctestrun`, and host logic tests. Filters map to `-only-testing`/`-skip-testing`. Xcode repetition/retry controls preserve every attempt and flaky status.

Parse `.xcresult` into test identifier, suite, status, duration, attempt, failure/source, stdout, screenshots, attachments, activities, crashes, diagnostics, coverage, and performance metrics when requested. Retain the original result bundle and produce normalized JUnit/xUnit.

## Capture and artifact security

Capture is keyed by `CaptureHandle.ID` and destination fence, never by automation session, so it may begin before `ios:open`.

Evidence can include screenshot, bounded WDA hierarchy, destination/app/context/orientation state, scoped unified log, Appium/WDA command logs, Simulator video, relevant crash/spin reports, and explicit diagnostic bundles. XCTest evidence comes primarily from `.xcresult`.

Screenshots, video, hierarchy, crash data, logs, pasteboard, and result bundles may contain secrets that regex redaction cannot reliably remove. Treat them as sensitive by construction: explicit opt-in, restrictive permissions, bounded retention/size, and authenticated encrypted sinks. Text metadata still redacts marked environment values, credentials, tokens, and configured patterns. Collection errors do not replace the primary failure.

Raw capabilities, build settings, executable paths, WDA URLs, and AFS destinations are privilege-bearing input. Canonicalize and allowlist them, reject unsafe schemes/paths and aliases, and execute tools with argv rather than shell strings.

## Complete Simulator E2E example

Endly stores each response using the exact YAML action name. The example uses camel-case keys and `${uuid.next}`.

```yaml
init:
  runID: ${uuid.next}
  appPath: $WorkingDirectory(./mobile-app/ios)
  artifactRoot: /tmp/endly/ios/$runID
  bundleID: com.example.shop.qa
  sessionID: ios-$runID

pipeline:
  doctor:
    action: ios:doctor
    target: {topology: local}
    requestedFeatures: [simulator, native, hybrid, deepLink, xctest]

  simulatorStart:
    action: ios:simulator-start
    target: {topology: local}
    baseName: endly-ios-base
    cloneName: endly-$runID
    erase: true
    bootTimeoutMs: 180000
    locale: en_US
    appearance: light
    artifactRoot: file://$artifactRoot

  buildTests:
    action: ios:build
    target: {topology: local}
    workspacePath: $appPath/Shop.xcworkspace
    scheme: Shop-QA
    configuration: Debug
    destination: $simulatorStart.Lease
    derivedDataPath: $artifactRoot/derived-data
    mode: buildForTesting
    artifactDir: file://$artifactRoot/build

  projectUITests:
    action: ios:test
    target: {topology: local}
    destination: $simulatorStart.Lease
    mode: withoutBuilding
    artifacts: $buildTests.Artifacts
    onlyTesting: [ShopUITests/CriticalSmokeTests]
    resultBundleURL: file://$artifactRoot/results/xctest.xcresult

  install:
    action: ios:install
    destination: $simulatorStart.Lease
    artifact: $buildTests.Selected.simulatorApp
    bundleID: $bundleID
    state: freshInstall

  serverStart:
    action: ios:server-start
    destination: $simulatorStart.Lease
    mode: managed
    address: 127.0.0.1
    driver: xcuitest
    logURL: file://$artifactRoot/appium.log

  captureStart:
    action: ios:capture-start
    destination: $simulatorStart.Lease
    bundleID: $bundleID
    unifiedLog: true
    video: true
    artifactDir: file://$artifactRoot/ui

  open:
    action: ios:open
    sessionID: $sessionID
    destination: $simulatorStart.Lease
    server: $serverStart.Server
    app: $buildTests.Selected.simulatorApp
    bundleID: $bundleID
    reset: none
    wdaMode: managed
    derivedDataPath: $artifactRoot/wda-derived-data

  checkout:
    tag: checkout
    description: A signed-in user can complete a purchase
    action: ios:run
    session: $open.Session
    actionTimeoutMs: 15000
    commands:
      - device.deepLink("shop://product/sku-42", "$bundleID")
      - app.getByTestId("add-to-cart").tap()
      - app.getByTestId("checkout").tap()
      - app.getByTestId("email").fill("qa@example.test")
      - app.getByTestId("place-order").tap()
      - orderID = app.getByTestId("order-id").text()
      - expect(app.getByText("Order confirmed")).toBeVisible(15000)
    expect:
      orderID: /ORD-[0-9]+/
    failureArtifacts:
      directory: file://$artifactRoot/ui/failures
      screenshot: true
      pageSource: true

  defer:
    cleanup:
      action: ios:cleanup
      session: $open.Session
      capture: $captureStart.Capture
      destination: $simulatorStart.Lease
      server: $serverStart.Server
```

`Selected.simulatorApp` is populated only when `ios:build` proves exactly one compatible simulator application; otherwise the build fails with an ambiguity requiring an explicit selector. The complete `Artifacts` array remains authoritative. Contract fixture tests must decode and execute this exact workflow before publication as runnable documentation.

## Physical-device differences

A physical-device workflow uses `ios:device-lease` with an exact UDID and declares signing/WDA strategy. It validates pairing, Developer Mode, device eligibility, app and WDA profiles/entitlements, transport support, and permissions before mutation. Cleanup terminates the requested app, restores reversible test settings, removes only explicitly installed test bundles if configured, and releases the lease without erase.

## Parallel CI

One atomic fenced lease owns `(Simulator clone or physical UDID, Appium port, WDA port, MJPEG port, DerivedData path, result path, workspace, artifact root)`. Every mutation verifies the fence. Takeover increments it.

Use compatible build-for-testing caching followed by test-without-building across destinations. Keep WDA alive within a compatible lease. Recommended tiers are PR unit/smoke; merge supported Simulator matrix; nightly locale/appearance/text/rotation/interruption/network/upgrade plus physical devices; release signed archive/export/install/entitlement/accessibility verification.

## Implementation stages

1. Add `internal/mobile` AST/protocol/retry/assertion/redaction utilities and iOS bootstrap import.
2. Implement macOS topology, fenced resource manager, doctor/provision, Appium lifecycle, and destination leases.
3. Implement build/product/signing validation, Simulator/device install, XCTest/xcresult, capture, and LIFO cleanup.
4. Implement WDA modes, sessions, locators, typed/string commands, assertions, and failure evidence.
5. Add hybrid/Safari, feature-gated extensions, remote workers, physical-device gates, and parallel stress tests.

## Acceptance criteria

- Every route has concrete contract, `Init`, `Validate`, conversion, and example-decode tests.
- Parser/AST, capability normalization, argv/path safety, redaction, signing, feature matrix, fencing, cleanup, and artifact selection have unit tests.
- Fake W3C/Appium tests verify exact protocol and extension payloads.
- Simulator integration builds a fixture, leases a clean clone, runs build-for-testing/test-without-building and native/hybrid DSL, intentionally fails, and verifies `.xcresult`, artifacts, and xUnit.
- Managed, remote-worker+tunnel, external Appium, and all WDA modes have lifecycle/compatibility tests.
- Gated physical-device tests cover signing/install/WDA reuse/app lifecycle/logs/cleanup without erase.
- Context cleanup is LIFO, idempotent, aggregate-reporting, and never touches unowned resources.
- Two parallel lanes have no destination, port, DerivedData, session, result, workspace, or artifact collisions.
- No secret appears in retained textual logs/metadata; visual/diagnostic artifacts are labeled sensitive with enforced storage policy.

## V1 non-goals

- replacing app-owned XCTest/XCUITest;
- installing, licensing, or globally selecting Xcode;
- bypassing signing, trust, Developer Mode, biometrics, device integrity, or security controls;
- image-only automation or visual-diff baselines;
- erasing physical devices;
- App Store Connect/TestFlight publication;
- a combined public mobile service or separately deployed runner daemon.
