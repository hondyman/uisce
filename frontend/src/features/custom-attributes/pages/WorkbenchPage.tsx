import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  Divider,
  FormControl,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { Link as RouterLink, useParams, useSearchParams } from 'react-router-dom';
import {
  createAttribute,
  deleteAttribute,
  listAttributes,
  previewAttributes,
  updateAttribute,
} from '../api';
import { AddFieldDialog } from '../components/AddFieldDialog';
import { DataPreviewGrid } from '../components/DataPreviewGrid';
import { FieldEditor } from '../components/FieldEditor';
import { FieldList } from '../components/FieldList';
import type { AttributeDef, CreateAttributeInput, PreviewResponse, UpdateAttributeInput } from '../types';

const ACCOUNT_TYPES = [
  'RETAIL',
  'INSTITUTIONAL',
  'CUSTODY',
  'PENSION',
  'RETIREMENT',
  'TRUST',
  'SMA',
  'UMA',
  'FAMILY_OFFICE',
  'PRIVATE_WEALTH',
  'INSURANCE',
  'ENDOWMENT',
];

export default function WorkbenchPage() {
  const { entityType: entityTypeParam } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const entityType = (entityTypeParam || '').toUpperCase();
  const tableRef = searchParams.get('table_ref') || guessTableRef(entityType);
  const accountType = searchParams.get('account_type') || '';

  const [fields, setFields] = useState<AttributeDef[]>([]);
  const [selected, setSelected] = useState<AttributeDef | null>(null);
  const [preview, setPreview] = useState<PreviewResponse | null>(null);
  const [loadingFields, setLoadingFields] = useState(true);
  const [loadingPreview, setLoadingPreview] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [editorError, setEditorError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [addOpen, setAddOpen] = useState(false);

  const refreshFields = useCallback(async () => {
    if (!entityType) return;
    setLoadingFields(true);
    setError(null);
    try {
      const data = await listAttributes(entityType, {
        accountType: entityType === 'ACCOUNT' ? accountType || undefined : undefined,
      });
      setFields(data);
      setSelected((prev) => {
        if (!prev) return data[0] || null;
        return data.find((f) => f.id === prev.id) || data[0] || null;
      });
    } catch (e: any) {
      setError(e?.message || 'Failed to load definitions');
    } finally {
      setLoadingFields(false);
    }
  }, [entityType, accountType]);

  const refreshPreview = useCallback(async () => {
    if (!entityType) return;
    setLoadingPreview(true);
    setPreviewError(null);
    try {
      const data = await previewAttributes({
        entityType,
        tableRef,
        limit: 50,
        accountType: entityType === 'ACCOUNT' ? accountType || undefined : undefined,
      });
      setPreview(data);
    } catch (e: any) {
      setPreview(null);
      setPreviewError(
        e?.message ||
          'Preview unavailable. Confirm the table exists and safe cast helpers are installed.',
      );
    } finally {
      setLoadingPreview(false);
    }
  }, [entityType, tableRef, accountType]);

  useEffect(() => {
    refreshFields();
  }, [refreshFields]);

  useEffect(() => {
    refreshPreview();
  }, [refreshPreview]);

  const handleCreate = async (input: CreateAttributeInput) => {
    await createAttribute(input);
    await refreshFields();
    await refreshPreview();
  };

  const handleSave = async (id: string, patch: UpdateAttributeInput) => {
    setSaving(true);
    setEditorError(null);
    try {
      await updateAttribute(id, patch);
      await refreshFields();
      await refreshPreview();
    } catch (e: any) {
      setEditorError(e?.message || 'Save failed');
      throw e;
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Deactivate this field definition? Existing values are left untouched.')) {
      return;
    }
    setSaving(true);
    setEditorError(null);
    try {
      await deleteAttribute(id);
      setSelected(null);
      await refreshFields();
      await refreshPreview();
    } catch (e: any) {
      setEditorError(e?.message || 'Deactivate failed');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box sx={{ height: 'calc(100vh - 64px)', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ px: 2, py: 1.5, display: 'flex', alignItems: 'center', gap: 2 }}>
        <Button
          component={RouterLink}
          to="/catalog/custom-fields"
          startIcon={<ArrowBackIcon />}
          size="small"
        >
          Entities
        </Button>
        <Typography variant="h6">Custom Fields · {entityType}</Typography>
        <Chip size="small" label={tableRef} />
        {entityType === 'ACCOUNT' && (
          <FormControl size="small" sx={{ minWidth: 180 }}>
            <InputLabel>Account type</InputLabel>
            <Select
              label="Account type"
              value={accountType}
              onChange={(e) => {
                const next = new URLSearchParams(searchParams);
                if (e.target.value) next.set('account_type', String(e.target.value));
                else next.delete('account_type');
                setSearchParams(next);
              }}
            >
              <MenuItem value="">All types</MenuItem>
              {ACCOUNT_TYPES.map((t) => (
                <MenuItem key={t} value={t}>
                  {t}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        )}
        <Button size="small" disabled variant="outlined" sx={{ ml: 'auto' }}>
          Validate (soon)
        </Button>
        <Button size="small" variant="outlined" onClick={() => { refreshFields(); refreshPreview(); }}>
          Refresh
        </Button>
      </Box>
      <Divider />

      {error && (
        <Alert severity="error" sx={{ m: 2 }}>
          {error}
        </Alert>
      )}

      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          display: 'grid',
          gridTemplateColumns: '280px 1fr 320px',
          gap: 0,
        }}
      >
        <Paper square elevation={0} sx={{ borderRight: 1, borderColor: 'divider', minHeight: 0 }}>
          {loadingFields ? (
            <Typography sx={{ p: 2 }} color="text.secondary">
              Loading fields…
            </Typography>
          ) : (
            <FieldList
              fields={fields}
              selectedId={selected?.id}
              onSelect={setSelected}
              onAdd={() => setAddOpen(true)}
            />
          )}
        </Paper>

        <Paper square elevation={0} sx={{ minHeight: 0, overflow: 'hidden' }}>
          <DataPreviewGrid
            preview={preview}
            loading={loadingPreview}
            error={previewError}
            highlightedField={selected?.field_cd}
            businessObjectPath="/business-objects"
          />
        </Paper>

        <Paper square elevation={0} sx={{ borderLeft: 1, borderColor: 'divider', minHeight: 0 }}>
          <FieldEditor
            field={selected}
            saving={saving}
            error={editorError}
            onSave={handleSave}
            onDelete={handleDelete}
          />
        </Paper>
      </Box>

      <AddFieldDialog
        open={addOpen}
        entityType={entityType}
        tableRef={tableRef}
        onClose={() => setAddOpen(false)}
        onCreate={handleCreate}
      />
    </Box>
  );
}

function guessTableRef(entityType: string): string {
  const map: Record<string, string> = {
    PRODUCT: 'mdm.product',
    COUNTERPARTY: 'mdm.counterparty',
    BENCHMARK: 'mdm.benchmark_master',
    CALENDAR: 'mdm.calendar_master',
    PRICE: 'mdm.price',
    CA_EVENT: 'mdm.ca_event',
    ACCOUNT: 'mdm.account_master',
    ORDER: 'foffice.order',
  };
  return map[entityType] || `mdm.${entityType.toLowerCase()}`;
}
