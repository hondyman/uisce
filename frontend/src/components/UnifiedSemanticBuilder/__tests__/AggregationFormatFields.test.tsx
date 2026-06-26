// React import removed (automatic JSX runtime)
import { render, screen, fireEvent } from '@testing-library/react';
import AggregationFormatFields from '../AggregationFormatFields';
import { vi } from 'vitest';

describe('AggregationFormatFields', () => {
  it('renders and updates values', () => {
    const setFormData = vi.fn();
    const formData = { aggregationType: 'sum', format: '#,##0.00' };
    render(<AggregationFormatFields formData={formData} setFormData={setFormData} />);
    expect(screen.getByTitle('Aggregation type')).toHaveValue('sum');
    fireEvent.change(screen.getByTitle('Aggregation type'), { target: { value: 'avg' } });
    expect(setFormData).toHaveBeenCalled();
  });
});
