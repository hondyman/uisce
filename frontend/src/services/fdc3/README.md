# FDC3-Compatible Context Interoperability Bus

This package provides cross-window and cross-tab context synchronization using the **FINOS FDC3-compatible context model** (`fdc3.instrument`, `fdc3.portfolio`, `fdc3.order`).

It replaces commercial containers like OpenFin with an open, zero-broker, sub-millisecond architecture that functions identically in standard web browsers and Wails v3 desktop WebViews.

---

## Key Features

1. **Pluggable Transports**:
   - `BroadcastChannelTransport` (Default on macOS & Web): Direct browser-level broadcast bus without server overhead.
   - `WailsRelayTransport` (Go-Bridge Fallback): Dispatches messages via Go `DeskWindowManager.RelayMessage` and listens for Wails events.
2. **Late-Joiner Context Replay (Hydration)**:
   - When a secondary window or monitor is launched 30 seconds after the user clicked an order on Monitor 1, it automatically hydrates the channel's `lastContext` from shared `localStorage` (`fdc3_last_{channelId}`).
   - All writes are JSON-serialized write-through operations wrapped in exception-safe guards.
3. **Echo Suppression**:
   - `sourceWindowId` prevents the originating window from echoing messages back to itself.
4. **Color-Coded User Channels**:
   - Implements the 8 financial desktop user channels: `red`, `orange`, `yellow`, `green`, `blue`, `purple`, `cyan`, `pink`.

---

## Transport Selection & Windows Deferred Verification

On **macOS (WKWebView)**, Phase 0 spike verification confirmed that separate native `WebviewWindow` instances share the same origin (`wails://localhost`), `BroadcastChannel` bus, and `localStorage` partition. Therefore, `BroadcastChannelTransport` is active by default.

### Windows (WebView2) Verification (Deferred):
Verification on Windows is scheduled for Phase 6. Because Windows assigns origin `http://wails.localhost`, verify whether:
1. `BroadcastChannel` works across separate WebView2 windows.
2. `localStorage` is visible across windows under the same user-data folder.

If separate WebView2 windows require Go-mediated IPC on Windows, switch the transport via:

```typescript
import { fdc3Agent, WailsRelayTransport } from './services/fdc3';

// Force Go relay transport on Windows or unsupported platforms:
if (navigator.userAgent.includes('Windows')) {
  fdc3Agent.setTransport(new WailsRelayTransport());
}
```

---

## Usage Example

```typescript
import { fdc3Agent, Fdc3InstrumentContext } from '@/services/fdc3';

// 1. Join a channel
fdc3Agent.joinUserChannel('blue');

// 2. Broadcast context (e.g. from Order Blotter)
fdc3Agent.broadcast({
  type: 'fdc3.instrument',
  name: 'Apple Inc.',
  id: { ticker: 'AAPL', ISIN: 'US0378331005' },
});

// 3. Listen for context updates (e.g. in Market Depth or Risk view)
const unsubscribe = fdc3Agent.addContextListener<Fdc3InstrumentContext>(
  'fdc3.instrument',
  (context) => {
    console.log('Selected ticker:', context.id.ticker);
  }
);
```

---

## Architectural Rules & Boundaries

### 1. Loop-Guard Rule
- **Rule**: Incoming context updates MUST only mutate local component/panel state. They MUST NEVER trigger an outgoing context broadcast.
- **Rationale**: An echo or cascade of broadcasts across connected windows causes infinite update loops and UI stutter. Only explicit, intentional user interactions (e.g., clicking a table row, choosing an instrument from a dropdown, clicking a quick-sync button) trigger `fdc3Agent.broadcast()`.

### 2. Market Data Flow Boundary: Tick Data Never Goes Over FDC3
- **Rule**: High-frequency financial tick data, order book L2/L3 quotes, and execution fills MUST NEVER travel over the FDC3 context bus.
- **Architecture**:
  - **Market Data Flow**: High-frequency tick streams flow directly per-window via dedicated WebSocket / gRPC-web connections from the backend streaming services to each specific chart/depth/grid view.
  - **FDC3 Context Flow**: The FDC3 bus carries **only user context** (e.g. which instrument is currently focused, which order was selected, which portfolio is active).
- **Rationale**: Broadcasting ticks over FDC3 creates an IPC bottleneck where every window is spammed with every symbol's stream, degrading frame rates and starving window messaging.

---

## Known Behaviors & Edge Cases

- **Advisory Staleness & Concurrent Write-Through**:
  If two windows broadcast on the exact same channel within the same clock tick, their write-throughs to `localStorage` may interleave. Live `BroadcastChannel` delivery is unaffected (handled immediately in-memory with last-write-wins per window). This only affects the *next* late-joining window that hydrates from `localStorage`. In alignment with the FDC3-compatible architectural contract, hydrated values on initial join are advisory and immediately superseded by any live incoming message.

---

## Automation & Test Hook: `window.__fdc3Agent`

For automated end-to-end multi-window test verification (e.g. Playwright smoke tests, Wails live verification suites), the agent instance is conditionally exposed on `window.__fdc3Agent`.

- **Security & Foot-gun Gating**:
  To protect institutional trading environments and reduce malicious script exposure, this global is strictly gated. It is instantiated only when:
  - Running under development (`import.meta.env.DEV`)
  - Running on `localhost` or `127.0.0.1`
  - URL explicitly contains `?verify=1` or `?test=1`
  - Explicitly opted in via `window.__ENABLE_FDC3_AUTOMATION_HOOK__ = true`
- **Production Staging/Live**: On non-localhost enterprise production hostnames, `window.__fdc3Agent` is omitted by default.


