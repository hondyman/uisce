import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Box, TextField, Checkbox, FormControlLabel, Button, Typography, Alert, CircularProgress, MenuItem, Accordion, AccordionSummary, AccordionDetails } from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { fetchBOSchema } from '../../features/query-builder/services/queryBuilderApi';
import type { BOSchema, BOSchemaField } from '../../features/query-builder/types/queryDef';
import { apiFetch } from '../../lib/apiClient';
import type { FieldLayoutEntry, FieldOverrideEntry } from '../../pages/page-studio/FormFieldsDesigner';
import { labelSxFromStyle } from '../../pages/page-studio/FormFieldsDesigner';
import { usePresentationRuntime } from '../../pages/page-studio/PresentationRuntime';

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
  /** Display-only lock from presentation events — does not change BO write policy. */
  readOnly?: boolean;
  componentId?: string;
  onRecordLoaded?: (values: Record<string, unknown>) => void;
  onFieldEdit?: (fieldName: string, value: unknown, values: Record<string, unknown>) => void;
  onFieldChange?: (fieldName: string, value: unknown, values: Record<string, unknown>) => void;
}

const inputTypeFor = (field: BOSchemaField): 'text' | 'number' | 'date' | 'checkbox' | 'select' => {
  // A field backed by a foreign key to another Business Object (referenceBoId)
  // gets a dropdown of that BO's real records, not a free-text id the user
  // would otherwise have to type in by hand - checked first since a reference
  // field's own `type` is usually just "string"/"uuid".
  if (field.referenceBoId || (field.enumValues && field.enumValues.length > 0)) return 'select';
  const t = (field.type || '').toLowerCase();
  if (['number', 'integer', 'float', 'decimal', 'currency'].includes(t)) return 'number';
  if (['date', 'datetime', 'timestamp'].includes(t)) return 'date';
  if (['boolean', 'bool'].includes(t)) return 'checkbox';
  return 'text';
};

/** Best-effort human label for a referenced record - prefers a name-ish field over the raw id. */
const labelForRecord = (record: Record<string, unknown>): string => {
  for (const key of ['name', 'display_name', 'displayName', 'label', 'title', 'acct_cd', 'bkr_cd', 'ticker', 'symbol', 'sec_id']) {
    const v = record[key] ?? record[key.toUpperCase()];
    if (v !== undefined && v !== null && String(v).trim()) {
      const name = record.name ?? record.NAME;
      if (name && key !== 'name') return `${v} — ${name}`;
      return String(v);
    }
  }
  return String(record.id ?? record.ID ?? '');
};

const todayISODate = () => new Date().toISOString().slice(0, 10);

const dateInputValue = (raw: unknown): string => {
  if (raw === null || raw === undefined || raw === '') return '';
  const s = String(raw);
  if (/^\d{4}-\d{2}-\d{2}/.test(s)) return s.slice(0, 10);
  return s;
};

