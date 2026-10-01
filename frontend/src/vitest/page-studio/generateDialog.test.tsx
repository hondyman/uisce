import React from 'react';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { registerOperations } from '../../studio-core/operations/registry';
import { GenerateFromOperationsDialog } from '../../pages/page-studio/app/GenerateFromOperationsDialog';
import { checkPage } from '../../pages/page-studio/app/pageChecker';

const run = async () => ({});
beforeAll(() => registerOperations([
  { id: 'dlg.list', domain: 'dlg', label: 'List widgets', kind: 'query', params: [], run, rowFields: [{ name: 'id' }, { name: 'name' }], rowsPath: 'rows' },
  { id: 'dlg.undescribed', domain: 'dlg', label: 'Mystery list', kind: 'query', params: [], run },
  { id: 'dlg.save', domain: 'dlg', label: 'Save widget', kind: 'mutation', params: [], run },
]));
afterEach(cleanup);

// MUI selects: open by role, choose by option text.
const choose = (label: RegExp, option: RegExp) => {
  fireEvent.mouseDown(screen.getByLabelText(label));
  fireEvent.click(within(screen.getByRole('listbox')).getByText(option));
};

describe('GenerateFromOperationsDialog', () => {
  it('offers only operations that describe their rows, and hands back a draft the page checker accepts', () => {
    const onGenerated = vi.fn();
    render(<GenerateFromOperationsDialog open onClose={() => undefined} onGenerated={onGenerated} />);
    expect(screen.getByRole('button', { name: 'Generate' })).toBeDisabled();
    fireEvent.change(screen.getByLabelText('Page title'), { target: { value: 'Widget catalogue' } });
    fireEvent.mouseDown(screen.getByLabelText(/Lists the rows/));
    const options = within(screen.getByRole('listbox')).getAllByRole('option').map((o) => o.textContent);
    expect(options.some((t) => /List widgets/.test(t ?? ''))).toBe(true);
    expect(options.some((t) => /Mystery list/.test(t ?? ''))).toBe(false);
    fireEvent.click(within(screen.getByRole('listbox')).getByText(/List widgets/));
    choose(/Creates a row/, /Save widget/);

    fireEvent.click(screen.getByRole('button', { name: 'Generate' }));
    expect(onGenerated).toHaveBeenCalledTimes(1);
    const draft = onGenerated.mock.calls[0][0];
    expect(draft.name).toBe('Widget catalogue');
    expect(draft.slug).toBe('widget-catalogue');
    expect(draft.status).toBe('draft');
    expect(checkPage(draft)).toEqual([]);
  });

  it('says why when the operations cannot make a page', () => {
    const onGenerated = vi.fn();
    render(<GenerateFromOperationsDialog open onClose={() => undefined} onGenerated={onGenerated} />);
    fireEvent.change(screen.getByLabelText('Page title'), { target: { value: 'Keyed oddly' } });
    fireEvent.mouseDown(screen.getByLabelText(/Lists the rows/));
    fireEvent.click(within(screen.getByRole('listbox')).getByText(/List widgets/));
    fireEvent.change(screen.getByLabelText('Row key field'), { target: { value: 'uuid' } });
    fireEvent.click(screen.getByRole('button', { name: 'Generate' }));
    expect(onGenerated).not.toHaveBeenCalled();
    expect(screen.getByRole('alert').textContent).toMatch(/no "uuid" field/);
  });
});
