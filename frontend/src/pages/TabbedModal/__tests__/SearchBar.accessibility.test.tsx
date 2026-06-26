// @vitest-environment jsdom
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import SearchBar from '../SearchBar';
import { test, expect, vi } from 'vitest';

const sampleResults = [
  { id: 'r1', type: 'table', label: 'Users', nodeId: 'n1', tableName: 'users' },
  { id: 'r2', type: 'column', label: 'email', nodeId: 'n1', tableName: 'users' }
];

test('SearchBar listbox sets aria-activedescendant when navigating with keyboard', () => {
  const onSearchChange = vi.fn();
  const onSearchSelect = vi.fn();

  render(
    <SearchBar
      searchTerm=""
      searchResults={sampleResults}
      onSearchChange={onSearchChange}
      onSearchSelect={onSearchSelect}
    />
  );

  const input = screen.getByPlaceholderText('Search tables and columns...');
  // First down arrow should highlight the first item
  fireEvent.keyDown(input, { key: 'ArrowDown' });

  const list = screen.getByRole('listbox', { name: 'Search results' });
  expect(list).toHaveAttribute('aria-activedescendant');
  const activeId = list.getAttribute('aria-activedescendant') || '';
  expect(activeId).toBe(`search-result-${sampleResults[0].id}`);
});
