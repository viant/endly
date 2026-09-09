# Android runner service and DSL proposal

> Status: accepted design contract. The `android` service implements the emulator/connected-device, build/deploy/session/test/capture/REPL lifecycle plus strict collections, W3C gestures, typed commands, hybrid contexts, richer element/device assertions, permissions/location/orientation/alerts, split APK/APKS/AAB deployment and signing inputs, process-shared leases, segmented video, automatic screenshot/source/log/video failure evidence, portable session descriptors, cross-process attach/reconnect, terminal completion, arrow-key editing, persistent history, provider-neutral external Appium/cloud-destination registration, and persistent authenticated Endly worker execution. The core paths, reconnect, intentional DSL and device-side instrumentation failures, checkpointed video, two concurrent isolated emulator/Appium lanes, authenticated multi-operation worker state, and leak-free cleanup are verified; split-package signing and cloud capability handoff currently have deterministic protocol tests. Endly cross-builds for Darwin, Linux, and Windows. Provider-specific farm upload APIs remain an adapter concern; inputs can be staged through AFS or supplied as provider app references.
>
> The filename preserves the requested `adntoid.md` spelling. The service ID and Go package are `android`.

## Decision

Add a new Endly service with ID `android` under `service/testing/runner/android`.

It is independent from both `webdriver` and `ios`: it owns its routes, public contracts, session registry, device leases, Appium processes, Android runner, artifacts, and tests. It is an in-process Endly service registered in the normal Endly binary, not a separately deployed daemon. A distributed runner protocol would be a separate proposal.

The service covers:

- Gradle build automation and artifact discovery;
- emulator and physical-device allocation;
- APK/APKS application deployment and reset policies;
- native, hybrid, and mobile-web E2E automation;
- app-owned Android instrumentation tests;
- screenshots, UI hierarchy, logcat, video, crash, and build/test evidence;
- local, remote-worker, and remote-Appium execution;
- safe parallel CI execution.

## Repository fit

The existing `webdriver` service provides patterns worth retaining:

- typed routes and `Init`/`Validate` request lifecycle;
- context state expansion immediately before execution;
- concise assigned commands and retrying expectations;
- bounded auto-waits which re-resolve stale elements;
- AFS-backed, opt-in failure evidence;
- `validator.Assert` and `Assertion()` integration with CLI/xUnit;
- Endly workflow tags, conditions, ranges, async actions, and explicit `defer` tasks.

Do not reuse browser-only implementation: DOM selector inference, JavaScript click, navigation stabilization, tabs, frames, CDP capture, and arbitrary reflected Selenium method calls do not belong in the Android runner.

The existing generic `build` service is deprecated. `android:build` must invoke the application's Gradle wrapper and return Android-aware products.

## Backend and integration

