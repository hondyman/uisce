import React, { useState } from 'react';
import { Box, Typography, TextField, Button, Paper, Chip, Select, MenuItem, FormControl, InputLabel } from '@mui/material';
import {
  Sparkles,
  Layers,
  Send,
  Star,
  CheckCircle2,
  Plus,
} from 'lucide-react';

interface SelectedField {
  termNodeId: string;
  termKey: string;
  fieldName: string;
  fieldRole: string; // DIMENSION, MEASURE, KEY
  bindingRequirement: string; // REQUIRED, OPTIONAL, BACKEND_SPECIFIC
  sourceNodeId?: string;
  sourceType: string;
  transformationType: string;
  transformationSql?: string;
  overrideReason?: string;
}

interface BindingCardState {
  backendId: string;
  drivingNodeId: string;
  isDefault: boolean;
  tableName: string;
  discoveredPK?: string;
  relatedTables: string[];
  fields: SelectedField[];
}

export const CreateBusinessObjectStudio: React.FC<{
  tenantId: string;
  modelId: string;
  onSuccess?: (boId: string) => void;
}> = ({ tenantId, modelId, onSuccess }) => {
  // 1. Definition State
  const [boName, setBoName] = useState('Customer');
  const [boKey, setBoKey] = useState('customer');
  const [boType, setBoType] = useState('ENTITY');
  const [businessKeyNodeId] = useState('');
  const [semanticIdNodeId] = useState('');

  // 2. Multi-Backend Binding Panels
  const [bindings, setBindings] = useState<BindingCardState[]>([
    {
      backendId: 'postgres-alpha-id',
      drivingNodeId: '',
      isDefault: true,
      tableName: '',
      relatedTables: [],
      fields: [],
    },
  ]);

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [statusMsg, setStatusMsg] = useState<string | null>(null);

  // Auto-Discovery Invocation
  const handleSelectDrivingTable = async (bindingIndex: number, drivingNodeId: string) => {
    try {
      const res = await fetch('/api/v1/business-objects/binding-context', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
          backendId: bindings[bindingIndex].backendId,
          drivingNodeId,
        }),
      });

      if (res.ok) {
        const data = await res.json();
        const updated = [...bindings];
        updated[bindingIndex].drivingNodeId = drivingNodeId;
        updated[bindingIndex].tableName = data.drivingTable.tableName;
        updated[bindingIndex].relatedTables = data.relatedTables.map((r: any) => r.tableName);

        // Auto-select discovered terms and auto-create field bindings
        updated[bindingIndex].fields = data.eligibleTerms.map((t: any) => ({
          termNodeId: t.termNodeId,
          termKey: t.termKey,
          fieldName: t.termKey,
          fieldRole: t.identityRole === 'BUSINESS_KEY' ? 'KEY' : 'DIMENSION',
          bindingRequirement: 'REQUIRED',
          sourceNodeId: t.mappings[0]?.columnNodeId,
          sourceType: 'COLUMN',
          transformationType: 'NONE',
        }));

        setBindings(updated);
      }
    } catch (err) {
      console.error('Failed auto-discovery:', err);
    }
  };

  const handleSaveAndPublish = async () => {
    setIsSubmitting(true);
    try {
      const payload = {
        tenantId,
        modelId,
        publish: true,
        businessObject: {
          boKey,
          boname: boName,
          boType,
          classificationNodeId: 'c0000000-0000-0000-0000-000000000003', // Level 3 Classification
          businessKeyNodeId: businessKeyNodeId || bindings[0]?.fields[0]?.termNodeId,
          semanticIdNodeId: semanticIdNodeId || bindings[0]?.fields[0]?.termNodeId,
          grainNodeId: businessKeyNodeId || bindings[0]?.fields[0]?.termNodeId,
        },
        bindings: bindings.map((b) => ({
          backendId: b.backendId,
          drivingNodeId: b.drivingNodeId,
          isDefault: b.isDefault,
          temporalOverride: 'NONE',
          fields: b.fields,
          relationships: [],
        })),
      };

      const res = await fetch('/api/v1/business-objects/save', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        const result = await res.json();
        setStatusMsg('Business Object & Bindings Published Successfully!');
        if (onSuccess) onSuccess(result.boId);
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', bgcolor: '#030914', color: '#f1f5f9', border: '1px solid #1e293b', borderRadius: 2, overflow: 'hidden', fontFamily: 'sans-serif' }}>
      <Box sx={{ p: 3, bgcolor: '#071526', borderBottom: '1px solid #1e293b', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#f1f5f9', display: 'flex', alignItems: 'center', gap: 1 }}>
            <Sparkles size={20} style={{ color: '#22d3ee' }} />
            Single-Screen Business Object Studio
          </Typography>
          <Typography variant="caption" sx={{ color: '#94a3b8', mt: 0.5, display: 'block' }}>
            Auto-discover tables, primary keys, relationships, and mapped semantic terms in one unified canvas.
          </Typography>
        </Box>
        <Button
          variant="contained"
          onClick={handleSaveAndPublish}
          disabled={isSubmitting}
          startIcon={<Send size={16} />}
          sx={{
            px: 3,
            py: 1.5,
            background: 'linear-gradient(to right, #06b6d4, #34d399)',
            '&:hover': { opacity: 0.95 },
            color: '#0f172a',
            fontWeight: 700,
            fontSize: '0.75rem',
            borderRadius: 1,
            textTransform: 'none',
            disabled: { opacity: 0.5 },
          }}
        >
          Save & Publish Business Object
        </Button>
      </Box>

      {statusMsg && (
        <Box sx={{ bgcolor: 'rgba(16, 185, 129, 0.2)', borderBottom: '1px solid rgba(16, 185, 129, 0.4)', px: 3, py: 1, display: 'flex', alignItems: 'center', gap: 1, color: '#6ee7b7', fontSize: '0.75rem' }}>
          <CheckCircle2 size={16} style={{ color: '#34d399' }} /> {statusMsg}
        </Box>
      )}

      <Box sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 3, flex: 1, overflowY: 'auto' }}>
        <Paper sx={{ p: 2.5, bgcolor: 'rgba(30, 41, 59, 0.6)', border: '1px solid #1e293b', borderRadius: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Typography variant="caption" sx={{ fontWeight: 700, color: '#cbd5e1', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            1. Semantic Contract Definition
          </Typography>
          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 2 }}>
            <Box>
              <Typography variant="caption" sx={{ fontWeight: 600, color: '#94a3b8', display: 'block', mb: 0.5 }}>BO Name</Typography>
              <TextField
                fullWidth
                size="small"
                value={boName}
                onChange={(e) => setBoName(e.target.value)}
                sx={{
                  '& .MuiOutlinedInput-root': { bgcolor: '#020617', borderColor: '#1e293b', '& fieldset': { borderColor: '#1e293b' } },
                  '& input': { color: '#f1f5f9', fontSize: '0.75rem', fontWeight: 600 },
                }}
              />
            </Box>
            <Box>
              <Typography variant="caption" sx={{ fontWeight: 600, color: '#94a3b8', display: 'block', mb: 0.5 }}>BO Key</Typography>
              <TextField
                fullWidth
                size="small"
                value={boKey}
                onChange={(e) => setBoKey(e.target.value)}
                sx={{
                  '& .MuiOutlinedInput-root': { bgcolor: '#020617', borderColor: '#1e293b', '& fieldset': { borderColor: '#1e293b' } },
                  '& input': { color: '#f1f5f9', fontSize: '0.75rem', fontFamily: 'monospace' },
                }}
              />
            </Box>
            <Box>
              <Typography variant="caption" sx={{ fontWeight: 600, color: '#94a3b8', display: 'block', mb: 0.5 }}>BO Type</Typography>
              <FormControl fullWidth size="small">
                <Select
                  value={boType}
                  onChange={(e) => setBoType(e.target.value)}
                  sx={{
                    bgcolor: '#020617',
                    color: '#f1f5f9',
                    fontSize: '0.75rem',
                    '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1e293b' },
                  }}
                >
                  <MenuItem value="ENTITY">ENTITY</MenuItem>
                  <MenuItem value="FACT">FACT</MenuItem>
                  <MenuItem value="DIMENSION">DIMENSION</MenuItem>
                </Select>
              </FormControl>
            </Box>
            <Box>
              <Typography variant="caption" sx={{ fontWeight: 600, color: '#94a3b8', display: 'block', mb: 0.5 }}>Level 3 Classification</Typography>
              <TextField
                fullWidth
                size="small"
                disabled
                value="Sales > Client > Client Entity"
                sx={{
                  '& .MuiOutlinedInput-root': { bgcolor: 'rgba(2, 6, 23, 0.6)', borderColor: '#1e293b', '& fieldset': { borderColor: '#1e293b' } },
                  '& input': { color: '#94a3b8', fontSize: '0.75rem' },
                }}
              />
            </Box>
          </Box>
        </Paper>

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Typography variant="caption" sx={{ fontWeight: 700, color: '#cbd5e1', textTransform: 'uppercase', letterSpacing: '0.05em', display: 'flex', alignItems: 'center', gap: 1 }}>
              <Layers size={16} style={{ color: '#22d3ee' }} />
              2. Backend Bindings & Scoped Term Discovery
            </Typography>
            <Button
              variant="outlined"
              size="small"
              onClick={() =>
                setBindings([
                  ...bindings,
                  {
                    backendId: 'starrocks-olap-id',
                    drivingNodeId: '',
                    isDefault: false,
                    tableName: '',
                    relatedTables: [],
                    fields: [],
                  },
                ])
              }
              startIcon={<Plus size={14} />}
              sx={{
                p: 1,
                bgcolor: '#0f172a',
                borderColor: '#1e293b',
                color: '#cbd5e1',
                fontSize: '0.75rem',
                textTransform: 'none',
                '&:hover': { bgcolor: '#1e293b' },
              }}
            >
              Add Backend Binding
            </Button>
          </Box>

          {bindings.map((b, idx) => (
            <Paper
              key={idx}
              sx={{ p: 2.5, bgcolor: 'rgba(30, 41, 59, 0.4)', border: '1px solid #1e293b', borderRadius: 2, display: 'flex', flexDirection: 'column', gap: 2 }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 1.5, borderBottom: '1px solid #1e293b' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                  <Chip
                    icon={<Star size={14} />}
                    label={b.isDefault ? 'Default Golden Binding' : `Binding ${idx + 1}`}
                    size="small"
                    sx={{
                      bgcolor: b.isDefault ? 'rgba(245, 158, 11, 0.2)' : '#1e293b',
                      color: b.isDefault ? '#fbbf24' : '#94a3b8',
                      fontWeight: 700,
                      fontSize: '0.75rem',
                      border: b.isDefault ? '1px solid rgba(245, 158, 11, 0.4)' : 'none',
                      '& .MuiChip-icon': { color: b.isDefault ? '#fbbf24' : '#94a3b8' },
                    }}
                  />
                  <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#cbd5e1' }}>{b.backendId}</Typography>
                </Box>

                <FormControl size="small" sx={{ minWidth: 180 }}>
                  <Select
                    native
                    onChange={(e) => handleSelectDrivingTable(idx, e.target.value as string)}
                    sx={{
                      bgcolor: '#020617',
                      color: '#f1f5f9',
                      fontSize: '0.75rem',
                      '& .MuiOutlinedInput-notchedOutline': { borderColor: '#1e293b' },
                    }}
                  >
                    <option value="">Select Driving Table...</option>
                    <option value="tbl-customers-node">Customers (Postgres Alpha)</option>
                    <option value="tbl-orders-node">Orders (Postgres Alpha)</option>
                  </Select>
                </FormControl>
              </Box>

              {b.tableName && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontSize: '0.75rem', color: '#94a3b8' }}>
                    <span>
                      Driving Table: <strong style={{ color: '#22d3ee', fontFamily: 'monospace' }}>{b.tableName}</strong>
                      {' '}| Related Discovered: <strong style={{ color: '#e2e8f0' }}>{b.relatedTables.join(', ') || 'None'}</strong>
                    </span>
                    <span style={{ color: '#34d399', fontWeight: 700 }}>{b.fields.length} Terms Auto-Mapped</span>
                  </Box>

                  <Box sx={{ border: '1px solid #1e293b', borderRadius: 1, bgcolor: 'rgba(2, 6, 23, 0.6)', overflow: 'hidden' }}>
                    {b.fields.map((f, fIdx) => (
                      <Box
                        key={fIdx}
                        sx={{ p: 1.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontSize: '0.75rem', fontFamily: 'monospace', borderBottom: fIdx < b.fields.length - 1 ? '1px solid rgba(30, 41, 59, 0.8)' : 'none' }}
                      >
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <CheckCircle2 size={14} style={{ color: '#34d399' }} />
                          <span style={{ color: '#e2e8f0', fontWeight: 600 }}>{f.fieldName}</span>
                          <Chip label={f.fieldRole} size="small" sx={{ bgcolor: '#1e293b', color: '#94a3b8', fontSize: '0.625rem', height: 18 }} />
                        </Box>

                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                          <span style={{ color: '#94a3b8' }}>{b.tableName}.{f.fieldName}</span>
                          <Chip label={f.bindingRequirement} size="small" sx={{ bgcolor: 'rgba(34, 211, 238, 0.2)', color: '#22d3ee', fontSize: '0.625rem', height: 18, fontWeight: 700, fontFamily: 'sans-serif' }} />
                        </Box>
                      </Box>
                    ))}
                  </Box>
                </Box>
              )}
            </Paper>
          ))}
        </Box>
      </Box>
    </Box>
  );
};
