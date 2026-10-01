import React, { useState } from 'react';
import { beforeAll, describe, expect, it } from 'vitest';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import '../../studio-core/registerDomains';
import { AppRuntimeProvider } from '../../pages/page-studio/app/AppRuntime';
import AppWidgetInspector from '../../pages/page-studio/app/AppWidgetInspector';
import { masteringConsoleBlueprint } from '../../pages/page-studio/app/blueprints/masteringConsole';
import { stagingBindingsBlueprint } from '../../pages/page-studio/app/blueprints/stagingBindings';
import { dataPipelineEditorBlueprint } from '../../pages/page-studio/app/blueprints/dataPipelines';
import { PAGE_BLUEPRINTS } from '../../pages/page-studio/app/blueprints';
import { APP_WIDGET_TYPES } from '../../pages/page-studio/app/AppWidgets';
import { ColorMapEditor } from '../../pages/page-studio/app/structuredEditors';
import type { CorePageDefinition } from '../../types/pageStudio';

/**
 * The structured editors that replaced raw-JSON boxes (colour maps, row
 * buttons, action forms, mapping tables, canvas categories, generated
 * columns, row detail), exercised on the shipped blueprints - the widgets
 * that really use those shapes - and a guard that no widget's form still
 * shows a JSON box.
 */

beforeAll(loadRuleEngine, 30000);

type Draft = CorePageDefinition;
const asDraft = (bp: unknown): Draft => ({ ...(bp as object), id: 'p1', createdAt: '', updatedAt: '', tenantId: 't' }) as Draft;

function Inspector({ bp, id, expose }: { bp: unknown; id: string; expose?: (d: Draft) => void }) {
  const [draft, setDraft] = useState<Draft>(() => asDraft(bp));
  expose?.(draft);
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><MemoryRouter>
      <AppRuntimeProvider mode="design" app={draft.app}>
        <AppWidgetInspector component={draft.components[id]} draft={draft} setDraft={setDraft} />
        <pre data-testid="props">{JSON.stringify(draft.components[id].props)}</pre>
      </AppRuntimeProvider>
    </MemoryRouter></QueryClientProvider>
  );
}
const props = () => JSON.parse(screen.getByTestId('props').textContent!);

describe('colour map', () => {
  function Map({ initial }: { initial?: Record<string, string> }) {
    const [v, setV] = useState(initial);
    return <><ColorMapEditor value={v} onChange={setV} /><pre data-testid="v">{JSON.stringify(v ?? null)}</pre></>;
  }
  const val = () => JSON.parse(screen.getByTestId('v').textContent!);

  it('edits rows without losing the ones being typed, and stores only complete ones', async () => {
    render(<Map initial={{ PUBLISHED: 'success', '*': 'default' }} />);
    expect(screen.getAllByRole('textbox', { name: 'When the value is' }).map((e: HTMLElement) => (e as HTMLInputElement).value)).toEqual(['PUBLISHED', '*']);
    fireEvent.click(screen.getByRole('button', { name: 'Add colour' }));
    // A blank row is shown but not stored.
    expect(screen.getAllByRole('textbox', { name: 'When the value is' })).toHaveLength(3);
    expect(val()).toEqual({ PUBLISHED: 'success', '*': 'default' });
    fireEvent.change(screen.getAllByRole('textbox', { name: 'When the value is' })[2], { target: { value: 'REVIEW' } });
    expect(val()).toEqual({ PUBLISHED: 'success', '*': 'default', REVIEW: 'default' });
    fireEvent.mouseDown(screen.getAllByRole('combobox', { name: 'Colour' })[2]);
    fireEvent.click(await screen.findByRole('option', { name: 'warning' }));
    expect(val().REVIEW).toBe('warning');
    fireEvent.click(screen.getByRole('button', { name: 'Remove PUBLISHED' }));
    expect(val()).toEqual({ '*': 'default', REVIEW: 'warning' });
  }, 30000);

  it('becomes nothing when every row is removed', () => {
    render(<Map initial={{ A: 'error' }} />);
    fireEvent.click(screen.getByRole('button', { name: 'Remove A' }));
    expect(val()).toBeNull();
  });
});

