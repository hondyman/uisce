# Uisce Multi-Monitor Workstation — Architectural Pillars & Sprint Roadmap

This document serves as the authoritative blueprint and engineering schedule for the Uisce Universal Browser & Native Wails v3 Desktop Multi-Monitor Workstation.

---

## 1. Executive Status: Sprint 1 & Sprint 2 Completed & Verified

Both Sprint 1 and Sprint 2 are **complete, fully verified, and passing all automated gates**:

- **Sprint 1 (Part 1 Blocker Closure & Hardening - Commit `1ffaab107`)**:
  - Closed ghost window bug via Go `desktop:window-closed` Wails event & frontend automatic unregistration.
  - Hardened token exchange, strict URL sanitization, and janitor lifecycle.
  - Multi-screen Playwright smoke test and live macOS 8/8 verification suite.

- **Sprint 2 (Internal Intents, PostgreSQL Layout Profiles & Travel Mode - Verified)**:
  - **Pillar 1: FDC3-Compatible Intent Resolution & Router**:
    - Implemented live cross-window handler mesh over system channel (`IntentRegistry.ts`).
    - Smart routing: direct route on single target; dark-themed `IntentResolverModal` on ambiguous multi-target matches.
    - Robust failure resilience: explicit acknowledgment contract (`fdc3.intent.ack`), 1500ms ack timeout with dead-handler purging (`AckTimeoutError`), automatic fallback re-resolution, and strict intent loop guard.
    - Wired views: `AIPortfolioRebalancer` (`ViewAnalysis`), `ScenarioAnalysisPro` (`ViewAnalysis`), `FixedIncomeDashboard` (`ViewInstrument`), and blotter trigger (`ViewAnalysis`).
    - Vitest unit tests: **8 / 8 passed** (`IntentRegistry.test.ts`).
  - **Pillar 2: PostgreSQL Multi-Tenant Layout Profiles**:
    - Migration: `public.user_workspace_layouts` table with tenant & user isolation (`20261019_001_workspace_layout_preferences.up.sql`).
    - Backend REST API: `GET/POST /api/user/preferences/workspace-layout` with 1MB payload ceiling via `http.MaxBytesReader`.
    - Go backend unit tests: **4 / 4 passed** (`workspace_layout_test.go`).
    - Frontend client & sync: single-writer rule (hub window writes, popouts read), server-authoritative merge, and offline cache fallback.
    - Vitest sync tests: **3 / 3 passed** (`LayoutProfileSync.test.ts`).
  - **Pillar 3: Travel Mode & Unified Alert Surface**:
    - Single cohesive alert strip in `UniversalWorkspaceHub` eliminating banner stacking.
    - Non-destructive view-time consolidation (`handleConsolidateToTabs`): collapses auxiliary windows into Dockview tabs without mutating saved multi-monitor profile.
    - Multi-monitor polling (4s interval) offering Travel Mode on display disconnect and restore prompt on display reconnect.
    - Vitest tests: **3 / 3 passed** (`TravelMode.test.ts`).
    - Playwright multi-screen smoke test: **Step 9 added, 9 / 9 checks passed** (`smoke_multiscreen.mjs`).

- **Full Verification Chain (Sprint 2)**:
  - **TypeScript Ratchet**: `node scripts/ts-ratchet.mjs` → **699 total error blocks / 383 unique signatures, 0 new** (baseline 702 / 386; exit code 0).
  - **Frontend Vitest**: **44 / 44 test files passed (195 tests, 0 failures)**.
  - **Production Build**: **PASS** (`npm run build` generated `frontend/dist` in 21.08s).
  - **Playwright Multi-Screen Smoke**: **9 / 9 checks passed** (`npm run test:multiscreen`), verifying real cross-window intent routing, UI reflection, and explicit ack.
  - **Go Backend Unit Tests**: **5 / 5 passed** (`go test -v ./internal/api -run TestWorkspaceLayout`), verifying cross-user and cross-tenant isolation.
  - **Go Desktop Unit Tests**: **22 / 22 passed** (`GOWORK=off go test -v ./manager ./...`).
  - **Live macOS Desktop Suite**: **9 / 9 passed** on real WKWebView windows (`./desktop/uisce-desk --verify`), verifying live cross-window intent resolution with explicit acknowledgment.

---

## 2. Five Pillars of Architectural Excellence

### Pillar 1: Internal Intent Resolution (FDC3-Compatible)
- **Concept**: Provide intent-based workflows (`fdc3.raiseIntent`) mapping trader actions to internal application views without an external app directory.
- **Target Intents & Handlers**:
  - `ViewInstrument`: Opens/focuses Market Depth or Security Master.
  - `ViewChart`: Opens/focuses Technical / Yield Curve charting.
  - `ViewOrders`: Opens/focuses OMS Order Blotter filtered by symbol.
  - `ViewAnalysis`: Opens/focuses AIPortfolioRebalancer or ScenarioAnalysisPro.
  - `ViewExecution`: Opens/focuses Execution Allocation drill-down.
