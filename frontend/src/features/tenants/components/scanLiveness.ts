/** A scan that has sent nothing for this long is treated as stalled (the server sends a heartbeat every ~5s). */
export const SCAN_STALL_AFTER_SECONDS = 30;

export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  if (h > 0) return `${h}h ${String(m).padStart(2, '0')}m`;
  if (m > 0) return `${m}m ${String(sec).padStart(2, '0')}s`;
  return `${sec}s`;
}

export interface ScanLiveness {
  elapsed: string;
  sinceLastUpdate: string;
  secondsSinceLastUpdate: number;
  stalled: boolean;
}

/**
 * How long the scan has been running and how long since the server last said anything. `stalled` is true once
 * nothing (not even a heartbeat) has arrived for SCAN_STALL_AFTER_SECONDS: slow work keeps sending heartbeats,
 * so silence means the scan or the connection is stuck.
 */
export function scanLiveness(startedAtMs: number, lastEventAtMs: number, nowMs: number): ScanLiveness {
  const since = Math.max(0, (nowMs - lastEventAtMs) / 1000);
  return {
    elapsed: formatDuration((nowMs - startedAtMs) / 1000),
    sinceLastUpdate: formatDuration(since),
    secondsSinceLastUpdate: since,
    stalled: since >= SCAN_STALL_AFTER_SECONDS,
  };
}
