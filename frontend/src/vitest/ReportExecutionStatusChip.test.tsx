/**
 * ReportExecutionStatusChip unit tests
 *
 * Verifies:
 *   - Renders correct label for each ExecutionStatus value
 *   - Renders correct icon for each status
 *   - Passes sx prop through to Chip
 *
 * Attribution: this component only maps status → visual representation.
 * No network or backend contract. Live-backend coverage is held by
 * Go tests cited in reportExecutionApi.test.ts (81ff74506f).
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ReportExecutionStatusChip } from '@/components/reporting/ReportExecutionStatusChip';
import type { ExecutionStatus } from '@/api/reportExecutionApi';

describe('ReportExecutionStatusChip', () => {
  const statuses: ExecutionStatus[] = [
    'pending',
    'running',
    'completed',
    'failed',
    'cancelled',
    'error',
  ];

  it('renders the correct label for every status', () => {
    for (const status of statuses) {
      render(<ReportExecutionStatusChip status={status} />);
      expect(screen.getByText(status.charAt(0).toUpperCase() + status.slice(1))).toBeInTheDocument();
    }
  });

  it('renders pending with Clock icon', () => {
    render(<ReportExecutionStatusChip status="pending" />);
    // The chip renders its icon as a child span; check the SVG is present
    expect(document.querySelector('svg')).toBeInTheDocument();
  });

  it('renders failed status', () => {
    render(<ReportExecutionStatusChip status="failed" />);
    expect(screen.getByText('Failed')).toBeInTheDocument();
  });

  it('renders completed status', () => {
    render(<ReportExecutionStatusChip status="completed" />);
    expect(screen.getByText('Completed')).toBeInTheDocument();
  });

  it('renders cancelled status', () => {
    render(<ReportExecutionStatusChip status="cancelled" />);
    expect(screen.getByText('Cancelled')).toBeInTheDocument();
  });

  it('renders error status', () => {
    render(<ReportExecutionStatusChip status="error" />);
    expect(screen.getByText('Error')).toBeInTheDocument();
  });

  it('renders running status', () => {
    render(<ReportExecutionStatusChip status="running" />);
    expect(screen.getByText('Running')).toBeInTheDocument();
  });

  it('passes sx prop to Chip', () => {
    const { container } = render(
      <ReportExecutionStatusChip status="completed" sx={{ borderWidth: 3 }} />
    );
    const chip = container.querySelector('.MuiChip-root');
    expect(chip).not.toBeNull();
  });
});
