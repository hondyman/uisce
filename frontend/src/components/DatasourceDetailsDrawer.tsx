// React default import removed — using automatic JSX runtime
import { Drawer, Box, Typography, IconButton, Divider } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';

const DatasourceDetailsDrawer: React.FC<{ open: boolean; datasource: any; onClose: () => void }> = ({ open, datasource, onClose }) => {
  if (!datasource) return null;
  return (
    <Drawer anchor="right" open={open} onClose={onClose} PaperProps={{ sx: { width: { xs: '100%', md: 420 } } }} aria-label="Datasource details drawer">
      <Box sx={{ p: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography variant="h6" sx={{ flexGrow: 1 }}>{datasource.alpha_datasource?.datasource_name || datasource.source_name || 'Datasource'}</Typography>
        <IconButton onClick={onClose} aria-label="close details"><CloseIcon /></IconButton>
      </Box>
      <Divider />
      <Box sx={{ p: 2 }} role="region" aria-labelledby="datasource-details-title">
        <Typography id="datasource-details-title" variant="subtitle2">ID</Typography>
        <Typography variant="body2" sx={{ wordBreak: 'break-all' }}>{String(datasource.id)}</Typography>
        <Typography variant="subtitle2" sx={{ mt: 2 }}>Type</Typography>
        <Typography variant="body2">{datasource.alpha_datasource?.datasource_type || datasource.type || 'N/A'}</Typography>
        <Typography variant="subtitle2" sx={{ mt: 2 }}>Source</Typography>
        <Typography variant="body2">{datasource.source_name}</Typography>
      </Box>
    </Drawer>
  );
};

export default DatasourceDetailsDrawer;