Use Appium 3 with the official UiAutomator2 driver for UI automation. UiAutomator2 supports native, hybrid, and mobile-web contexts through W3C WebDriver plus Appium extensions. Appium installs drivers separately, so the server and driver versions must both be pinned and verified. See the official [Appium driver list](https://github.com/appium/appium/blob/master/packages/appium/docs/en/ecosystem/drivers.md), [UiAutomator2 documentation](https://github.com/appium/appium-uiautomator2-driver), and [capability examples](https://github.com/appium/appium-uiautomator2-driver/blob/master/docs/capability-sets.md).

Use a narrow internal W3C/Appium HTTP client rather than exposing the older `tebeka/selenium` API. Use `adb`, emulator tools, the Gradle wrapper, and bundletool directly for lifecycle, build, deployment, instrumentation, and diagnostics.

```text
service/testing/runner/internal/mobile/ # protocol-neutral, non-public helpers only
  ast.go parser.go protocol.go retry.go assertion.go redaction.go artifact.go
service/testing/runner/android/
  init.go contract.go service.go session.go resource_manager.go
  server.go device.go build.go deploy.go instrumentation.go
  capture.go selector.go event.go
  test/
```

`android` must not import iOS contracts or share an iOS session map. `service/bootstrap/bootstrap.go` must blank-import the Android package so normal Endly binaries register it.

```go
const ServiceID = "android"

func init() {
    endly.Registry.Register(func() endly.Service { return New() })
}
```

## Execution topology

Every action resolves one execution target before doing work:

| Mode | Platform commands | Appium | Client connectivity |
| --- | --- | --- | --- |
| `local` | local Endly host | managed locally or external | direct loopback/reachable URL |
| `remoteWorker` | Endly `Target` over SSH | managed on that worker | Endly-owned authenticated SSH forward from coordinator to worker loopback |
| `external` | local or selected worker | externally managed | explicitly reachable protected `serverURL` |

Rules:

- `HostPath` means a path on the resolved target; `ArtifactURL` means an AFS location. They are never interchangeable.
- Inputs are staged from AFS into an owned target workspace before a tool consumes them.
- Outputs are checksummed on the target and uploaded to AFS before workspace cleanup.
- A managed Appium server binds to `127.0.0.1`; remote-worker mode owns the SSH forward and returns a coordinator-reachable endpoint.
- External endpoints must use an allowlisted scheme/host and their ownership is always `external`, so Endly never stops them.
- Target identity is retained in every lease, server, session, and capture handle so cleanup runs on the correct host.

## V1 route surface

| Action | Required input | Output |
| --- | --- | --- |
| `android:doctor` | target, requested features | checks and feature matrix |
| `android:provision` | target, pinned manifest, explicit mutation policy | installed tool manifest |
| `android:server-start` | fenced device lease and managed/external server options | `ServerHandle` |
| `android:server-stop` | owned server handle | stop/cleanup report |
| `android:device-start` | target and device profile/serial | fenced `DeviceLease` |
| `android:device-stop` | fenced lease | release/cleanup report |
| `android:device-register` | provider and external device ID | fenced external `DeviceLease` |
| `android:device-release` | fenced external lease | non-mutating release report |
| `android:build` | target, project, module, variant, tasks | `[]Artifact` and reports |
| `android:install` | fenced lease, artifacts, package, state | installed package metadata |
| `android:uninstall` | fenced lease and exact package | removal status |
| `android:test` | fenced lease, app/test artifacts and test selection | normalized test result |
| `android:capture-start` | fenced lease and capture options | `CaptureHandle` |
| `android:capture-stop` | capture handle | artifact manifest |
| `android:open` | fenced lease, server handle, app identity | `SessionHandle` |
| `android:attach` | descriptor or endpoint/backend session ID | reconnected `SessionHandle` |
| `android:close` | session handle | close/cleanup report |
| `android:cleanup` | acquired session/capture/device/server handles | aggregate LIFO cleanup report |
| `android:run` | session handle, commands, expectations | data, validations, steps, failures |
| `android:repl` | optional session ID, terminal and inspector options | interactive history/data/artifacts summary |
| `android:artifact` | session or fenced lease, artifact kinds | artifact manifest |

Device lifecycle, Appium lifecycle, capture lifecycle, and automation-session lifecycle are separate. A suite may reuse a booted emulator and server while resetting the application between tests.

## Frozen V1 data model

The exact Go field names below define the YAML names through Endly's normal conversion. Examples in this document use the same nesting.

```go
type TargetRef struct {
    Resource *location.Resource
    Topology string // local, remoteWorker, external
}

type LeaseRef struct {
    ID     string
    Fence  uint64
    Serial string
    Target TargetRef
}

type PortSet struct {
    EmulatorConsole int
    Appium          int
    UiAutomator2    int
    ChromeDriver    int
    MJPEG           int
}

type DeviceLease struct {
    LeaseRef
    AVD        string
    RealDevice bool
    Ports      PortSet
    Workspace  string // HostPath
    ArtifactRoot string // AFS URL reserved for the lane
    ExpiresAt  time.Time
}

type ServerHandle struct {
    ID        string
    Fence     uint64
    Target    TargetRef
    Endpoint  string // coordinator-reachable
    Ownership string // managed or external
    PID       int
    BasePath  string
    Log       Artifact
}

type SessionHandle struct {
    ID               string // Endly session ID
    BackendSessionID string
    Lease            LeaseRef
    Server           ServerHandle
}

type CaptureHandle struct {
    ID      string
    Fence   uint64
    Lease   LeaseRef
    Started time.Time
}

type Artifact struct {
    ID            string
    Kind          string // appAPK, testAPK, apks, aab, junit, log, mapping, symbols...
    Module        string
    Variant       string
    ABI           string
    Package       string
    VersionCode   string
    VersionName   string
    SigningDigest string
    SHA256        string
    HostPath      string
    ArtifactURL   string
    Sensitive     bool
}

type SigningProfile struct {
    Mode               string // debug, existing, bundletool
    Keystore           *location.Resource
    Alias               string
    StorePasswordSecret string
    KeyPasswordSecret   string
}

type BuildResponse struct {
    Artifacts  []Artifact
    Selected   map[string]Artifact // populated only for a unique compatible kind
    Reports    []Artifact
    DurationMs int
}

type ServerStartResponse struct { Server ServerHandle }
type DeviceStartResponse struct { Lease DeviceLease }
type CaptureStartResponse struct { Capture CaptureHandle }
type OpenResponse struct { Session SessionHandle }
```

Every mutating device action requires `LeaseRef.ID`, `Fence`, `Serial`, and matching `Target`. The fencing token is checked immediately before install, uninstall, clear, permission, launch, session creation, capture mutation, wipe, or stop. Expiry alone never transfers ownership while the prior process is still active.

Important route contracts:

```go
type ServerStartRequest struct {
    Lease           LeaseRef // target and reserved ports come from the atomic device lease
    Mode            string // managed or external
    ServerURL       string // required only for external
    AppiumExecutable string
    AppiumHome      string
    Address         string // managed must be loopback; default 127.0.0.1
    BasePath        string
    Driver          string // fixed to uiautomator2 in V1
    LogURL          string
    AllowInsecure   []string // allowlist; empty by default
    StartupTimeoutMs int
}

type BuildRequest struct {
    Target       TargetRef
    ProjectDir   string // HostPath
    Module       string
    Variant      string
    Tasks        []string
    GradleArgs   []string // argv entries, not a shell fragment
    Env          map[string]string
    Signing      *SigningProfile
    TimeoutMs    int
    ArtifactDir  string // AFS URL
}

type InstallRequest struct {
    Lease      LeaseRef
    Artifacts  []Artifact
    Kinds      []string // normally appAPK; may include testAPK
    Package    string
    State      string // freshInstall, cleanData, preserve, upgrade
    Grant      []string
    TimeoutMs  int
}

type TestRequest struct {
    Lease       LeaseRef
    Mode        string // gradle, instrument, orchestrator, managedDevice
    Artifacts   []Artifact // appAPK and testAPK for direct instrumentation
    Install     bool // install required artifacts before direct instrumentation
    AppPackage  string
    TestPackage string
    Runner      string
    Classes     []string
    Annotations []string
    Arguments   map[string]string
    ArtifactDir string
    TimeoutMs   int
}

type OpenRequest struct {
    SessionID        string
    Lease            LeaseRef
    Server           ServerHandle
    Package          string
    Activity         string
    App              *Artifact
    Reset            string // none, appData, reinstall
    AutoLaunch       *bool
    Capabilities     map[string]interface{}
}

type CaptureStartRequest struct {
    Lease       LeaseRef
    Package     string
    Logcat      bool
    Video       bool
    ArtifactDir string
    MaxBytes    int64
    SegmentMs   int
}

type CaptureStopRequest struct {
    Capture CaptureHandle
}

type CleanupRequest struct {
    Session *SessionHandle
    Capture *CaptureHandle
    Lease   *DeviceLease
    Server  *ServerHandle
}

type RunRequest struct {
    Session          SessionHandle
    Commands         []interface{}
    ActionTimeoutMs  int
    PollIntervalMs   int
    ActionDelaysMs   int
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
    AllowLicenses  bool
    AllowDownloads bool
    ArtifactDir    string
}

type ServerStopRequest struct { Server ServerHandle }

type DeviceStartRequest struct {
    Target        TargetRef
    Profile       string
    Serial        string
    WipeData      bool
    Animations    bool
    BootTimeoutMs int
    LeaseTTLms    int
    ArtifactRoot  string
}

type DeviceStopRequest struct { Lease DeviceLease }

type UninstallRequest struct {
    Lease   LeaseRef
    Package string
}

type CloseRequest struct {
    Session      SessionHandle
    TerminateApp bool
}

type ArtifactRequest struct {
    Session   *SessionHandle
    Lease     *LeaseRef
    Kinds     []string
    Directory string
    MaxBytes  int64
}

type FailureArtifactOptions struct {
    Directory  string
    Screenshot bool
    PageSource bool
    Logcat     bool
    Video      bool
    Bugreport  bool
    MaxBytes   int64
}

type TestCase struct {
    ID, Class, Status, Failure, Stack, Stdout string
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

Each route gets concrete `Init()` and `Validate()` tests. Unknown enum values, duplicate capability aliases, unsupported capability namespaces, ambiguous devices/artifacts, mismatched targets/fences, empty packages, unsafe paths, and port collisions fail before mutation.

Raw capabilities are an advanced escape hatch. Keys are canonicalized to W3C/Appium form, aliases and duplicate normalized keys are rejected, and Endly-owned platform/device/port/security values cannot be overridden.

## Appium server ownership and security

Managed mode:

- uses a pinned executable and isolated `APPIUM_HOME` from the provision manifest;
- verifies the installed UiAutomator2 version before launch;
- allocates the port as part of the fenced lease tuple;
- starts a new process group, probes `/status`, captures bounded logs, and records PID/start token;
- binds loopback and never enables `--relaxed-security`;
- permits only explicitly allowlisted insecure features;
- stops only a process whose PID and start token still match the handle.

External mode health-checks but never mutates the server. Appium endpoints must not be exposed to untrusted networks; see [Appium server security](https://appium.io/docs/en/3.3/guides/security/).

## Resource management and cleanup

Endly's current `Context.Deffer` runs callbacks FIFO and drops their errors. The Android service therefore registers one context callback for one platform `ResourceManager`. A device lease atomically reserves the device, all required ports, workspace, and artifact root before Appium starts. Internally the manager holds an idempotent LIFO stack:

```text
session -> capture -> managed Appium/tunnel -> device -> workspace
```

`CloseAll` continues after individual failures, aggregates them into `CleanupReport`, uploads the report when an artifact root exists, stores it under the Android service state, and publishes a cleanup event. The `android:cleanup` route accepts all acquired handles and performs the same LIFO best-effort cleanup as one reportable deferred action. Multiple deferred child actions are unsafe because an early child failure can suppress later cleanup. Automatic context cleanup is the best-effort safety net.

No cleanup operation may stop an external server, wipe a physical device, stop an unowned emulator, or act after a fence mismatch.

## DSL grammar and typed form

The string DSL is compiled into a closed AST; it never invokes arbitrary methods by reflection.

```ebnf
command     = [ identifier, "=" ], invocation | expectation ;
invocation  = namespace, ".", call, { ".", call } ;
expectation = "expect", "(", invocation, ")", ".", call ;
namespace   = "app" | "device" ;
call        = identifier, "(", [ argument, { ",", argument } ], ")" ;
argument    = string | number | boolean | null ;
identifier  = letter, { letter | digit | "_" } ;
```

Strings support single/double quotes and backslash escaping. Variables are expanded after parsing and immediately before a step executes. Regex is accepted only in expectation values using Endly's `/pattern/` convention. A locator that matches zero elements waits; one element acts; more than one fails unless an explicit `.first()`, `.nth(index)`, or count expectation appears.

Typed commands avoid quoting ambiguity:

```yaml
commands:
  - locator: {strategy: resourceId, value: "com.example.shop.qa:id/email"}
    action: fill
    args: [qa@example.test]
  - expect:
      locator: {strategy: accessibilityId, value: order-confirmed}
      matcher: visible
      timeoutMs: 15000
```

`fill` clears then replaces a field value; `type` appends key input. Attribute results retain backend JSON types. Cancellation propagates from the action context through polling and HTTP requests.

## Locator policy

`getByTestId` requires an explicit session profile; there is no universal Android mapping.

| Profile | Mapping | Application contract |
| --- | --- | --- |
| `viewResourceId` | W3C ID | stable Android View resource ID |
| `composeResourceId` | W3C ID | Compose test tag exposed as resource ID through the supported UiAutomator2/Compose configuration |
| `accessibilityId` | accessibility ID | intentional user-facing content description/accessibility contract |

Do not misuse content descriptions as hidden test data. Android's [Compose interoperability guidance](https://developer.android.com/develop/ui/compose/testing/interoperability) explains test-tag exposure considerations.

Additional explicit locators are `getByResourceId`, `getByAccessibilityId`, `getByClass`, `getByUiAutomator`, `getByXPath`, and `locator(strategy,value)`. `getByText` is a localization-sensitive convenience. UiSelector/UiScrollable behavior is legacy and limited in multi-window cases, so it is not a primary durable locator. XPath is a last resort.

Portable actions include `tap`, `doubleTap`, `longPress`, `fill`, `type`, `clear`, `swipe`, `scroll`, `dragTo`, state reads, and visible/hidden waits. Android device commands include back/home, notifications, keyboard, rotation, location, deep links, exact-package lifecycle, permissions, and explicit context switching. Coordinate operations are explicit escape hatches and derive coordinates from a current element rectangle, never a screenshot.

## Assertions and error model

`RunResponse` contains `Data`, `Steps`, `Validations`, `Failures`, and `Warnings`. `Assertion()` returns all inline expectation validations plus the final `expect:` map validation. Expected value, final actual value, locator, elapsed time, attempt count, and last backend error are retained.

Assertion mismatches, instrumentation failures, and flaky-pass attempts are test results. Host, lease, Appium, adb, device-health, install, protocol, and artifact failures are infrastructure errors. A retry that eventually passes remains marked flaky with every attempt preserved; Endly never reports it as an ordinary pass.

## Build, signing, and artifact selection

Run the project-owned `./gradlew` with argv-safe execution. Android's official guidance uses wrapper tasks such as `assembleDebug`, `installDebug`, and `connectedVariantAndroidTest`; see [Build from the command line](https://developer.android.com/build/building-cmdline) and [Test from the command line](https://developer.android.com/studio/test/command-line).

Return `[]Artifact`, not singular APK fields. Each product includes kind, module, variant, ABI/density when applicable, package/version, signing digest, checksum, `HostPath`, and `ArtifactURL`. A consuming action filters by kind and must fail on an ambiguous selection.

AAB-to-APKS conversion uses a pinned, checksummed bundletool. Debug/existing/bundletool signing is explicit. Keystore passwords are secret references, passed through protected input/environment/file descriptors where supported and never placed in retained argv or logs.

Build reports include unit-test XML/HTML, lint/SARIF, mappings, native symbols, and command metadata. Gradle caches and workspaces are isolated or safely keyed per lane.

## Provisioning and deployment

`android:doctor` is read-only and reports JDK, SDK paths/packages/licenses, adb devices, emulator acceleration, AVDs, disk, Node/Appium/driver versions, features, ports, and remediation. It may invoke `appium driver doctor uiautomator2`.

`android:provision` is explicitly mutating. It consumes a pinned manifest for JDK/Node/Appium/driver/command-line tools/platform-tools/build-tools/emulator/platform/system image/bundletool. It does not install unpinned “latest”, edit shell profiles, or accept licenses unless the manifest and worker policy explicitly authorize that operation. Static Endly deployment descriptors may stage archives; procedural verification uses `sdkmanager`, npm/Appium driver commands, and checksums.

Application state is explicit:

| State | Behavior |
| --- | --- |
| `freshInstall` | uninstall exact package, install selected artifacts |
| `cleanData` | preserve binary, clear exact package data |
| `preserve` | preserve binary and data |
| `upgrade` | replace binary while preserving data |

`.apk` uses explicit adb install flags; split APKs use validated `install-multiple`; `.apks` uses pinned bundletool; `.aab` is converted and signed before installation. Never wipe a physical device.

## Instrumentation

`android:test` supports Gradle connected tests, direct `am instrument`, Android Test Orchestrator, and project-owned Gradle Managed Devices. Direct instrumentation requires both app and test APK artifacts. With `Install: true`, the route selects and installs both before invocation; otherwise it verifies both packages are already installed and compatible.

Normalize each case into identifier, class, duration, attempt, status, failure/stack, stdout, and evidence. Retain original and synthesized JUnit reports. Android documents direct instrumentation and Gradle connected tasks in its [command-line testing guide](https://developer.android.com/studio/test/command-line).

## Capture and artifacts

Capture is keyed by `CaptureHandle.ID` plus destination fence, never by automation session. It may start before `android:open`.

Artifacts can include screenshot, bounded UI XML, logcat from capture start, crash buffer, Appium command timings/log slice, instrumentation reports, video segments, and optional explicit bugreport. Android `adb shell screenrecord` has a 180-second maximum documented in the [adb screenrecord reference](https://developer.android.com/tools/adb#screenrecord); long capture rotates bounded segments and returns a segment manifest, or uses a configured MJPEG/host encoder. Parallel MJPEG streaming uses unique ports.

Visual and diagnostic artifacts may contain secrets that text redaction cannot guarantee to remove. They are marked sensitive, opt-in, permission-restricted, retention-bounded, and sent only to authenticated/encrypted sinks. Textual metadata still applies header/token/secret-pattern redaction. Collection errors accompany rather than replace the primary failure.

## Complete E2E example

Endly stores an action response under the exact YAML action name. The example therefore uses camel-case action keys and the documented `${uuid.next}` UDF.

```yaml
init:
  runID: ${uuid.next}
  appPath: $WorkingDirectory(./mobile-app)
  artifactRoot: /tmp/endly/android/$runID
  packageName: com.example.shop.qa
  sessionID: android-$runID

pipeline:
  doctor:
    action: android:doctor
    target: {topology: local}
    requestedFeatures: [emulator, native, hybrid, instrumentation]

  deviceStart:
    action: android:device-start
    target: {topology: local}
    profile: pixel_api_35
    wipeData: true
    animations: false
    bootTimeoutMs: 180000
    artifactRoot: file://$artifactRoot

  build:
    action: android:build
    target: {topology: local}
    projectDir: $appPath
    module: app
    variant: qaDebug
    tasks:
      - ":app:testQaDebugUnitTest"
      - ":app:assembleQaDebug"
      - ":app:assembleQaDebugAndroidTest"
    artifactDir: file://$artifactRoot/build

  instrumentation:
    action: android:test
    lease: $deviceStart.Lease
    mode: instrument
    artifacts: $build.Artifacts
    install: true
    appPackage: $packageName
    testPackage: com.example.shop.qa.test
    runner: androidx.test.runner.AndroidJUnitRunner
    artifactDir: file://$artifactRoot/instrumentation

  install:
    action: android:install
    lease: $deviceStart.Lease
    artifacts: $build.Artifacts
    kinds: [appAPK]
    package: $packageName
    state: cleanData

  serverStart:
    action: android:server-start
    lease: $deviceStart.Lease
    mode: managed
    address: 127.0.0.1
    driver: uiautomator2
    logURL: file://$artifactRoot/appium.log

  captureStart:
    action: android:capture-start
    lease: $deviceStart.Lease
    package: $packageName
    logcat: true
    video: true
    segmentMs: 170000
    artifactDir: file://$artifactRoot/ui

  open:
    action: android:open
    sessionID: $sessionID
    lease: $deviceStart.Lease
    server: $serverStart.Server
    package: $packageName
    activity: .MainActivity
    testIDStrategy: accessibilityId
    reset: none

  checkout:
    tag: checkout
    description: A signed-in user can complete a purchase
    action: android:run
    session: $open.Session
    actionTimeoutMs: 15000
    commands:
      - device.deepLink("shop://product/sku-42", "$packageName")
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
      action: android:cleanup
      session: $open.Session
      capture: $captureStart.Capture
      lease: $deviceStart.Lease
      server: $serverStart.Server
```

Contract fixture tests must decode this exact YAML and execute it against a fixture app before it is promoted from proposal to runnable example.

## Parallel CI

A single atomic fenced lease owns `(serial, emulator console port, Appium port, systemPort, chromedriver port, MJPEG port, workspace, artifact root)`. Every mutation verifies its fence. The lease has owner, heartbeat, expiry, and explicit release; takeover increments the fence.

Recommended tiers are: PR unit/lint/smoke; merge supported API/ABI critical journeys; nightly locale/display/network/upgrade and selected physical devices; release signed-artifact installation and production-like E2E. Remote device farms implement the same service contracts through external protected Appium profiles.

## Implementation stages

1. Add `internal/mobile` AST/protocol/retry/assertion/redaction utilities and Android bootstrap import.
2. Implement target topology, fenced resource manager, doctor/provision, server lifecycle, and device lease.
3. Implement build/artifact discovery, signing, install, instrumentation, capture, and LIFO cleanup.
4. Implement sessions, locators, core actions, typed/string commands, assertions, and failure evidence.
5. Add hybrid contexts, gestures, device extensions, remote workers, physical-device gates, and parallel stress tests.

## Acceptance criteria

- Every route has concrete contract, `Init`, `Validate`, conversion, and example-decode tests.
- Parser/AST, capability normalization, argv safety, redaction, retry, fencing, cleanup, and artifact selection have unit tests.
- Fake W3C/Appium tests verify exact protocol and extension requests.
- Fixture integration builds app and test APKs, leases a clean AVD, runs instrumentation and native/hybrid DSL flows, intentionally fails, and verifies evidence/xUnit.
- Managed, remote-worker+tunnel, and external Appium modes have lifecycle tests.
- Context cleanup is LIFO, idempotent, aggregate-reporting, and never touches unowned resources.
- Two parallel lanes have no device, port, session, workspace, cache, or artifact collisions.
- No secret appears in retained textual logs/metadata; visual artifacts are labeled sensitive with enforced storage policy.

## V1 non-goals

- replacing app-owned Espresso or Compose tests;
- image-only automation or visual-diff baselines;
- bypassing CAPTCHA, biometrics, device integrity, or security controls;
- wiping physical devices;
- Play Store publication;
- a combined public mobile service or separately deployed runner daemon.
