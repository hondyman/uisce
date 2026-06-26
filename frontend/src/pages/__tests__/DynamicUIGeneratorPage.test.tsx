// @vitest-environment jsdom
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
// Note: jest-dom matchers are available via vitest setup; avoid direct import here to prevent environment issues
import { vi, describe, it, beforeEach, afterEach, expect } from 'vitest';

// Mock dnd-kit to avoid hook/context dependency during unit tests
vi.mock('@dnd-kit/core', () => {
  return {
    DndContext: ({ children }: any) => children,
    DragOverlay: ({ children }: any) => children,
    closestCenter: () => null,
    PointerSensor: () => null,
    KeyboardSensor: () => null,
    useSensor: () => null,
    useSensors: (...s: any[]) => s,
  };
});

vi.mock('@dnd-kit/sortable', () => {
  return {
    SortableContext: ({ children }: any) => children,
    arrayMove: (arr: any[], a: number, b: number) => {
      const copy = arr.slice();
      const v = copy.splice(a, 1)[0];
      copy.splice(b, 0, v);
      return copy;
    },
  useSortable: (_opts: any) => ({ attributes: {}, listeners: {}, setNodeRef: () => {}, transform: null, transition: null, isDragging: false }),
    verticalListSortingStrategy: () => null,
    sortableKeyboardCoordinates: () => null,
  };
});

vi.mock('@dnd-kit/utilities', () => ({ CSS: { Transform: { toString: () => '' } } }));

import Wrapped from '../DynamicUIGeneratorPage';

// Simple fetch mock helper
function mockFetchOnce(response: any, ok = true, status = 200) {
  (global as any).fetch = vi.fn().mockResolvedValueOnce({ ok, status, json: async () => response });
}

describe('DynamicUIGeneratorPage saved layouts', () => {
  beforeEach(() => {
    // simple localStorage mock for node test env
    const store: Record<string, string> = {};
    const mockLocalStorage = {
      getItem: (k: string) => (k in store ? store[k] : null),
      setItem: (k: string, v: string) => { store[k] = String(v); },
      removeItem: (k: string) => { delete store[k]; },
      clear: () => { for (const k in store) delete store[k]; },
    };
    vi.stubGlobal('localStorage', mockLocalStorage as any);
    vi.stubGlobal('fetch', vi.fn());
    vi.resetAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });
  it('saves locally when server unavailable', async () => {
    // Setup: no tenant in localStorage -> serverUnavailable
    render(<Wrapped />);
    const saveBtn = await screen.findByRole('button', { name: /save layout/i });
    fireEvent.click(saveBtn);
    await waitFor(() => expect(localStorage.getItem('dui_layout_v1')).toBeTruthy());
  });

  it('saves to server when tenant set', async () => {
    // seed tenant info
    localStorage.setItem('selected_tenant', JSON.stringify({ id: '00000000-0000-0000-0000-000000000000' }));
    localStorage.setItem('selected_datasource', JSON.stringify({ id: '11111111-1111-1111-1111-111111111111' }));

    // mock save response
    mockFetchOnce({ id: 'abc', name: 'My Save' });

    render(<Wrapped />);
  // open save panel by clicking Save to Server (if multiple match, use the first)
  const serverSaves = await screen.findAllByRole('button', { name: /save to server/i });
  fireEvent.click(serverSaves[0]);

    await waitFor(() => expect((global as any).fetch).toHaveBeenCalled());
    // returned data should be handled (toast shows), but we assert fetch was called
  });
});
