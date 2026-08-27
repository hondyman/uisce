import React, { useState, useEffect } from 'react';
import {
  Box, Card, CardHeader, CardContent, Button,
  Grid, Typography, Alert, CircularProgress
} from '@mui/material';
import SaveIcon from '@mui/icons-material/Save';
import { PageWidgetDef } from '../PageDesignerTypes';
import { usePageEventBus } from '../PageEventBusContext';
import { DynamicFormField } from '../DynamicFormField';
import { DynamicFieldUIConfig } from '../DynamicPropertyTypes';

interface DynamicBOFormWidgetProps {
  widget: PageWidgetDef;
}

export const DynamicBOFormWidget: React.FC<DynamicBOFormWidgetProps> = ({ widget }) => {
  const { parameters, setParameter } = usePageEventBus();
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Determine active record ID from subscribed parameters
  const activeRecordId = widget.subscribedParams.length > 0 
    ? parameters[widget.subscribedParams[0]] 
    : null;

  useEffect(() => {
    if (!activeRecordId || !widget.boKey) {
      setFormData({});
      return;
    }

    // Fetch entity record by ID
    setLoading(true);
    fetch(`/api/v1/bo/${widget.boKey}/records/${activeRecordId}`)
      .then(async (res) => {
        if (!res.ok) throw new Error(await res.text());
        return res.json();
      })
      .then((data) => {
        setFormData(data || {});
        setMessage(null);
      })
      .catch((err) => setMessage({ type: 'error', text: 'Failed to hydrate form data: ' + (err.message || '') }))
      .finally(() => setLoading(false));
  }, [activeRecordId, widget.boKey]);

  const handleFieldChange = (fieldKey: string, value: any) => {
    setFormData((prev) => ({ ...prev, [fieldKey]: value }));
  };

  const handleSave = async () => {
    if (!widget.boKey) return;
    setSaving(true);
    try {
      const endpoint = activeRecordId
        ? `/api/v1/bo/${widget.boKey}/records/${activeRecordId}`
        : `/api/v1/bo/${widget.boKey}/records`;
      const method = activeRecordId ? 'PUT' : 'POST';

      const res = await fetch(endpoint, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });

      if (!res.ok) throw new Error(await res.text());
      const saved = await res.json();
      setMessage({ type: 'success', text: 'Record committed successfully.' });

      // Dispatch Form Submit Events to Bus
      if (widget.events?.onFormSubmit) {
        widget.events.onFormSubmit.forEach((evt) => {
          if (evt.actionType === 'SET_PARAMETER') {
            const val = saved[evt.sourcePropertyKey] ?? saved.id;
            setParameter(evt.targetChannel, val);
          }
        });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err.message || 'Mutation failed' });
    } finally {
      setSaving(false);
    }
  };

  if (!activeRecordId && !widget.entitlements?.allowCreate) {
    return (
      <Card sx={{ bgcolor: '#071526', border: '1px solid #1E293B', color: '#64748B', p: 3, textAlign: 'center' }}>
        <Typography variant="body2">Select a record from the grid or chart to inspect and edit details.</Typography>
      </Card>
    );
  }

  // Derive form fields from formSpec or fallback to standard schema keys
  const sections = widget.formSpec?.sections || [
    {
      id: 'default_sec',
      title: 'Fields',
      columns: 2 as const,
      items: Object.keys(formData).length > 0
        ? Object.keys(formData).map((key) => ({
            id: key,
            type: 'SEMANTIC_FIELD' as const,
            label: key.replace(/_/g, ' ').toUpperCase(),
            fieldKey: key,
            colSpan: 6 as const,
            isReadOnly: key === 'id' || key === 'tenant_id' || key === 'created_at',
            uiConfig: (widget as any).fieldUIConfigs?.[key] as DynamicFieldUIConfig | undefined,
          }))
        : [
            { id: 'f_name', type: 'SEMANTIC_FIELD' as const, label: 'Name / Title', fieldKey: 'name', colSpan: 6 as const, uiConfig: undefined },
            { id: 'f_status', type: 'SEMANTIC_FIELD' as const, label: 'Status', fieldKey: 'status', colSpan: 6 as const, uiConfig: undefined },
            { id: 'f_desc', type: 'SEMANTIC_FIELD' as const, label: 'Description', fieldKey: 'description', colSpan: 12 as const, uiConfig: undefined },
          ],
    },
  ];

  return (
    <Card sx={{ bgcolor: '#071526', border: '1px solid #1E293B', color: '#F8FAFC' }}>
      <CardHeader
        title={<Typography variant="subtitle2" fontWeight={700} color="#00D4FF">{widget.title}</Typography>}
        action={
          widget.entitlements?.allowUpdate !== false && (
            <Button
              size="small"
              variant="contained"
              startIcon={saving ? <CircularProgress size={12} /> : <SaveIcon />}
              onClick={handleSave}
              disabled={saving}
              sx={{ bgcolor: '#0284C7', textTransform: 'none', fontSize: 11 }}
            >
              Save Record
            </Button>
          )
        }
        sx={{ borderBottom: '1px solid #1E293B', py: 1 }}
      />
      <CardContent sx={{ p: 2 }}>
        {message && <Alert severity={message.type} sx={{ mb: 2, fontSize: 11 }}>{message.text}</Alert>}

        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}><CircularProgress size={24} /></Box>
        ) : (
          <Grid container spacing={2}>
            {sections.flatMap((s) => s.items).map((item) => (
              <Grid size={{ xs: 12, sm: item.colSpan }} key={item.id}>
                <DynamicFormField
                  label={item.label}
                  fieldKey={item.fieldKey || ''}
                  value={formData[item.fieldKey || ''] ?? ''}
                  onChange={(val) => handleFieldChange(item.fieldKey || '', val)}
                  uiConfig={(item as any).uiConfig || {
                    isReadOnly: { isExpression: false, value: !!item.isReadOnly },
                  }}
                  rowData={formData}
                />
              </Grid>
            ))}
          </Grid>
        )}
      </CardContent>
    </Card>
  );
};
