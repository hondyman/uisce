// React default import removed — using automatic JSX runtime
import { Chip, Tooltip } from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';

interface SchemaHashTagProps {
  hash: string;
}

const SchemaHashTag: React.FC<SchemaHashTagProps> = ({ hash }) => {
  const handleCopy = () => {
    navigator.clipboard.writeText(hash);
    // Optional: show a toast notification
  };

  return (
    <Tooltip title="Copy Schema Hash">
      <Chip
        label={`Hash: ${hash}`}
        variant="outlined"
        size="small"
        onClick={handleCopy}
        onDelete={handleCopy}
        deleteIcon={<ContentCopyIcon />}
        sx={{ fontFamily: 'monospace' }}
      />
    </Tooltip>
  );
};

export default SchemaHashTag;