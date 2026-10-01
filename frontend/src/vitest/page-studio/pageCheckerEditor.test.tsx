import React from 'react';
import { beforeAll, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';

/** The page checker in the editor: the problem count on the Check button, and Publish refused while errors remain. */

const setStatus = vi.hoisted(() => vi.fn());
vi.mock('../../api/pageStudio', async (orig) => {
  const real = await orig<typeof import('../../api/pageStudio')>();
  return { ...real, PageStudioApi: { ...real.PageStudioApi, setStatus } };
});

import '../../studio-core/registerDomains';
import { schedulesBlueprint } from '../../pages/page-studio/app/blueprints/schedules';

beforeAll(loadRuleEngine, 30000);

describe('page checker in the editor', () => {
  it('counts the problems and refuses to publish while an error remains', async () => {
    const { default: PageEditor } = await import('../../pages/page-studio/PageEditor');
    const bp = schedulesBlueprint();
    // A broken binding: the grid's empty text reads a variable nobody declared.
    (bp.components.grid.props as Record<string, unknown>).emptyText = '{{vars.oops}}';
    const page = { ...bp, id: 'p1', createdAt: '', updatedAt: '', tenantId: 't1' };
    render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><MemoryRouter>
      <PageEditor page={page as never} onSave={() => {}} />
    </MemoryRouter></QueryClientProvider>);

    const check = await screen.findByRole('button', { name: /Check/ });
    expect(check.closest('.MuiBadge-root')?.textContent).toContain('1');
    fireEvent.click(screen.getByRole('button', { name: 'Publish' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('Not published: fix the errors below first.')).toBeTruthy();
    expect(within(dialog).getByText(/reads the variable "oops", which is not declared/)).toBeTruthy();
    expect(setStatus).not.toHaveBeenCalled();
  }, 30000);
});
