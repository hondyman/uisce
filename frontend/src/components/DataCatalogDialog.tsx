// React import not required with the new JSX transform
import {
  Dialog,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Box,
} from '@mui/material';
import ModalHeader from '@/components/ModalHeader';

interface DataCatalogDialogProps {
  open: boolean;
  sourceName: string | null;
  onClose: () => void;
}

export const DataCatalogDialog: React.FC<DataCatalogDialogProps> = ({
  open,
  sourceName,
  onClose,
}) => {
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
  <ModalHeader title="Data Catalog" onClose={onClose} />
      <DialogContent>
        <Box sx={{ p: 2, minHeight: '100px', display: 'flex', alignItems: 'center' }}>
          <Typography variant="h6">
            Details for: <strong>{sourceName ? String(sourceName) : ''}</strong>
          </Typography>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};