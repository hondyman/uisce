import { createContext, useContext, FC, ReactNode } from 'react';

export type TelemetryFn = (event: string, payload?: any) => void;

export interface TelemetryOptions {
  endpoint?: string; // optional server endpoint to POST events to
  enabled?: boolean; // opt-in flag
}

const defaultTelemetry: TelemetryFn = () => {
  // Default telemetry handler - no-op
};

const TelemetryContext = createContext<TelemetryFn>(defaultTelemetry);

export const useTelemetry = () => useContext(TelemetryContext);

// small helper that returns a telemetry function which optionally POSTs to an endpoint
export const createTelemetry = (opts?: TelemetryOptions): TelemetryFn => {
  const enabled = opts?.enabled ?? false;
  const endpoint = opts?.endpoint;
  return (event: string, payload?: any) => {
    // always safe no-op in tests/dev if disabled
    if (!enabled || !endpoint) return;
    // best-effort fire-and-forget
    try {
      void fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ event, payload, ts: Date.now() })
      }).catch(() => { /* swallow network errors */ });
    } catch (e) {
      // swallow errors
    }
  };
};

export const TelemetryProvider: FC<{ children: ReactNode; telemetry?: TelemetryFn } & ({ options?: TelemetryOptions } | {})> = ({ children, telemetry }) => {
  return (
    <TelemetryContext.Provider value={telemetry || defaultTelemetry}>
      {children}
    </TelemetryContext.Provider>
  );
};
