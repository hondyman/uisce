// React import not required with the new JSX transform
import { DialogTitle, Box, Typography, IconButton, Chip, Divider } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';

interface ModalHeaderProps {
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  chipLabel?: React.ReactNode;
  onClose?: () => void;
  /** background color token, e.g. 'primary.light' or 'grey.50' */
  bg?: string;
}

export default function ModalHeader({ title, subtitle, chipLabel, onClose, bg }: ModalHeaderProps) {
  const titleSx = bg ? { p: 2, backgroundColor: bg, color: 'text.primary' } : { p: 2 };
  const subtitleColor = bg ? 'rgba(0,0,0,0.7)' : 'text.secondary';

  return (
    <>
      <DialogTitle sx={titleSx}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Box>
            <Typography variant="h6" component="div" sx={{ fontWeight: 700 }}> {title} </Typography>
            {subtitle && (
              <Typography variant="subtitle2" sx={{ color: subtitleColor }}>{subtitle}</Typography>
            )}
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {chipLabel && <Chip label={chipLabel} size="small" />}
            {onClose && (
              <IconButton aria-label="close" size="small" onClick={onClose}>
                <CloseIcon fontSize="small" />
              </IconButton>
            )}
          </Box>
        </Box>
      </DialogTitle>
      <Divider sx={{ borderColor: 'divider', mb: 1 }} />
    </>
  );
}
