# iOS runner

The `ios` Endly service provides iOS Simulator lifecycle and, incrementally, build/signing/deployment and E2E automation described in [`ios.md`](../../../../ios.md).

Implemented actions:

| Action | Description |
| --- | --- |
| `ios:doctor` | Report Xcode tools, installed Simulator runtimes, and available Simulators |
| `ios:build` | Run Simulator build/build-for-testing and discover/checksum `.app`/`.xctestrun` products |
| `ios:simulator-start` | Attach to, clone, or create and boot a Simulator under a fenced lease |
| `ios:simulator-stop` | Release the lease, shut down the Simulator, and delete only an owned clone |
| `ios:device-list` | List paired physical iOS/iPadOS devices known to CoreDevice |
| `ios:device-lease` | Acquire a process-shared fenced lease for one exact physical-device UDID |
| `ios:device-release` | Release the lease without terminating, shutting down, or erasing the device |
| `ios:destination-register` | Fence a provider/cloud device ID without local CoreDevice mutation |
| `ios:destination-release` | Release an external registration without stopping provider infrastructure |
| `ios:server-start` | Health-check/register external Appium or start a loopback-only managed server |
| `ios:server-stop` | Release external registration or idempotently stop an owned server |
| `ios:install` | Install one compatible Simulator or signed device `.app` with an explicit policy |
| `ios:uninstall` | Remove an exact bundle through a valid destination lease |
| `ios:launch` | Launch an exact bundle with argv-safe arguments/environment and return its PID |
| `ios:terminate` | Terminate a Simulator bundle or exact physical-device PID through a valid lease |
| `ios:test` | Run XCTest/test-without-building and normalize modern `.xcresult` summaries into assertions |
| `ios:capture-start` | Start an owned Simulator unified-log stream with an optional predicate |
| `ios:capture-stop` | Stop the stream and return sensitive log evidence metadata |
| `ios:open` | Open an Appium XCUITest session with managed/prebuilt/preinstalled/external WDA |
| `ios:attach` | Reconnect this process to an existing Appium session or descriptor |
| `ios:run` | Run assigned `app.*`/`device.*` commands and retrying inline expectations |
| `ios:repl` | Run a live terminal DSL, hierarchy inspector, screenshot tool, and command history |
| `ios:artifact` | Store sensitive screenshot and bounded accessibility-source evidence through AFS |
| `ios:close` | Idempotently close the Appium session |
| `ios:cleanup` | Run every registered teardown in LIFO order and report all errors |

The implementation uses argv-safe local command execution, Xcode destination build/test, archive/export signing inputs, `devicectl` physical-device lifecycle, and `.xcresult` tooling, a small W3C/Appium client, a closed mobile DSL AST, and injectable backends for deterministic lifecycle/protocol tests. The DSL supports strict plural lookup, `first`/`last`/`nth`/`count`, typed and textual commands, pointer gestures, richer element/device assertions, contexts, orientation, location, alerts, permissions, appearance, biometrics, and application lifecycle. Simulator capture can rotate bounded video segments; failed runs automatically retain screenshot, bounded source, bounded active-log tail, a checkpointed valid video segment, and a manifest while recording continues. Managed destinations and Appium ports use process-shared fenced leases, while owned processes still unwind through the LIFO context cleanup stack. Physical-device release never shuts down or erases hardware. Portable 0600 session descriptors let a second Endly process run `ios:attach` or inline `ios:repl.attach`; attachment is non-owning unless `takeOwnership` is explicit. A gated two-Simulator stress test verifies distinct Simulator/Appium/WDA/MJPEG/DerivedData resources under concurrent sessions. The bundled project contains one passing and one intentionally failing UI test; `ios:test` retains and normalizes the real `.xcresult`. `collectDiagnostics` defaults to `never` to avoid Xcode's ten-minute failure sysdiagnose and accepts `on-failure` when those diagnostics are wanted. `open.wda` owns all WDA lifecycle/signing/port capabilities and supports `managed`, `prebuilt`, `preinstalled`, and `external`; raw conflicting capabilities are rejected. Current compatibility rules reject preinstalled WDA below iOS 17 and on iOS 27+ Simulators, where direct XCTest-runner launch is not viable, with guidance to use another mode. Provider-neutral cloud mode fences an external destination, health-checks external Appium, supplies `appReference` plus namespaced provider capabilities, and never invokes local build/install/capture/stop against farm infrastructure. Persistent remote execution uses Endly's authenticated control-plane sessions; see [mobile remote workers](../MOBILE_REMOTE_WORKERS.md). Provider-specific upload APIs remain adapters around `appReference` rather than runner-owned credentials. On non-Darwin hosts, `ios` registers the same action-compatible unsupported-platform stub so Endly still builds and reports a useful error. Connected-device hardware validation remains.

