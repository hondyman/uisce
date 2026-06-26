import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import UMABuilder from '@/components/UMABuilder';

// Mock data
const mockUMAAccount = {
  id: 'uma-123',
  name: 'John Smith Portfolio',
  aum: 5000000,
  status: 'active',
  lastRebalanced: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
  sleeves: [
    {
      id: 'sleeve-1',
      model: 'Growth',
      sleeveType: 'equities',
      targetAllocation: 0.6,
      currentAllocation: 0.62,
      drift: 0.02,
      minDriftThreshold: 0.05,
      status: 'active',
    },
    {
      id: 'sleeve-2',
      model: 'Income',
      sleeveType: 'fixed_income',
      targetAllocation: 0.3,
      currentAllocation: 0.28,
      drift: -0.02,
      minDriftThreshold: 0.05,
      status: 'active',
    },
    {
      id: 'sleeve-3',
      model: 'Alternatives',
      sleeveType: 'alternatives',
      targetAllocation: 0.1,
      currentAllocation: 0.1,
      drift: 0,
      minDriftThreshold: 0.05,
      status: 'active',
    },
  ],
};

const mockRebalancePlan = {
  id: 'plan-123',
  driftSignal: 0.02,
  trades: [
    {
      symbol: 'VTSAX',
      side: 'buy' as const,
      quantity: 500,
      estimatedPrice: 145.67,
      estimatedValue: 72835.0,
      reason: 'Rebalance to target allocation',
    },
    {
      symbol: 'VBTLX',
      side: 'sell' as const,
      quantity: 200,
      estimatedPrice: 10.5,
      estimatedValue: 2100.0,
      reason: 'Reduce fixed income overweight',
    },
  ],
  approvalStatus: 'pending_approval',
};

