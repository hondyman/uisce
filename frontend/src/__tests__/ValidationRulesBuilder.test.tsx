import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ValidationRulesBuilderPage } from '../pages/ValidationRulesBuilderPage';
import { TenantProvider } from '../contexts/TenantContext';
import '@testing-library/jest-dom';

// Mock tenant context
const mockTenant = {
  id: 'tenant-123',
  name: 'Test Tenant',
};

const mockDatasource = {
  id: 'datasource-456',
  name: 'Test Datasource',
};

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false },
    mutations: { retry: false },
  },
});

const renderWithProviders = (component: React.ReactElement) => {
  return render(
    <QueryClientProvider client={queryClient}>
      <TenantProvider value={{ tenant: mockTenant, datasource: mockDatasource }}>
        {component}
      </TenantProvider>
    </QueryClientProvider>
  );
};

describe('ValidationRulesBuilderPage', () => {
  beforeEach(() => {
    queryClient.clear();
    global.fetch = jest.fn();
  });

  describe('Initial Load', () => {
    it('should display tenant selection message when no tenant', () => {
      render(
        <QueryClientProvider client={queryClient}>
          <TenantProvider value={{ tenant: null, datasource: null }}>
            <ValidationRulesBuilderPage />
          </TenantProvider>
        </QueryClientProvider>
      );

      expect(screen.getByText(/select a tenant and datasource/i)).toBeInTheDocument();
    });

    it('should load and display rules on mount', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: [
            {
              id: 'rule-1',
              name: 'Max Position Concentration',
              ruleType: 'position_limit',
              severity: 'BLOCK',
              isActive: true,
            },
          ],
        }),
      });

      renderWithProviders(<ValidationRulesBuilderPage />);

      await waitFor(() => {
        expect(screen.getByText('Max Position Concentration')).toBeInTheDocument();
      });
    });

    it('should display loading state', () => {
      (global.fetch as jest.Mock).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      renderWithProviders(<ValidationRulesBuilderPage />);

      expect(screen.getByText(/loading/i)).toBeInTheDocument();
    });

    it('should handle API errors gracefully', async () => {
      (global.fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      renderWithProviders(<ValidationRulesBuilderPage />);

      await waitFor(() => {
        expect(screen.getByText(/error/i)).toBeInTheDocument();
      });
    });
  });

  describe('Create New Rule', () => {
    it('should open form when New Rule button clicked', () => {
      renderWithProviders(<ValidationRulesBuilderPage />);

      const newButton = screen.getByRole('button', { name: /new rule/i });
      fireEvent.click(newButton);

      expect(screen.getByText(/create new rule/i)).toBeInTheDocument();
    });

    it('should validate required fields', async () => {
      renderWithProviders(<ValidationRulesBuilderPage />);

      fireEvent.click(screen.getByRole('button', {name: /new rule/i }));

      const saveButton = screen.getByRole('button', { name: /save rule/i });
      fireEvent.click(saveButton);

      await waitFor(() => {
        expect(screen.getByText(/name is required/i)).toBeInTheDocument();
      });
    });

    it('should create rule with valid data', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: 'new-rule-1' }),
      });

      renderWithProviders(<ValidationRulesBuilderPage />);

      fireEvent.click(screen.getByRole('button', { name: /new rule/i }));

      // Fill form
      fireEvent.change(screen.getByLabelText(/rule name/i), {
        target: { value: 'New Rule' },
      });

      fireEvent.click(screen.getByRole('button', { name: /save rule/i }));

      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/validation-rules'),
          expect.objectContaining({
            method: 'POST',
          })
        );
      });
    });
  });

  describe('Rule Type Selection', () => {
    const ruleTypes = [
      'position_limit',
      'concentration_limit',
      'sector_exposure',
      'risk_score',
      'cash_reserve',
      'rebalancing_threshold',
      'tax_loss_harvesting',
      'withdrawal_rule',
      'contribution_limit',
      'asset_allocation',
      'esg_screening',
    ];

    ruleTypes.forEach((ruleType) => {
      it(`should handle ${ruleType} rule type`, () => {
        renderWithProviders(<ValidationRulesBuilderPage />);

        fireEvent.click(screen.getByRole('button', { name: /new rule/i }));

        const select = screen.getByLabelText(/rule type/i);
        fireEvent.change(select, { target: { value: ruleType } });

        expect(select).toHaveValue(ruleType);
      });
    });
  });

  describe('Parameter Validation', () => {
    it('should validate numeric parameters', async () => {
      renderWithProviders(<ValidationRulesBuilderPage />);

      fireEvent.click(screen.getByRole('button', { name: /new rule/i }));

      const select = screen.getByLabelText(/rule type/i);
      fireEvent.change(select, { target: { value: 'position_limit' } });

      const maxInput = screen.getByLabelText(/max percentage/i);
      fireEvent.change(maxInput, { target: { value: '150' } }); // Invalid: > 100

      await waitFor(() => {
        expect(screen.getByText(/must be between 0 and 100/i)).toBeInTheDocument();
      });
    });

    it('should accept valid parameters', async () => {
      renderWithProviders(<ValidationRulesBuilderPage />);

      fireEvent.click(screen.getByRole('button', { name: /new rule/i }));

      const select = screen.getByLabelText(/rule type/i);
      fireEvent.change(select, { target: { value: 'position_limit' } });

      const maxInput = screen.getByLabelText(/max percentage/i);
      fireEvent.change(maxInput, { target: { value: '25' } });

      expect(maxInput).toHaveValue(25);
    });
  });

  describe('Edit Rule', () => {
    it('should populate form with existing rule data', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: [
            {
              id: 'rule-1',
              name: 'Existing Rule',
              ruleType: 'position_limit',
              parameters: { maxPercentage: 30 },
            },
          ],
        }),
      });

      renderWithProviders(<ValidationRulesBuilderPage />);

      await waitFor(() => {
        expect(screen.getByText('Existing Rule')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByRole('button', { name: /edit/i }));

      expect(screen.getByDisplayValue('Existing Rule')).toBeInTheDocument();
      expect(screen.getByDisplayValue('30')).toBeInTheDocument();
    });
  });

  describe('Delete Rule', () => {
    it('should delete rule after confirmation', async () => {
      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            data: [{ id: 'rule-1', name: 'Rule to Delete' }],
          }),
        })
        .mockResolvedValueOnce({ ok: true });

      renderWithProviders(<ValidationRulesBuilderPage />);

      await waitFor(() => {
        expect(screen.getByText('Rule to Delete')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByRole('button', { name: /delete/i }));

      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/validation-rules/rule-1'),
          expect.objectContaining({ method: 'DELETE' })
        );
      });
    });
  });

  describe('Search and Filter', () => {
    it('should filter rules by search term', async() => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: [
            { id: '1', name: 'Position Limit', ruleType: 'position_limit' },
            { id: '2', name: 'Cash Reserve', ruleType: 'cash_reserve' },
          ],
        }),
      });

      renderWithProviders(<ValidationRulesBuilderPage />);

      await waitFor(() => {
        expect(screen.getByText('Position Limit')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText(/search rules/i);
      fireEvent.change(searchInput, { target: { value: 'position' } });

      // Cash Reserve should be filtered out
      await waitFor(() => {
        expect(screen.queryByText('Cash Reserve')).not.toBeInTheDocument();
      });
    });

    it('should filter by rule type', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: [
            { id: '1', name: 'Rule 1', ruleType: 'position_limit' },
            { id: '2', name: 'Rule 2', ruleType: 'cash_reserve' },
          ],
        }),
      });

      renderWithProviders(<ValidationRulesBuilderPage />);

      await waitFor(() => {
        expect(screen.getByText('Rule 1')).toBeInTheDocument();
      });

      const filterSelect = screen.getByLabelText(/filter by rule type/i);
      fireEvent.change(filterSelect, { target: { value: 'position_limit' } });

      await waitFor(() => {
        expect(screen.queryByText('Rule 2')).not.toBeInTheDocument();
      });
    });
  });
});
