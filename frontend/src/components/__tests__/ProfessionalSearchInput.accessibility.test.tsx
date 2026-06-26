// @vitest-environment jsdom
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import ProfessionalSearchInput from '../ProfessionalSearchInput';
import { test, expect, vi } from 'vitest';

const DATA = [
  { id: 'alpha', text: 'Alpha', subtext: 'First' },
  { id: 'beta', text: 'Beta', subtext: 'Second' }
];

test('ProfessionalSearchInput sets aria-activedescendant on keyboard navigation', async () => {
  vi.useFakeTimers();

  const onSelect = vi.fn();
  const onSearch = vi.fn();

  render(
    <ProfessionalSearchInput
      data={DATA}
      onSelect={onSelect}
      onSearch={onSearch}
      debounceMs={200}
    />
  );

  const input = screen.getByPlaceholderText('Type to search...');
  // Type a query to open the results (debounced)
  fireEvent.change(input, { target: { value: 'A' } });
  // advance timers so debounced effect runs
  vi.advanceTimersByTime(250);

  const list = await screen.findByRole('listbox', { name: 'Search results' });
  fireEvent.keyDown(input, { key: 'ArrowDown' });

  expect(list).toHaveAttribute('aria-activedescendant');
  const active = list.getAttribute('aria-activedescendant') || '';
  expect(active).toBe('search-option-alpha');

  vi.useRealTimers();
});