describe('row buttons and caption, on the mastering console\'s override button', () => {
  const bp = masteringConsoleBlueprint();
  it('shows the button as a form and changes only what is edited', async () => {
    render(<Inspector bp={bp} id="gd_fields" />);
    const before = (bp.components.gd_fields.props as { columns: { id: string; cell?: { kind: string; buttons?: unknown[] } }[] }).columns.find((c) => c.id === 'act')!.cell!.buttons![0] as Record<string, unknown>;
    // Open the actions column and read the button.
    fireEvent.click(screen.getByText('act'));
    expect(await screen.findByDisplayValue(before.icon as string)).toBeTruthy();
    // The label is an i18n key with a value filled in ({field}): the key and its values each have a box.
    expect(before.label).toEqual({ t: 'mastering.overrides.action', params: { field: '{{row.attribute}}' } });
    const key = await screen.findByDisplayValue('mastering.overrides.action');
    expect(screen.getAllByDisplayValue('{{row.attribute}}').length).toBeGreaterThan(0); // the label's value (and the action's own param)
    fireEvent.change(key, { target: { value: 'mastering.overrides.edit' } });
    const cols = props().columns as { id: string; cell: { buttons: Record<string, unknown>[] } }[];
    const after = cols.find((c) => c.id === 'act')!.cell.buttons[0];
    // The label changed; everything else about the button is as it was: icon, show-when and the form-asking action.
    expect(after.label).toEqual({ t: 'mastering.overrides.edit', params: { field: '{{row.attribute}}' } });
    expect(after.icon).toBe(before.icon);
    expect(after.visibleWhen).toEqual(before.visibleWhen);
    expect(after.onClick).toEqual(before.onClick);
  }, 30000);
});

describe('action form and confirm, on the same button', () => {
  const bp = masteringConsoleBlueprint();
  it('shows the override form the action asks for, and the confirm switch off', async () => {
    render(<Inspector bp={bp} id="gd_fields" />);
    fireEvent.click(screen.getByText('act'));
    await screen.findByText('On click');
    expect(screen.getAllByRole('switch', { name: 'Ask for input first' })[0]).toHaveProperty('checked', true);
    expect(screen.getAllByRole('switch', { name: 'Ask to confirm first' })[0]).toHaveProperty('checked', false);
    // The form's fields are the ones the blueprint declares.
    const fields = (bp.components.gd_fields.props as { columns: { id: string; cell?: { buttons?: { onClick: { form: { fields: { name: string }[] } }[] }[] } }[] })
      .columns.find((c) => c.id === 'act')!.cell!.buttons![0].onClick[0].form.fields.map((f) => f.name);
    for (const name of fields) expect(screen.getAllByDisplayValue(name).length).toBeGreaterThan(0);
  }, 30000);
});

