# Android runner

The `android` Endly service provides Android emulator lifecycle and, incrementally, build/deployment and E2E automation described in [`adntoid.md`](../../../../adntoid.md).

Implemented actions:

| Action | Description |
| --- | --- |
| `android:doctor` | Report adb, emulator, Appium, connected devices, and available AVD readiness |
| `android:build` | Run argv-safe Gradle-wrapper tasks and discover/checksum variant APK/AAB/APKS products |
| `android:device-start` | Attach to an exact connected serial or start an owned AVD and wait for `sys.boot_completed=1` |
| `android:device-stop` | Release a fenced lease and stop only an emulator started by this service |
| `android:device-register` | Fence a provider/cloud device ID without local adb mutation |
| `android:device-release` | Release an external registration without stopping provider infrastructure |
| `android:server-start` | Health-check/register external Appium or start a loopback-only managed server |
| `android:server-stop` | Release external registration or idempotently stop an owned server |
| `android:install` | Apply an explicit state policy and install an APK, split APKs, APKS, or AAB |
| `android:uninstall` | Remove an exact package through a valid device lease |
| `android:launch` | Resolve the launch activity and start the exact package |
| `android:terminate` | Force-stop the exact package through a valid lease |
| `android:test` | Optionally install app and test APKs, run `am instrument`, and expose failures as assertions |
| `android:capture-start` | Start owned logcat and optional segmented video capture |
| `android:capture-stop` | Stop capture and return sensitive log/video evidence metadata |
| `android:open` | Open an Appium UiAutomator2 session against valid device and server handles |
| `android:attach` | Reconnect this process to an existing Appium session or descriptor |
| `android:run` | Run assigned `app.*`/`device.*` commands and retrying inline expectations |
| `android:repl` | Run a live terminal DSL, hierarchy inspector, screenshot tool, and command history |
| `android:artifact` | Store sensitive screenshot and bounded UI-source evidence through AFS |
| `android:close` | Idempotently close the Appium session |
| `android:cleanup` | Run every registered teardown in LIFO order and report all errors |

The implementation uses argv-safe local command execution, the project Gradle wrapper, Android instrumentation, a small W3C/Appium client, a closed mobile DSL AST, and injectable backends for deterministic lifecycle/protocol tests. The DSL supports strict plural lookup, `first`/`last`/`nth`/`count`, typed and textual commands, pointer gestures, richer element/device assertions, contexts, orientation, location, permissions, alerts, and application lifecycle. Deployment accepts a single APK, split APKs, APKS, or an AAB with file-backed signing secrets; capture can rotate bounded screen-recording segments. Failed runs automatically retain screenshot, bounded source, bounded active-log tail, a checkpointed valid video segment, and a manifest while recording continues. Managed devices and Appium ports use process-shared fenced leases, while owned processes still unwind through the LIFO context cleanup stack. `open` requires a registered `ServerHandle` and explicit `testIDStrategy`. Portable 0600 session descriptors let a second Endly process run `android:attach` or inline `android:repl.attach`; attachment is non-owning unless `takeOwnership` is explicit. A gated two-AVD stress test verifies distinct emulator/Appium/UiAutomator2/MJPEG resources under concurrent sessions. The real integration also installs a minimal device-side instrumentation APK and verifies that its intentional assertion failure is normalized rather than returned as infrastructure failure. Provider-neutral cloud mode fences an external device ID, health-checks external Appium, supplies `appReference` plus namespaced provider capabilities, and never runs adb install/capture/stop against farm infrastructure. Persistent remote execution uses Endly's authenticated control-plane sessions; see [mobile remote workers](../MOBILE_REMOTE_WORKERS.md). Provider-specific upload APIs remain adapters around `appReference` rather than runner-owned credentials.

External farm shape:

```yaml
- action: android:device-register
  request: {provider: example-farm, deviceID: pixel-remote, platformVersion: "15"}
- action: android:server-start
  request: {lease: $deviceRegister.Lease, mode: external, serverURL: $secureAppiumURL}
- action: android:open
  request:
    lease: $deviceRegister.Lease
    server: $serverStart.Server
    appReference: farm://apps/build-123
    testIDStrategy: accessibilityId
    capabilities:
      farm:options: {project: endly}
```

Run the real host check:

```bash
endly -r=service/testing/runner/android/test/doctor.yaml
```

Run the complete emulator integration after provisioning an SDK and AVD:

```bash
ANDROID_SDK_ROOT=/path/to/android-sdk \
ENDLY_ANDROID_EMULATOR_INTEGRATION=1 \
ENDLY_ANDROID_TEST_AVD=endly_api_35 \
ENDLY_ANDROID_TEST_PROJECT=/path/to/android-project \
ENDLY_ANDROID_TEST_PACKAGE=com.example.app \
  go test ./service/testing/runner/android -run '^TestAndroidEmulatorIntegration$' -v -count=1
```

The gated test builds with the project Gradle wrapper, starts a clean owned AVD, installs and launches the APK, captures logcat and a non-empty screenshot, terminates the app, and stops the emulator. Set the documented Appium environment variables to include UiAutomator2 session and source capture.

The reference validation uses API 35 `endly_api_35` on arm64 with Appium 3.7.0 and UiAutomator2 8.6.1. It additionally opens a managed session, resolves a native view through the DSL, asserts visibility, captures Appium screenshot/source evidence, and verifies that no emulator or Appium process remains.

Add `ENDLY_ANDROID_AAB_INTEGRATION=1` and point `ENDLY_ANDROID_BUNDLETOOL` at the official `bundletool-all` JAR to build an AAB, generate device-targeted signed APKS using file-backed passwords, and install it on the real emulator before the remaining suite runs. This path is verified with bundletool 1.18.3; Endly passes its already-resolved adb path explicitly rather than relying on `ANDROID_HOME` or PATH.

Run the two-lane isolation stress test with two provisioned AVD names:

```bash
ENDLY_ANDROID_PARALLEL_INTEGRATION=1 \
ENDLY_ANDROID_PARALLEL_AVDS=api35_lane_1,api35_lane_2 \
  go test ./service/testing/runner/android -run '^TestAndroidParallelEmulatorStress$' -v -count=1
```

Run a live inspector with `test/repl.yaml`. Once setup completes, Endly displays an `android[session]>` prompt and executes each entered DSL command immediately:

```text
:status
:tree workspace
heading = app.getByText("Choose your workspace").text()
app.getByClass("android.widget.EditText").fill("https://example.test")
expect(app.getByClass("android.widget.EditText")).toHaveText("https://example.test", 10000)
:screenshot
:history
:quit
```

Inspector commands are `:status`, `:source`, `:tree [filter]`, `:find <text>`, `:screenshot`, `:history`, `!<number>`, `:help`, `:close`, and `:quit`. In a real terminal, Tab completes DSL/meta commands and Up/Down navigate persistent history. Command errors are printed and the prompt continues unless `failOnError` is enabled. Ctrl-C returns to the workflow so deferred cleanup runs.

For a deliberate cross-process handoff, create the session on an attached device and external Appium server with `descriptorPath` plus `keepSession: true`. A later process can enter the inspector directly:

```yaml
action: android:repl
request:
  attach:
    descriptorPath: /secure/run/android-session.json
    takeOwnership: false
  historyPath: /secure/run/android-history.jsonl
```

Use `takeOwnership: true` only when `:close` or cleanup should delete the remote Appium session and its descriptor.
