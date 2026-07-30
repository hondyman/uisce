import type { FC } from 'react';
import { Button, ButtonGroup } from '@mui/material';
import SaveIcon from '@mui/icons-material/Save';
import VisibilityIcon from '@mui/icons-material/Visibility';
import PrintIcon from '@mui/icons-material/Print';
import DownloadIcon from '@mui/icons-material/Download';
import StorageIcon from '@mui/icons-material/Storage';
import FilterListIcon from '@mui/icons-material/FilterList';

type Props = {
  canUndo: boolean;
  canRedo: boolean;
  onUndo: () => void;
  onRedo: () => void;
  onSave: () => void;
  onPreview: () => void;
  onPrint: () => void;
  onExport: () => void;
  onOpenDataSources: () => void;
  onOpenParameters: () => void;
};

const TopControls: FC<Props> = ({ canUndo, canRedo, onUndo, onRedo, onSave, onPreview, onPrint, onExport, onOpenDataSources, onOpenParameters }) => {
  return (
    <>
      <ButtonGroup variant="contained" sx={{ mr: 2 }}>
        <Button disabled={!canUndo} onClick={onUndo} size="small">Undo</Button>
        <Button disabled={!canRedo} onClick={onRedo} size="small">Redo</Button>
        <Button startIcon={<SaveIcon />} size="small" onClick={onSave}>Save</Button>
        <Button startIcon={<VisibilityIcon />} size="small" onClick={onPreview}>Preview</Button>
        <Button startIcon={<PrintIcon />} size="small" onClick={onPrint}>Print</Button>
        <Button startIcon={<DownloadIcon />} size="small" onClick={onExport}>Export</Button>
      </ButtonGroup>

      <Button startIcon={<StorageIcon />} size="small" sx={{ mr: 1 }} onClick={onOpenDataSources}>Data Sources</Button>
      <Button startIcon={<FilterListIcon />} size="small" onClick={onOpenParameters}>Parameters</Button>
    </>
  );
};

export default TopControls;
