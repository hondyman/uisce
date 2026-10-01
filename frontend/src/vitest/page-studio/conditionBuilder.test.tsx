import React, { useState } from 'react';
import { beforeAll, describe, expect, it } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import { AppRuntimeProvider } from '../../pages/page-studio/app/AppRuntime';
import { ConditionEditor } from '../../pages/page-studio/app/editors';
import type { ConditionNode } from '../../pages/page-studio/app/appModel';

/**
 * The condition builder, used as a designer would: pick a field, get the
 * operators that fit its kind, fill the value the operator takes, nest
 * groups, and see problems and the sentence it reads as.
 */

beforeAll(loadRuleEngine, 30000);

const PATHS = ['vars.count', 'vars.flag', 'vars.name', 'vars.tab', 'row.code'];

function Harness({ initial }: { initial?: ConditionNode }) {
  const [value, setValue] = useState<ConditionNode | undefined>(initial);
  return (
    <QueryClientProvider client={new QueryClient()}><MemoryRouter>
      <AppRuntimeProvider mode="design" app={{ variables: [
        { name: 'count', default: 5 }, { name: 'flag', default: true }, { name: 'name', default: 'abc' }, { name: 'tab', default: 'runs' },
      ] }}>
        <ConditionEditor label="Show when" value={value} onChange={setValue} paths={PATHS} />
        <pre data-testid="stored">{JSON.stringify(value ?? null)}</pre>
      </AppRuntimeProvider>
    </MemoryRouter></QueryClientProvider>
  );
}
const stored = () => JSON.parse(screen.getByTestId('stored').textContent!);
const setField = (path: string) => fireEvent.change(screen.getAllByRole('combobox', { name: 'Check' })[0], { target: { value: path } });
async function operators() {
  fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Is' }));
  const list = await screen.findByRole('listbox');
  return { list, names: within(list).getAllByRole('option').map((o: HTMLElement) => o.textContent) };
}
const choose = async (label: string) => {
  const { list } = await operators();
  fireEvent.click(within(list).getByRole('option', { name: label }));
};

describe('building a condition', () => {
  it('starts as Always and stores a single condition plainly (not a group)', async () => {
    render(<Harness />);
    expect(screen.getByText('Always')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Add condition' }));
    expect(stored()).toEqual({ type: 'condition', field: 'vars.count', operator: 'is_not_empty' });
    expect(await screen.findByText('When vars.count is not empty')).toBeTruthy();
    // It reads page data that is there now, so it can say whether it holds.
    expect(await screen.findByText('true now')).toBeTruthy();
  }, 30000);

  it('offers number operators for a number field, and takes a range as two numbers', async () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('button', { name: 'Add condition' }));
    setField('vars.count');
    const { names, list } = await operators();
    expect(names).toContain('is between');
    expect(names).toContain('is at least');
    expect(names).not.toContain('starts with');
    expect(names).not.toContain('is true');
    fireEvent.click(within(list).getByRole('option', { name: 'is between' }));
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '1' } });
    fireEvent.change(screen.getByLabelText('To'), { target: { value: '9' } });
    expect(stored()).toEqual({ type: 'condition', field: 'vars.count', operator: 'between', value: 1, secondValue: 9 });
    expect(await screen.findByText('When vars.count is between 1 and 9')).toBeTruthy();
    await waitFor(() => expect(screen.getByText('true now')).toBeTruthy());
    fireEvent.change(screen.getByLabelText('To'), { target: { value: '3' } });
    await waitFor(() => expect(screen.getByText('false now')).toBeTruthy());
  }, 30000);

  it('offers truth tests for a yes/no field with no value box, and true/false for equals', async () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('button', { name: 'Add condition' }));
    setField('vars.flag');
    expect((await operators()).names).toEqual(['equals', 'does not equal', 'is empty', 'is not empty', 'is not set', 'is set', 'is true', 'is false']);
    fireEvent.click(screen.getByRole('option', { name: 'is true' }));
    expect(screen.queryByLabelText('Value')).toBeNull();
    expect(stored()).toEqual({ type: 'condition', field: 'vars.flag', operator: 'is_true' });
    await choose('equals');
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Value' }));
    fireEvent.click(await screen.findByRole('option', { name: 'false' }));
    expect(stored()).toEqual({ type: 'condition', field: 'vars.flag', operator: 'equals', value: false });
  }, 30000);

  it('keeps text as text, and takes a list as chips', async () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole('button', { name: 'Add condition' }));
    setField('vars.name');
    await choose('equals');
    fireEvent.change(screen.getByLabelText('Value'), { target: { value: '007' } });
    expect(stored().value).toBe('007');
    await choose('is one of');
    const values = screen.getByRole('combobox', { name: 'Values' });
    fireEvent.change(values, { target: { value: 'abc' } });
    fireEvent.keyDown(values, { key: 'Enter' });
    fireEvent.change(values, { target: { value: 'xyz' } });
    fireEvent.keyDown(values, { key: 'Enter' });
    expect(stored()).toMatchObject({ operator: 'in', value: ['abc', 'xyz'] });
  }, 30000);

  it('nests groups and switches between all and any', async () => {
    render(<Harness initial={{ type: 'condition', field: 'vars.tab', operator: 'equals', value: 'runs' }} />);
    fireEvent.click(screen.getByRole('button', { name: 'Add group' }));
    expect(stored()).toMatchObject({ type: 'group', operator: 'AND' });
    expect(stored().conditions[1]).toMatchObject({ type: 'group', operator: 'OR' });
    expect(await screen.findByText(/When vars\.tab equals runs and /)).toBeTruthy();
    fireEvent.click(screen.getAllByRole('button', { name: 'any' })[0]);
    expect(stored().operator).toBe('OR');
    // Removing the nested group puts it back to the one condition.
    fireEvent.click(screen.getByRole('button', { name: 'Remove group' }));
    expect(stored()).toEqual({ type: 'condition', field: 'vars.tab', operator: 'equals', value: 'runs' });
  }, 30000);
});

describe('problems are shown beside the condition', () => {
  it('flags a variable the page does not have, a missing value and a poor fit', async () => {
    const { rerender } = render(<Harness initial={{ type: 'condition', field: 'vars.nope', operator: 'equals' }} />);
    expect(await screen.findByText('The page has no variable "nope".')).toBeTruthy();
    expect(screen.getByText('"equals" needs a value.')).toBeTruthy();
    rerender(<Harness initial={{ type: 'condition', field: 'vars.name', operator: 'greater_than', value: 3 }} key="fit" />);
    expect(await screen.findByText(/"is more than" is meant for number fields, and this one is text\./)).toBeTruthy();
  }, 30000);

  it('keeps an operator the engine lacks visible and says so', async () => {
    render(<Harness initial={{ type: 'condition', field: 'vars.name', operator: 'sounds_like', value: 'x' }} />);
    expect(await screen.findByText('The rule engine has no operator "sounds_like".')).toBeTruthy();
    expect(screen.getByRole('combobox', { name: 'Is' }).textContent).toContain('sounds_like (unknown)');
  }, 30000);
});
