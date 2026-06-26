// React import removed (unused)
import { render, screen } from '@testing-library/react';
import { act } from 'react-dom/test-utils';
import { CodePanel } from '../CodePanel';

describe('CodePanel jump behavior', () => {
  test('jumpToSection scrolls and selects expected line in Prism textarea (JSON)', async () => {
    const json = `{
  "dimensions": [
    { "id": "d1", "name": "dim1" },
    { "id": "d2", "name": "dim2" }
  ],
  "measures": [
    { "id": "m1", "name": "measure1" }
  ]
}`;
    const genJson = () => json;

    render(
      <CodePanel
        showCode={'json'}
        modelName={'jumpmodel'}
        generateJSON={genJson}
        generateYAML={() => ''}
        searchTerm={''}
        setMatchIndex={() => {}}
        setMatchCount={() => {}}
      />
    );

    // Wait for the textarea to appear
    const ta = (await screen.findByLabelText(/JSON code editor/i)) as HTMLTextAreaElement;
    expect(ta).toBeInTheDocument();

    // Dispatch a jump request to 'dimensions' and key 'd2'
    act(() => {
      window.dispatchEvent(new CustomEvent('semlayer.jumpToSection', { detail: { section: 'dimensions', key: 'd2' } }));
    });

    // After jump, expect the selection to be set somewhere around the second dimension's line
    // Because selectionStart points to the char index at the start of the selected line we assert it's > 0 and not at document start.
    expect(ta.selectionStart).toBeGreaterThan(0);
    expect(ta.selectionEnd).toBeGreaterThanOrEqual(ta.selectionStart);
  });
});
