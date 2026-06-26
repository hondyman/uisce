// React import not required with the new JSX transform
import { Dialog, DialogActions, DialogContent, DialogContentText, Button } from '@mui/material';
import ModalHeader from '@/components/ModalHeader';

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

const ConfirmDialog: React.FC<ConfirmDialogProps> = ({ open, title, message, onConfirm, onCancel }) => {
  return (
    <Dialog open={open} onClose={onCancel}>
      <ModalHeader title={String(title)} onClose={onCancel} />
      <DialogContent>
        <DialogContentText>{String(message)}</DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onCancel}>Cancel</Button>
        <Button onClick={onConfirm} color="primary" autoFocus>
          Confirm
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default ConfirmDialog;