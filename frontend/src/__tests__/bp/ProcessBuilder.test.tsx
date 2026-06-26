import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ProcessBuilder } from '../ProcessBuilder';
import '@testing-library/jest-dom';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
});

const renderWithProviders = (component: React.ReactElement) => {
  return render(
    <QueryClientProvider client={queryClient}>{component}</QueryClientProvider>
  );
};

describe('ProcessBuilder', () => {
  beforeEach(() => {
    queryClient.clear();
    global.fetch = jest.fn();
  });

  describe('Initialization', () => {
    it('should render empty process builder', () => {
      renderWithProviders(<ProcessBuilder />);
      expect(screen.getByPlaceholderText('Process Name')).toBeInTheDocument();
    });

    it('should show activity palette', () => {
      renderWithProviders(<ProcessBuilder />);
      expect(screen.getByText('Activity Palette')).toBeInTheDocument();
      expect(screen.getByText('Manual Task')).toBeInTheDocument();
    });
  });

  describe('Activity Management', () => {
    it('should add activity when clicked from palette', () => {
      renderWithProviders(<ProcessBuilder />);
      
      const manualTaskButton = screen.getByText('Manual Task').closest('div');
      fireEvent.click(manualTaskButton!);
      
      expect(screen.getByText(/manual_task 1/i)).toBeInTheDocument();
    });

    it('should select activity on click', () => {
      renderWithProviders(<ProcessBuilder />);
      
      fireEvent.click(screen.getByText('Manual Task').closest('div')!);
      const activity = screen.getByText(/manual_task 1/i);
      fireEvent.click(activity);
      
      expect(screen.getByText('Activity Properties')).toBeInTheDocument();
    });

    it('should delete activity', async () => {
      renderWithProviders(<ProcessBuilder />);
      
      fireEvent.click(screen.getByText('Manual Task').closest('div')!);
      const activity = screen.getByText(/manual_task 1/i);
      fireEvent.click(activity);
      
      const deleteButton = screen.getByText('Delete Activity');
      fireEvent.click(deleteButton);
      
      await waitFor(() => {
        expect(screen.queryByText(/manual_task 1/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('Process Operations', () => {
    it('should mark as dirty when name changes', () => {
      renderWithProviders(<ProcessBuilder />);
      
      const nameInput = screen.getByPlaceholderText('Process Name');
      fireEvent.change(nameInput, { target: { value: 'Test Process' } });
      
      expect(screen.getByText('Unsaved changes')).toBeInTheDocument();
    });

    it('should save process', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: 'process-123' }),
      });

      renderWithProviders(<ProcessBuilder />);
      
      const nameInput = screen.getByPlaceholderText('Process Name');
      fireEvent.change(nameInput, { target: { value: 'Test Process' } });
      
      const saveButton = screen.getByText('Save');
      fireEvent.click(saveButton);
      
      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith(
          '/api/bp/processes',
          expect.objectContaining({ method: 'POST' })
        );
      });
    });

    it('should export process', () => {
      renderWithProviders(<ProcessBuilder />);
      
      const nameInput = screen.getByPlaceholderText('Process Name');
      fireEvent.change(nameInput, { target: { value: 'Export Test' } });
      
      const exportButton = screen.getByText('Export');
      
      // Mock createElement and click
      const createElementSpy = jest.spyOn(document, 'createElement');
      fireEvent.click(exportButton);
      
      expect(createElementSpy).toHaveBeenCalledWith('a');
    });
  });

  describe('Transitions', () => {
    it('should create transition between activities', () => {
      renderWithProviders(<ProcessBuilder />);
      
      // Add two activities
      fireEvent.click(screen.getByText('Manual Task').closest('div')!);
      fireEvent.click(screen.getByText('Approval').closest('div')!);
      
      // Switch to transitions tab
      fireEvent.click(screen.getByText('Transitions'));
      
      // Would need to interact with canvas to create actual transition
      // This is a simplified test
    });
  });

  describe('Validation', () => {
    it('should validate empty process name', () => {
      renderWithProviders(<ProcessBuilder />);
      
      fireEvent.click(screen.getByText('Preview'));
      
      expect(screen.getByText(/Process name is required/i)).toBeInTheDocument();
    });
  });
});
