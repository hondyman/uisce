import type { FC } from 'react';
import { Button, ButtonGroup } from '@mui/material';
import { Save, Eye, Printer, Download, Database, Filter } from 'lucide-react';

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
        <Button startIcon={<Save />} size="small" onClick={onSave}>Save</Button>
        <Button startIcon={<Eye />} size="small" onClick={onPreview}>Preview</Button>
        <Button startIcon={<Printer />} size="small" onClick={onPrint}>Print</Button>
        <Button startIcon={<Download />} size="small" onClick={onExport}>Export</Button>
      </ButtonGroup>

      <Button startIcon={<Database />} size="small" sx={{ mr: 1 }} onClick={onOpenDataSources}>Data Sources</Button>
      <Button startIcon={<Filter />} size="small" onClick={onOpenParameters}>Parameters</Button>
    </>
  );
};

export default TopControls;
