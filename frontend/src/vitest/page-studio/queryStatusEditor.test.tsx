import React from 'react';
import { beforeAll, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import { arrange } from './schedulesScenario';

/** Live query status in the real editor: the Schedules page's grid shows what its query returned, and the App tab shows each query's state. */

const api = vi.hoisted(() => ({ list: vi.fn(), get: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn(), pause: vi.fn(), resume: vi.fn(), runNow: vi.fn(), kinds: vi.fn(), targets: vi.fn(), calendars: vi.fn(), preview: vi.fn(), runs: vi.fn(), run: vi.fn() }));
vi.mock('../../features/schedules/api', async (orig) => {
  const real = await orig<typeof import('../../features/schedules/api')>();
  return { ...real, schedulesApi: { ...real.schedulesApi, ...api } };
});

import '../../studio-core/registerDomains';
import { schedulesBlueprint } from '../../pages/page-studio/app/blueprints/schedules';

beforeAll(loadRuleEngine, 30000);

describe('live query status in the editor', () => {
  it('shows each widget\'s query on the canvas, and every query on the App tab, with their state', async () => {
    arrange(api);
    const { default: PageEditor } = await import('../../pages/page-studio/PageEditor');
    const bp = schedulesBlueprint();
    const page = { ...bp, id: 'p1', createdAt: '', updatedAt: '', tenantId: 't1' };
    render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><MemoryRouter>
      <PageEditor page={page as never} onSave={() => {}} />
    </MemoryRouter></QueryClientProvider>);

    // The schedules grid: its query returned the three schedules.
    expect(await screen.findByText('schedules · 3 rows')).toBeTruthy();
    // The kind picker reads `kinds`; the run history's query waits for its tab.
    expect((await screen.findAllByText(/^kinds · 3 rows$/)).length).toBeGreaterThan(0);

    // The App tab lists every query with its state, and what the paused one waits for.
    fireEvent.click(screen.getByRole('tab', { name: /^app$/i }));
    const runs = (await screen.findAllByText('runs · paused'))[0];
    expect(runs).toBeTruthy();
    fireEvent.mouseOver(runs);
    expect(await screen.findByText(/Its "Run only when" condition does not hold yet\./)).toBeTruthy();
    expect(within(document.body).getAllByText('schedStart · paused').length).toBeGreaterThan(0);
  }, 60000);
});
