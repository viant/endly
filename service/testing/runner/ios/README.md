# iOS runner

The `ios` Endly service provides iOS Simulator lifecycle and, incrementally, build/signing/deployment and E2E automation described in [`ios.md`](../../../../ios.md).

Implemented actions:

| Action | Description |
| --- | --- |
| `ios:doctor` | Report Xcode tools, installed Simulator runtimes, and available Simulators |
| `ios:build` | Run Simulator build/build-for-testing and discover/checksum `.app`/`.xctestrun` products |
| `ios:simulator-start` | Attach to, clone, or create and boot a Simulator under a fenced lease |
| `ios:simulator-stop` | Release the lease, shut down the Simulator, and delete only an owned clone |
| `ios:server-start` | Health-check/register external Appium or start a loopback-only managed server |
| `ios:server-stop` | Release external registration or idempotently stop an owned server |
| `ios:install` | Install one Simulator `.app` with an explicit fresh/preserve policy |
| `ios:uninstall` | Remove an exact bundle through a valid Simulator lease |
| `ios:launch` | Launch an exact Simulator bundle with argv-safe arguments/environment |
| `ios:terminate` | Terminate an exact Simulator bundle through a valid lease |
| `ios:test` | Run XCTest/test-without-building and normalize modern `.xcresult` summaries into assertions |
| `ios:capture-start` | Start an owned Simulator unified-log stream with an optional predicate |
| `ios:capture-stop` | Stop the stream and return sensitive log evidence metadata |
| `ios:open` | Open an Appium XCUITest session against valid Simulator and server handles |
| `ios:run` | Run assigned `app.*`/`device.*` commands and retrying inline expectations |
| `ios:repl` | Run a live terminal DSL, hierarchy inspector, screenshot tool, and command history |
| `ios:artifact` | Store sensitive screenshot and bounded accessibility-source evidence through AFS |
| `ios:close` | Idempotently close the Appium session |
| `ios:cleanup` | Run every registered teardown in LIFO order and report all errors |

The implementation uses argv-safe local command execution, Xcode Simulator build/test and `.xcresult` tooling, a small W3C/Appium client, a closed mobile DSL AST, and injectable backends for deterministic lifecycle/protocol tests. Managed Appium and unified-log processes are owned by a LIFO context cleanup stack. `open` requires a registered `ServerHandle`. Remote macOS targets, advanced WDA modes, signing/archive, physical-device deployment, Simulator video, and richer failure manifests remain to be implemented.

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

Inspector commands are `:status`, `:source`, `:tree [filter]`, `:find <text>`, `:screenshot`, `:history`, `!<number>`, `:help`, `:close`, and `:quit`. Command errors are printed and the prompt continues unless `failOnError` is enabled. Ctrl-C returns to the workflow so deferred cleanup runs.