- **Smart Routing & UI Ergonomics**:
  - If a single target view handles the intent, **route directly without showing a modal** (preserves speed and minimizes trader friction).
  - Surface an intent resolver modal **only when multiple possible target views exist**.
- **Minimal Result Contract**:
  - v1 uses a fire-and-forget model with visible acknowledgment and focus state in the target window.

### Pillar 2: High-Density Canvas Grids & Market Data Boundary
- **Concept**: Support 10,000+ streaming rows with sub-millisecond sorting, grouping, and virtualized scrolling in Dockview panels.
- **STRICT ARCHITECTURAL BOUNDARY — Market Data vs FDC3 Context**:
  - **Tick Data Never Goes Over FDC3**: High-frequency financial tick data, L2/L3 order book quotes, and execution fills MUST NEVER travel over the FDC3 context bus.
  - **Market Data Flow**: Ticks and order book updates flow per-window via direct WebSocket / gRPC-web connections from backend streaming services to each specific chart/grid view.
  - **FDC3 Context Flow**: The FDC3 bus carries **only user context** (e.g. which symbol or order is currently focused). Broadcasting ticks over FDC3 creates an IPC bottleneck that spams every window with every symbol's stream.
- **Loop-Guard Rule**:
  - Incoming context updates only mutate local state; user interactions trigger outgoing broadcasts.

### Pillar 3: Multi-Tenant PostgreSQL Presets & Travel Mode
- **Concept**: Roaming workspace configurations stored in PostgreSQL under tenant isolation.
- **Backend Architecture & Boundary**:
  - Restful profile endpoint `/api/user/preferences/workspace-layout` in `backend/internal/api`.
  - Frontend talks to API via standard HTTP client.
  - **Zero imports from `backend/` into `desktop/`** (preserves clean packaging and isolation).
- **Travel Mode (View-Time Dynamic Mode)**:
  - When an external monitor is disconnected, the system enters "Travel Mode" (collapsing auxiliary views into tabbed Dockview panels on the primary screen).
  - **Rule**: Travel Mode is applied as a **view-time dynamic layout transformation**, NOT a destructive mutation of the saved multi-monitor layout in the database or localStorage. When the user reconnects to their multi-monitor desk, the original layout restores seamlessly.

### Pillar 4: Unified Keyboard Command Bar & Global Shortcuts
- **Concept**: Rapid keyboard-first navigation for portfolio managers and traders.
- **Internal Command Bar (`Cmd+K` / `Ctrl+K`)**:
  - Instant searchable command palette for searching instruments, switching FDC3 color channels, opening views, and applying presets.
  - **Naming Discipline**: This is an **internal command bar**, NOT an "FDC3 App Directory".
- **Global vs Local Shortcuts**:
  - **Desktop Mode**: Global OS shortcuts (`Cmd+1`, `Cmd+2`) registered via Go AppKit/Win32 APIs to focus windows across physical displays even when another window is focused.
  - **Browser Mode**: Keyboard navigation operates within the active browser window/tab.

### Pillar 5: Institutional Packaging & Enterprise Distribution
- **Concept**: Enterprise-ready binaries with modern code signing and zero unnecessary OS permissions.
- **Security Audit Compliance**:
  - **Zero Camera/Microphone Entitlements**: Financial workstations have no business requesting audio/video recording permissions; omitting them removes audit red flags during institutional security diligence.
- **Packaging Targets**:
  - **macOS**: Signed and notarized Universal DMG / `.app` bundle via Apple Developer ID and `notarytool`.
  - **Windows**: **MSIX package** preferred over NSIS (modern signing, auto-update, enterprise Microsoft Intune & Group Policy deployment).
  - **Tooling Verification**: Pin and verify Wails v3 packaging commands per release before production builds.

---

## 3. Strict Naming Discipline

To ensure audit readiness and avoid diligence issues:
- **Use**: *"FDC3-compatible context model"*, *"FDC3-compatible context bus"*, *"FDC3-aligned data shapes"*.
- **Never Use**: *"FINOS FDC3 Certified"*, *"FDC3 App Directory"*, *"Full FDC3 Desktop Agent"*.
- Uisce provides a specialized, ultra-low-latency financial context interoperability bus inspired by the FINOS FDC3 standard, tailored for browser and Wails desktop multi-screen environments.

---

## 4. Multi-Sprint Implementation Roadmap

