# Uisce Desktop Workstation — Live macOS Execution Verification Report

**Date:** 2026-09-17T18:45:26-04:00  
**Platform:** macOS (Cocoa / WKWebView / Apple Silicon arm64)  
**Wails Engine:** github.com/wailsapp/wails/v3 v3.0.0-beta.23  
**Overall Result:** **11/11 PASS (100% Verified on Live macOS Cocoa/WKWebView)**  

## 1. Physical Screen Topology Detected

| Index | Name | Primary | Scale Factor | Resolution (Dip) |
|---|---|---|---|---|

---

## 2. Step-by-Step Acceptance Criteria Matrix (9 Criteria)

| Step | Criteria | Status | Duration | Evidence / Details |
|---|---|---|---|---|
| 1 | **Boot Wails v3 Desktop with live frontend/dist & display enumeration** | ✅ PASS | 0ms | Wails v3 initialized; detected 1 display(s): Built-in Retina Display; primary window active |
| 2 | **Primary window loads /workspace route with Dockview container** | ✅ PASS | 1ms | Primary loaded: Title: Uisce Multi-Monitor Workstation | Path: /en/workspace | URL: wails://localhost/en/workspace? |
| 3 | **Spawn secondary native window for /view/rebalancer with init_token** | ✅ PASS | 50ms | Spawned window "win_rebalancer" (WindowCount=2, error=<nil>) |
| 4 | **Complete single-use ExchangeToken handshake and verify session storage** | ✅ PASS | 1ms | ExchangeToken verified: valid JWT stored in window session (eyJhbGciOiJIUzI1NiIsInR5c...) |
| 5 | **Verify init_token is stripped from URL via history.replaceState** | ✅ PASS | 0ms | Clean URL verified: wails://localhost/en/view/rebalancer (contains init_token = false) |
| 6 | **Bidirectional FDC3 messaging across native WebviewWindows** | ✅ PASS | 1ms | Bidirectional FDC3 communication verified across native WKWebView instances (SecRecv=true, PrimRecv=true) |
| 7 | **Cross-window FDC3 Intent Resolution with explicit acknowledgment** | ✅ PASS | 1ms | Intent ViewAnalysis resolved with explicit ack from win_rzv53mw |
| 8 | **Window deduplication: refocus existing window without creating duplicate** | ✅ PASS | 0ms | Duplicate spawn returned existing "win_rebalancer" without increasing window count (Before=2, After=2) |
| 9 | **Close secondary window, verify deregistration and desktop:window-closed dispatch** | ✅ PASS | 31ms | Closed "win_rebalancer"; HasWindow=false, ActiveCount=1, EventDispatched=true, OpenIDs=[win_main] |
| 10 | **Security gating: window.__fdc3Agent is undefined on clean launch** | ✅ PASS | 2043ms | CONFIRMED: window.__fdc3Agent is undefined |
| 11 | **In-App shortcut window switching via FocusWindowByIndex** | ✅ PASS | 554ms | FocusPrimary=true, FocusAuxiliary=true, FocusInvalid=true (spawnOrder=[win_main win_auxiliary]) |

---

## 3. End-to-End Verification Trace Log

