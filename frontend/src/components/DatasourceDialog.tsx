import { useState, useEffect, Suspense } from 'react';
import {
  Dialog, DialogContent, DialogActions, Button, FormControl,
  InputLabel, Select, MenuItem, Alert, Box, Typography,
  FormControlLabel, Switch, TextField
} from '@mui/material';
import ModalHeader from '@/components/ModalHeader';
import MonacoCodeEditor from './UnifiedSemanticBuilder/MonacoCodeEditor.lazy';
import { useQuery } from '@apollo/client';
import { Product, DataSource } from '../types';
import { GET_ALPHA_DATASOURCES } from '../graphql/queries/tenantQueries';


interface TenantProductDatasource {
  id: string;
  source_name: string;
  config: object;
  alpha_tenant_instance_id: string;
}

interface DatasourceDialogProps {
  open: boolean;
  product: Product | null;
  datasource: DataSource | null;
  onClose: () => void;
  onSave: (id: string, config: object, isActive: boolean, sourceName: string, isNew: boolean) => Promise<void>;
}

export const DatasourceDialog: React.FC<DatasourceDialogProps> = ({
  open, product, datasource, onClose, onSave
}) => {
  const [sourceName, setSourceName] = useState<string>('');
  const [selectedDatasourceId, setSelectedDatasourceId] = useState<string>('');
  const [config, setConfig] = useState<string>('');
  const [isActive, setIsActive] = useState<boolean>(true);
  const [error, setError] = useState<string>('');
  const [saving, setSaving] = useState<boolean>(false);

  const { data: alphaDatasourceData, loading: datasourcesLoading } = useQuery(GET_ALPHA_DATASOURCES);
  const alphaDatasources = alphaDatasourceData?.alpha_datasource || [];


  const isEditMode = !!datasource;
  const dialogTitle = isEditMode
    ? `Edit Datasource for ${product?.alpha_product?.product_name}`
    : `Add Datasource to ${product?.alpha_product?.product_name}`;

  useEffect(() => {
    if (open) {
      setError('');
      if (isEditMode && datasource) {
        setSourceName(datasource.source_name || '');
        setSelectedDatasourceId(datasource.alpha_tenant_instance_id || '');
        setConfig(JSON.stringify(datasource.config || {}, null, 2));
        setIsActive(datasource.is_active);
      } else {
        setSourceName('');
        setSelectedDatasourceId('');
        setConfig('');
        setIsActive(true);
      }
    }
  }, [open, datasource, isEditMode]);

  useEffect(() => {
    if (!isEditMode && selectedDatasourceId && alphaDatasources.length > 0) {
      const selectedDs = alphaDatasources.find(
        (ds: any) => ds.id === selectedDatasourceId
      );

      if (selectedDs) {
        // For new datasources from template, we might start with empty config or template config
        setConfig(JSON.stringify(selectedDs.config || {}, null, 2));
      }
    }
  }, [selectedDatasourceId, isEditMode, alphaDatasources]);


  const handleSave = async () => {
    setError('');
    if (!sourceName.trim()) {
      setError('Source Name is a required field.');
      return;
    }
    if (!isEditMode && !selectedDatasourceId) {
      setError('Please select a datasource type.');
      return;
    }
    let parsedConfig: object;
    try {
      parsedConfig = JSON.parse(config);
    } catch (err) {
      setError('The configuration is not valid JSON.');
      return;
    }
    setSaving(true);
    try {
      const idToSave = isEditMode ? datasource!.id : selectedDatasourceId;
      await onSave(idToSave, parsedConfig, isActive, sourceName, !isEditMode);
      onClose();
    } catch (err) {
      setError('Failed to save the datasource. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onClose={() => !saving && onClose()} fullWidth maxWidth="md">
      <ModalHeader title={dialogTitle} onClose={() => !saving && onClose()} />
      <DialogContent>
        <Box component="form" sx={{ display: 'flex', flexDirection: 'column', gap: 3, pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          
          <TextField
            autoFocus
            required
            margin="dense"
            id="source_name"
            label="Source Name"
            type="text"
            fullWidth
            variant="outlined"
            value={sourceName}
            onChange={(e) => setSourceName(e.target.value)}
            disabled={saving}
          />
          
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 2 }}>
            {!isEditMode ? (
              <FormControl fullWidth required>
                <InputLabel id="datasource-select-label">Datasource Type</InputLabel>
                <Select
                  labelId="datasource-select-label"
                  value={selectedDatasourceId}
                  label="Datasource Type"
                  onChange={(e) => setSelectedDatasourceId(e.target.value)}
                  disabled={saving || datasourcesLoading}
                >
                  {alphaDatasources.map((type: any) => (
                    <MenuItem key={type.id} value={type.id}>
                      {type.datasource_name} ({type.datasource_type})
                    </MenuItem>
                  ))}

                </Select>
              </FormControl>
            ) : (
              <Typography variant="h6">
                Type: {datasource?.alpha_datasource?.datasource_name}
              </Typography>
            )}
            <FormControlLabel
              control={<Switch checked={isActive} onChange={(e) => setIsActive(e.target.checked)} />}
              label="Active"
              sx={{ whiteSpace: 'nowrap' }}
              disabled={saving}
            />
          </Box>
          
          {(selectedDatasourceId || isEditMode) && (
          <Box>
            <Typography variant="subtitle1" sx={{mb: 1}}>Configuration</Typography>
            <Box border={1} borderColor="grey.400" borderRadius={1} sx={{ height: 400 }}>
              <Suspense fallback={<div>Loading editor...</div>}>
                  <div className="editor-wrapper-full editor-h-400">
                    <MonacoCodeEditor
                      value={typeof config === 'string' ? config : JSON.stringify(config, null, 2)}
                      language="json"
                      onChange={(val: string) => setConfig(val)}
                    />
                  </div>
              </Suspense>
            </Box>
        </Box>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => !saving && onClose()}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={saving || !sourceName.trim()}>
          {saving ? 'Saving...' : (isEditMode ? 'Update' : 'Add')}
        </Button>
      </DialogActions>
    </Dialog>
  );
};