External farm shape:

```yaml
- action: ios:destination-register
  request: {provider: example-farm, deviceID: iphone-remote, platformVersion: "18.5"}
- action: ios:server-start
  request: {destination: $destinationRegister.Lease, mode: external, serverURL: $secureAppiumURL}
- action: ios:open
  request:
    destination: $destinationRegister.Lease
    server: $serverStart.Server
    appReference: farm://apps/build-456
    capabilities:
      farm:options: {project: endly}
```

Run the real host check:

```bash
endly -r=service/testing/runner/ios/test/doctor.yaml
```

Run the complete bundled-fixture integration on an installed Simulator runtime:

```bash
ENDLY_IOS_EMULATOR_INTEGRATION=1 \
  go test ./service/testing/runner/ios -run '^TestIOSSimulatorIntegration$' -v -count=1
```

This creates an owned Simulator, builds and checksums `test/fixture/FixtureApp`, installs and launches it, captures unified logs and a non-empty screenshot, terminates the app, and deletes the Simulator.

To include the complete managed-Appium/XCUITest DSL path:

```bash
export PATH=/path/to/node-v24/bin:$PATH
ENDLY_IOS_EMULATOR_INTEGRATION=1 \
ENDLY_IOS_APPIUM_INTEGRATION=1 \
ENDLY_IOS_APPIUM_EXECUTABLE=/path/to/appium \
ENDLY_IOS_APPIUM_HOME=/path/to/appium-home \
  go test ./service/testing/runner/ios -run '^TestIOSSimulatorIntegration$' -v -count=1
```

That path additionally starts owned Appium/WDA processes, locates the fixture by accessibility identifier, verifies its initial text, taps the increment button, waits for `Count: 1`, captures Appium screenshot/source evidence, and verifies complete cleanup.

Run the two-lane isolation stress test:

```bash
ENDLY_IOS_PARALLEL_INTEGRATION=1 \
  go test ./service/testing/runner/ios -run '^TestIOSParallelSimulatorStress$' -v -count=1
```

Run the WDA lifecycle compatibility suite:

```bash
ENDLY_IOS_WDA_MODES_INTEGRATION=1 \
  go test ./service/testing/runner/ios -run '^TestIOSWDAModesIntegration$' -v -count=1
```

Query CoreDevice through the physical-device parser:

```bash
ENDLY_IOS_DEVICE_LIST_INTEGRATION=1 \
  go test ./service/testing/runner/ios -run '^TestPhysicalDeviceListIntegration$' -v -count=1
```

Run `test/repl.yaml` to enter the live inspector. Once setup completes, Endly displays an `ios[session]>` prompt:

```text
:status
:tree increment
count = app.getByTestId("count").text()
app.getByTestId("increment").tap()
expect(app.getByTestId("count")).toHaveText("Count: 1", 10000)
:screenshot
:history
:quit
```

Inspector commands are `:status`, `:source`, `:tree [filter]`, `:find <text>`, `:screenshot`, `:history`, `!<number>`, `:help`, `:close`, and `:quit`. In a real terminal, Tab completes DSL/meta commands and Up/Down navigate persistent history. Command errors are printed and the prompt continues unless `failOnError` is enabled. Ctrl-C returns to the workflow so deferred cleanup runs.

For a deliberate cross-process handoff, select an existing Simulator with `keepBooted: true`, use an external Appium server, and open with `descriptorPath` plus `keepSession: true`. A later process can enter the inspector directly:

```yaml
action: ios:repl
request:
  attach:
    descriptorPath: /secure/run/ios-session.json
    takeOwnership: false
  historyPath: /secure/run/ios-history.jsonl
```

Use `takeOwnership: true` only when `:close` or cleanup should delete the remote Appium session and its descriptor.
