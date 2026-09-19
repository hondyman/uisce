import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Paper, ButtonBase, Typography, Dialog, DialogTitle, DialogContent, IconButton } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { ComponentDefinition } from '../../types/pageStudio';
import EmbeddedPageContent from './EmbeddedPageContent';
import { useTenant } from '../../contexts/TenantContext';

interface TileWidgetProps {
  component: ComponentDefinition;
}

/**
 * A clickable summary card meant for a row of tiles across the top of a
 * page (e.g. "Open Orders: 42"). Same three navigation modes as
 * ButtonWidget.tsx (navigate/openModal/openUrl) - kept as a separate
 * component rather than reusing ButtonWidget directly because a tile's
 * layout (title + big value, card shape) is a different shape than a
 * button, not just a different style.
 */
const TileWidget: React.FC<TileWidgetProps> = ({ component }) => {
  const navigate = useNavigate();
  const { tenant } = useTenant();
  const [modalOpen, setModalOpen] = useState(false);

  const title = (component.props?.title as string) || 'Tile';
  const value = component.props?.value as string | undefined;
  const action = (component.props?.action as string) || 'navigate';
  const targetPageSlug = component.props?.targetPageSlug as string | undefined;
  const url = component.props?.url as string | undefined;
  const clickable = action === 'openModal' ? !!targetPageSlug : action === 'openUrl' ? !!url : !!targetPageSlug;

  const handleClick = () => {
    if (action === 'openModal' && targetPageSlug) { setModalOpen(true); return; }
    if (action === 'openUrl' && url) { window.open(url, '_blank', 'noopener,noreferrer'); return; }
    if (action === 'navigate' && targetPageSlug) { navigate(`/pages/${targetPageSlug}`); return; }
  };

  const content = (
    <Paper
      variant="outlined"
      sx={{
        p: 2,
        minWidth: 160,
        borderRadius: 2,
        textAlign: 'left',
        transition: 'box-shadow 0.15s ease',
        '&:hover': clickable ? { boxShadow: 2 } : undefined,
      }}
    >
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>{title}</Typography>
      {value !== undefined && <Typography variant="h5" fontWeight={800}>{value}</Typography>}
    </Paper>
  );

  return (
    <>
      {clickable ? (
        <ButtonBase onClick={handleClick} sx={{ display: 'block', textAlign: 'left', borderRadius: 2 }}>{content}</ButtonBase>
      ) : content}
      {action === 'openModal' && targetPageSlug && (
        <Dialog open={modalOpen} onClose={() => setModalOpen(false)} maxWidth="md" fullWidth>
          <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            {title}
            <IconButton size="small" onClick={() => setModalOpen(false)}><CloseIcon fontSize="small" /></IconButton>
          </DialogTitle>
          <DialogContent>
            <EmbeddedPageContent slug={targetPageSlug} tenantId={tenant?.id || ''} />
          </DialogContent>
        </Dialog>
      )}
    </>
  );
};

export default TileWidget;
