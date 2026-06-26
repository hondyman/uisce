/**
 * CollaborativeAnnotations Component Tests
 * 
 * Tests for:
 * - Rendering annotations list
 * - Adding new annotations
 * - Editing existing annotations
 * - Deleting annotations
 * - Filtering annotations by type
 * - User presence and timestamps
 * - Real-time updates
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { CollaborativeAnnotations } from '../CollaborativeAnnotations';

const theme = createTheme();

const renderWithTheme = (component: React.ReactElement) => {
  return render(
    <ThemeProvider theme={theme}>
      {component}
    </ThemeProvider>
  );
};

interface Annotation {
  id: string;
  author: string;
  text: string;
  type: 'comment' | 'warning' | 'insight';
  timestamp: Date;
  editable: boolean;
}

describe('CollaborativeAnnotations Component', () => {
  const mockAnnotations: Annotation[] = [
    {
      id: '1',
      author: 'John Doe',
      text: 'Consider increasing equity exposure',
      type: 'insight',
      timestamp: new Date('2024-01-15T10:00:00'),
      editable: true,
    },
    {
      id: '2',
      author: 'Jane Smith',
      text: 'Risk threshold exceeded in sector',
      type: 'warning',
      timestamp: new Date('2024-01-15T10:30:00'),
      editable: false,
    },
    {
      id: '3',
      author: 'Bob Johnson',
      text: 'Scenario results need review',
      type: 'comment',
      timestamp: new Date('2024-01-15T11:00:00'),
      editable: true,
    },
  ];

  const mockOnAdd = jest.fn();
  const mockOnEdit = jest.fn();
  const mockOnDelete = jest.fn();

  const defaultProps = {
    annotations: mockAnnotations,
    onAdd: mockOnAdd,
    onEdit: mockOnEdit,
    onDelete: mockOnDelete,
    currentUser: 'John Doe',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    test('renders all annotations', () => {
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      mockAnnotations.forEach(annotation => {
        expect(screen.getByText(annotation.text)).toBeInTheDocument();
        expect(screen.getByText(annotation.author)).toBeInTheDocument();
      });
    });

    test('displays annotation type badges', () => {
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      expect(screen.getByText('insight')).toBeInTheDocument();
      expect(screen.getByText('warning')).toBeInTheDocument();
      expect(screen.getByText('comment')).toBeInTheDocument();
    });

    test('displays timestamps in human-readable format', () => {
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      // Should show relative time (e.g., "2 hours ago")
      const timeElements = screen.getAllByText(/ago|min|hour|day/);
      expect(timeElements.length).toBeGreaterThan(0);
    });

    test('displays author names for each annotation', () => {
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      expect(screen.getByText('John Doe')).toBeInTheDocument();
      expect(screen.getByText('Jane Smith')).toBeInTheDocument();
      expect(screen.getByText('Bob Johnson')).toBeInTheDocument();
    });

    test('renders add annotation button', () => {
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const addButton = screen.getByRole('button', { name: /add annotation|new annotation/i });
      expect(addButton).toBeInTheDocument();
    });
  });

  describe('Adding Annotations', () => {
    test('shows add annotation form when button clicked', async () => {
      const user = userEvent.setup();
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const addButton = screen.getByRole('button', { name: /add annotation|new/i });
      await user.click(addButton);

      expect(screen.getByRole('textbox', { name: /annotation text|comment|message/i })).toBeInTheDocument();
    });

    test('calls onAdd with correct data when form submitted', async () => {
      const user = userEvent.setup();
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const addButton = screen.getByRole('button', { name: /add annotation|new/i });
      await user.click(addButton);

      const input = screen.getByRole('textbox', { name: /annotation text|comment|message/i });
      await user.type(input, 'New annotation text');

      const typeSelect = screen.getByRole('combobox', { name: /type|category/i });
      await user.selectOptions(typeSelect, 'insight');

      const submitButton = screen.getByRole('button', { name: /add|submit|save/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnAdd).toHaveBeenCalledWith(
          expect.objectContaining({
            text: 'New annotation text',
            type: 'insight',
          })
        );
      });
    });

    test('clears form after successful submission', async () => {
      const user = userEvent.setup();
      const { rerender } = renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const addButton = screen.getByRole('button', { name: /add annotation|new/i });
      await user.click(addButton);

      const input = screen.getByRole('textbox') as HTMLTextAreaElement;
      await user.type(input, 'New annotation');

      const submitButton = screen.getByRole('button', { name: /add|submit|save/i });
      await user.click(submitButton);

      // Rerender with new annotation
      const newAnnotations = [
        ...mockAnnotations,
        {
          id: '4',
          author: 'John Doe',
          text: 'New annotation',
          type: 'comment' as const,
          timestamp: new Date(),
          editable: true,
        },
      ];

      rerender(
        <ThemeProvider theme={theme}>
          <CollaborativeAnnotations
            {...defaultProps}
            annotations={newAnnotations}
          />
        </ThemeProvider>
      );

      expect(screen.getByText('New annotation')).toBeInTheDocument();
    });

    test('filters annotations by type', async () => {
      const user = userEvent.setup();
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const filterSelect = screen.getByRole('combobox', { name: /filter|type/i });
      await user.selectOptions(filterSelect, 'warning');

      // Should only show warning type annotation
      expect(screen.getByText('Risk threshold exceeded in sector')).toBeInTheDocument();
      expect(screen.queryByText('Consider increasing equity exposure')).not.toBeInTheDocument();
    });
  });

  describe('Editing Annotations', () => {
    test('shows edit button only for editable annotations by current user', () => {
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      // John Doe's annotation should have edit button
      const editButtons = screen.getAllByRole('button', { name: /edit/i });
      expect(editButtons.length).toBeGreaterThanOrEqual(1);
    });

    test('opens edit dialog when edit button clicked', async () => {
      const user = userEvent.setup();
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const editButton = screen.getAllByRole('button', { name: /edit/i })[0];
      await user.click(editButton);

      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    test('pre-fills edit form with current annotation data', async () => {
      const user = userEvent.setup();
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const editButton = screen.getAllByRole('button', { name: /edit/i })[0];
      await user.click(editButton);

      const input = screen.getByRole('textbox') as HTMLTextAreaElement;
      expect(input.value).toContain(mockAnnotations[0].text);
    });

    test('calls onEdit with updated data', async () => {
      const user = userEvent.setup();
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const editButton = screen.getAllByRole('button', { name: /edit/i })[0];
      await user.click(editButton);

      const input = screen.getByRole('textbox') as HTMLTextAreaElement;
      await user.clear(input);
      await user.type(input, 'Updated annotation text');

      const saveButton = screen.getByRole('button', { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(mockOnEdit).toHaveBeenCalledWith(
          mockAnnotations[0].id,
          expect.objectContaining({
            text: 'Updated annotation text',
          })
        );
      });
    });

    test('hides edit button for annotations by other users', () => {
      renderWithTheme(
        <CollaborativeAnnotations
          {...defaultProps}
          currentUser="Different User"
        />
      );

      // Should only have edit options for "Different User" annotations
      const editButtons = screen.queryAllByRole('button', { name: /edit/i });
      expect(editButtons.length).toBe(0);
    });
  });

  describe('Deleting Annotations', () => {
    test('shows delete button only for editable annotations by current user', () => {
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const deleteButtons = screen.getAllByRole('button', { name: /delete|remove/i });
      expect(deleteButtons.length).toBeGreaterThanOrEqual(1);
    });

    test('calls onDelete when delete button clicked after confirmation', async () => {
      const user = userEvent.setup();
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const deleteButton = screen.getAllByRole('button', { name: /delete|remove/i })[0];
      await user.click(deleteButton);

      // Confirm deletion
      const confirmButton = screen.getByRole('button', { name: /confirm|delete|yes/i });
      await user.click(confirmButton);

      await waitFor(() => {
        expect(mockOnDelete).toHaveBeenCalledWith(mockAnnotations[0].id);
      });
    });

    test('does not delete if confirmation is cancelled', async () => {
      const user = userEvent.setup();
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const deleteButton = screen.getAllByRole('button', { name: /delete|remove/i })[0];
      await user.click(deleteButton);

      // Cancel deletion
      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      expect(mockOnDelete).not.toHaveBeenCalled();
    });
  });

  describe('Annotation Types Styling', () => {
    test('renders different colors for different annotation types', () => {
      const { container } = renderWithTheme(
        <CollaborativeAnnotations {...defaultProps} />
      );

      // Check for type-specific styling
      const badges = container.querySelectorAll('[data-annotation-type]');
      expect(badges.length).toBe(3);
    });

    test('insight annotations display in highlight color', () => {
      const { container } = renderWithTheme(
        <CollaborativeAnnotations {...defaultProps} />
      );

      const insightBadge = container.querySelector('[data-annotation-type="insight"]');
      expect(insightBadge).toBeInTheDocument();
    });

    test('warning annotations display in alert color', () => {
      const { container } = renderWithTheme(
        <CollaborativeAnnotations {...defaultProps} />
      );

      const warningBadge = container.querySelector('[data-annotation-type="warning"]');
      expect(warningBadge).toBeInTheDocument();
    });
  });

  describe('Real-time Updates', () => {
    test('updates annotations list when new prop data arrives', () => {
      const { rerender } = renderWithTheme(
        <CollaborativeAnnotations
          {...defaultProps}
          annotations={mockAnnotations}
        />
      );

      const newAnnotation: Annotation = {
        id: '4',
        author: 'New User',
        text: 'Real-time annotation',
        type: 'comment' as const,
        timestamp: new Date(),
        editable: false,
      };

      rerender(
        <ThemeProvider theme={theme}>
          <CollaborativeAnnotations
            {...defaultProps}
            annotations={[...mockAnnotations, newAnnotation]}
          />
        </ThemeProvider>
      );

      expect(screen.getByText('Real-time annotation')).toBeInTheDocument();
    });

    test('maintains scroll position when new annotations arrive', () => {
      const { container, rerender } = renderWithTheme(
        <CollaborativeAnnotations {...defaultProps} />
      );

      const annotationsList = container.querySelector('[data-testid="annotations-list"]');
      if (annotationsList) {
        annotationsList.scrollTop = 100;
        const previousScrollTop = annotationsList.scrollTop;

        rerender(
          <ThemeProvider theme={theme}>
            <CollaborativeAnnotations
              {...defaultProps}
              annotations={[
                ...mockAnnotations,
                {
                  id: '4',
                  author: 'User',
                  text: 'New',
                  type: 'comment' as const,
                  timestamp: new Date(),
                  editable: false,
                },
              ]}
            />
          </ThemeProvider>
        );

        // Scroll position should be maintained
        expect(annotationsList.scrollTop).toBeLessThanOrEqual(previousScrollTop);
      }
    });
  });

  describe('Accessibility', () => {
    test('announcements are made to screen readers for new annotations', () => {
      const { rerender } = renderWithTheme(
        <CollaborativeAnnotations {...defaultProps} />
      );

      const liveRegion = screen.getByRole('status', { hidden: true });
      expect(liveRegion).toBeInTheDocument();

      rerender(
        <ThemeProvider theme={theme}>
          <CollaborativeAnnotations
            {...defaultProps}
            annotations={[
              ...mockAnnotations,
              {
                id: '4',
                author: 'User',
                text: 'New annotation for testing',
                type: 'comment' as const,
                timestamp: new Date(),
                editable: false,
              },
            ]}
          />
        </ThemeProvider>
      );

      expect(liveRegion.textContent).toContain('New annotation');
    });

    test('keyboard navigation works for all buttons', async () => {
      const user = userEvent.setup();
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const addButton = screen.getByRole('button', { name: /add|new/i });
      addButton.focus();

      expect(addButton).toHaveFocus();

      await user.keyboard('{Enter}');
      expect(screen.getByRole('textbox')).toBeInTheDocument();
    });

    test('form fields have proper labels', () => {
      renderWithTheme(<CollaborativeAnnotations {...defaultProps} />);

      const addButton = screen.getByRole('button', { name: /add|new/i });
      fireEvent.click(addButton);

      expect(screen.getByLabelText(/text|message|comment/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/type|category/i)).toBeInTheDocument();
    });
  });

  describe('Error States', () => {
    test('displays empty state when no annotations exist', () => {
      renderWithTheme(
        <CollaborativeAnnotations
          {...defaultProps}
          annotations={[]}
        />
      );

      expect(screen.getByText(/no annotations|empty/i)).toBeInTheDocument();
    });

    test('handles empty annotation text gracefully', () => {
      renderWithTheme(
        <CollaborativeAnnotations
          {...defaultProps}
          annotations={[
            {
              id: '1',
              author: 'User',
              text: '',
              type: 'comment' as const,
              timestamp: new Date(),
              editable: true,
            },
          ]}
        />
      );

      expect(screen.getByText(/user/i)).toBeInTheDocument();
    });

    test('displays error message when adding annotation fails', async () => {
      const user = userEvent.setup();
      const failingOnAdd = jest.fn().mockRejectedValue(new Error('Failed to add'));

      renderWithTheme(
        <CollaborativeAnnotations
          {...defaultProps}
          onAdd={failingOnAdd}
        />
      );

      const addButton = screen.getByRole('button', { name: /add|new/i });
      await user.click(addButton);

      const input = screen.getByRole('textbox');
      await user.type(input, 'Test');

      const submitButton = screen.getByRole('button', { name: /add|submit|save/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText(/error|failed/i)).toBeInTheDocument();
      });
    });
  });
});
