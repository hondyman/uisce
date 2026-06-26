// React import removed (automatic JSX runtime)
import { render, screen, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
import AddElementOverride from '../AddElementOverride';

describe('AddElementOverride', () => {
  const mockCore = [
    { name: 'm1', title: 'Measure 1', sourceTable: 't1', description: 'd1' },
    { name: 'm2', title: 'Measure 2', sourceTable: 't2', description: 'd2' }
  ];
  it('renders list and allows selection', () => {
  const setCoreSelected = vi.fn();
  const setCoreSearch = vi.fn();
  render(<AddElementOverride filteredCore={mockCore} coreSelected={''} setCoreSelected={setCoreSelected} coreSearch={''} setCoreSearch={setCoreSearch} kind={'measure'} onBack={() => {}} onCreateOverride={() => {}} />);
    expect(screen.getByText('Measure 1')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Measure 2'));
    expect(setCoreSelected).toHaveBeenCalledWith('m2');
  });
});
