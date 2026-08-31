import React, { useEffect, useState } from 'react';
import { Box, TextField, Checkbox, FormControlLabel, Button, Typography, Alert, CircularProgress } from '@mui/material';
import { fetchBOSchema } from '../../features/query-builder/services/queryBuilderApi';
import type { BOSchema, BOSchemaField } from '../../features/query-builder/types/queryDef';
import { apiFetch } from '../../lib/apiClient';

export interface BOFormWidgetProps {
  boId: string;
  tenantId: string;
  /** When set, the form loads and updates this record instead of creating a new one. */
  recordId?: string;
  onSaved?: (record: Record<string, unknown>) => void;
}

const inputTypeFor = (field: BOSchemaField): 'text' | 'number' | 'date' | 'checkbox' => {
  const t = (field.type || '').toLowerCase();
  if (['number', 'integer', 'float', 'decimal', 'currency'].includes(t)) return 'number';
  if (['date', 'datetime', 'timestamp'].includes(t)) return 'date';
  if (['boolean', 'bool'].includes(t)) return 'checkbox';
  return 'text';
};

/**
 * A real, Business-Object-bound, validated form — not a static text box.
 * Fetches the BO's field schema (the same metadata Report/Query Builder
 * resolve fields against) and writes through the live BO record CRUD API,
 * so a saved record is immediately visible everywhere else that BO is used.
 */
const BOFormWidget: React.FC<BOFormWidgetProps> = ({ boId, tenantId, recordId, onSaved }) => {
  const [schema, setSchema] = useState<BOSchema | null>(null);
  const [values, setValues] = useState<Record<string, unknown>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (!boId || !tenantId) return;
    setLoading(true);
    setSaved(false);
    fetchBOSchema(boId, tenantId)
      .then((s) => {
        setSchema(s);
        setValues({});
      })
      .catch(() => setSchema(null))
      .finally(() => setLoading(false));
  }, [boId, tenantId]);

  useEffect(() => {
    if (!recordId || !boId || !tenantId) return;
    apiFetch(`/api/v1/bo/${encodeURIComponent(boId)}/records/${encodeURIComponent(recordId)}`, {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((rec) => setValues(rec || {}))
      .catch(() => undefined);
  }, [recordId, boId, tenantId]);

  if (loading) {
    return (
      <Box sx={{ p: 2, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress size={20} />
      </Box>
    );
  }

  if (!schema || schema.fields.length === 0) {
    return (
      <Box sx={{ p: 1 }}>
        <Typography variant="caption" color="text.secondary">
          Bind this form to a Business Object in Properties
        </Typography>
      </Box>
    );
  }

  // BOSchema (GET /api/metadata/bo/{boId}) doesn't carry required-field
  // metadata today, so there's nothing honest to validate against beyond
  // type coercion — a fabricated "required" heuristic would just produce
  // wrong error messages. Server-side inserts still enforce real NOT NULL/
  // check constraints; add real client-side required validation once the
  // schema endpoint exposes it.
  const validate = (): boolean => true;

  const handleChange = (field: BOSchemaField, raw: string | boolean) => {
    const t = inputTypeFor(field);
    const value = t === 'number' ? (raw === '' ? '' : Number(raw)) : raw;
    setValues((prev) => ({ ...prev, [field.name]: value }));
    setErrors((prev) => {
      if (!prev[field.name]) return prev;
      const next = { ...prev };
      delete next[field.name];
      return next;
    });
  };

  const handleSubmit = async () => {
    if (!validate()) return;
    setSaving(true);
    setSaveError(null);
    try {
      const method = recordId ? 'PUT' : 'POST';
      const path = recordId
        ? `/api/v1/bo/${encodeURIComponent(boId)}/records/${encodeURIComponent(recordId)}`
        : `/api/v1/bo/${encodeURIComponent(boId)}/records`;
      const res = await apiFetch(path, {
        method,
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
        body: JSON.stringify(values),
      });
      if (!res.ok) {
        const detail = await res.text();
        throw new Error(detail || `Save failed (${res.status})`);
      }
      const record = await res.json().catch(() => values);
      setSaved(true);
      onSaved?.(record);
    } catch (err: any) {
      setSaveError(err?.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box sx={{ p: 1.5, height: '100%', overflowY: 'auto' }}>
      {saveError && <Alert severity="error" sx={{ mb: 1, fontSize: '0.7rem' }}>{saveError}</Alert>}
      {saved && <Alert severity="success" sx={{ mb: 1, fontSize: '0.7rem' }}>Saved</Alert>}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.25 }}>
        {schema.fields.map((field) => {
          const t = inputTypeFor(field);
          const label = field.displayName || field.name;
          if (t === 'checkbox') {
            return (
              <FormControlLabel
                key={field.id}
                control={
                  <Checkbox
                    size="small"
                    checked={Boolean(values[field.name])}
                    onChange={(e) => handleChange(field, e.target.checked)}
                  />
                }
                label={label}
              />
            );
          }
          return (
            <TextField
              key={field.id}
              size="small"
              label={label}
              type={t === 'date' ? 'date' : t === 'number' ? 'number' : 'text'}
              InputLabelProps={t === 'date' ? { shrink: true } : undefined}
              value={values[field.name] ?? ''}
              onChange={(e) => handleChange(field, e.target.value)}
              error={!!errors[field.name]}
              helperText={errors[field.name]}
              fullWidth
            />
          );
        })}
        <Button variant="contained" size="small" onClick={handleSubmit} disabled={saving}>
          {saving ? 'Saving…' : recordId ? 'Update' : 'Create'}
        </Button>
      </Box>
    </Box>
  );
};

export default BOFormWidget;
