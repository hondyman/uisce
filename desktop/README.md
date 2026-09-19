# Uisce Multi-Monitor Workstation (Desktop Module)

This directory contains the native desktop application for the **Uisce Multi-Monitor Workstation**, built on **Wails v3 (`v3.0.0-beta.23`)**. It provides an institutional-grade, multi-window trading workspace supporting multi-monitor display targeting, FINOS FDC3-compatible context interoperability, and zero-trust authentication.

---

## Architecture & Isolation

- **Fully Independent Module**: Rooted at `github.com/hondyman/uisce/desktop` with its own `go.mod` and `go.sum`.
- **Zero Backend Imports**: Does not import any packages from `backend/`. Operates strictly as a client to the Uisce platform.
- **Go Workspace Notice**: The repository contains a root `go.work` file. All Go commands inside this directory must specify `GOWORK=off`.

---

## Build Targets & Tags

The desktop module uses Go build tags to strictly segregate automated test and verification scaffolding from production artifacts:

### 1. Production Build
Production builds exclude all verification test endpoints (`/api/desk-verify/report`) and verification harness code:
```bash
GOWORK=off go build -o uisce-desk .
```
Production files:
- `spa_handler_prod.go` (`//go:build !verify`): No-op verification handler initialization.
- `verify_stub.go` (`//go:build !verify`): Stubbed test harness runner.

### 2. Verification Build
Verification builds compile the live 8-step macOS desktop test suite:
```bash
GOWORK=off go build -tags verify -o uisce-desk .
```
Verification files:
- `spa_handler_verify.go` (`//go:build verify`): Mounts `/api/desk-verify/report` and exposes `GetVerifyReportsChannel()`.
- `verify_suite.go` (`//go:build verify`): Automated macOS WKWebView test harness exercising all 8 acceptance criteria.

---

## Running the Workstation

### Prerequisites
Ensure the frontend production bundle is compiled:
```bash
cd ../frontend && npm run build
```

### Standard Launch
```bash
./uisce-desk
```

### Automated Live Verification (macOS)
Executes the honest 8-step acceptance suite against live Cocoa WKWebView windows and exits with report:
```bash
./uisce-desk --verify
# Or via npm script:
cd ../frontend && npm run test:desktop
```

---

## Key Subsystems

1. **`manager.TokenVault`**: Thread-safe single-use ephemeral token vault with 60-second TTL and background janitor. Cleanly stopped on application termination via `app.OnShutdown`.
2. **`DeskWindowManager`**: Multi-window lifecycle manager supporting monitor index targeting (`opts.Screen`), window deduplication, and display hotplug re-clamping (`ReclampOrphanedWindows`).
3. **`SPAHandler`**: Custom `http.Handler` serving `frontend/dist` with client-side SPA routing fallbacks for `/view/page/:slug`, `/view/rebalancer`, and `/workspace`.
4. **FDC3 Context Interoperability**: Native macOS WKWebViews communicate directly via same-origin `BroadcastChannel` (`wails://localhost`). Dual-broadcast `WailsRelayTransport` is available as a fallback for platforms requiring Go IPC.