const sectionForField = (field: BOSchemaField, layout?: Record<string, FieldLayoutEntry>): string => {
  if (layout?.[field.name]?.section) return layout[field.name].section as string;
  const k = (field.physicalColumn || field.name).toLowerCase().replace(/_/g, '');
  if (['status', 'side', 'ordertype', 'timeinforce', 'tif', 'secid', 'securityid', 'securitiesid'].includes(k)) return 'Identity';
  if (['targetqty', 'executedqty', 'leavesqty', 'limitprice', 'avgprice', 'averageprice', 'price', 'quantity'].includes(k)) return 'Economics';
  if ((field.type || '').toLowerCase().includes('date') || k.includes('date')) return 'Dates';
  return 'Details';
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
const BOFormWidget: React.FC<BOFormWidgetProps> = ({
  boId, tenantId, recordId, onSaved, fieldLayout, fieldOverrides,
  readOnly, componentId, onRecordLoaded, onFieldEdit, onFieldChange,
}) => {
  const [schema, setSchema] = useState<BOSchema | null>(null);
  const [values, setValues] = useState<Record<string, unknown>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [refOptions, setRefOptions] = useState<Record<string, ReferenceOption[]>>({});
  const { overlayFor } = usePresentationRuntime();
  const navigate = useNavigate();
  const { slug } = useParams<{ slug?: string }>();

  useEffect(() => {
    if (!boId || !tenantId) return;
    setLoading(true);
    setSaved(false);
    fetchBOSchema(boId, tenantId)
      .then((s) => {
        setSchema(s);
        if (!recordId) {
          const defaults: Record<string, unknown> = {};
          for (const f of s.fields) {
            if (f.defaultValue === 'today') defaults[f.name] = todayISODate();
          }
          setValues(defaults);
        } else {
          setValues({});
        }
      })
      .catch(() => setSchema(null))
      .finally(() => setLoading(false));
  }, [boId, tenantId, recordId]);

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
          const valueKey = field.referenceValueField || 'id';
          setRefOptions((prev) => ({
            ...prev,
            [field.name]: rows.map((row) => {
              const raw = row[valueKey] ?? row[valueKey.toUpperCase()] ?? row.id ?? row.ID ?? '';
              return { value: String(raw), label: labelForRecord(row) };
            }),
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
            const raw = rec[col];
            mapped[field.name] = inputTypeFor(field) === 'date' ? dateInputValue(raw) : raw;
          }
        }
        setValues(mapped);
        onRecordLoaded?.(mapped);
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
    const next = { ...values, [field.name]: value };
    setValues(next);
    onFieldEdit?.(field.name, value, next);
    setErrors((prev) => {
      if (!prev[field.name]) return prev;
      const next = { ...prev };
      delete next[field.name];
      return next;
    });
  };

  const handleBlur = (field: BOSchemaField) => {
    onFieldChange?.(field.name, values[field.name], values);
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
      const record = await res.json().catch(() => values) as Record<string, unknown>;
      setSaved(true);
      onSaved?.(record);
      const newId = record?.id ?? record?.ID;
      if (!recordId && slug && newId) {
        navigate(`/pages/${slug}/${newId}`, { replace: true });
      }
    } catch (err: any) {
      setSaveError(err?.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const visibleFields = orderedFields.filter((field) => {
    const pres = componentId ? overlayFor(componentId, field.name) : undefined;
    if (fieldOverrides?.[field.name]?.hidden || pres?.hidden) return false;
    const k = (field.physicalColumn || field.name).toLowerCase();
    if (!recordId && (k === 'id' || k === 'created_at' || k === 'updated_at' || k === 'createdat' || k === 'updatedat')) return false;
    return true;
  });
  const grouped = new Map<string, BOSchemaField[]>();
  for (const field of visibleFields) {
    const name = sectionForField(field, fieldLayout);
    const list = grouped.get(name) || [];
    list.push(field);
    grouped.set(name, list);
  }
  const sectionList = Array.from(grouped.entries());

  const renderField = (field: BOSchemaField) => {
    const pres = componentId ? overlayFor(componentId, field.name) : undefined;
    const t = inputTypeFor(field);
    const label = pres?.label || fieldOverrides?.[field.name]?.label || field.displayName || field.name;
    const colSpan = Math.min(GRID_COLS, Math.max(1, fieldLayout?.[field.name]?.colSpan ?? 6));
    const gridColumn = `span ${colSpan}`;
    const labelSx = labelSxFromStyle({
      ...fieldOverrides?.[field.name]?.style,
      ...(pres?.style?.color ? { color: pres.style.color } : {}),
      ...(pres?.style?.fontWeight ? { bold: pres.style.fontWeight === '700' || pres.style.fontWeight === 'bold' } : {}),
    });
    const locked = readOnly || pres?.readOnly;
    if (t === 'checkbox') {
      return (
        <FormControlLabel
          key={field.id}
          sx={{ gridColumn }}
          control={
            <Checkbox
              size="small"
              checked={Boolean(values[field.name])}
              disabled={locked}
              onChange={(e) => handleChange(field, e.target.checked)}
              onBlur={() => handleBlur(field)}
            />
          }
          label={<Typography component="span" sx={labelSx}>{label}</Typography>}
        />
      );
    }
    if (t === 'select') {
      const options = (field.enumValues && field.enumValues.length > 0)
        ? field.enumValues.map((e) => ({ value: e.value, label: e.label }))
        : (refOptions[field.name] || []);
      const loadingRefs = !field.enumValues?.length && field.referenceBoId && options.length === 0;
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
          onBlur={() => handleBlur(field)}
          disabled={locked}
          error={!!errors[field.name]}
          helperText={errors[field.name] || (loadingRefs ? 'Loading options…' : undefined)}
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
        value={t === 'date' ? dateInputValue(values[field.name]) : (values[field.name] ?? '')}
        onChange={(e) => handleChange(field, e.target.value)}
        onBlur={() => handleBlur(field)}
        disabled={locked}
        error={!!errors[field.name]}
        helperText={errors[field.name]}
        fullWidth
      />
    );
  };

  return (
    <Box sx={{ p: 1.5, height: '100%', overflowY: 'auto' }}>
      {saveError && <Alert severity="error" sx={{ mb: 1, fontSize: '0.7rem' }}>{saveError}</Alert>}
      {saved && <Alert severity="success" sx={{ mb: 1, fontSize: '0.7rem' }}>Saved</Alert>}
      {sectionList.map(([sectionName, fields]) => (
        <Accordion key={sectionName} defaultExpanded disableGutters sx={{ mb: 1, '&:before': { display: 'none' }, boxShadow: 'none', border: '1px solid', borderColor: 'divider' }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight={700}>{sectionName}</Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Box sx={{ display: 'grid', gridTemplateColumns: `repeat(${GRID_COLS}, 1fr)`, gap: 1.25 }}>
              {fields.map(renderField)}
            </Box>
          </AccordionDetails>
        </Accordion>
      ))}
      <Button variant="contained" size="small" onClick={handleSubmit} disabled={saving || readOnly} sx={{ mt: 1 }}>
        {saving ? 'Saving…' : recordId ? 'Update' : 'Create'}
      </Button>
    </Box>
  );
};

export default BOFormWidget;
