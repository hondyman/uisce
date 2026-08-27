import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  TextField,
  MenuItem,
  Button,
  Chip,
  Grid,
  Divider,
  Checkbox,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Radio,
  RadioGroup,
  FormControlLabel,
  Alert,
  IconButton,
  Collapse,
  Card,
  CardContent
} from '@mui/material';
import {
  Storage as TableIcon,
  CheckCircle as ValidIcon,
  Star as StarIcon,
  StarBorder as StarBorderIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Save as SaveIcon,
  AccountTree as RelationshipIcon
} from '@mui/icons-material';

interface ColumnMapping {
  columnNodeId: string;
  columnName: string;
  tableName: string;
  sourceType: 'DIRECT' | 'RELATED';
  isPrimarySource: boolean;
}

interface EligibleTerm {
  termNodeId: string;
  termKey: string;
  termName: string;
  termType: string;
  sourceType: 'DIRECT' | 'RELATED' | 'CALCULATED';
  selected: boolean;
  selectedMappingIndex: number;
  mappings: ColumnMapping[];
}

interface BackendBindingConfig {
  backendId: string;
  backendName: string;
  drivingNodeId: string;
  drivingTableName: string;
  isDefault: boolean;
  pkColumn: string;
  suggestedBk: string;
  relatedTables: string[];
  terms: EligibleTerm[];
}

