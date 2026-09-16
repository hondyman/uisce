import React, { useEffect, useState } from 'react';
import { Box, TextField, Checkbox, FormControlLabel, Button, Typography, Alert, CircularProgress, MenuItem } from '@mui/material';
import { fetchBOSchema } from '../../features/query-builder/services/queryBuilderApi';
import type { BOSchema, BOSchemaField } from '../../features/query-builder/types/queryDef';
import { apiFetch } from '../../lib/apiClient';
import type { FieldLayoutEntry, FieldOverrideEntry } from '../../pages/page-studio/FormFieldsDesigner';
import { labelSxFromStyle } from '../../pages/page-studio/FormFieldsDesigner';

const GRID_COLS = 12;

/** One selectable option for a reference (foreign-key) field's dropdown. */
interface ReferenceOption {
  value: string;
  label: string;
}

export interface BOFormWidgetProps {
  boId: string;
  tenantId: string;
  /** When set, the form loads and updates this record instead of creating a new one. */
  recordId?: string;
  onSaved?: (record: Record<string, unknown>) => void;
  /** Per-field grid position/size and label/required/hidden overrides, set via
   * FormFieldsDesigner.tsx (Page Studio's Design-mode field editor) - keeps
   * the real, data-bound form's layout in sync with what was designed. */
  fieldLayout?: Record<string, FieldLayoutEntry>;
  fieldOverrides?: Record<string, FieldOverrideEntry>;
}

const inputTypeFor = (field: BOSchemaField): 'text' | 'number' | 'date' | 'checkbox' | 'select' => {
  // A field backed by a foreign key to another Business Object (referenceBoId)
  // gets a dropdown of that BO's real records, not a free-text id the user
  // would otherwise have to type in by hand - checked first since a reference
  // field's own `type` is usually just "string"/"uuid".
  if (field.referenceBoId) return 'select';
  const t = (field.type || '').toLowerCase();
  if (['number', 'integer', 'float', 'decimal', 'currency'].includes(t)) return 'number';
  if (['date', 'datetime', 'timestamp'].includes(t)) return 'date';
  if (['boolean', 'bool'].includes(t)) return 'checkbox';
  return 'text';
};

/** Best-effort human label for a referenced record - prefers a name-ish field over the raw id. */
const labelForRecord = (record: Record<string, unknown>): string => {
  for (const key of ['name', 'display_name', 'displayName', 'label', 'title']) {
    const v = record[key];
    if (typeof v === 'string' && v.trim()) return v;
  }
  return String(record.id ?? '');
};

/**
 * A real, Business-Object-bound, validated form — not a static text box.
 * Fetches the BO's field schema (the same metadata Report/Query Builder
 * resolve fields against) and writes through the live BO record CRUD API,
 * so a saved record is immediately visible everywhere else that BO is used.
 *
 * Endpoint contract (matches backend/internal/api/bo_crud_handler.go):
 *   GET    /api/bo/{boKey}/schema                       -> BOSchema
 *   GET    /api/bo/{boKey}/records/{recordId}           -> single record
 *   POST   /api/bo/{boKey}/records                      -> create
 *   PUT    /api/bo/{boKey}/records/{recordId}           -> update
 * The earlier `/api/v1/bo/...` paths 404'd because the real router is
 * mounted at `/bo/...` inside the `/api` group, not `/v1/bo/...`.
 */
