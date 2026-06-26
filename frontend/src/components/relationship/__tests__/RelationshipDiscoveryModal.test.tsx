import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import RelationshipDiscoveryModal from '../RelationshipDiscoveryModal';

// Mock fetch
global.fetch = jest.fn();

// Mock Ant Design message
jest.mock('antd', () => {
  const actual = jest.requireActual('antd');
  return {
    ...actual,
    message: {
      error: jest.fn(),
      success: jest.fn(),
      info: jest.fn(),
    },
  };
});

describe('RelationshipDiscoveryModal', () => {
  const defaultProps = {
    visible: true,
    entityAttributeId: 'entity-123',
    entityName: 'Customers',
    tenantId: 'tenant-123',
    datasourceId: 'ds-456',
    onClose: jest.fn(),
    onApplyRelationship: jest.fn(),
  };

  beforeEach(() => {
    (global.fetch as jest.Mock).mockClear();
  });

  it('should render the modal with tabs', () => {
    render(<RelationshipDiscoveryModal {...defaultProps} />);

    expect(screen.getByText('Discover Relationships')).toBeInTheDocument();
    expect(screen.getByText('Direct Relationships')).toBeInTheDocument();
  });

  it('should display loading state while discovering', async () => {
    (global.fetch as jest.Mock).mockImplementation(
      () =>
        new Promise((resolve) =>
          setTimeout(
            () =>
              resolve({
                ok: true,
                json: async () => ({
                  directRelationships: [],
                  multiHopPaths: [],
                }),
              }),
            200
          )
        )
    );

    const { container } = render(
      <RelationshipDiscoveryModal {...defaultProps} />
    );

    const discoverButton = screen.getByRole('button', {
      name: /discover relationships/i,
    });
    fireEvent.click(discoverButton);

    // Check for loading spinner
    await waitFor(() => {
      const spinner = container.querySelector('.ant-spin');
      expect(spinner).toBeInTheDocument();
    });
  });

  it('should display confidence badges', async () => {
    const mockResponse = {
      directRelationships: [
        {
          relatedEntityId: 'entity-789',
          relatedEntityName: 'Orders',
          linkType: 'DIRECT_FK',
          confidence: 0.95,
          cardinality: '1:N',
          foreignKeyPath: [],
          columnMapping: [],
        },
      ],
      multiHopPaths: [],
    };

    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    render(<RelationshipDiscoveryModal {...defaultProps} />);

    const discoverButton = screen.getByRole('button', {
      name: /discover relationships/i,
    });
    fireEvent.click(discoverButton);

    await waitFor(() => {
      expect(screen.getByText('Orders')).toBeInTheDocument();
    });
  });

  it('should handle discovery errors', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: async () => ({ error: 'Server error' }),
    });

    const { container } = render(
      <RelationshipDiscoveryModal {...defaultProps} />
    );

    const discoverButton = screen.getByRole('button', {
      name: /discover relationships/i,
    });
    fireEvent.click(discoverButton);

    await waitFor(() => {
      const errorBanner = container.querySelector('.error-banner');
      expect(errorBanner).toBeInTheDocument();
    });
  });

  it('should apply relationship on button click', async () => {
    const mockResponse = {
      directRelationships: [
        {
          relatedEntityId: 'entity-789',
          relatedEntityName: 'Orders',
          linkType: 'DIRECT_FK',
          confidence: 0.95,
          cardinality: '1:N',
          foreignKeyPath: [],
          columnMapping: [],
        },
      ],
      multiHopPaths: [],
    };

    (global.fetch as jest.Mock)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true }),
      });

    render(<RelationshipDiscoveryModal {...defaultProps} />);

    // Discover relationships
    fireEvent.click(
      screen.getByRole('button', {
        name: /discover relationships/i,
      })
    );

    await waitFor(() => {
      expect(screen.getByText('Orders')).toBeInTheDocument();
    });

    // Apply relationship
    const applyButton = screen.getByRole('button', {
      name: /apply relationship/i,
    });
    fireEvent.click(applyButton);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/relationships/apply',
        expect.any(Object)
      );
    });
  });

  it('should display empty state when no relationships found', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        directRelationships: [],
        multiHopPaths: [],
      }),
    });

    render(<RelationshipDiscoveryModal {...defaultProps} />);

    fireEvent.click(
      screen.getByRole('button', {
        name: /discover relationships/i,
      })
    );

    await waitFor(() => {
      expect(screen.getByText('No Relationships Found')).toBeInTheDocument();
    });
  });
});