export const SingleScreenBOStudio: React.FC<{ tenantId?: string }> = ({ tenantId: _tenantId }) => {
  const [boName, setBoName] = useState('Customer');
  const [boKey, setBoKey] = useState('customer');
  const [boType, setBoType] = useState('ENTITY');
  const [classificationNodeId, setClassificationNodeId] = useState('820b1234-0000-4000-a000-000000000001');

  const [bindings, setBindings] = useState<BackendBindingConfig[]>([
    {
      backendId: 'a0000001-0000-4000-a000-000000000001',
      backendName: 'Northwind Postgres (Hot)',
      drivingNodeId: 'tbl-customers-001',
      drivingTableName: 'Customers',
      isDefault: true,
      pkColumn: 'CustomerID',
      suggestedBk: 'customer_bk',
      relatedTables: ['Orders (via CustomerID, 1:M)', 'CustomerCustomerDemo (via CustomerID, 1:M)'],
      terms: [
        {
          termNodeId: 'term-01',
          termKey: 'customer_identifier',
          termName: 'Customer Identifier',
          termType: 'KEY',
          sourceType: 'DIRECT',
          selected: true,
          selectedMappingIndex: 0,
          mappings: [
            { columnNodeId: 'col-c-id', columnName: 'CustomerID', tableName: 'Customers', sourceType: 'DIRECT', isPrimarySource: true },
            { columnNodeId: 'col-o-cid', columnName: 'CustomerID', tableName: 'Orders', sourceType: 'RELATED', isPrimarySource: false }
          ]
        },
        {
          termNodeId: 'term-02',
          termKey: 'customer_bk',
          termName: 'Customer Business Key',
          termType: 'KEY',
          sourceType: 'DIRECT',
          selected: true,
          selectedMappingIndex: 0,
          mappings: [
            { columnNodeId: 'col-c-id', columnName: 'CustomerID', tableName: 'Customers', sourceType: 'DIRECT', isPrimarySource: true }
          ]
        },
        {
          termNodeId: 'term-03',
          termKey: 'company_name',
          termName: 'Company Name',
          termType: 'DIMENSION',
          sourceType: 'DIRECT',
          selected: true,
          selectedMappingIndex: 0,
          mappings: [
            { columnNodeId: 'col-c-name', columnName: 'CompanyName', tableName: 'Customers', sourceType: 'DIRECT', isPrimarySource: true }
          ]
        }
      ]
    }
  ]);

  const [expandedBindingIndex, setExpandedBindingIndex] = useState<number>(0);
  const [saveSuccess, setSaveSuccess] = useState(false);

  const handleToggleTerm = (bindingIdx: number, termIdx: number) => {
    setBindings(prev => {
      const updated = [...prev];
      const targetTerm = updated[bindingIdx].terms[termIdx];
      targetTerm.selected = !targetTerm.selected;
      return updated;
    });
  };

  const handleSelectMapping = (bindingIdx: number, termIdx: number, mapIdx: number) => {
    setBindings(prev => {
      const updated = [...prev];
      updated[bindingIdx].terms[termIdx].selectedMappingIndex = mapIdx;
      return updated;
    });
  };

  const handleSetDefault = (targetIdx: number) => {
    setBindings(prev =>
      prev.map((b, idx) => ({
        ...b,
        isDefault: idx === targetIdx
      }))
    );
  };

  const handleSaveAndPublish = () => {
    setSaveSuccess(true);
  };

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <TableIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Business Object Studio & Multi-Backend Auto-Mapper
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Single-screen semantic definition, driving table discovery, and cross-backend binding resolution
            </Typography>
          </Box>
        </Stack>

        <Button
          variant="contained"
          startIcon={<SaveIcon />}
          onClick={handleSaveAndPublish}
          sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 600, '&:hover': { bgcolor: '#0369A1' } }}
        >
          Save & Publish Business Object
        </Button>
      </Box>

      {saveSuccess && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC', border: '1px solid #10B981' }}>
          Business Object [<strong>{boName}</strong>] and bindings published successfully to the Semantic Graph.
        </Alert>
      )}

      <Card sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', mb: 3, color: '#F8FAFC' }}>
        <CardContent>
          <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', display: 'block', mb: 2 }}>
            1. Semantic Contract Definition
          </Typography>
          <Grid container spacing={2}>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="BO Name"
                value={boName}
                onChange={e => setBoName(e.target.value)}
                sx={{ input: { color: '#F8FAFC' }, label: { color: '#94A3B8' } }}
              />
            </Grid>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <TextField
                fullWidth
                size="small"
                label="BO Key"
                value={boKey}
                onChange={e => setBoKey(e.target.value)}
                sx={{ input: { color: '#F8FAFC' }, label: { color: '#94A3B8' } }}
              />
            </Grid>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <TextField
                select
                fullWidth
                size="small"
                label="BO Type"
                value={boType}
                onChange={e => setBoType(e.target.value)}
                sx={{ '& .MuiSelect-select': { color: '#F8FAFC' }, label: { color: '#94A3B8' } }}
              >
                <MenuItem value="ENTITY">ENTITY</MenuItem>
                <MenuItem value="FACT">FACT</MenuItem>
                <MenuItem value="DIMENSION">DIMENSION</MenuItem>
              </TextField>
            </Grid>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <TextField
                select
                fullWidth
                size="small"
                label="Level 3 Classification"
                value={classificationNodeId}
                onChange={e => setClassificationNodeId(e.target.value)}
                sx={{ '& .MuiSelect-select': { color: '#F8FAFC' }, label: { color: '#94A3B8' } }}
              >
                <MenuItem value="820b1234-0000-4000-a000-000000000001">Sales &gt; Client &gt; Client Entity</MenuItem>
                <MenuItem value="820b1234-0000-4000-a000-000000000002">Finance &gt; General Ledger &gt; Account Master</MenuItem>
              </TextField>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', display: 'block', mb: 1 }}>
        2. Physical Backend Bindings & Scoped Term Discovery
      </Typography>

      {bindings.map((b, bIdx) => (
        <Card key={b.backendId} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', mb: 2, color: '#F8FAFC' }}>
          <Box
            display="flex"
            justifyContent="space-between"
            alignItems="center"
            p={2}
            sx={{ cursor: 'pointer', '&:hover': { bgcolor: '#0E2442' } }}
            onClick={() => setExpandedBindingIndex(expandedBindingIndex === bIdx ? -1 : bIdx)}
          >
            <Stack direction="row" spacing={2} alignItems="center">
              <IconButton
                size="small"
                onClick={e => {
                  e.stopPropagation();
                  handleSetDefault(bIdx);
                }}
                sx={{ color: b.isDefault ? '#F59E0B' : '#64748B' }}
              >
                {b.isDefault ? <StarIcon /> : <StarBorderIcon />}
              </IconButton>
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#F8FAFC' }}>
                  {b.backendName} {b.isDefault && <Chip label="DEFAULT" size="small" sx={{ bgcolor: '#451A03', color: '#FBBF24', fontSize: 10, height: 18, ml: 1 }} />}
                </Typography>
                <Typography variant="caption" sx={{ color: '#94A3B8' }}>
                  Driving Table: <strong>{b.drivingTableName}</strong> | Primary Key: <code>{b.pkColumn}</code> ➔ <code>{b.suggestedBk}</code>
                </Typography>
              </Box>
            </Stack>

            <Stack direction="row" spacing={1} alignItems="center">
              <Chip
                label={`${b.terms.filter(t => t.selected).length}/${b.terms.length} Terms Selected`}
                size="small"
                sx={{ bgcolor: '#071526', color: '#38BDF8', fontSize: 11 }}
              />
              {expandedBindingIndex === bIdx ? <ExpandLessIcon /> : <ExpandMoreIcon />}
            </Stack>
          </Box>

          <Collapse in={expandedBindingIndex === bIdx}>
            <Divider sx={{ borderColor: '#1E293B' }} />
            <CardContent>
              <Box sx={{ p: 1.5, bgcolor: '#071526', borderRadius: 1, border: '1px solid #1E293B', mb: 2 }}>
                <Stack direction="row" spacing={1} alignItems="center">
                  <RelationshipIcon sx={{ color: '#00D4FF', fontSize: 18 }} />
                  <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600 }}>
                    Discovered Graph Relationships (FK / JOINS_TO):
                  </Typography>
                  {b.relatedTables.map((rt, idx) => (
                    <Chip key={idx} label={rt} size="small" sx={{ bgcolor: '#1E293B', color: '#CBD5E1', fontSize: 10 }} />
                  ))}
                </Stack>
              </Box>

              <TableContainer component={Paper} sx={{ bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 1 }}>
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
                      <TableCell width={50}>Select</TableCell>
                      <TableCell>Semantic Term Key</TableCell>
                      <TableCell>Term Name</TableCell>
                      <TableCell align="center">Source</TableCell>
                      <TableCell>Column Mapping (Direct / Disambiguated)</TableCell>
                      <TableCell align="center">Status</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {b.terms.map((t, tIdx) => (
                      <TableRow key={t.termNodeId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                        <TableCell>
                          <Checkbox
                            size="small"
                            checked={t.selected}
                            onChange={() => handleToggleTerm(bIdx, tIdx)}
                            sx={{ color: '#64748B', '&.Mui-checked': { color: '#00D4FF' } }}
                          />
                        </TableCell>
                        <TableCell sx={{ fontFamily: 'monospace', fontWeight: 600, color: '#38BDF8', fontSize: 12 }}>
                          {t.termKey}
                        </TableCell>
                        <TableCell sx={{ fontSize: 12 }}>{t.termName}</TableCell>
                        <TableCell align="center">
                          <Chip
                            label={t.sourceType}
                            size="small"
                            sx={{
                              bgcolor: t.sourceType === 'DIRECT' ? '#064E3B' : '#082F49',
                              color: t.sourceType === 'DIRECT' ? '#34D399' : '#38BDF8',
                              fontSize: 10,
                              fontWeight: 700
                            }}
                          />
                        </TableCell>
                        <TableCell>
                          {t.mappings.length === 1 ? (
                            <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#CBD5E1' }}>
                              {t.mappings[0].tableName}.{t.mappings[0].columnName}
                            </Typography>
                          ) : (
                            <RadioGroup
                              row
                              value={t.selectedMappingIndex}
                              onChange={e => handleSelectMapping(bIdx, tIdx, parseInt(e.target.value))}
                            >
                              {t.mappings.map((m, mIdx) => (
                                <FormControlLabel
                                  key={m.columnNodeId}
                                  value={mIdx}
                                  control={<Radio size="small" sx={{ p: 0.5, color: '#64748B', '&.Mui-checked': { color: '#00D4FF' } }} />}
                                  label={
                                    <Typography variant="caption" sx={{ fontFamily: 'monospace', fontSize: 10 }}>
                                      {m.tableName}.{m.columnName} ({m.sourceType})
                                    </Typography>
                                  }
                                />
                              ))}
                            </RadioGroup>
                          )}
                        </TableCell>
                        <TableCell align="center">
                          {t.selected ? (
                            <Chip icon={<ValidIcon sx={{ fontSize: 12, color: '#10B981 !important' }} />} label="Resolved" size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontSize: 10, fontWeight: 700 }} />
                          ) : (
                            <Typography variant="caption" sx={{ color: '#64748B' }}>Unbound</Typography>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Collapse>
        </Card>
      ))}

      <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5, mt: 3 }}>
        <Stack direction="row" spacing={3} alignItems="center" justifyContent="space-between">
          <Stack direction="row" spacing={2} alignItems="center">
            <Chip icon={<ValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />} label="Identity Bound: BK + SID" size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }} />
            <Chip icon={<ValidIcon sx={{ fontSize: 14, color: '#10B981 !important' }} />} label="Required Fields: 3/3 Resolved" size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }} />
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>Publish Status: <strong>READY TO PUBLISH</strong></Typography>
          </Stack>
        </Stack>
      </Paper>
    </Paper>
  );
};

export default SingleScreenBOStudio;
