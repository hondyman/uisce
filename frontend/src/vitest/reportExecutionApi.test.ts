/**
 * reportExecutionApi unit tests
 *
 * Coverage:
 *   - triggerScheduleRun: 202 accepted path
 *   - triggerScheduleRun: 503 dispatch-failed path
 *   - triggerScheduleRun: non-2xx error path
 *   - getExecution: happy path
 *   - getExecution: apiFetch throws (non-2xx) — propagates as thrown ApiError
 *   - pollExecution: polls until terminal state, respects AbortSignal
 *
 * Attribution: these tests mock the network layer (fetch, apiFetch).
 * The live-backend contract is verified by Go integration tests in:
 *   backend/internal/api/report_handlers_test.go
 *   Commit: 81ff74506f
 *   Named families: TriggerScheduleRun_-_Success_returns_202...,
 *                  TriggerScheduleRun_-_DispatchError_returns_503...,
 *                  GetExecution_*
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { triggerScheduleRun, getExecution, pollExecution } from '@/api/reportExecutionApi';
import * as apiClient from '@/lib/apiClient';

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockFetch = vi.fn();
const mockApiFetch = vi.fn();

// Patch the global fetch used inside triggerScheduleRun (it calls fetch directly,
// not apiFetch, so we mock the global).
global.fetch = mockFetch as unknown as typeof fetch;

vi.spyOn(apiClient, 'apiFetch').mockImplementation(mockApiFetch as unknown as typeof apiClient.apiFetch);

describe('reportExecutionApi', () => {
  beforeEach(() => {
    mockFetch.mockReset();
    mockApiFetch.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  // -------------------------------------------------------------------------
  // triggerScheduleRun
  // -------------------------------------------------------------------------

  describe('triggerScheduleRun', () => {
    it('returns accepted result on 202', async () => {
      mockFetch.mockResolvedValueOnce({
        status: 202,
        ok: true,
        json: () => Promise.resolve({ execution_id: 'exec-123', workflow_id: 'wf-456' }),
      } as unknown as Response);

      const result = await triggerScheduleRun('report-1', 'sched-1');

      expect(result).toEqual({ kind: 'accepted', execution_id: 'exec-123', workflow_id: 'wf-456' });
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/reports/report-1/schedules/sched-1/run',
        expect.objectContaining({ method: 'POST' })
      );
    });

    it('returns dispatch_failed result on 503', async () => {
      mockFetch.mockResolvedValueOnce({
        status: 503,
        ok: false,
        json: () => Promise.resolve({ execution_id: 'exec-789', error: 'Temporal unavailable' }),
      } as unknown as Response);

      const result = await triggerScheduleRun('report-1', 'sched-1');

      expect(result).toEqual({
        kind: 'dispatch_failed',
        execution_id: 'exec-789',
        error: 'Temporal unavailable',
      });
    });

    it('returns error result on non-2xx non-503', async () => {
      mockFetch.mockResolvedValueOnce({
        status: 401,
        ok: false,
        statusText: 'Unauthorized',
        text: () => Promise.resolve(''),
      } as unknown as Response);

      const result = await triggerScheduleRun('report-1', 'sched-1');

      expect(result.kind).toBe('error');
      expect((result as { kind: 'error' }).status).toBe(401);
    });

    it('returns error on network failure', async () => {
      mockFetch.mockRejectedValueOnce(new Error('net::ERR_INTERNET_DISCONNECTED'));

      const result = await triggerScheduleRun('report-1', 'sched-1');

      expect(result.kind).toBe('error');
      expect((result as { kind: 'error' }).message).toBe('net::ERR_INTERNET_DISCONNECTED');
    });
  });

  // -------------------------------------------------------------------------
  // getExecution
  // -------------------------------------------------------------------------

  describe('getExecution', () => {
    it('returns ExecutionRecord on 200', async () => {
      const fixture: import('@/api/reportExecutionApi').ExecutionRecord = {
        id: 'exec-1',
        tenant_id: 'tenant-1',
        template_id: 'tmpl-1',
        report_key: 'valuation',
        status: 'completed',
        output_url: 'https://storage.example.com/out.pdf',
        rows_processed: 1420,
        execution_time_ms: 3200,
        created_at: '2026-09-11T10:00:00Z',
        completed_at: '2026-09-11T10:00:03Z',
        is_personal: false,
      };
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(fixture),
      } as unknown as Response);

      const record = await getExecution('exec-1');

      expect(record.status).toBe('completed');
      expect(record.output_url).toBe('https://storage.example.com/out.pdf');
      expect(record.rows_processed).toBe(1420);
    });

    it('apiFetch throws on non-2xx — caller must handle', async () => {
      mockApiFetch.mockRejectedValueOnce(new apiClient.ApiError('Unauthorized', 401, 'Unauthorized', {} as Response));

      await expect(getExecution('exec-1')).rejects.toThrow('Unauthorized');
    });
  });

  // -------------------------------------------------------------------------
  // pollExecution
  // -------------------------------------------------------------------------

  describe('pollExecution', () => {
    it('stops polling and returns record when status is completed', async () => {
      const abortController = new AbortController();

      mockApiFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ status: 'pending' }),
        } as unknown as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ status: 'running' }),
        } as unknown as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ status: 'completed', output_url: 'https://out.pdf' }),
        } as unknown as Response);

      vi.useFakeTimers();

      const pollPromise = pollExecution('exec-1', abortController.signal);
      await vi.advanceTimersByTimeAsync(100); // let promise start
      await vi.runAllTimersAsync(); // advance past all sleeps

      const record = await pollPromise;
      expect(record.status).toBe('completed');
      expect(mockApiFetch).toHaveBeenCalledTimes(3);
    });

    it('stops polling immediately when aborted', async () => {
      const abortController = new AbortController();

      mockApiFetch.mockImplementation(
        () =>
          new Promise((resolve) =>
            setTimeout(() => resolve({ ok: true, json: () => Promise.resolve({ status: 'pending' }) }), 100)
          ) as unknown as Response
      );

      const pollPromise = pollExecution('exec-1', abortController.signal);

      setTimeout(() => abortController.abort(), 50);

      await expect(pollPromise).rejects.toThrow('Poll aborted');
    });
  });
});
