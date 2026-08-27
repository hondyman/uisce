import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  TextField,
  MenuItem,
  Button,
  Chip,
  Radio,
  RadioGroup,
  FormControlLabel,
  Stack,
  Divider,
  Collapse,
  IconButton,
  Alert,
} from '@mui/material';
import StarIcon from '@mui/icons-material/Star';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';

interface ColumnMapping {
  columnNodeId: string;
  columnName: string;
  tableName: string;
  sourceType: string;
  isPrimarySource: boolean;
}

interface DiscoveredTerm {
  termNodeId: string;
  termKey: string;
  termName: string;
  termType: string;
  dataType: string;
  sourceType: string;
  mappings: ColumnMapping[];
}

interface BindingState {
  backendId: string;
  backendType: string;
  drivingNodeId: string;
  isDefault: boolean;
  discoveredPK?: string;
  selectedTerms: Record<string, { term: DiscoveredTerm; chosenMapping: ColumnMapping; role: string }>;
}

export const BusinessObjectStudioPage: React.FC<{ tenantId?: string }> = ({ tenantId }) => {
  const [boName, setBoName] = useState('Investment Account');
  const [boKey, setBoKey] = useState('account');
  const [boType, setBoType] = useState('ENTITY');
  const [bkTermId, setBkTermId] = useState('3a9ea353-0000-0000-0000-000000000001');
  const [sidTermId, setSidTermId] = useState('3a9ea353-0000-0000-0000-000000000002');

  const [bindings, setBindings] = useState<BindingState[]>([
    {
      backendId: 'b1-postgres',
      backendType: 'POSTGRES',
      drivingNodeId: 'f53ad6ce-a247-4950-bf1f-0a27e9ad5df4',
      isDefault: true,
      selectedTerms: {},
    },
  ]);

  const [discoveredTerms, setDiscoveredTerms] = useState<DiscoveredTerm[]>([]);
  const [expandedField, setExpandedField] = useState<string | null>(null);

  useEffect(() => {
    // Simulated fetch to /api/business-objects/binding-context
    setDiscoveredTerms([
      {
        termNodeId: 'term-1',
        termKey: 'account_identifier',
        termName: 'Account Identifier',
        termType: 'KEY',
        dataType: 'UUID',
        sourceType: 'DIRECT',
        mappings: [
          { columnNodeId: 'c1', columnName: 'account_id', tableName: 'oms.account', sourceType: 'DIRECT', isPrimarySource: true },
          { columnNodeId: 'c2', columnName: 'account_id', tableName: 'oms.orders', sourceType: 'RELATED', isPrimarySource: false },
        ],
      },
      {
        termNodeId: 'term-2',
        termKey: 'account_name',
        termName: 'Account Name',
        termType: 'ATTRIBUTE',
        dataType: 'VARCHAR',
        sourceType: 'DIRECT',
        mappings: [
          { columnNodeId: 'c3', columnName: 'account_name', tableName: 'oms.account', sourceType: 'DIRECT', isPrimarySource: true },
        ],
      },
    ]);
  }, []);

  const handleToggleTerm = (bIdx: number, term: DiscoveredTerm) => {
    const next = [...bindings];
    const target = next[bIdx].selectedTerms;
    if (target[term.termNodeId]) {
      delete target[term.termNodeId];
    } else {
      target[term.termNodeId] = {
        term,
        chosenMapping: term.mappings[0],
        role: term.termType === 'KEY' ? 'KEY' : 'DIMENSION',
      };
    }
    setBindings(next);
  };

  const handleSelectMapping = (bIdx: number, termId: string, mapping: ColumnMapping) => {
    const next = [...bindings];
    if (next[bIdx].selectedTerms[termId]) {
      next[bIdx].selectedTerms[termId].chosenMapping = mapping;
      setBindings(next);
    }
  };

  const handleSetDefaultBinding = (bIdx: number) => {
    const next = bindings.map((b, i) => ({ ...b, isDefault: i === bIdx }));
    setBindings(next);
  };

  return (
    <Box sx={{ p: 4, bgcolor: '#030914', minHeight: '100vh', color: '#F8FAFC' }}>
      {/* 1. Header & Semantic Shell */}
      <Paper sx={{ p: 3, mb: 3, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 2 }}>
        <Typography variant="h6" sx={{ fontWeight: 700, mb: 2, color: '#00D4FF' }}>
          Business Object Studio (Single-Screen Multi-Binding Model)
        </Typography>
        <Grid container spacing={2}>
          <Grid   size={{ xs: 12, md: 3 }}>
            <TextField fullWidth label="BO Name" value={boName} onChange={(e) => setBoName(e.target.value)} size="small" />
          </Grid>
          <Grid   size={{ xs: 12, md: 3 }}>
            <TextField fullWidth label="BO Key" value={boKey} onChange={(e) => setBoKey(e.target.value)} size="small" />
          </Grid>
          <Grid   size={{ xs: 12, md: 3 }}>
            <TextField fullWidth select label="BO Type" value={boType} onChange={(e) => setBoType(e.target.value)} size="small">
              <MenuItem value="ENTITY">ENTITY</MenuItem>
              <MenuItem value="FACT">FACT</MenuItem>
              <MenuItem value="DIMENSION">DIMENSION</MenuItem>
            </TextField>
          </Grid>
          <Grid   size={{ xs: 12, md: 3 }}>
            <Chip label="Level 3 Classification: Client Portfolio" sx={{ bgcolor: 'rgba(0, 212, 255, 0.1)', color: '#38BDF8', mt: 1 }} />
          </Grid>
        </Grid>
      </Paper>

      {/* 2. Side-by-Side Multi-Backend Bindings */}
      <Stack spacing={3} sx={{ mb: 3 }}>
        {bindings.map((b, bIdx) => (
          <Paper key={b.backendId} sx={{ p: 3, bgcolor: '#0A1B30', border: '1px solid #1E293B', borderRadius: 2 }}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
              <Stack direction="row" spacing={1.5} alignItems="center">
                <Typography variant="subtitle1" sx={{ fontWeight: 700, color: '#F1F5F9' }}>
                  Binding {bIdx + 1}: {b.backendType}
                </Typography>
                {b.isDefault ? (
                  <Chip icon={<StarIcon sx={{ fontSize: '14px !important', color: '#F59E0B' }} />} label="DEFAULT" size="small" sx={{ bgcolor: 'rgba(245, 158, 11, 0.1)', color: '#FBBF24' }} />
                ) : (
                  <Button size="small" onClick={() => handleSetDefaultBinding(bIdx)} sx={{ color: '#94A3B8' }}>Make Default</Button>
                )}
              </Stack>
            </Box>

            {/* Terms & Multi-Mapping Resolution */}
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase' }}>
              Eligible Semantic Terms (Auto-Discovered)
            </Typography>
            <Stack spacing={1.5} sx={{ mt: 1 }}>
              {discoveredTerms.map((term) => {
                const isSelected = !!b.selectedTerms[term.termNodeId];
                const activeMapping = b.selectedTerms[term.termNodeId]?.chosenMapping;

                return (
                  <Paper key={term.termNodeId} sx={{ p: 1.5, bgcolor: '#061324', border: '1px solid #1E293B', borderRadius: 1.5 }}>
                    <Box display="flex" justifyContent="space-between" alignItems="center">
                      <Stack direction="row" spacing={1.5} alignItems="center">
                        <Button
                          variant={isSelected ? 'contained' : 'outlined'}
                          size="small"
                          onClick={() => handleToggleTerm(bIdx, term)}
                          sx={{ minWidth: 28, height: 28, p: 0 }}
                        >
                          {isSelected ? '✓' : '+'}
                        </Button>
                        <Box>
                          <Typography variant="body2" sx={{ fontWeight: 600 }}>{term.termName}</Typography>
                          <Typography variant="caption" sx={{ color: '#64748B' }}>{term.termKey} • {term.dataType}</Typography>
                        </Box>
                      </Stack>
                      {isSelected && activeMapping && (
                        <Chip label={`${activeMapping.tableName}.${activeMapping.columnName}`} size="small" sx={{ bgcolor: '#0284C7', color: '#fff', fontSize: 11 }} />
                      )}
                    </Box>

                    {/* Multi-Mapping Radio Resolution */}
                    {isSelected && term.mappings.length > 1 && (
                      <Box sx={{ mt: 1.5, pl: 5 }}>
                        <Typography variant="caption" sx={{ color: '#F59E0B', fontWeight: 600 }}>Multiple physical columns detected. Select mapping:</Typography>
                        <RadioGroup
                          value={activeMapping?.columnNodeId}
                          onChange={(e) => {
                            const chosen = term.mappings.find((m) => m.columnNodeId === e.target.value);
                            if (chosen) handleSelectMapping(bIdx, term.termNodeId, chosen);
                          }}
                        >
                          {term.mappings.map((m) => (
                            <FormControlLabel
                              key={m.columnNodeId}
                              value={m.columnNodeId}
                              control={<Radio size="small" sx={{ color: '#38BDF8' }} />}
                              label={<Typography variant="caption">{m.tableName}.{m.columnName} ({m.sourceType})</Typography>}
                            />
                          ))}
                        </RadioGroup>
                      </Box>
                    )}
                  </Paper>
                );
              })}
            </Stack>
          </Paper>
        ))}
      </Stack>

      {/* 3. Validation Summary & Publish Rail */}
      <Paper sx={{ p: 2.5, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Stack direction="row" spacing={2} alignItems="center">
          <CheckCircleIcon sx={{ color: '#10B981' }} />
          <Box>
            <Typography variant="body2" sx={{ fontWeight: 700 }}>Validation Summary</Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>All required identity terms (BK, SID) mapped across 1 default binding.</Typography>
          </Box>
        </Stack>
        <Button variant="contained" sx={{ bgcolor: '#00D4FF', color: '#030914', fontWeight: 700 }}>
          🚀 Publish Business Object
        </Button>
      </Paper>
    </Box>
  );
};

export default BusinessObjectStudioPage;
