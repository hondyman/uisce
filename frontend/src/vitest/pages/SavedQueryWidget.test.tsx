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

// These tests pin the COMPOSITION of the two-signal guard at the widget
// call site, not just the predicate units (those live in
// savedQueryApi.test.ts). The exact failure this PR exists to kill is
// "an AVG column summed across rows shows a wrong number" - catching
// a future refactor that drops isAdditiveSafe from the conjunction, or
// "widens" it to accept avg, only registers at the widget level. The
// helpers stay unit-testable in isolation; these are the integration
// pins that hold the call-site-and-isAdditiveSafe pairing together.
//
// We assert on presence/absence of "Needs review" rather than the
// rendered number to avoid coupling to the host's toLocaleString()
// locale setting; the absence of the guard's failure text is sufficient
// proof the gate green-lit the column.
describe('SavedQueryWidget - rollup-safety gate composition', () => {
  it('renders the sum (does NOT show "Needs review") when aggregation is "sum" on a clean-grain result', async () => {
    const result: SavedQueryRunResult = {
      columns: [{ name: 'total', type: 'number', aggregation: 'sum' }],
      rows: [{ total: 100 }, { total: 200 }],
      rowCount: 2,
      chartType: 'bar',
      name: 'sum-test',
      // hasRelatedBOs: false → isRowGrainIntact passes by construction;
      // isSafeToRollUpAcrossRows short-circuits; only isAdditiveSafe
      // remains, which passes for aggregation: 'sum'. Both signals
      // green → no "Needs review."
      hasRelatedBOs: false,
    };
    mockRunSavedQuery.mockResolvedValueOnce(result);

    render(<SavedQueryWidget savedQueryId="q1" widgetType="gauge" />);

    // The h4 total (sum of 100 + 200 = 300) is what proves the guard
    // passed without depending on the host's toLocaleString() locale
    // or on the mock's module-level call count (which accumulates
    // across tests in this file). If "Needs review" had rendered
    // instead, the h4 wouldn't exist and findByText would time out.
    const totalEl = await screen.findByText(/^\d+$/, { selector: 'h4' });
    expect(totalEl).toBeInTheDocument();
    expect(screen.queryByText('Needs review')).not.toBeInTheDocument();
    expect(screen.queryByText('No data')).not.toBeInTheDocument();
  });

  it('renders "Needs review" when aggregation is "avg" on a clean-grain result - additivity signal refuses alone', async () => {
    // Central bug the PR exists to kill: per-row AVG values are
    // correct; summed across rows they are wrong. Grain/ownership are
    // pristine (hasRelatedBOs: false short-circuits both); ONLY
    // isAdditiveSafe needs to refuse, and the widget-level
    // composition must surface that refusal. A future refactor that
    // drops isAdditiveSafe (or generalizes it past 'sum') regresses
    // this exactly and fails here.
    const result: SavedQueryRunResult = {
      columns: [{ name: 'avg_col', type: 'number', aggregation: 'avg' }],
      rows: [{ avg_col: 50 }, { avg_col: 75 }],
      rowCount: 2,
      chartType: 'bar',
      name: 'avg-test',
      hasRelatedBOs: false,
    };
    mockRunSavedQuery.mockResolvedValueOnce(result);

    render(<SavedQueryWidget savedQueryId="q1" widgetType="gauge" />);

    await waitFor(() => {
      expect(screen.getByText('Needs review')).toBeInTheDocument();
    });
  });

  it('renders "Needs review" when the aggregation field is missing - single-BO wire shape, fail-safe by default', async () => {
    // Single-BO queries do not populate QueryResultColumn at all
    // today (see boresolver.QueryResultColumn's empty-columns contract
    // and the additivity gate's doc comment - wired as a deliberate
    // fail state, not a bug to paper over). Aggregation arriving as
    // undefined is the wire shape that produces the "missing
    // aggregation ⇒ unsafe" branch of isAdditiveSafe. This pins that
    // fail-safe behavior at the call site; if a future change
    // special-cases missing aggregation as "trivially safe," it has
    // to revert THIS test along with the predicate.
    const result: SavedQueryRunResult = {
      columns: [{ name: 'col', type: 'number' }], // no aggregation field
      rows: [{ col: 100 }, { col: 200 }],
      rowCount: 2,
      chartType: 'bar',
      name: 'missing-aggregation-test',
      hasRelatedBOs: false,
    };
    mockRunSavedQuery.mockResolvedValueOnce(result);

    render(<SavedQueryWidget savedQueryId="q1" widgetType="gauge" />);

    await waitFor(() => {
      expect(screen.getByText('Needs review')).toBeInTheDocument();
    });
  });

  // Pins the post-fix single-BO wire shape end-to-end: with boId now
  // populated on the saved-query preview response (backend/internal/querybuilder/service.go),
  // the gate must still
  // render "Needs review" because Aggregation stays empty under
  // omitempty (row-grain output). This is the wire-side companion to
  // TestSingleBOPreviewColumns_PopulatesBOID_LeavesAggregationEmpty
  // (which pins JSON shape); this one pins the consumer-side
  // outcome so a future "widening" of isAdditiveSafe — or a future
  // refactor that special-cases missing-Aggregation when boId is
  // set — regresses here with the failure text rather than silently.
  it('renders "Needs review" for a single-BO column with boId populated but aggregation absent', async () => {
    const result: SavedQueryRunResult = {
      columns: [
        { name: 'Order ID', type: 'number', boId: 'bo-orders' },
        { name: 'Total Amount', type: 'number', boId: 'bo-orders' },
      ],
      rows: [{ 'Order ID': 1, 'Total Amount': 100 }, { 'Order ID': 2, 'Total Amount': 200 }],
      rowCount: 2,
      chartType: 'bar',
      name: 'populated-boId-single-bo',
      hasRelatedBOs: false,
    };
    mockRunSavedQuery.mockResolvedValueOnce(result);

    render(<SavedQueryWidget savedQueryId="q1" widgetType="gauge" />);

    await waitFor(() => {
      expect(screen.getByText('Needs review')).toBeInTheDocument();
    });
  });

  // β gate-open proof: aggregation:"sum" on a single-BO column (hasRelatedBOs:false)
  // passes both isSafeToRollUpAcrossRows and isAdditiveSafe, so the gauge renders
  // the summed value instead of "Needs review". This is a gate-predicate proof with
  // a mock shape; the actual end-to-end claim (backend emits aggregation:"sum" →
  // widget renders) is proven by the combination of querybuilder wire tests
  // (TestSingleBOAggregatedMeasure_WireShape) + this test.
  it('renders the sum for an aggregated single-BO column with hasRelatedBOs:false and aggregation sum', async () => {
    const result: SavedQueryRunResult = {
      columns: [
        { name: 'Total Amount', type: 'number', boId: 'bo-orders', aggregation: 'sum' },
      ],
      rows: [{ 'Total Amount': 300 }],
      rowCount: 1,
      chartType: 'gauge',
      name: 'aggregated-single-bo',
      hasRelatedBOs: false,
    };
    mockRunSavedQuery.mockResolvedValueOnce(result);

    render(<SavedQueryWidget savedQueryId="q1" widgetType="gauge" />);

    await waitFor(() => {
      expect(screen.getByText('300')).toBeInTheDocument();
    });
    expect(screen.queryByText('Needs review')).not.toBeInTheDocument();
  });
});