```mermaid
flowchart TD
    S1[Sprint 1: Hardening & Blocker Closure<br>COMPLETED - Commit 1ffaab107] --> S2[Sprint 2: Internal Intents & Layout Presets<br>COMPLETED & Fully Verified]
    S2 --> S3[Sprint 3: High-Density Grids & Command Bar<br>WebSocket Tick Boundary + Cmd+K]
    S3 --> S4[Sprint 4: Multi-Monitor Hardware Lab & Windows WebView2 Protocol<br>Mixed-DPI + WINDOWS_VERIFICATION_CHECKLIST.md]
    S4 -->|HARD GATE: Checklist Passed| S5[Sprint 5: Enterprise Packaging & Signing<br>macOS Notarization + Windows MSIX]
```

### Sprint 1: Part 1 Completion & Blocker Closure (COMPLETED)
- **Focus**: Close the 4 blockers from review; harden token exchange, URL hygiene, loop guards, and ghost window prevention.
- **Deliverables & Evidence**:
  - DeskWindowManager closing hooks + DOM event dispatching.
  - LayoutManager automatic unregistration on close + dismiss cleanup.
  - Full Vitest suite passing (41/41 suites, 181 tests).
  - Multi-screen Playwright smoke with cross-view sync and ghost window repro assertions (7/7 passed).
  - Live macOS verification suite (8/8 passed).
  - Commit `1ffaab107` landed on `main`.

### Sprint 2: Internal Intent Resolution, PostgreSQL Profiles & Travel Mode (COMPLETED)
- **Focus**: Windows-independent functional elevation.
- **Deliverables & Evidence**:
  - **FDC3-Compatible Intent Router**: Live handler registry (`IntentRegistry.ts`), closed standard vocabulary, 1500ms ack timeout with dead-handler purging (`AckTimeoutError`), fallback re-resolution, intent loop guard, and dark-themed `IntentResolverModal`. Live verified across real windows in both Playwright smoke (Step 9) and macOS desktop suite (Step 7).
  - **PostgreSQL Layout Profile Service**: `public.user_workspace_layouts` migration, REST API with 1MB payload limit, single-writer rule (hub writes, popouts read), and server-authoritative merge. Cross-user and cross-tenant isolation verified by test (`TestWorkspaceLayout_TenantAndUserIsolation`).
  - **Travel Mode & Unified Alert Surface**: Non-destructive view-time consolidation into Dockview tabs without mutating saved layouts; single alert strip in `UniversalWorkspaceHub` for browser restore, monitor disconnect, and reconnect prompts.
  - **Verification**: Vitest (44/44 suites, 195 tests), TS ratchet (699 blocks / 383 unique signatures, 0 new), Production build (21.08s), Playwright multi-screen smoke (9/9 passed, including live intent routing & ack), Go backend layout tests (5/5 passed, including isolation), Go desktop unit tests (22/22 passed), Live macOS desktop suite (9/9 passed, including live cross-window intent resolution with explicit ack).

### Sprint 3: High-Density Grids & Command Bar
- **Focus**: High-throughput trading UI and keyboard ergonomics.
- **Scope**:
  - **Market Data Streaming Boundary**: Direct WebSocket market data feeds to blotter & depth grids; FDC3 context strictly limited to selection/focus.
  - **Internal Command Bar (`Cmd+K`)**: Fuzzy-search navigation for views, instruments, channels, and presets.
  - **Shortcut Engine**: Window focus shortcuts across displays in Go desktop mode; scoped within window in browser mode.

### Sprint 4: Multi-Monitor Hardware Lab, Mixed-DPI & Windows WebView2 Protocol
- **Focus**: Physical display testing, DPI scaling, and Windows verification.
- **Scope**:
  - **Physical Multi-Monitor Validation**: Verify window clamping, menu positioning, and DPI scaling across 1x, 2x, and mixed-DPI multi-monitor desks.
  - **Windows WebView2 6-Point Verification**:
    - Execute [`WINDOWS_VERIFICATION_CHECKLIST.md`](file:///Users/eganpj/GitHub/uisce/desktop/WINDOWS_VERIFICATION_CHECKLIST.md) on Windows 11 under WebView2 runtime.
    - Verify origin behavior (`http://wails.localhost`).
    - Verify `BroadcastChannel` cross-window delivery under shared user data folder.
    - Verify Go `WailsRelayTransport` fallback if `BroadcastChannel` is partitioned.

### Sprint 5: Enterprise Packaging, Installer Signing & Distribution
- **HARD GATE**: **Sprint 5 Windows packaging is blocked until Sprint 4 Windows WebView2 verification passes.**
- **Scope**:
  - **macOS Build**: Notarized, codesigned `.app` bundle and drag-and-drop DMG with zero camera/mic entitlements.
  - **Windows Build**: Signed **MSIX** enterprise installer with code-signing certificate (ready for Intune and Group Policy deployment).
  - **Automated Release Workflow**: Pinned Wails v3 packaging automation in GitHub Actions.
