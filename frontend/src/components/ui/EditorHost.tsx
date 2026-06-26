import type { FC, RefObject, ReactNode } from 'react';
import { Modal } from '../ui/Modal';
import { SlideOver } from '../ui/SlideOver';

export type EditorMode = 'modal' | 'panel' | 'auto';

export type EditorHostProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  mode?: EditorMode; // 'auto' decides based on complexity
  estimatedComplexity?: 'short' | 'long'; // optional hint from caller
  initialFocusRef?: RefObject<HTMLElement>;
  children: ReactNode;
};

const decide = (mode: EditorMode, complexity: 'short' | 'long') =>
  mode === 'auto' ? (complexity === 'short' ? 'modal' : 'panel') : mode;

export const EditorHost: FC<EditorHostProps> = ({
  open, onClose, title, mode = 'auto', estimatedComplexity = 'short', initialFocusRef, children
}) => {
  const resolved = decide(mode, estimatedComplexity);
  if (resolved === 'modal') {
    return (
      <Modal open={open} onClose={onClose} title={title} initialFocusRef={initialFocusRef}>
        {children}
      </Modal>
    );
  }
  return (
    <SlideOver open={open} onClose={onClose} title={title} side="right" width={560} modal>
      {children}
    </SlideOver>
  );
};