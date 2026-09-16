import React, { createContext, useCallback, useContext, useMemo, useState } from 'react';

/**
 * The record currently "drilled into" on a page: a master Table's row was
 * clicked, and every other widget on the page (any tab) that filters by
 * this master's records reacts to it - a child Table scoped via
 * BusinessObjectDataSourceConfig.masterFilter, or a Form showing/editing
 * this exact record. One selection per page, shared across all of a page's
 * tabs (not reset on tab switch) so a Workday-style "pick an order, then
 * work across several tabs for that one order" flow works.
 */
export interface RecordSelection {
  boId: string;
  recordId: string;
  /** Cosmetic only - lets a detail header show "Order #1234" instead of a raw id. */
  label?: string;
}

interface SelectionContextValue {
  selection: RecordSelection | null;
  select: (selection: RecordSelection) => void;
  clear: () => void;
}

const SelectionContext = createContext<SelectionContextValue>({
  selection: null,
  select: () => {},
  clear: () => {},
});

export const SelectionProvider: React.FC<{ children: React.ReactNode; initialSelection?: RecordSelection | null }> = ({ children, initialSelection }) => {
  // Seeds the page's selection from a URL-carried record id (a List page's
  // row navigated here) instead of requiring a master Table click inside
  // this page - see PageBrowser.tsx's PageContent, which remounts this
  // provider (key includes recordId) whenever the URL's record changes, so
  // this initial value is only ever read once per mount, same as useState's
  // normal initializer semantics.
  const [selection, setSelection] = useState<RecordSelection | null>(initialSelection ?? null);
  const select = useCallback((next: RecordSelection) => setSelection(next), []);
  const clear = useCallback(() => setSelection(null), []);
  const value = useMemo(() => ({ selection, select, clear }), [selection, select, clear]);
  return <SelectionContext.Provider value={value}>{children}</SelectionContext.Provider>;
};

export const useSelection = () => useContext(SelectionContext);
