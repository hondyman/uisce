// React import removed (unused)
import { render, screen, fireEvent } from '@testing-library/react';
import TypeAndDefaultFields from '../TypeAndDefaultFields';
import { vi } from 'vitest';

describe('TypeAndDefaultFields', () => {
  it('renders type select and default value for filter', () => {
    const setFormData = vi.fn();
    const formData = { type: 'string', defaultValue: 'x' };
    render(<TypeAndDefaultFields kind={'filter'} formData={formData} setFormData={setFormData} />);
    expect(screen.getByTitle('Data type')).toBeInTheDocument();
    expect(screen.getByTitle('Default value')).toHaveValue('x');
    fireEvent.change(screen.getByTitle('Data type'), { target: { value: 'number' } });
    expect(setFormData).toHaveBeenCalled();
  });
});
