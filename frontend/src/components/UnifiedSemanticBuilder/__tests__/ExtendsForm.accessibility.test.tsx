// @vitest-environment jsdom
import { render, screen, fireEvent } from '@testing-library/react';
import ExtendsForm, { BaseModelOption } from '../ExtendsForm';
import { test, expect, vi } from 'vitest';

const options: BaseModelOption[] = [
  { key: '/base/a', label: 'Base A', kind: 'core' },
  { key: '/base/b', label: 'Base B', kind: 'custom' },
];

test('ExtendsForm exposes ARIA combobox semantics and can be disabled', () => {
  const onChange = vi.fn();
  render(
    <ExtendsForm
      currentBase={null}
      options={options}
      disabled={true}
      onChange={onChange}
    />
  );

  const input = screen.getByTestId('extends-typeahead-input');
  expect(input).toHaveAttribute('role', 'combobox');
  expect(input).toHaveAttribute('aria-autocomplete', 'list');
  expect(input).toHaveAttribute('aria-expanded', 'false');
  expect(input).toHaveAttribute('aria-controls');
  expect(input).toBeDisabled();
});

test('ExtendsForm opens list and sets activedescendant', () => {
  const onChange = vi.fn();
  render(
    <ExtendsForm
      currentBase={null}
      options={options}
      disabled={false}
      onChange={onChange}
    />
  );

  const input = screen.getByTestId('extends-typeahead-input');
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: 'Base' } });
  fireEvent.keyDown(input, { key: 'ArrowDown' });

  expect(input).toHaveAttribute('aria-expanded', 'true');
  expect(input.getAttribute('aria-activedescendant')).toMatch(/^extends-option-/);
});
