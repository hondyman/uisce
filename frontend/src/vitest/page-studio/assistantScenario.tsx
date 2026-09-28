import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { expect, vi } from 'vitest';

/**
 * The pipeline assistant scenario shared by the parity recording and test:
 * the empty state, a starter, a reply with a proposal (changes, remaining
 * issues), Apply, and an error reply.
 */

export const START_SPEC = { version: 1, nodes: [{ id: 'file_1', type: 'file_source', label: 'Vendor file', config: { uri: 'a.csv', format: 'csv', columns: [] } }], edges: [] };
export const PROPOSED = {
  ...START_SPEC,
  nodes: [...START_SPEC.nodes, { id: 'bo_sink_2', type: 'bo_sink', label: 'Write funds', config: { bo_key: 'fund', mode: 'upsert' } }],
  edges: [{ from: 'file_1', to: 'bo_sink_2' }],
};

// jsdom has no layout: the chat scrolls to its end.
if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};

const sq = (x: string | null | undefined) => (x ?? '').replace(/[\s​]+/g, '');

/** The chat panel: the element holding the "Pipeline assistant" title. */
const panel = () => screen.getByText('Pipeline assistant').closest('.MuiPaper-root') as HTMLElement;

export async function assistantScenario(api: ReturnType<typeof vi.fn>, applied: () => unknown) {
  const out: Record<string, unknown> = {};
  await screen.findByText('Pipeline assistant');
  out.empty = sq(panel().textContent);
  api.mockResolvedValueOnce({ reply: 'Added a step that writes funds.', spec: PROPOSED, changes: ['Added Write funds', 'Connected Vendor file → Write funds'],
    issues: [{ node_id: 'bo_sink_2', message: 'Pick key fields' }] });
  fireEvent.click(screen.getByText('Load the uploaded FactSet file into the Fund business object, updating funds that already exist'));
  await within(panel()).findByText('Added a step that writes funds.');
  out.request = api.mock.calls[0];
  out.proposal = sq(panel().textContent);
  fireEvent.click(within(panel()).getByRole('button', { name: 'Apply to canvas' }));
  await within(panel()).findByRole('button', { name: 'Applied' });
  out.applied = applied();
  out.appliedButtonDisabled = (within(panel()).getByRole('button', { name: 'Applied' }) as HTMLButtonElement).disabled;
  api.mockRejectedValueOnce(new Error('API Error: 503 Service Unavailable - no LLM'));
  const input = within(panel()).getByPlaceholderText('e.g. only load funds with AUM over 1m');
  fireEvent.change(input, { target: { value: 'Why were rows rejected?' } });
  fireEvent.keyDown(input, { key: 'Enter' });
  await within(panel()).findByText(/not set up here yet/);
  out.request2 = api.mock.calls[1];
  out.error = sq(panel().textContent);
  await waitFor(() => expect(api).toHaveBeenCalledTimes(2));
  return out;
}
