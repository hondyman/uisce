import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { WorkflowOrchestrator } from '../components/WorkflowOrchestrator';

// Mock the ABAC hook
vi.mock('../hooks/useABAC', () => ({
  useABAC: () => ({
    evaluate: vi.fn().mockResolvedValue(true)
  })
}));

// Mock fetch
global.fetch = vi.fn();

const createTestQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
    mutations: {
      retry: false,
    },
  },
});

const renderWithProviders = (component: React.ReactElement) => {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      {component}
    </QueryClientProvider>
  );
};

describe('WorkflowOrchestrator', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the component with initial elements', () => {
    renderWithProviders(<WorkflowOrchestrator />);

    expect(screen.getByText('AI Workflow Orchestrator')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/describe the workflow/i)).toBeInTheDocument();
    expect(screen.getByText('🤖 AI Suggest')).toBeInTheDocument();
    expect(screen.getByText('Clear')).toBeInTheDocument();
  });

  it('allows adding nodes via the palette', () => {
    renderWithProviders(<WorkflowOrchestrator />);

    const startButton = screen.getByText('+ Start');
    fireEvent.click(startButton);

    // Check that the status shows more nodes
    expect(screen.getByText(/nodes: 2/i)).toBeInTheDocument();
  });

  it('handles AI suggestion workflow', async () => {
    const mockResponse = {
      suggestion: {
        elements: [
          {
            id: 'ai-1',
            type: 'action',
            position: { x: 100, y: 100 },
            data: { label: 'AI Generated Action' }
          }
        ]
      }
    };

    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockResponse)
    });

    renderWithProviders(<WorkflowOrchestrator />);

    const textarea = screen.getByPlaceholderText(/describe the workflow/i);
    const suggestButton = screen.getByText('🤖 AI Suggest');

    fireEvent.change(textarea, { target: { value: 'Create a simple approval workflow' } });
    fireEvent.click(suggestButton);

    await waitFor(() => {
      expect(screen.getByText('✓ AI suggestion applied')).toBeInTheDocument();
    });

    expect(global.fetch).toHaveBeenCalledWith('http://localhost:8081/workflows/suggest', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        description: 'Create a simple approval workflow',
        context: {
          industry: 'wealth-management',
          compliance: 'FINRA',
          user: 'advisor'
        }
      })
    });
  });

  it('shows error when AI suggestion fails', async () => {
    (global.fetch as any).mockRejectedValueOnce(new Error('Network error'));

    renderWithProviders(<WorkflowOrchestrator />);

    const textarea = screen.getByPlaceholderText(/describe the workflow/i);
    const suggestButton = screen.getByText('🤖 AI Suggest');

    fireEvent.change(textarea, { target: { value: 'Test workflow' } });
    fireEvent.click(suggestButton);

    await waitFor(() => {
      expect(screen.getByText(/error:/i)).toBeInTheDocument();
    });
  });

  it('clears workflow when clear button is clicked', () => {
    renderWithProviders(<WorkflowOrchestrator />);

    const startButton = screen.getByText('+ Start');
    fireEvent.click(startButton);

    expect(screen.getByText(/nodes: 2/i)).toBeInTheDocument();

    const clearButton = screen.getByText('Clear');
    fireEvent.click(clearButton);

    expect(screen.getByText(/nodes: 1/i)).toBeInTheDocument();
  });
});