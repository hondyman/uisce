import { useEffect } from 'react';

export interface UseDialogProps {
  open: boolean;
  onClose: () => void;
  initialFocusRef?: React.RefObject<HTMLElement>;
  containerRef?: React.RefObject<HTMLElement>;
}

/**
 * useDialog: Hook for managing dialog/modal accessibility
 * Handles focus management, escape key, and scroll lock
 */
export const useDialog = ({
  open,
  onClose,
  initialFocusRef,
  containerRef,
}: UseDialogProps) => {
  useEffect(() => {
    if (!open) return;

    // Lock body scroll
    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    // Focus management
    const previousActiveElement = document.activeElement as HTMLElement;
    const focusTarget = initialFocusRef?.current || containerRef?.current;
    if (focusTarget) {
      focusTarget.focus();
    }

    // Handle escape key
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = originalOverflow;
      if (previousActiveElement && previousActiveElement.focus) {
        previousActiveElement.focus();
      }
    };
  }, [open, onClose, initialFocusRef, containerRef]);
};

export default useDialog;
