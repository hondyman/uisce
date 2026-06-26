/**
 * @vitest-environment jsdom
 */
 
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { within } from '@testing-library/react';

// Mock dnd-kit to avoid hook/provider issues during unit tests
vi.mock('@dnd-kit/core', () => ({
  DndContext: ({ children }: any) => <div>{children}</div>,
  DragOverlay: ({ children }: any) => <div>{children}</div>,
  closestCenter: () => {},
  useSensor: () => {},
  useSensors: () => [],
  PointerSensor: () => {},
  KeyboardSensor: () => {},
}));

vi.mock('@dnd-kit/sortable', () => ({
  SortableContext: ({ children }: any) => <div>{children}</div>,
  verticalListSortingStrategy: () => {},
  useSortable: () => ({ attributes: {}, listeners: {}, setNodeRef: () => {}, transform: null, transition: null, isDragging: false }),
  sortableKeyboardCoordinates: () => {},
  arrayMove: (arr: any[], a: number, b: number) => {
    const copy = [...arr];
    const [v] = copy.splice(a, 1);
    copy.splice(b, 0, v);
    return copy;
  }
}));

import Wrapped from '../DynamicUIGeneratorPage';

describe('DynamicUIGeneratorPage features', () => {
  beforeEach(() => {
    // reset localStorage keys used by the page
    localStorage.removeItem('dui_layout_v1');
    localStorage.removeItem('dui_default_save_to_server');
  });

  it('shows confirm modal when changing primary BO and applies change on confirm', async () => {
    render(<Wrapped />);

    // primary BO select exists
    const select = screen.getByLabelText(/Primary Business Object/i) as HTMLSelectElement;
  expect(select).toBeTruthy();

    // currently default is a BO; choose a different one
    const initial = select.value;
    // pick the next option
    const optionToPick = Array.from(select.options).find(o => o.value !== initial) as HTMLOptionElement;
    fireEvent.change(select, { target: { value: optionToPick.value } });

    // confirm modal should appear (button label Confirm)
  await waitFor(() => expect(screen.getByRole('button', { name: /Confirm/i })).toBeTruthy());

  const confirmBtn = screen.getByRole('button', { name: /Confirm/i });
    fireEvent.click(confirmBtn);

  // after confirming, the select value should update to the picked BO
  await waitFor(() => expect(select.value).toBe(optionToPick.value));
  });

  it('validates and opens ErrorSummary, and jump focuses the layout name input', async () => {
    render(<Wrapped />);

  const nameInput = screen.getAllByPlaceholderText('Layout name')[0] as HTMLInputElement;
    // clear name
    fireEvent.change(nameInput, { target: { value: '' } });

  // click Save Layout
  const saveBtn = screen.getAllByRole('button', { name: /Save Layout/i })[0];
  fireEvent.click(saveBtn);

    // ErrorSummary should show
  await waitFor(() => expect(screen.getByText(/Validation errors/i)).toBeTruthy());

    // Click the layout name error link inside the ErrorSummary dialog (should focus input)
  const layoutCandidates = screen.getAllByText(/Layout name/i);
  const layoutErrorBtn = layoutCandidates.find((el: HTMLElement) => el.closest('[role="dialog"]')) as HTMLElement;
    expect(layoutErrorBtn).toBeTruthy();
    fireEvent.click(layoutErrorBtn);

  await waitFor(() => expect(document.activeElement).toBe(nameInput));
  });

  it('opens SlideOver editor for a section and saves edits', async () => {
    render(<Wrapped />);

    // Find an Edit (settings) button for a section
    const editButtons = screen.getAllByTitle('Edit section');
    expect(editButtons.length).toBeGreaterThan(0);

    fireEvent.click(editButtons[0]);

    // SlideOver should show with 'Save' button
  await waitFor(() => expect(screen.getByPlaceholderText('Section title')).toBeTruthy());

  // change section title input (in SlideOver), which has placeholder 'Section title'
  const slTitle = screen.getByPlaceholderText('Section title') as HTMLInputElement;
    fireEvent.change(slTitle, { target: { value: 'Updated Section Title' } });

    // click Save inside SlideOver
  // locate the Save button near the section title input using within()
  const dialog = slTitle.closest('[role="dialog"]') as Element | null;
  expect(dialog).toBeTruthy();
  const saveBtn = within(dialog as Element).getByRole('button', { name: /Save/i });
  fireEvent.click(saveBtn);

    // the preview or section title input in the main UI should reflect the updated title
  await waitFor(() => expect(screen.getAllByDisplayValue('Updated Section Title').length).toBeGreaterThan(0));
  });
});
