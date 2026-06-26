// React import removed (not needed with the new JSX transform)
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { vi, describe, it, expect } from 'vitest';
import { MantineProvider } from '@mantine/core';
import axios from 'axios';
import ScanResultsModal from '../../components/ScanResultsModal';

vi.mock('axios');

const mockAxios = axios as unknown as { post: any };
// jsdom (used by Vitest) doesn't implement matchMedia; Mantine calls it. Provide a lightweight polyfill.
if (typeof window !== 'undefined' && !window.matchMedia) {
  (window as any).matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  });
}

describe('ScanResultsModal', () => {
  it('renders results and retries a failed datasource', async () => {
    const results = [
      { tenant_instance_id: 'ds1', name: 'DS One', success: true },
      { tenant_instance_id: 'ds2', name: 'DS Two', success: false, error: 'boom' },
    ];

    const onRetry = vi.fn(async (_id) => {
      // simulate axios returning updated results for the retried id
      mockAxios.post = vi.fn().mockResolvedValueOnce({ data: { results: [ { tenant_instance_id: 'ds1', success: true }, { tenant_instance_id: 'ds2', success: true } ] } });
    });

    render(
      <MantineProvider theme={{}}>
        <ScanResultsModal opened={true} onClose={() => {}} results={results as any} onRetry={onRetry as any} />
      </MantineProvider>
    );

    expect(screen.getByText('DS One')).toBeTruthy();
    expect(screen.getByText('DS Two')).toBeTruthy();
    expect(screen.getByText('boom')).toBeTruthy();

    const retryBtn = screen.getByText('Retry');
    fireEvent.click(retryBtn);

    await waitFor(() => expect(onRetry).toHaveBeenCalledWith('ds2'));
  });
});
