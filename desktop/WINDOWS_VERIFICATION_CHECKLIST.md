# Windows (WebView2) Verification Checklist & Protocol

This document outlines the acceptance criteria, verification checklist, and contingency actions for executing Phase 6 Windows validation of the Uisce Multi-Monitor Desktop Workstation.

---

## Background & Architecture Divergence

On **macOS (WKWebView)**, Wails v3 assigns the custom origin `wails://localhost`. Multiple top-level Cocoa `WebviewWindow` instances share the same WebKit process pool, allowing browser-native `BroadcastChannel` and shared `localStorage` to work out of the box with sub-millisecond latency.

On **Windows (WebView2)**, Chromium-based WebView2 assigns the origin `http://wails.localhost`. While Chromium supports `BroadcastChannel` within the same User Data Directory (UDD), WebView2 process partitioning behavior across top-level Win32 windows requires explicit validation.

---

## Pre-Verification Requirements

1. **Environment**:
   - Windows 10 (21H2+) or Windows 11 (x64 / ARM64).
   - Microsoft Edge WebView2 Runtime installed (Evergreen).
   - Go 1.22+ and GCC (e.g. MinGW-w64 or MSYS2) for CGO compilation.
   - Dual-monitor setup with heterogeneous DPI scaling (e.g. 150% scaling on Display 1, 100% scaling on Display 2).

2. **Build**:
   ```powershell
   cd desktop
   $env:GOWORK="off"
   go build -tags verify -o uisce-desk.exe .
   ```

---

## 6-Point Verification Checklist

### 1. Same-Origin & Storage Partitioning
- [ ] Inspect window origin via DevTools console (`window.location.origin`). Confirm `http://wails.localhost`.
- [ ] In Window 1 (`/workspace`), execute: `localStorage.setItem('win_test', 'val123')`.
- [ ] In Window 2 (`/view/rebalancer`), execute: `localStorage.getItem('win_test')`.
- [ ] **Expected Result**: Window 2 returns `'val123'`.
- [ ] **Contingency**: If `localStorage` is partitioned, verify Wails v3 `application.Options.Windows` settings for shared user data directory.

### 2. Cross-Window `BroadcastChannel`
- [ ] In Window 1: `const bc = new BroadcastChannel('fdc3.channel.blue'); bc.onmessage = e => console.log('RECV:', e.data);`
- [ ] In Window 2: `const bc = new BroadcastChannel('fdc3.channel.blue'); bc.postMessage({ test: 'hello' });`
- [ ] **Pass Criteria**: Window 1 logs `RECV: { test: 'hello' }`.
- [ ] **Contingency**: If `BroadcastChannel` does not cross window boundaries in WebView2, activate the pre-built `WailsRelayTransport`:
  ```typescript
  // In frontend/src/services/fdc3/Fdc3DesktopAgent.ts:
  if (navigator.userAgent.includes('Windows')) {
    fdc3Agent.setTransport(new WailsRelayTransport());
  }
  ```

### 3. Ephemeral Single-Use Token Exchange
- [ ] Spawn secondary window with `?init_token=<UUID>`.
- [ ] Verify `window.go.main.DeskWindowManager.ExchangeToken(token)` succeeds on WebView2.
- [ ] Verify `history.replaceState` immediately strips `?init_token` without page reload.
- [ ] Verify second invocation with same token fails (`token invalid or already consumed`).

### 4. Heterogeneous Multi-Monitor Targeting & DPI Scaling
- [ ] Execute "Distribute to Multi-Monitor".
- [ ] Verify Window 1 centers on Display 1 (150% DPI).
- [ ] Verify Window 2 centers on Display 2 (100% DPI).
- [ ] Verify window dimensions (`1400x900 DIPs`) scale appropriately across monitors without blurriness or incorrect hit-testing.

### 5. Display Hotplug & Disconnect Re-Clamping
- [ ] Move secondary window to external monitor.
- [ ] Disconnect external monitor cable.
- [ ] Verify Wails triggers screen change event and executes `ReclampOrphanedWindows()`.
- [ ] Verify secondary window moves onto the primary laptop display rather than remaining off-screen.

### 6. Frameless Window Non-Client Drag Regions
- [ ] Verify custom dark window chrome (`DeskWindowFrame.tsx`).
- [ ] Verify dragging on the titlebar header (`--wails-non-client-region: drag`) moves the Win32 window smoothly.
- [ ] Verify double-clicking titlebar maximizes / restores window.
- [ ] Verify close, minimize, maximize buttons function via `window.wails.Window`.
