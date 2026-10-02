import { render, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';

// apiClient is a plain function (not an object with .get/.post/.delete) that
// already parses JSON responses. These managers used to call apiClient.get(...)
// and res.json(), which threw a TypeError on mount.
const apiClient = vi.fn();
vi.mock('../../../utils/apiClient', () => ({ apiClient: (...a: unknown[]) => apiClient(...a) }));

import { SoDRulesManager } from '../../../components/Compliance/SoDRulesManager';
import { TAMMatrixManager } from '../../../components/Compliance/TAMMatrixManager';
import { GSIFIEventRegistry } from '../../../components/Compliance/GSIFIEventRegistry';

const cases: Array<[string, () => JSX.Element, string]> = [
  ['SoDRulesManager', () => <SoDRulesManager />, '/api/compliance/gsifi/sod'],
  ['TAMMatrixManager', () => <TAMMatrixManager />, '/api/compliance/gsifi/tam'],
  ['GSIFIEventRegistry', () => <GSIFIEventRegistry />, '/api/compliance/gsifi/events'],
];

describe.each(cases)('%s', (_name, ui, url) => {
  beforeEach(() => {
    apiClient.mockReset();
  });

  it('loads its list by calling apiClient as a function', async () => {
    apiClient.mockResolvedValue([]);
    render(ui());
    await waitFor(() => expect(apiClient).toHaveBeenCalledWith(url));
  });

  it('tolerates a non-array response without throwing', async () => {
    apiClient.mockResolvedValue({ unexpected: true });
    const { container } = render(ui());
    await waitFor(() => expect(apiClient).toHaveBeenCalled());
    expect(container).toBeTruthy();
  });

  it('survives a failed load', async () => {
    // mockImplementation, not mockRejectedValue: in this vitest version the latter
    // surfaces as an unhandled rejection before the component can catch it.
    apiClient.mockImplementation(() => Promise.reject(new Error('boom')));
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const { container } = render(ui());
    await waitFor(() => expect(spy).toHaveBeenCalled());
    expect(container).toBeTruthy();
    spy.mockRestore();
  });
});
