import { Suspense } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button } from '@mui/material';
import DualLineageViewer from '../../pages/TabbedModal/Catalog/DualLineageViewer';

interface LineageModalProps {
  open: boolean;
  onClose: () => void;
  selectedAsset: any;
  lineageData: any;
  loading?: boolean;
}

export function LineageModal({ open, onClose, selectedAsset, lineageData, loading }: LineageModalProps) {
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="lg">
      <DialogTitle>Lineage for {selectedAsset?.name || 'Selected Term'}</DialogTitle>
      <DialogContent dividers sx={{ height: '70vh', minHeight: 400 }}>
        <Suspense fallback={<div>Loading lineage...</div>}>
          <DualLineageViewer
            selectedAsset={selectedAsset}
            technicalData={lineageData?.technicalData}
            semanticData={lineageData?.semanticData}
            onAssetClick={() => {}}
            onRelationshipClick={() => {}}
            onToggleFullScreen={() => {}}
            isFullScreen={false}
            forceLineageType={undefined}
          />
        </Suspense>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
}
