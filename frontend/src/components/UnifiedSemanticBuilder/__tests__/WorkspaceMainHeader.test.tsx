// React import removed (unused)
import { vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WorkspaceMain from '../WorkspaceMain';

const noop = () => {};

const baseProps: any = {
  isOver: false,
  drop: null,
  activeWorkspaceTab: 'canvas',
  setActiveWorkspaceTab: noop,
  selectedColumn: null,
  addDimension: noop,
  addMeasure: noop,
  addFilter: noop,
  getBusinessTermForColumn: () => null,
  semanticModel: { dimensions: [], measures: [], filters: [], joins: [] },
  setSemanticModel: noop,
  modelName: 'm',
  showCode: null,
  setShowCode: noop,
  rawGenerateJSON: () => '',
  rawGenerateYAML: () => '',
  selectedModel: null,
  openAddModal: noop,
  enhancedRemoveSemanticElement: noop,
  toggleElementEdit: noop,
  updateSemanticElement: noop,
  coreOptions: [],
  formatType: 'json',
  setFormatType: noop,
  rawOpen: false,
  setRawOpen: noop,
  rawFormat: 'json',
  setRawFormat: noop,
  activeEditorTab: 'custom',
  setActiveEditorTab: noop,
  generateCoreJSON: () => '',
  generateCoreYAML: () => '',
  generateCustomJSON: () => '',
  generateCustomYAML: () => '',
  generateMergedModelObject: () => ({}),
  generateJSON: () => '',
  generateYAML: () => '',
  refreshCompatibility: async () => {},
  compatLoading: false,
  issueLevelFilter: 'all',
  setIssueLevelFilter: noop,
  issueCodeFilter: '',
  setIssueCodeFilter: noop,
  compatErr: null,
  filteredCompat: [],
  expandIssues: {},
  setExpandIssues: noop,
  expandChanges: {},
  setExpandChanges: noop,
  isCodeDirty: false,
  setIsCodeDirty: noop,
  editMode: false,
  setEditMode: noop,
};

describe('WorkspaceMain header search', () => {
  test('calls setSearchTerm when search input changes and badge is interactive', async () => {
    const setSearchTerm = vi.fn();
    render(<WorkspaceMain {...baseProps} setSearchTerm={setSearchTerm} filteredCompat={[1,2,3]} />);
    const input = screen.getByPlaceholderText('Search catalog');
    const user = userEvent.setup();
  await user.type(input, 'abc');
  // userEvent types char-by-char; component is controlled so onChange receives chars — join calls
  const calls = setSearchTerm.mock.calls.map(c => String(c[0]));
  expect(calls.join('')).toBe('abc');
    // stats badge should show length of filteredCompat and be clickable
    const badge = screen.getByRole('button', { name: /Compatibility issues/i });
    expect(badge.textContent).toBe('3');
  const handler = vi.fn();
  window.addEventListener('compatibility.badge.click', handler as any);
  await user.click(badge);
  expect(handler).toHaveBeenCalled();
  // panel should appear
  expect(screen.getByTestId('compat-panel')).toBeInTheDocument();
  // close it
  await user.click(screen.getByLabelText('Close compatibility panel'));
  expect(screen.queryByTestId('compat-panel')).toBeNull();
  window.removeEventListener('compatibility.badge.click', handler as any);
  });
});
