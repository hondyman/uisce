import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { vi, describe, it, expect } from 'vitest';

// Mock heavy lazy-loaded children to keep test fast and deterministic
vi.mock('../LineageFlow', () => ({
  __esModule: true,
  default: () => <div data-testid="lineage-flow">LineageFlow</div>
}));
vi.mock('../LineageTypeSelector', () => ({
  __esModule: true,
  default: () => <div data-testid="lineage-type-selector">TypeSelector</div>
}));
vi.mock('./DetailsPane', () => ({
  __esModule: true,
  default: () => <div data-testid="details-pane">Details</div>
}));

import DualLineageViewer from '../DualLineageViewer';

const baseAsset = { type: 'table', id: 't1', name: 'table_one' } as any;

describe('DualLineageViewer fullscreen behaviour', () => {
  it('calls onToggleFullScreen when fullscreen button clicked (inline)', async () => {
    const onToggle = vi.fn();
    render(
      <DualLineageViewer
        selectedAsset={baseAsset}
        technicalData={{ nodes: [], edges: [], metadata: {} } as any}
        semanticData={undefined}
        onToggleFullScreen={onToggle}
        isFullScreen={false}
      />
    );

    const btn = await screen.findByText(/Fullscreen/i);
    fireEvent.click(btn);
    expect(onToggle).toHaveBeenCalled();
  });

  it('renders overlay when isFullScreen=true and shows open class, and the fullscreen button calls handler', async () => {
    const onToggle = vi.fn();
    render(
      <DualLineageViewer
        selectedAsset={baseAsset}
        technicalData={{ nodes: [{ id: 'n1' }], edges: [{ id: 'e1' }], metadata: {} } as any}
        semanticData={undefined}
        onToggleFullScreen={onToggle}
        isFullScreen={true}
      />
    );

  // overlay should be present and eventually have the open class
  const overlayByClass = document.querySelector('.dlv-overlay');
    expect(overlayByClass).toBeTruthy();

    await waitFor(() => {
      expect(overlayByClass).toHaveClass('open');
    });

  // Click the fullscreen button inside overlay - it triggers a small closing animation
  const btn = await screen.findByText(/Fullscreen/i);
  fireEvent.click(btn);
  await waitFor(() => expect(onToggle).toHaveBeenCalled(), { timeout: 500 });
  });
});
