import React, { useMemo, useState, useEffect, useCallback } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, TextField, MenuItem, Switch, FormControlLabel, Chip, Box, IconButton, FormHelperText } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import useFormStore from '../store/formStore';
import { devDebug, devError } from '../utils/devLogger';
import KeyValueEditor from './KeyValueEditor';

interface EntityRecord {
  entity_name: string;
  display_name: string;
  default_schema?: any;
  subtypes?: any[];
}

interface SemanticTerm {
  id: string;
  name: string;
  label: string;
  description?: string;
  type: 'text' | 'number' | 'date' | 'select' | 'json' | 'switch' | 'tag-input';
  options?: string[];
  required?: boolean;
  subtypes?: any[];
}

interface Props {
  open: boolean;
  onClose: () => void;
  // onSubmit should return a Promise that may resolve with { validation_errors } when server-side validation fails
  onSubmit: (payload: any) => Promise<any>;
  initialValues?: any | null;
  serverErrors?: Record<string, string> | null;
  onClearServerErrors?: () => void;
}

const DynamicEntityForm: React.FC<Props> = ({ open, onClose, onSubmit, initialValues = null, serverErrors = null, onClearServerErrors }) => {
  const [entity, setEntity] = useState<string | null>(null);
  const [subtype, setSubtype] = useState<string | null>(null);
  const [values, setValues] = useState<Record<string, any>>({});
  const [tagInput, setTagInput] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [semanticTerms, setSemanticTerms] = useState<Record<string, SemanticTerm>>({});
  const [entities, setEntities] = useState<EntityRecord[]>([]);

  // Fetch entities from API
  useEffect(() => {
    const fetchEntities = async () => {
      try {
        const res = await fetch('/api/entity_registry');

        // If server returned an error status, capture body for diagnostics
        if (!res.ok) {
          const text = await res.text().catch(() => '');
          devError('[DynamicEntityForm] Failed to fetch entities - non-OK status', { status: res.status, statusText: res.statusText, body: text });
          throw new Error(`Failed to fetch entity registry: ${res.status} ${res.statusText}`);
        }

        const contentType = res.headers.get('content-type') || '';
        if (!contentType.includes('application/json')) {
          // If we unexpectedly received HTML (vite index.html) or other text, surface a clearer error
          const text = await res.text().catch(() => '');
          devError('[DynamicEntityForm] Expected JSON but received:', { contentType, bodyPreview: text.substring(0, 1024) });
          throw new Error('Expected JSON response from /api/entity_registry but received HTML or non-JSON. Check backend/API Gateway routing.');
        }

        const data = await res.json();
        const rows = data?.entity_registry || [];
        setEntities(rows);
      } catch (err) {
  devError('Failed to fetch entities:', err);
        setEntities([]);
      }
    };
    fetchEntities();
  }, []);

  const selectedEntity = entities.find(e => e.entity_name === entity);

  const fetchSemanticTerm = useCallback(async (termId: string) => {
    // Avoid re-fetching
    if (semanticTerms[termId]) return semanticTerms[termId];

    try {
      // In a real app, this would be a dedicated API call: /api/semantic-terms/${termId}
      // For this demo, we'll mock the fetch based on your schema.
  devDebug(`[DynamicForm] Fetching metadata for semantic term: ${termId}`);
      const mockTerms: Record<string, SemanticTerm> = {
        'st_trade_ticker': { id: 'st_trade_ticker', name: 'security_ticker', label: 'Ticker', description: 'The stock symbol for the security.', type: 'text', required: true },
        'st_trade_date': { id: 'st_trade_date', name: 'trade_date', label: 'Trade Date', description: 'The date the trade was executed.', type: 'date', required: true },
        'st_trade_side': { id: 'st_trade_side', name: 'side', label: 'Side', description: 'Whether the trade was a buy or a sell.', type: 'select', options: ['Buy', 'Sell'], required: true },
        'st_trade_quantity': { id: 'st_trade_quantity', name: 'quantity', label: 'Quantity', description: 'The number of shares traded.', type: 'number', required: true },
        'st_trade_price': { id: 'st_trade_price', name: 'price', label: 'Price', description: 'The price per share.', type: 'number', required: true },
        'st_trade_commission': { id: 'st_trade_commission', name: 'commission', label: 'Commission', description: 'Fees paid for the execution.', type: 'number', required: false },
      };
      const term = mockTerms[termId];
      if (term) {
        setSemanticTerms(prev => ({ ...prev, [termId]: term }));
        return term;
      }
    } catch (err) {
      devError(`Failed to fetch semantic term ${termId}:`, err);
    }
    return null;
  }, [semanticTerms]);

  const visibleFields: Array<any> = useMemo(() => {
    if (!selectedEntity) return [] as any[];
    const schema = selectedEntity.default_schema?.[selectedEntity.entity_name] || selectedEntity.default_schema;
    if (!schema || !schema.fields) return [];

    const allFields = schema.fields.map((field: any) => {
      if (field.semantic_term_id && semanticTerms[field.semantic_term_id]) {
        // If metadata is fetched, use it
        return { ...semanticTerms[field.semantic_term_id], key: field.key };
      }
      // Fallback to the hardcoded definition in the schema
      return field;
    });

    return allFields.filter((field: any) => {
      if (!field.condition) {
        return true; // Always show fields without a condition
      }
      try {
        // Safely evaluate the condition against the current form values
        // eslint-disable-next-line no-new-func
        const conditionFunc = new Function('values', `return ${field.condition}`);
        return conditionFunc(values);
        } catch (e) {
        devError(`Error evaluating field condition "${field.condition}":`, e);
        return false; // Hide field if condition is malformed
      }
    });
  }, [selectedEntity, semanticTerms, values]);

  const handleChange = (key: string, value: any) => {
    setValues(v => ({ ...v, [key]: value }));
    // clear error on change
    setErrors(e => {
      const next = { ...e };
      delete next[key];
      return next;
    });
  };

  // When the entity changes, fetch all required semantic terms
  useEffect(() => {
    if (selectedEntity) {
      const schema = selectedEntity.default_schema?.[selectedEntity.entity_name];
      if (schema && schema.fields) {
        schema.fields.forEach((field: any) => {
          if (field.semantic_term_id) {
            fetchSemanticTerm(field.semantic_term_id);
          }
        });
      }
    }
  }, [selectedEntity, fetchSemanticTerm]);

  // persist draft in store so accidental modal close doesn't lose state
  const setDraft = useFormStore(state => state.setDraft);
  const draft = useFormStore(state => state.draft);

  useEffect(() => {
    if (open) {
      if (initialValues) {
        // populate
        setEntity(initialValues.entity || initialValues.entity_type || null);
        setSubtype(initialValues.type || initialValues.subtype || null);
        setValues(initialValues || {});
      } else if (draft) {
        // restore draft
        setValues(draft);
      }
    } else {
      // when closing, persist draft
      setDraft(values);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const handleAddTag = (key: string) => {
    if (!tagInput) return;
    const existing = values[key] || [];
    // de-duplicate and trim
    const cleaned = tagInput.trim();
    if (cleaned === '') return;
    if (existing.includes(cleaned)) {
      setTagInput('');
      return;
    }
    handleChange(key, [...existing, cleaned]);
    setTagInput('');
  };

  const handleRemoveTag = (key: string, idx: number) => {
    const existing = values[key] || [];
    const next = [...existing.slice(0, idx), ...existing.slice(idx+1)];
    handleChange(key, next);
  };

  const submit = async () => {
    // Basic validation
    const nextErrors: Record<string, string> = {};
    visibleFields.forEach((f: any) => {
      if (f.required) {
        const v = values[f.key];
        if (v === undefined || v === null || (typeof v === 'string' && v.trim() === '') || (Array.isArray(v) && v.length === 0)) {
          nextErrors[f.key] = 'Required';
        }
      }
      if (f.type === 'number' && values[f.key] !== undefined && values[f.key] !== null) {
        if (Number.isNaN(Number(values[f.key]))) nextErrors[f.key] = 'Must be a number';
      }
      if (f.type === 'select' && f.options && values[f.key] && !f.options.includes(values[f.key])) {
        nextErrors[f.key] = 'Invalid selection';
      }

      // Apply dynamic validation rules from the entity schema
      const validationRules = selectedEntity?.default_schema?.[selectedEntity.entity_name]?.validation_rules?.[f.key];
      if (validationRules) {
        const value = values[f.key];
        if (validationRules.min !== undefined && Number(value) < validationRules.min) {
          nextErrors[f.key] = `Must be at least ${validationRules.min}`;
        }
        if (validationRules.max !== undefined && Number(value) > validationRules.max) {
          nextErrors[f.key] = `Must be no more than ${validationRules.max}`;
        }
        if (validationRules.regex && !new RegExp(validationRules.regex).test(value)) {
          nextErrors[f.key] = `Does not match required format.`;
        }
      }
    });
    if (Object.keys(nextErrors).length > 0) {
      setErrors(nextErrors);
      return;
    }
    const payload = { type: subtype, ...values };
    try {
      const resp = await onSubmit({ entity_type: entity, data: payload });
      // resp may contain validation_errors
      if (resp && resp.validation_errors) {
        setErrors(resp.validation_errors);
        return;
      }
      // success: clear draft so reopening form starts fresh
      setDraft({});
      if (onClearServerErrors) onClearServerErrors();
      onClose();
    } catch (err: any) {
      // if the handler throws a structured validation error, try to display it
      const body = err?.response || err;
      if (body && body.validation_errors) {
        setErrors(body.validation_errors);
        return;
      }
      // fallback: set generic error
      setErrors({ _error: err?.message || String(err) });
    }
  };

  return (
  <Dialog open={open} onClose={() => { if (onClearServerErrors) onClearServerErrors(); onClose(); }} fullWidth maxWidth="md">
      <DialogTitle>Create New Entity</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
          <TextField select label="Entity Category" value={entity || ''} onChange={(e) => { setEntity(e.target.value); setSubtype(null); setValues({}); }} fullWidth>
            {entities.map(e => <MenuItem key={e.entity_name} value={e.entity_name}>{e.display_name}</MenuItem>)}
          </TextField>
          {entity && selectedEntity && (
            <TextField select label="Subtype" value={subtype || ''} onChange={(e) => { setSubtype(e.target.value); setValues({}); }} fullWidth>
              {(selectedEntity.subtypes || []).map((s: any) => <MenuItem key={s.key || s} value={s.key || s}>{s.label || s}</MenuItem>)}
            </TextField>
          )}
        </Box>

        {visibleFields.map(field => (
          <Box key={field.key} sx={{ mb: 2 }}>
            {field.type === 'text' && (
              <>
                <TextField fullWidth label={field.label} value={values[field.key] || ''} onChange={(e) => handleChange(field.key, e.target.value)} helperText={errors[field.key]} error={!!errors[field.key]} />
              </>
            )}
            {field.type === 'number' && (
              <>
                <TextField type="number" fullWidth label={field.label} value={values[field.key] || ''} onChange={(e) => handleChange(field.key, e.target.value)} helperText={errors[field.key]} error={!!errors[field.key]} />
              </>
            )}
            {field.type === 'select' && (
              <>
                <TextField select fullWidth label={field.label} value={values[field.key] || ''} onChange={(e) => handleChange(field.key, e.target.value)} helperText={errors[field.key]} error={!!errors[field.key]}>
                  {(field.options || []).map((opt: string) => <MenuItem key={opt} value={opt}>{opt}</MenuItem>)}
                </TextField>
              </>
            )}
            {field.type === 'switch' && (
              <FormControlLabel control={<Switch checked={!!values[field.key]} onChange={(e) => handleChange(field.key, e.target.checked)} />} label={field.label} />
            )}
            {field.type === 'json' && (
              <>
                <KeyValueEditor value={values[field.key] || {}} onChange={(v) => handleChange(field.key, v)} />
                {(errors[field.key] || serverErrors?.[field.key]) && <FormHelperText error>{errors[field.key] || serverErrors?.[field.key]}</FormHelperText>}
              </>
            )}
            {field.type === 'tag-input' && (
              <Box>
                <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
                  <TextField value={tagInput} onChange={(e) => setTagInput(e.target.value)} placeholder="Add tag" onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); handleAddTag(field.key); } }} />
                  <IconButton onClick={() => handleAddTag(field.key)} aria-label="add tag"><AddIcon /></IconButton>
                </Box>
                <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                  {(values[field.key] || []).map((t: string, idx: number) => (
                    <Chip key={t + idx} label={t} onDelete={() => handleRemoveTag(field.key, idx)} />
                  ))}
                </Box>
                {errors[field.key] && <FormHelperText error>{errors[field.key]}</FormHelperText>}
              </Box>
            )}
          </Box>
        ))}
      </DialogContent>
      <DialogActions>
  <Button onClick={() => { if (onClearServerErrors) onClearServerErrors(); onClose(); }}>Cancel</Button>
        <Button variant="contained" onClick={submit}>Create</Button>
      </DialogActions>
    </Dialog>
  );
}

export default DynamicEntityForm;
