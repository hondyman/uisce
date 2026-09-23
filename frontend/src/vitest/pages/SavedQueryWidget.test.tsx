import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import SavedQueryWidget from '../../pages/page-studio/SavedQueryWidget';
import type { SavedQueryRunResult } from '../../features/query-builder/services/savedQueryApi';

// Pins the guard two prior findings in this review round rested on
// (retracted "gate unreachable for empty rows," and the request to
// confirm it isn't just true by inspection): result.rows.length === 0
// must render "No data" and return before the gauge branch ever reaches
// result.rows[0][c.name] - which would throw on an actually-empty rows
// array. This renders the REAL component, not just the exported helper
// functions, so a future refactor that moves or removes that guard would
// fail this test by actually crashing during render.

const { mockRunSavedQuery } = vi.hoisted(() => ({ mockRunSavedQuery: vi.fn() }));
vi.mock('../../features/query-builder/services/savedQueryApi', async () => {
  const actual = await vi.importActual<typeof import('../../features/query-builder/services/savedQueryApi')>(
    '../../features/query-builder/services/savedQueryApi',
  );
  return { ...actual, runSavedQuery: mockRunSavedQuery };
});

describe('SavedQueryWidget - empty result guard', () => {
  it('renders "No data" for a gauge widget when the saved query returns zero rows, without throwing', async () => {
    const emptyResult: SavedQueryRunResult = {
      columns: [{ name: 'total', type: 'number' }],
      rows: [],
      rowCount: 0,
      chartType: 'bar',
      name: 'empty-query',
    };
    mockRunSavedQuery.mockResolvedValueOnce(emptyResult);

    render(<SavedQueryWidget savedQueryId="q1" widgetType="gauge" />);

    await waitFor(() => {
      expect(screen.getByText('No data')).toBeInTheDocument();
    });
    // If the gauge branch's result.rows[0] access were ever reached with
    // an empty rows array, render() above would have thrown before this
    // line - so reaching it at all is part of the proof.
    expect(screen.queryByText('Needs review')).not.toBeInTheDocument();
  });
});
