import { useRef, useState } from 'react';
import { EditorHost } from '../src/components/editor/EditorHost';

// Use loose typing for Storybook stories to avoid consuming storybook-only types in the strict tsc run
const meta: any = {
  title: 'Editor/EditorHost',
  component: EditorHost,
  parameters: { layout: 'centered' }
};
export default meta;
type Story = any;

export const ModalShort: Story = {
  render: () => {
    const Demo = () => {
      const [open, setOpen] = useState(false);
      const btnRef = useRef<HTMLButtonElement>(null);
      return (
        <>
          <button ref={btnRef} onClick={() => setOpen(true)}>Open Modal</button>
          <EditorHost
            open={open}
            onClose={() => setOpen(false)}
            title="Configure Section"
            mode="modal"
            estimatedComplexity="short"
            initialFocusRef={btnRef}
          >
            <input placeholder="Name" className="demo-input" />
            <input placeholder="Label" className="demo-input" />
            <button>Primary Action</button>
          </EditorHost>
        </>
      );
    };
    return <Demo />;
  },
  play: async (_ctx: any) => {
    // Add a style tag to the document head for story-specific styles
    const style = document.createElement('style');
    style.innerHTML = `
      .demo-input { display: block; margin-bottom: 12px; }
      .demo-scroll-content { height: 800px; }
    `;
    document.head.appendChild(style);
    // Optional: add Storybook Testing Library interactions
  }
};

export const PanelLong: Story = {
  render: () => {
    const Demo = () => {
      const [open, setOpen] = useState(false);
      return (
        <>
          <button onClick={() => setOpen(true)}>Open Panel</button>
          <EditorHost
            open={open}
            onClose={() => setOpen(false)}
            title="Related Records"
            mode="panel"
            estimatedComplexity="long"
          >
            <div className="demo-scroll-content">
              <p>Scroll content</p>
              <input placeholder="Filter" />
              {/* long list to verify scroll locking */}
              {Array.from({ length: 50 }).map((_, i) => <div key={i}>Row {i+1}</div>)}
            </div>
          </EditorHost>
        </>
      );
    };
    return <Demo />;
  }
};