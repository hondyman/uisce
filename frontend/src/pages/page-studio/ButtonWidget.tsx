import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Dialog, DialogTitle, DialogContent, IconButton } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { ComponentDefinition } from '../../types/pageStudio';
import EmbeddedPageContent from './EmbeddedPageContent';
import { useTenant } from '../../contexts/TenantContext';
import { useSelection } from './SelectionContext';

interface ButtonWidgetProps {
  component: ComponentDefinition;
}

/**
 * A clickable action widget with three modes (`props.action`):
 * - 'navigate': go to another page in this app (react-router, in-app)
 * - 'openUrl': open an external URL in a new tab
 * - 'openModal': show another page's content in a Dialog without
 *   leaving the current page - reuses EmbeddedPageContent, the same
 *   renderer PageBrowser.tsx uses for its top-level layout, so a page
 *   opened as a modal behaves identically to visiting it directly.
 */
const ButtonWidget: React.FC<ButtonWidgetProps> = ({ component }) => {
  const navigate = useNavigate();
  const { tenant } = useTenant();
  const { selection } = useSelection();
  const [modalOpen, setModalOpen] = useState(false);

  const label = (component.props?.label as string) || 'Button';
  const action = (component.props?.action as string) || 'navigate';
  const targetPageSlug = component.props?.targetPageSlug as string | undefined;
  const url = component.props?.url as string | undefined;
  const variant = (component.props?.variant as 'contained' | 'outlined' | 'text') || 'contained';
  const recordPath = targetPageSlug && selection?.recordId
    ? `/pages/${targetPageSlug}/${selection.recordId}`
    : targetPageSlug
      ? `/pages/${targetPageSlug}`
      : undefined;

  const handleClick = () => {
    if (action === 'openModal') { setModalOpen(true); return; }
    if (action === 'openUrl' && url) { window.open(url, '_blank', 'noopener,noreferrer'); return; }
    if (action === 'navigate' && recordPath) { navigate(recordPath); return; }
  };

  return (
    <>
      <Button variant={variant} onClick={handleClick}>{label}</Button>
      {action === 'openModal' && targetPageSlug && (
        <Dialog open={modalOpen} onClose={() => setModalOpen(false)} maxWidth="md" fullWidth>
          <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            {label}
            <IconButton size="small" onClick={() => setModalOpen(false)}><CloseIcon fontSize="small" /></IconButton>
          </DialogTitle>
          <DialogContent>
            <EmbeddedPageContent slug={targetPageSlug} tenantId={tenant?.id || ''} recordId={selection?.recordId} />
          </DialogContent>
        </Dialog>
      )}
    </>
  );
};

export default ButtonWidget;
