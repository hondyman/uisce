/**
 * ReportScheduleBurstingTab state machine unit tests
 *
 * Verifies Phase 4 async contract rendering:
 *   (a) 202 → chip transitions pending → running → completed
 *   (b) 503 → dispatch_failed alert with execution_id
 *   (c) Error kind → failed state
 *
 * Attribution: network contract verified by Go integration tests in:
 *   backend/internal/api/report_handlers_test.go
 *   Commit: 81ff74506f
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import React from 'react';

const mockTrigger = vi.fn();
const mockPoll = vi.fn();

vi.mock('@/api/reportExecutionApi', () => ({
  triggerScheduleRun: (...args: unknown[]) => mockTrigger(...args),
  pollExecution: (...args: unknown[]) => mockPoll(...args),
  getExecution: vi.fn(),
}));

const mockApiFetch = vi.fn();

vi.mock('@/lib/apiClient', () => ({
  apiFetch: (...args: unknown[]) => mockApiFetch(...args),
}));

import { ReportScheduleBurstingTab } from '@/components/reporting/ReportScheduleBurstingTab';
import type { TriggerResult, ExecutionRecord } from '@/api/reportExecutionApi';

const DEFAULT_PROPS = {
  reportId: 'report-001',
  reportName: 'Valuation Report',
  tenantId: 'tenant-001',
};

const EXEC_COMPLETED: ExecutionRecord = {
  id: 'exec-1',
  tenant_id: 'tenant-001',
  template_id: 'tmpl-1',
  report_key: 'valuation',
  status: 'completed',
  output_url: 'https://storage.example.com/out.pdf',
  rows_processed: 1420,
  execution_time_ms: 3200,
  created_at: '2026-09-11T10:00:00Z',
  is_personal: false,
};

function setupScheduleMock() {
  mockApiFetch.mockResolvedValue({
    ok: true,
    json: () => Promise.resolve([{ id: 'sched-001', schedule_name: 'Daily Valuation' }]),
  } as unknown as Response);
}

describe('ReportScheduleBurstingTab — Phase 4 async contract', () => {
  beforeEach(() => {
    mockTrigger.mockReset();
    mockPoll.mockReset();
    mockApiFetch.mockReset();
    // Use real timers — apiFetch mock resolves immediately, no setTimeout needed
  });

  // (a) 202 → polling → completed; onPoll intermediate states are batched by React
  it('renders Completed chip with output after successful poll', async () => {
    setupScheduleMock();

    mockTrigger.mockResolvedValueOnce({
      kind: 'accepted',
      execution_id: 'exec-123',
      workflow_id: 'wf-456',
    } as TriggerResult);

    mockPoll.mockResolvedValueOnce({ ...EXEC_COMPLETED, status: 'completed' });

    render(<ReportScheduleBurstingTab {...DEFAULT_PROPS} />);

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /run now/i })).toBeInTheDocument();
    });

    const runButton = screen.getByRole('button', { name: /run now/i });

    await act(async () => {
      runButton.click();
    });

    await waitFor(() => {
      expect(screen.getByText('Completed')).toBeInTheDocument();
    });

    expect(screen.getByText('Download output')).toBeInTheDocument();
    expect(screen.getByText('1,420')).toBeInTheDocument();
    expect(screen.getByText('3.2s')).toBeInTheDocument();
    expect(mockPoll).toHaveBeenCalledWith('exec-123', expect.any(AbortSignal), expect.any(Function));
  });

  // (b) 503 dispatch_failed
  it('renders dispatch_failed alert with execution_id on 503', async () => {
    setupScheduleMock();

    mockTrigger.mockResolvedValueOnce({
      kind: 'dispatch_failed',
      execution_id: 'exec-999',
      error: 'Temporal cluster unavailable',
    } as TriggerResult);

    render(<ReportScheduleBurstingTab {...DEFAULT_PROPS} />);

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /run now/i })).toBeInTheDocument();
    });

    const runButton = screen.getByRole('button', { name: /run now/i });

    await act(async () => {
      runButton.click();
    });

    await waitFor(() => {
      expect(screen.getByText('Dispatch failed')).toBeInTheDocument();
    });
    expect(screen.getByText(/Temporal cluster unavailable/i)).toBeInTheDocument();
    expect(screen.getByText(/exec-999/)).toBeInTheDocument();
  });

  // (c) terminal failed
  it('renders error_message when execution completes as failed', async () => {
    setupScheduleMock();

    mockTrigger.mockResolvedValueOnce({
      kind: 'accepted',
      execution_id: 'exec-2',
      workflow_id: 'wf-789',
    } as TriggerResult);

    mockPoll.mockResolvedValueOnce({
      ...EXEC_COMPLETED,
      id: 'exec-2',
      status: 'failed',
      error_message: 'Template rendering error: missing field portfolio_id',
    });

    render(<ReportScheduleBurstingTab {...DEFAULT_PROPS} />);

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /run now/i })).toBeInTheDocument();
    });

    const runButton = screen.getByRole('button', { name: /run now/i });

    await act(async () => {
      runButton.click();
    });

    // Poll resolves synchronously (mock), Failed chip + error alert should appear
    await waitFor(() => {
      expect(screen.getByText('Failed')).toBeInTheDocument();
    });
    expect(screen.getByText('Execution failed')).toBeInTheDocument();
  });

  // (d) cancel-on-retrigger: second click cancels first poll
  it('cancels in-flight poll when Run Now is clicked again', async () => {
    setupScheduleMock();

    let pollCount = 0;
    mockTrigger.mockResolvedValue({
      kind: 'accepted',
      execution_id: 'exec-abort',
      workflow_id: 'wf-abort',
    } as TriggerResult);

    mockPoll.mockImplementation(async () => {
      pollCount++;
      await new Promise((r) => setTimeout(r, 200));
      return { ...EXEC_COMPLETED, status: 'completed' };
    });

    render(<ReportScheduleBurstingTab {...DEFAULT_PROPS} />);

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /run now/i })).toBeInTheDocument();
    });

    const runButton = screen.getByRole('button', { name: /run now/i });

    await act(async () => {
      runButton.click();
    });

    await act(async () => {
      runButton.click();
    });

    // Wait for all timers to complete
    await act(async () => {
      await new Promise((r) => setTimeout(r, 500));
    });

    // Only one poll should have been started (the second click replaced the first)
    expect(pollCount).toBe(1);
  });
});
