# Mobile remote workers and external farms

Android and iOS runner workflows can execute inside a persistent Endly control-plane session on a machine that owns the SDK, devices, signing material, Appium, and artifact workspace. Each action operation reuses that session's Endly context, so fenced leases, managed servers, sessions, captures, deferred cleanup, and workflow state survive between client calls.

Two deployment shapes are supported:

1. Keep `endly serve` on worker loopback and expose it through an authenticated SSH forward. Appium can remain bound to worker loopback because mobile actions execute on the worker.
2. Bind the worker to a protected network interface with `--allow-remote`. Endly then requires JWT authentication and explicit `--allow-action` entries. Use narrowly scoped mobile actions rather than `*` in production.

Example worker:

```bash
endly serve \
  --addr 127.0.0.1:8080 \
  --max-sessions 8 \
  --session-ttl 45m \
  --allow-action android:* \
  --allow-action ios:*
```

Example coordinator tunnel and client session:

```bash
ssh -N -L 18080:127.0.0.1:8080 mobile-worker.example
endly client --endpoint http://127.0.0.1:18080 open --name mobile-ci
endly client load --url mobile-e2e.yaml --alias mobile-e2e
endly client run mobile-e2e
endly client status
endly client close
```

For a directly reachable worker, start `endly serve` with `--allow-remote`, a `--jwt-public-key`, issuer/audience/scope settings, and action allowlists. Configure the client with the matching private-key profile. The control API isolates sessions by JWT subject and rejects cross-subject access.

Input files should be staged through an authenticated AFS URL or already exist at a declared worker `HostPath`. Outputs should be uploaded to AFS before the session closes when they must outlive the worker workspace. Passwords, signing material, provider credentials, and JWT keys remain worker-side or in Endly/Scy secret resources; do not put them in workflow literals or provider capability maps.

External device farms use `android:device-register` or `ios:destination-register`, followed by an external Appium server and provider-issued `appReference`. Provider-specific W3C namespaced capabilities pass through unchanged, while Endly-owned platform, destination, app, and WDA capabilities are protected. Release removes only Endly's fenced registration and never invokes local SDK mutation or provider teardown.

The control-plane integration tests execute complete Android and iOS external-Appium lifecycles across separate authenticated action operations and verify cleanup and JSON-stable fences.