const BOFormWidget: React.FC<BOFormWidgetProps> = ({ boId, tenantId, recordId, onSaved, fieldLayout, fieldOverrides }) => {
  const [schema, setSchema] = useState<BOSchema | null>(null);
  const [values, setValues] = useState<Record<string, unknown>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [refOptions, setRefOptions] = useState<Record<string, ReferenceOption[]>>({});

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

  // For every reference field, load the referenced BO's records once to
  // populate its dropdown - a small, static picklist is the common case for
  // these (accounts, statuses, owning entities), so no search/paging here.
  useEffect(() => {
    if (!schema) return;
    const refFields = schema.fields.filter((f) => f.referenceBoId);
    if (refFields.length === 0) return;
    let cancelled = false;
    refFields.forEach((field) => {
      const refBoId = field.referenceBoId!;
      apiFetch(`/api/bo/${encodeURIComponent(refBoId)}/records?limit=200`, { headers: { 'X-Tenant-ID': tenantId } })
        .then((r) => r.json())
        .then((data) => {
          if (cancelled) return;
          const rows: Record<string, unknown>[] = Array.isArray(data) ? data : data?.rows || data?.records || [];
          setRefOptions((prev) => ({
            ...prev,
            [field.name]: rows.map((row) => ({ value: String(row.id ?? ''), label: labelForRecord(row) })),
          }));
        })
        .catch(() => { if (!cancelled) setRefOptions((prev) => ({ ...prev, [field.name]: [] })); });
    });
    return () => { cancelled = true; };
  }, [schema, tenantId]);

  useEffect(() => {
    if (!recordId || !boId || !tenantId || !schema) return;
    apiFetch(`/api/bo/${encodeURIComponent(boId)}/records/${encodeURIComponent(recordId)}`, {
      headers: { 'X-Tenant-ID': tenantId },
    })
      .then((r) => r.json())
      .then((rec) => {
        // The record comes back keyed by physical column (avg_price,
        // order_type, ...) - the same raw shape bo_crud_handler.go's
        // SELECT * returns - but every field below reads/writes
        // values[field.name], the semantic field name (AveragePrice,
        // OrderType, ...). Remap through each field's physicalColumn so a
        // fetched record actually populates the form instead of every
        // input rendering empty.
        const mapped: Record<string, unknown> = {};
        for (const field of schema.fields) {
          const col = field.physicalColumn || field.name;
          if (rec && Object.prototype.hasOwnProperty.call(rec, col)) {
            mapped[field.name] = rec[col];
          }
        }
        setValues(mapped);
      })
      .catch(() => undefined);
  }, [recordId, boId, tenantId, schema]);

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

  // Same ordering FormFieldsDesigner.tsx's design-mode grid uses, so a
  // record's real form matches what was designed - fields without an
  // explicit order sort after every explicitly ordered one, in their
  // original schema order (stable sort).
  const orderedFields = [...schema.fields].sort((a, b) => {
    const oa = fieldLayout?.[a.name]?.order ?? 999;
    const ob = fieldLayout?.[b.name]?.order ?? 999;
    return oa - ob;
  });

  // The new schema endpoint carries `required: true` per field, sourced from
  // business_object_fields.is_required and binding_requirement='REQUIRED'
  // - so client-side required validation is now honest, not a heuristic.
  // Server-side still enforces NOT NULL/check constraints on submit (the
  // server is always the source of truth); this client check is purely a
  // UX nicety to surface the problem before round-tripping.
  const validate = (): boolean => {
    const nextErrors: Record<string, string> = {};
    let ok = true;
    for (const field of schema.fields) {
      if (!field.required) continue;
      const v = values[field.name];
      if (v === undefined || v === null || v === '') {
        nextErrors[field.name] = `${field.displayName || field.name} is required`;
        ok = false;
      }
    }
    setErrors(nextErrors);
    return ok;
  };

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
        ? `/api/bo/${encodeURIComponent(boId)}/records/${encodeURIComponent(recordId)}`
        : `/api/bo/${encodeURIComponent(boId)}/records`;
      // Reverse of the fetch-side remap: bo_crud_handler.go's writable-column
      // allowlist checks against physical column names, not semantic field
      // names, so `values` (keyed by field.name) must go back through
      // physicalColumn before it's sent.
      const payload: Record<string, unknown> = {};
      for (const field of schema.fields) {
        if (!Object.prototype.hasOwnProperty.call(values, field.name)) continue;
        payload[field.physicalColumn || field.name] = values[field.name];
      }
      const res = await apiFetch(path, {
        method,
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
        body: JSON.stringify(payload),
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
      <Box sx={{ display: 'grid', gridTemplateColumns: `repeat(${GRID_COLS}, 1fr)`, gap: 1.25 }}>
        {orderedFields.map((field) => {
          if (fieldOverrides?.[field.name]?.hidden) return null;
          const t = inputTypeFor(field);
          const label = fieldOverrides?.[field.name]?.label || field.displayName || field.name;
          const colSpan = Math.min(GRID_COLS, Math.max(1, fieldLayout?.[field.name]?.colSpan ?? GRID_COLS));
          const gridColumn = `span ${colSpan}`;
          const labelSx = labelSxFromStyle(fieldOverrides?.[field.name]?.style);
          if (t === 'checkbox') {
            return (
              <FormControlLabel
                key={field.id}
                sx={{ gridColumn }}
                control={
                  <Checkbox
                    size="small"
                    checked={Boolean(values[field.name])}
                    onChange={(e) => handleChange(field, e.target.checked)}
                  />
                }
                label={<Typography component="span" sx={labelSx}>{label}</Typography>}
              />
            );
          }
          if (t === 'select') {
            const options = refOptions[field.name] || [];
            return (
              <TextField
                key={field.id}
                select
                size="small"
                sx={{ gridColumn }}
                label={label}
                InputLabelProps={{ sx: labelSx }}
                value={values[field.name] ?? ''}
                onChange={(e) => handleChange(field, e.target.value)}
                error={!!errors[field.name]}
                helperText={errors[field.name] || (options.length === 0 ? 'Loading options…' : undefined)}
                fullWidth
              >
                {options.map((opt) => <MenuItem key={opt.value} value={opt.value}>{opt.label}</MenuItem>)}
              </TextField>
            );
          }
          return (
            <TextField
              key={field.id}
              size="small"
              sx={{ gridColumn }}
              label={label}
              type={t === 'date' ? 'date' : t === 'number' ? 'number' : 'text'}
              InputLabelProps={{ sx: labelSx, ...(t === 'date' ? { shrink: true } : undefined) }}
              value={values[field.name] ?? ''}
              onChange={(e) => handleChange(field, e.target.value)}
              error={!!errors[field.name]}
              helperText={errors[field.name]}
              fullWidth
            />
          );
        })}
        <Button variant="contained" size="small" onClick={handleSubmit} disabled={saving} sx={{ gridColumn: `span ${GRID_COLS}`, justifySelf: 'start' }}>
          {saving ? 'Saving…' : recordId ? 'Update' : 'Create'}
        </Button>
      </Box>
    </Box>
  );
};

export default BOFormWidget;
