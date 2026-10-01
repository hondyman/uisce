import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';

vi.mock('../../api/pageStudio', () => ({
  PageStudioApi: { listTemplates: vi.fn(), instantiateTemplate: vi.fn(), publishTemplate: vi.fn() },
}));
import { PageStudioApi } from '../../api/pageStudio';
import { SaveAsTemplateDialog, TemplateGalleryDialog } from '../../pages/page-studio/app/TemplateDialogs';
import type { CorePageDefinition } from '../../types/pageStudio';

const api = PageStudioApi as unknown as Record<'listTemplates' | 'instantiateTemplate' | 'publishTemplate', ReturnType<typeof vi.fn>>;
const tpl = (slug: string, category: string, isCore = false) => ({ id: slug, slug, version: 2, name: `${slug} template`, description: `About ${slug}`, category, isCore, createdAt: '' });

beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);

describe('TemplateGalleryDialog', () => {
  it('lists templates, filters by category, and starts a page from the one picked', async () => {
    api.listTemplates.mockResolvedValue([tpl('orders', 'list'), tpl('kpis', 'dashboard', true)]);
    api.instantiateTemplate.mockResolvedValue({ plan: { create: [], reuse: [], conflicts: [] }, page: { name: 'My orders', slug: 'my-orders' }, applied: true });
    const onChosen = vi.fn();
    render(<TemplateGalleryDialog open onClose={() => undefined} onChosen={onChosen} />);

    await screen.findByText('orders template');
    expect(screen.getByText('Core')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'dashboard' }));
    expect(screen.queryByText('orders template')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'All' }));

    expect(screen.getByRole('button', { name: 'Start' })).toBeDisabled();
    fireEvent.click(screen.getByText('orders template'));
    fireEvent.change(screen.getByLabelText('New page name'), { target: { value: 'My orders' } });
    expect(screen.getByText('/my-orders')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Start' }));

    await waitFor(() => expect(onChosen).toHaveBeenCalledWith({ name: 'My orders', slug: 'my-orders' }));
    expect(api.instantiateTemplate).toHaveBeenCalledWith('orders', 2, { name: 'My orders', slug: 'my-orders' });
  });

  it('says so when there are none', async () => {
    api.listTemplates.mockResolvedValue([]);
    render(<TemplateGalleryDialog open onClose={() => undefined} onChosen={() => undefined} />);
    await screen.findByText(/No templates yet/);
  });

  it('shows why a start failed and does not open a draft', async () => {
    api.listTemplates.mockResolvedValue([tpl('orders', 'list')]);
    api.instantiateTemplate.mockRejectedValue(new Error('"orders-list" version 1 already exists here with different content'));
    const onChosen = vi.fn();
    render(<TemplateGalleryDialog open onClose={() => undefined} onChosen={onChosen} />);
    fireEvent.click(await screen.findByText('orders template'));
    fireEvent.click(screen.getByRole('button', { name: 'Start' }));
    expect((await screen.findByRole('alert')).textContent).toMatch(/different content/);
    expect(onChosen).not.toHaveBeenCalled();
  });

  it('reports a failed load instead of looking empty', async () => {
    api.listTemplates.mockRejectedValue(new Error('boom'));
    render(<TemplateGalleryDialog open onClose={() => undefined} onChosen={() => undefined} />);
    expect((await screen.findByRole('alert')).textContent).toBe('boom');
  });
});

describe('SaveAsTemplateDialog', () => {
  const page = { id: 'p1', name: 'Order list', description: 'All orders' } as CorePageDefinition;

  it('saves the page as a template with the name, id, description and category given', async () => {
    api.publishTemplate.mockResolvedValue(tpl('order-list', 'list'));
    const onSaved = vi.fn();
    render(<SaveAsTemplateDialog page={page} onClose={() => undefined} onSaved={onSaved} />);
    expect((screen.getByLabelText('Template id') as HTMLInputElement).value).toBe('order-list');
    fireEvent.click(screen.getByRole('button', { name: 'Save template' }));
    await waitFor(() => expect(onSaved).toHaveBeenCalled());
    expect(api.publishTemplate).toHaveBeenCalledWith({ slug: 'order-list', name: 'Order list', description: 'All orders', category: 'general', pageId: 'p1' });
  });

  it('will not save with an id the server would refuse', () => {
    render(<SaveAsTemplateDialog page={page} onClose={() => undefined} onSaved={() => undefined} />);
    fireEvent.change(screen.getByLabelText('Template id'), { target: { value: 'Bad Id' } });
    expect(screen.getByRole('button', { name: 'Save template' })).toBeDisabled();
  });

  it('is closed with no page', () => {
    render(<SaveAsTemplateDialog page={null} onClose={() => undefined} onSaved={() => undefined} />);
    expect(screen.queryByText('Save as template')).toBeNull();
  });
});