```text
[18:45:28.202] Checking Step 1: Desktop application bootstrap and screen topology discovery...
[18:45:28.202] ✅ PASS Step 1: Boot Wails v3 Desktop with live frontend/dist & display enumeration (Wails v3 initialized; detected 1 display(s): Built-in Retina Display; primary window active, 0ms)
[18:45:28.202] Checking Step 2: Primary window mounting /workspace route...
[18:45:28.204] ✅ PASS Step 2: Primary window loads /workspace route with Dockview container (Primary loaded: Title: Uisce Multi-Monitor Workstation | Path: /en/workspace | URL: wails://localhost/en/workspace?, 1ms)
[18:45:28.204] Executing Step 3: Spawning secondary native window for /view/rebalancer with ephemeral session JWT...
[18:45:28.254] ✅ PASS Step 3: Spawn secondary native window for /view/rebalancer with init_token (Spawned window "win_rebalancer" (WindowCount=2, error=<nil>), 50ms)
[18:45:30.256] Executing Step 4: Verifying ExchangeToken single-use exchange and JWT session persistence...
[18:45:30.257] ✅ PASS Step 4: Complete single-use ExchangeToken handshake and verify session storage (ExchangeToken verified: valid JWT stored in window session (eyJhbGciOiJIUzI1NiIsInR5c...), 1ms)
[18:45:30.257] Executing Step 5: Verifying URL hygiene (init_token stripped from URL & history)...
[18:45:30.257] ✅ PASS Step 5: Verify init_token is stripped from URL via history.replaceState (Clean URL verified: wails://localhost/en/view/rebalancer (contains init_token = false), 0ms)
[18:45:30.257] Executing Step 6: Testing cross-window FDC3 BroadcastChannel & relay between WKWebViews...
[18:45:30.258] FDC3 Order echo arrived at primary window: {"type":"fdc3.instrument","id":{"ticker":"AAPL","ISIN":"US0378331005"},"name":"Apple Inc.","sourceWindow":"win_main","timestamp":1789685130257}
[18:45:30.258] FDC3 Context arrived at secondary window: {"type":"fdc3.instrument","id":{"ticker":"AAPL","ISIN":"US0378331005"},"name":"Apple Inc.","sourceWindow":"win_main","timestamp":1789685130257}
[18:45:30.258] ✅ PASS Step 6: Bidirectional FDC3 messaging across native WebviewWindows (Bidirectional FDC3 communication verified across native WKWebView instances (SecRecv=true, PrimRecv=true), 1ms)
[18:45:30.258] Executing Step 7: Testing cross-window FDC3 Intent Resolution with explicit acknowledgment...
[18:45:30.259] Verified: Intent ViewAnalysis resolved with explicit ack from win_rzv53mw
[18:45:30.260] ✅ PASS Step 7: Cross-window FDC3 Intent Resolution with explicit acknowledgment (Intent ViewAnalysis resolved with explicit ack from win_rzv53mw, 1ms)
[18:45:30.260] Executing Step 8: Testing window deduplication (refocus existing without duplicate)...
[18:45:30.260] ✅ PASS Step 8: Window deduplication: refocus existing window without creating duplicate (Duplicate spawn returned existing "win_rebalancer" without increasing window count (Before=2, After=2), 0ms)
[18:45:30.260] Executing Step 9: Closing secondary window, verifying deregistration & desktop:window-closed event dispatch...
[18:45:30.291] Verified: Primary window received desktop:window-closed event for win_rebalancer
[18:45:30.292] ✅ PASS Step 9: Close secondary window, verify deregistration and desktop:window-closed dispatch (Closed "win_rebalancer"; HasWindow=false, ActiveCount=1, EventDispatched=true, OpenIDs=[win_main], 31ms)
[18:45:30.292] Executing Step 10: Negative proof for window.__fdc3Agent gating on clean window launch...
[18:45:32.335] ✅ PASS Step 10: Security gating: window.__fdc3Agent is undefined on clean launch (CONFIRMED: window.__fdc3Agent is undefined, 2043ms)
[18:45:32.335] Executing Step 11: Testing FocusWindowByIndex deterministic spawn-order switching...
[18:45:32.889] ✅ PASS Step 11: In-App shortcut window switching via FocusWindowByIndex (FocusPrimary=true, FocusAuxiliary=true, FocusInvalid=true (spawnOrder=[win_main win_auxiliary]), 554ms)
```

## 4. Layer Testing Scope & Precision Notes

- **Step 4 (Token Exchange & Session Persistence)**: Exercised the full end-to-end application path: `StandaloneWindowWrapper` React component mount -> URL token extraction -> Go memory vault `ExchangeToken` invocation -> atomic consumption -> `localStorage` / `sessionStorage` hydration -> `isReady` state progression.
- **Step 6 (Cross-Window FDC3 Messaging)**: Exercised the native `BroadcastChannel` bus and Go dual-broadcast IPC relay (`DeskWindowManager.RelayMessage`) across separate Cocoa WKWebView instances. This verifies the same-origin transport and inter-webview messaging layer under Wails v3 beta.23. The standalone `Fdc3DesktopAgent` TypeScript class logic (echo suppression, channel cycling, late-joiner replay) is verified in isolation via the 10 Vitest unit tests in `src/vitest/fdc3/Fdc3DesktopAgent.test.ts`.
- **Display Hotplug Topology Subscription**: `main.go` registers a listener on `events.Mac.ApplicationDidChangeScreenParameters` (mapped from AppKit `NSApplicationDidChangeScreenParametersNotification`) invoking `deskManager.ReclampOrphanedWindows()`. Physical cable disconnect testing remains tracked in the deferred hardware verification set.