describe('generated columns and row detail, on the golden record drawer', () => {
  it('shows the matrix\'s generated columns as a form', async () => {
    const bp = masteringConsoleBlueprint();
    render(<Inspector bp={bp} id="gd_matrix" />);
    expect(screen.getByRole('switch', { name: 'Generate columns from data' })).toHaveProperty('checked', true);
    const dyn = (bp.components.gd_matrix.props as { dynamicColumns: { idField: string; valuePath: string; insertAt: number } }).dynamicColumns;
    expect(screen.getByLabelText('Field that names each column')).toHaveProperty('value', dyn.idField);
    expect(screen.getByLabelText('Row field holding each column\'s value')).toHaveProperty('value', dyn.valuePath);
    fireEvent.change(screen.getByLabelText('Min width (px)'), { target: { value: '200' } });
    expect(props().dynamicColumns).toEqual({ ...dyn, minWidth: 200 });
    fireEvent.click(screen.getByRole('switch', { name: 'Generate columns from data' }));
    expect(props().dynamicColumns).toBeUndefined();
  }, 30000);

  it('shows row detail with its nested table, and a text-only one as an alert', async () => {
    const bp = masteringConsoleBlueprint();
    render(<Inspector bp={bp} id="gd_fields" />);
    expect(screen.getByRole('switch', { name: 'An expandable detail under each row' })).toHaveProperty('checked', true);
    expect(screen.getByRole('switch', { name: 'A nested table of rows' })).toHaveProperty('checked', true);
    // Turning the table off keeps the text and drops the table's settings.
    const before = props().rowDetail;
    fireEvent.click(screen.getByRole('switch', { name: 'A nested table of rows' }));
    expect(props().rowDetail).toEqual({ text: before.text, when: before.when });
  }, 30000);
});

describe('canvas categories, on the pipeline editor', () => {
  it('edits a category\'s colour and handles, storing only what differs from the default', async () => {
    const bp = dataPipelineEditorBlueprint();
    render(<Inspector bp={bp} id="canvas" />);
    expect(screen.getAllByRole('textbox', { name: 'Category' }).map((e: HTMLElement) => (e as HTMLInputElement).value)).toEqual(['source', 'step', 'destination']);
    const before = props().categories;
    expect(before.source).toEqual({ color: 'info', inputs: false });
    // A source gets an input: the "false" goes away rather than becoming "true".
    fireEvent.click(screen.getAllByRole('switch', { name: 'Takes an input' })[0]);
    expect(props().categories.source).toEqual({ color: 'info' });
    // A destination gives no output (stored as false); turning it on removes the false, and off puts it back.
    expect(before.destination).toEqual({ color: 'success', outputs: false });
    fireEvent.click(screen.getAllByRole('switch', { name: 'Gives an output' })[2]);
    expect(props().categories.destination).toEqual({ color: 'success' });
    fireEvent.click(screen.getAllByRole('switch', { name: 'Gives an output' })[2]);
    expect(props().categories.destination).toEqual({ color: 'success', outputs: false });
  }, 30000);
});

describe('mapping table, on the staging binding editor', () => {
  it('shows its groups and add controls and keeps them when the headers change', async () => {
    const bp = stagingBindingsBlueprint();
    render(<Inspector bp={bp} id="sb_form" />);
    const fields = (bp.components.sb_form.props as { fields: { name: string; map?: unknown }[] }).fields;
    const map = fields.find((f) => f.name === 'fields')!.map as { groups: unknown[]; add: unknown[]; rows: string };
    expect(map.groups).toHaveLength(2);
    // Open the "fields" field (the mapping table) in the form's field list.
    const fieldBox = screen.getAllByDisplayValue('fields')[0].closest('.MuiPaper-root') as HTMLElement;
    expect(within(fieldBox).getByDisplayValue(map.rows)).toBeTruthy();
    expect(within(fieldBox).getAllByRole('textbox', { name: 'Key prefix' }).map((e: HTMLElement) => (e as HTMLInputElement).value)).toEqual(['id:', 'value:']);
  }, 30000);
});

describe('no widget in the app still shows a JSON box in its form', () => {
  it('on every widget of every shipped page', async () => {
    const seen: string[] = [];
    for (const b of PAGE_BLUEPRINTS) {
      const bp = b.build();
      for (const [id, c] of Object.entries(bp.components)) {
        if (!(APP_WIDGET_TYPES as readonly string[]).includes(c.type)) continue;
        const { unmount } = render(<Inspector bp={bp} id={id} />);
        if (screen.queryAllByText('JSON - applied when valid').length) seen.push(`${b.id}/${id} (${c.type})`);
        unmount();
        cleanup();
      }
    }
    expect(seen).toEqual([]);
  }, 180000);
});
