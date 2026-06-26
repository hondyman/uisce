// React import not required with the new JSX transform
import {
  Dialog,
  DialogContent,
  DialogActions,
  Button,
  CircularProgress
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { Alert, AlertTitle } from '@mui/material';

interface ConnectionTestDialogProps {
  open: boolean;
  loading: boolean;
  result: { success: boolean; message: string; } | null;
  onClose: () => void;
}

export const ConnectionTestDialog: React.FC<ConnectionTestDialogProps> = ({
  open,
  loading,
  result,
  onClose,
}) => {
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <ModalHeader title="Connection Test Result" onClose={onClose} />
      <DialogContent sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '150px' }}>
        {loading && <CircularProgress />}
        {!loading && result && (
          <Alert
            severity={result.success ? 'success' : 'error'}
            iconMapping={{
              success: <CheckCircleOutlineIcon fontSize="inherit" />,
              error: <ErrorOutlineIcon fontSize="inherit" />,
            }}
            sx={{ width: '100%' }}
          >
            <AlertTitle>{result.success ? 'Success' : 'Failed'}</AlertTitle>
            {result && typeof result.message === 'object' ? JSON.stringify(result.message) : String(result?.message ?? '')}
          </Alert>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};