/**
 * reportExecutionApi.ts
 *
 * Typed client for the Phase 3 async execution contract.
 *
 * Live-backend coverage:
 *   POST /api/v1/reports/{id}/schedules/{sid}/run → 202 Accepted / 503 DispatchError
 *   GET  /api/v1/reports/executions/{id}          → execution row
 *
 * Verified by Go integration tests in:
 *   backend/internal/api/report_handlers_test.go
 *   Commit: 81ff74506f ("feat(reporting): Phase 3 — async 202 contract, typed
 *   DispatchError, and pinned execution visibility")
 *   Named test families: TriggerScheduleRun_-_Success_returns_202...,
 *   TriggerScheduleRun_-_DispatchError_returns_503..., GetExecution_*
 */

import { apiFetch } from '@/lib/apiClient';

export type ExecutionStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'error';

export interface ExecutionRecord {
  id: string;
  tenant_id: string;
  template_id: string;
  report_key: string;
  status: ExecutionStatus;
  parameters?: Record<string, unknown>;
  output_url?: string;
  output_size_bytes?: number;
  rows_processed?: number;
  execution_time_ms?: number;
  error_message?: string;
  workflow_id?: string;
  run_id?: string;
  requested_by?: string;
  triggered_by?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  completed_at?: string;
  is_personal: boolean;
}

export type TriggerResult =
  | { kind: 'accepted'; execution_id: string; workflow_id: string }
  | { kind: 'dispatch_failed'; execution_id: string; error: string }
  | { kind: 'error'; status: number; message: string };

/**
 * POST /api/v1/reports/{id}/schedules/{sid}/run
 *
 * Uses fetch (not apiFetch) because apiFetch throws on non-2xx and we need
 * to read the 503 body for dispatch-failed executions.
 */
export async function triggerScheduleRun(
  reportId: string,
  scheduleId: string
): Promise<TriggerResult> {
  const endpoint = `/api/v1/reports/${reportId}/schedules/${scheduleId}/run`;

  let res: Response;
  try {
    res = await fetch(endpoint, {
      method: 'POST',
      credentials: 'include',
    });
  } catch (err) {
    return { kind: 'error', status: 0, message: (err as Error).message };
  }

  if (res.status === 202) {
    const body = await res.json().catch(() => ({}));
    return {
      kind: 'accepted',
      execution_id: String(body.execution_id ?? ''),
      workflow_id: String(body.workflow_id ?? ''),
    };
  }

  if (res.status === 503) {
    const body = await res.json().catch(() => ({}));
    return {
      kind: 'dispatch_failed',
      execution_id: String(body.execution_id ?? ''),
      error: String(body.error ?? 'Dispatch failed'),
    };
  }

  const body = await res.text().catch(() => '');
  return {
    kind: 'error',
    status: res.status,
    message: body.slice(0, 200) || res.statusText,
  };
}

/**
 * GET /api/v1/reports/executions/{id}
 *
 * Uses apiFetch — throws on non-2xx, which callers handle explicitly.
 */
export async function getExecution(executionId: string): Promise<ExecutionRecord> {
  const res = await apiFetch(`/api/v1/reports/executions/${executionId}`);
  return res.json() as Promise<ExecutionRecord>;
}

/**
 * Poll an execution until a terminal state is reached.
 *
 * @param executionId  the execution_id from the 202 body
 * @param signal     AbortSignal to cancel the poll (e.g. dialog unmount)
 * @param onPoll      called on each poll with the current ExecutionRecord
 * @returns the terminal ExecutionRecord
 *
 * Backoff: 1s → 2s → 5s, cap 5s, max ~60s (12 polls).
 */
export async function pollExecution(
  executionId: string,
  signal: AbortSignal,
  onPoll?: (record: ExecutionRecord) => void
): Promise<ExecutionRecord> {
  const delays = [1000, 2000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000];
  let offset = 0;

  while (true) {
    if (signal.aborted) {
      throw new DOMException('Poll aborted', 'AbortError');
    }

    const record = await getExecution(executionId);
    onPoll?.(record);

    if (isTerminal(record.status)) {
      return record;
    }

    if (offset >= delays.length) {
      return record; // return last state even if not terminal after max polls
    }

    await sleep(delays[offset]);
    offset++;
  }
}

function isTerminal(status: ExecutionStatus): boolean {
  return status === 'completed' || status === 'failed' || status === 'cancelled' || status === 'error';
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
