# Hardware Verification Matrix: Multi-Monitor & Mixed-DPI Lab

This document defines the physical hardware verification protocol for Uisce Multi-Monitor Workstation on macOS (Apple Silicon).
It serves as the execution checklist for the moment an external display (USB-C/HDMI/DisplayPort) is connected.

---

## Hardware Under Test

- **Primary Display**: Apple M1 Pro Built-in Liquid Retina XDR (3024 × 1964 @ 2x Retina scale)
- **Secondary Display (Required)**: External monitor (1080p, 1440p, or 4K @ 1x or 2x scale)

---

## 5-Step Physical Verification Protocol

| Step | Test Objective | Procedure | Expected Outcome | Pass/Fail | Evidence / Notes |
| :--- | :--- | :--- | :--- | :---: | :--- |
| **1. Physical Enumeration** | `GetMonitors()` enumerates both displays accurately | Launch desktop app with external monitor plugged in. Check `window.go.main.DeskWindowManager.GetMonitors()`. | Returns array of length 2 with valid coordinates, dimensions, and distinct names/indices. | [ ] | Screenshot of devtools / status bar showing 2 displays. |
| **2. Multi-Screen Distribution** | "Distribute to Multi-Monitor" targets physical screen 1 | Click "Distribute to Multi-Monitor" in Hub toolbar. | Secondary window (e.g. `AI Portfolio Rebalancer`) spawns cleanly on the external screen. | [ ] | Photo/screenshot of dual screens with windows placed. |
| **3. Mixed-DPI Seam & Rendering** | Zero clipping/misalignment between 2x Retina and 1x External | Drag or spawn window on external monitor. Check Canvas blotter, command bar, and window borders. | Sharp typography, canvas DPR scaling matches display, zero coordinate shift or clipping at seam. | [ ] | Visual inspection of canvas text and command bar overlay. |
| **4. Physical Cable Disconnect** | Live hotplug reclamping mid-session | Unplug external display cable while secondary window is open. | macOS fires `ApplicationDidChangeScreenParameters`. Go `ReclampOrphanedWindows` runs. Secondary window smoothly relocates onto primary display. | [ ] | Verify window is visible and accessible on primary screen. |
| **5. Layout Save & Restore** | Multi-monitor desk geometry persistence across restarts | With 2 displays active, arrange windows and click "Save". Quit app, re-launch, click "Restore Desk". | Both windows re-appear in exact positions and dimensions on their respective physical screens. | [ ] | Confirm multi-screen layout persists in PostgreSQL/localStorage. |
| **6. Production-Scale Search Latency** | Database trigram index search under 100ms across 25k+ securities | In live database with >=25k securities populated, run `GET /api/instruments/search?q=<token>&limit=20` and assert server processing time. | Trigram & B-tree index scans return ranked results in <100ms p95; zero sequential scan bottlenecks. | [ ] | Query execution plan & API latency metrics. |

---

## Quick Verification Command

When an external monitor is attached, run:
```bash
cd desktop
GOWORK=off go build -o uisce-desk . && ./uisce-desk
```
Open DevTools on the primary window (`win_main`) and run:
```javascript
await window.go.main.DeskWindowManager.GetMonitors();
```
Observe the return value and proceed through the 5 steps above.