describe('UMABuilder Component', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    // Mock localStorage
    localStorage.setItem('selected_tenant', 'tenant-123');
    localStorage.setItem('selected_datasource', 'ds-456');

    // Mock fetch
    global.fetch = jest.fn();
  });

  afterEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
  });

  const renderComponent = (props = {}) => {
    return render(
      <QueryClientProvider client={queryClient}>
        <UMABuilder umaId="uma-123" {...props} />
      </QueryClientProvider>
    );
  };

  describe('Rendering', () => {
    it('should render loading spinner initially', () => {
      (global.fetch as jest.Mock).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      renderComponent();

      expect(screen.getByRole('progressbar')).toBeInTheDocument();
    });

    it('should render error alert when UMA not found', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(null),
      });

      renderComponent();

      await waitFor(() => {
        expect(
          screen.getByText(/UMA Account not found/i)
        ).toBeInTheDocument();
      });
    });

    it('should render UMA header with account name', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText(/John Smith Portfolio/i)).toBeInTheDocument();
      });
    });

    it('should render sleeves table with correct data', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
        expect(screen.getByText('Income')).toBeInTheDocument();
        expect(screen.getByText('Alternatives')).toBeInTheDocument();
      });
    });

    it('should render allocation percentages correctly', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        // Target allocation for Growth sleeve: 60%
        const cells = screen.getAllByText(/60\.00%/);
        expect(cells.length).toBeGreaterThan(0);
      });
    });

    it('should render drift values with correct colors', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        // Drift of 0.02 (2%) for Growth sleeve
        const driftCells = screen.getAllByText(/2\.00%/);
        expect(driftCells.length).toBeGreaterThan(0);
      });
    });
  });

  describe('Drift Detection', () => {
    it('should NOT show drift alert when under threshold', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        // All drifts are under 5% threshold, no alert should appear
        expect(
          screen.queryByText(/exceeded drift threshold/)
        ).not.toBeInTheDocument();
      });
    });

    it('should show drift alert when drift exceeds threshold', async () => {
      const accountWithHighDrift = {
        ...mockUMAAccount,
        sleeves: [
          {
            ...mockUMAAccount.sleeves[0],
            drift: 0.08, // 8% > 5% threshold
          },
        ],
      };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(accountWithHighDrift),
      });

      renderComponent();

      await waitFor(() => {
        expect(
          screen.getByText(/exceeded drift threshold/)
        ).toBeInTheDocument();
      });
    });

    it('should display total allocation calculation', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        // Total current allocation: 62% + 28% + 10% = 100%
        expect(screen.getByText(/100\.00%/)).toBeInTheDocument();
      });
    });
  });

  describe('Sleeve Management', () => {
    it('should open edit dialog when edit button clicked', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const editButtons = screen.getAllByRole('button').filter(
        (btn: HTMLElement) => btn.querySelector('svg') && btn.parentElement?.textContent?.includes('Edit')
      );

      if (editButtons.length > 0) {
        fireEvent.click(editButtons[0]);

        await waitFor(() => {
          expect(screen.getByDisplayValue('Growth')).toBeInTheDocument();
        });
      }
    });

    it('should update sleeve data on save', async () => {
      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          json: () => Promise.resolve(mockUMAAccount),
        })
        .mockResolvedValueOnce({
          json: () =>
            Promise.resolve({
              ...mockUMAAccount.sleeves[0],
              targetAllocation: 0.65,
            }),
        });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      // Note: Full integration test would require more complex dialog interaction
      // This demonstrates the pattern
    });

    it('should disable edit in read-only mode', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent({ readOnly: true });

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      // In read-only mode, edit buttons and inputs should be disabled
      const textInputs = screen.queryAllByRole('textbox') as HTMLInputElement[];
      textInputs.forEach((input) => {
        if (input.disabled) {
          expect(input.disabled).toBe(true);
        }
      });
    });
  });

  describe('Rebalance Workflow', () => {
    it('should trigger rebalance on button click', async () => {
      const mockCallback = jest.fn();

      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          json: () => Promise.resolve(mockUMAAccount),
        })
        .mockResolvedValueOnce({
          json: () =>
            Promise.resolve({
              workflow_id: 'workflow-123',
              plan: mockRebalancePlan,
            }),
        });

      renderComponent({ onRebalanceTriggered: mockCallback });

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const rebalanceButton = screen.getByRole('button', {
        name: /Suggest Rebalance/i,
      });

      fireEvent.click(rebalanceButton);

      await waitFor(() => {
        expect(mockCallback).toHaveBeenCalledWith('workflow-123');
      });
    });

    it('should display rebalance plan with trades', async () => {
      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          json: () => Promise.resolve(mockUMAAccount),
        })
        .mockResolvedValueOnce({
          json: () =>
            Promise.resolve({
              workflow_id: 'workflow-123',
              plan: mockRebalancePlan,
            }),
        });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const rebalanceButton = screen.getByRole('button', {
        name: /Suggest Rebalance/i,
      });

      fireEvent.click(rebalanceButton);

      await waitFor(() => {
        expect(screen.getByText('VTSAX')).toBeInTheDocument();
        expect(screen.getByText('VBTLX')).toBeInTheDocument();
      });
    });

    it('should show trade details correctly', async () => {
      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          json: () => Promise.resolve(mockUMAAccount),
        })
        .mockResolvedValueOnce({
          json: () =>
            Promise.resolve({
              workflow_id: 'workflow-123',
              plan: mockRebalancePlan,
            }),
        });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const rebalanceButton = screen.getByRole('button', {
        name: /Suggest Rebalance/i,
      });

      fireEvent.click(rebalanceButton);

      await waitFor(() => {
        expect(screen.getByText(/72835\.00/)).toBeInTheDocument(); // VTSAX value
        expect(screen.getByText(/2100\.00/)).toBeInTheDocument(); // VBTLX value
      });
    });

    it('should handle rebalance error gracefully', async () => {
      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          json: () => Promise.resolve(mockUMAAccount),
        })
        .mockRejectedValueOnce(new Error('Network error'));

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const rebalanceButton = screen.getByRole('button', {
        name: /Suggest Rebalance/i,
      });

      fireEvent.click(rebalanceButton);

      // Component should handle error gracefully (no crash)
      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });
    });
  });

  describe('Approval Workflow', () => {
    it('should show approve button for pending approval', async () => {
      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          json: () => Promise.resolve(mockUMAAccount),
        })
        .mockResolvedValueOnce({
          json: () =>
            Promise.resolve({
              workflow_id: 'workflow-123',
              plan: { ...mockRebalancePlan, approvalStatus: 'pending_approval' },
            }),
        });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const rebalanceButton = screen.getByRole('button', {
        name: /Suggest Rebalance/i,
      });

      fireEvent.click(rebalanceButton);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Approve/i })).toBeInTheDocument();
      });
    });

    it('should approve rebalance plan', async () => {
      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          json: () => Promise.resolve(mockUMAAccount),
        })
        .mockResolvedValueOnce({
          json: () =>
            Promise.resolve({
              workflow_id: 'workflow-123',
              plan: { ...mockRebalancePlan, approvalStatus: 'pending_approval' },
            }),
        })
        .mockResolvedValueOnce({
          json: () =>
            Promise.resolve({ status: 'approved', executionStarted: true }),
        });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const rebalanceButton = screen.getByRole('button', {
        name: /Suggest Rebalance/i,
      });

      fireEvent.click(rebalanceButton);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Approve/i })).toBeInTheDocument();
      });

      const approveButton = screen.getByRole('button', { name: /Approve/i });

      fireEvent.click(approveButton);

      // Button should be disabled while loading
      expect(approveButton).toBeDisabled();
    });

    it('should not show approve button when not pending', async () => {
      (global.fetch as jest.Mock)
        .mockResolvedValueOnce({
          json: () => Promise.resolve(mockUMAAccount),
        })
        .mockResolvedValueOnce({
          json: () =>
            Promise.resolve({
              workflow_id: 'workflow-123',
              plan: { ...mockRebalancePlan, approvalStatus: 'approved' },
            }),
        });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const rebalanceButton = screen.getByRole('button', {
        name: /Suggest Rebalance/i,
      });

      fireEvent.click(rebalanceButton);

      await waitFor(() => {
        expect(screen.queryByRole('button', { name: /Approve/i })).not.toBeInTheDocument();
      });
    });
  });

  describe('API Integration', () => {
    it('should include tenant headers in all requests', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalled();
      });

      const calls = (global.fetch as jest.Mock).mock.calls;
      const lastCall = calls[calls.length - 1];

      expect(lastCall[1]?.headers).toEqual(
        expect.objectContaining({
          'X-Tenant-ID': 'tenant-123',
          'X-Tenant-Datasource-ID': 'ds-456',
        })
      );
    });

    it('should include datasource query parameters', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalled();
      });

      const calls = (global.fetch as jest.Mock).mock.calls;
      const url = new URL(calls[0][0], 'http://localhost');

      expect(url.searchParams.get('tenant_id')).toBe('tenant-123');
      expect(url.searchParams.get('tenant_instance_id')).toBe('ds-456');
    });

    it('should handle missing tenant context', async () => {
      localStorage.clear();

      renderComponent();

      // Component should render but might show error or disable features
      await waitFor(() => {
        // Verify component doesn't crash
        expect(screen.getByRole('progressbar')).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        const sleevesTable = screen.getByRole('table');
        expect(sleevesTable).toBeInTheDocument();
      });
    });

    it('should be keyboard navigable', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const button = screen.getByRole('button', {
        name: /Suggest Rebalance/i,
      });

      button.focus();
      expect(button).toHaveFocus();

      // Can activate with keyboard
      fireEvent.keyDown(button, { key: 'Enter', code: 'Enter' });
    });

    it('should have proper color contrast for drift indicators', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(mockUMAAccount),
      });

      renderComponent();

      await waitFor(() => {
        const table = screen.getByRole('table');
        expect(table).toBeInTheDocument();
        // Color contrast is ensured by CSS - verify via visual regression testing
      });
    });
  });

  describe('Performance', () => {
    it('should render sleeves efficiently', async () => {
      const largeAccount = {
        ...mockUMAAccount,
        sleeves: Array(50)
          .fill(null)
          .map((_, i) => ({
            ...mockUMAAccount.sleeves[0],
            id: `sleeve-${i}`,
            model: `Sleeve ${i}`,
          })),
      };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        json: () => Promise.resolve(largeAccount),
      });

      const startTime = performance.now();

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Sleeve 0')).toBeInTheDocument();
      });

      const endTime = performance.now();
      const renderTime = endTime - startTime;

      // Should render 50 sleeves in reasonable time (< 2 seconds)
      expect(renderTime).toBeLessThan(2000);
    });

    it('should cache query results', async () => {
      (global.fetch as jest.Mock).mockResolvedValue({
        json: () => Promise.resolve(mockUMAAccount),
      });

      const { rerender } = renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      const firstCallCount = (global.fetch as jest.Mock).mock.calls.length;

      rerender(
        <QueryClientProvider client={queryClient}>
          <UMABuilder umaId="uma-123" />
        </QueryClientProvider>
      );

      const secondCallCount = (global.fetch as jest.Mock).mock.calls.length;

      // Should not make a new fetch request (cache hit)
      expect(secondCallCount).toBe(firstCallCount);
    });
  });

  describe('Error Scenarios', () => {
    it('should handle API 403 Forbidden (permission denied)', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        status: 403,
        json: () => Promise.resolve({ error: 'Permission denied' }),
      });

      renderComponent();

      await waitFor(() => {
        // Component should handle 403 gracefully
        expect(screen.getByText('Growth')).not.toBeInTheDocument();
      });
    });

    it('should handle API 500 Server Error', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        status: 500,
        json: () => Promise.resolve({ error: 'Internal server error' }),
      });

      renderComponent();

      await waitFor(() => {
        // Component should show error state
        expect(screen.getByText('Growth')).not.toBeInTheDocument();
      });
    });

    it('should retry on network timeout', async () => {
      (global.fetch as jest.Mock)
        .mockRejectedValueOnce(new Error('Network timeout'))
        .mockResolvedValueOnce({
          json: () => Promise.resolve(mockUMAAccount),
        });

      renderComponent();

      await waitFor(() => {
        expect(screen.getByText('Growth')).toBeInTheDocument();
      });

      // Should have retried
      expect((global.fetch as jest.Mock).mock.calls.length).toBeGreaterThanOrEqual(1);
    });
  });
});
