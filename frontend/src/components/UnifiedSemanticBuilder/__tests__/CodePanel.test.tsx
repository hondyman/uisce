// React import removed (unused)
import { render, screen } from '@testing-library/react';
import { CodePanel } from '../CodePanel';

describe('CodePanel', () => {
  test('renders and shows code content with format toggles (copy/download moved to header)', async () => {
    const mockGenJson = () => JSON.stringify({ dimensions: {}, measures: {} }, null, 2);
    const mockGenYaml = () => 'dimensions:\n  - id: a\nmeasures:\n  - id: m\n';

    render(
      <CodePanel
        showCode={'yaml'}
        modelName={'testmodel'}
  searchTerm={''}
  setMatchIndex={() => {}}
  setMatchCount={() => {}}
        generateJSON={mockGenJson}
        generateYAML={mockGenYaml}
        codeEditable={true}
        onImportCode={async () => {}}
        extendsModel={'base.core'}
      />
    );

    // Editor rendered (match by aria-label)
    const ta = (await screen.findByLabelText(
      /YAML code editor/i
    )) as HTMLTextAreaElement;
    expect(ta).toBeInTheDocument();

  // Copy/Download buttons were removed from CodePanel to avoid duplication; they live in the header
  expect(screen.queryByRole('button', { name: /Copy/i })).toBeNull();
  expect(screen.queryByRole('button', { name: /Download/i })).toBeNull();

    // code content contains expected sections
    expect(ta.value).toMatch(/dimensions:/i);
    expect(ta.value).toMatch(/measures:/i);
  });
});
