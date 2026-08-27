import React, { useState, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Box,
  Typography,
  Grid,
  TextField,
  Button,
  Chip,
  Tabs,
  Tab,
  Paper,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Tooltip,
  IconButton,
  Alert,
  Divider,
} from '@mui/material';
import {
  Code2,
  Folder,
  FolderOpen,
  FileText,
  Sparkles,
  Check,
  Copy,
  Layers,
  Split,
  Calendar,
  Settings,
  HelpCircle,
  Play,
  RotateCcw,
} from 'lucide-react';
import {
  SYSTEM_VARIABLES,
  PATH_EXPRESSION_PRESETS,
  PathEvaluationContext,
  getDefaultEvaluationContext,
  evaluatePathExpression,
  VariableDef,
} from './pathExpressionEvaluator';

interface ReportPathExpressionBuilderModalProps {
  open: boolean;
  onClose: () => void;
  folderPath: string;
  fileNamePattern: string;
  exportFormat: 'PDF' | 'EXCEL' | 'BOTH';
  onApply: (newFolderPath: string, newFileNamePattern: string) => void;
  reportName?: string;
  reportId?: string;
  tenantId?: string;
}

export const ReportPathExpressionBuilderModal: React.FC<ReportPathExpressionBuilderModalProps> = ({
  open,
  onClose,
  folderPath: initialFolderPath,
  fileNamePattern: initialFileNamePattern,
  exportFormat,
  onApply,
  reportName,
  reportId,
  tenantId,
}) => {
  const [activeTab, setActiveTab] = useState<'folder' | 'filename' | 'sandbox'>('folder');
  const [currentFolderPath, setCurrentFolderPath] = useState(initialFolderPath);
  const [currentFileNamePattern, setCurrentFileNamePattern] = useState(initialFileNamePattern);

  // Simulation test parameters
  const [simTenantCode, setSimTenantCode] = useState('acme_wealth');
  const [simIsCore, setSimIsCore] = useState(false);
  const [simSliceKey, setSimSliceKey] = useState('client-001');
  const [simSeq, setSimSeq] = useState('001');
  const [selectedCategory, setSelectedCategory] = useState<'all' | 'system' | 'burst' | 'datetime'>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [copiedText, setCopiedText] = useState<string | null>(null);

  // Sync initial props when opening
  React.useEffect(() => {
    if (open) {
      setCurrentFolderPath(initialFolderPath);
      setCurrentFileNamePattern(initialFileNamePattern);
    }
  }, [open, initialFolderPath, initialFileNamePattern]);

  // Build evaluation context
  const testContext = useMemo<PathEvaluationContext>(() => {
    return getDefaultEvaluationContext({
      tenant_code: simTenantCode,
      tenant_id: tenantId || '8f3a9e22-1d54-4f9e-a612-88231901df42',
      tenant_name: simTenantCode.replace('_', ' ').toUpperCase() + ' Management',
      is_core: simIsCore,
      gold_copy: simIsCore,
      report_name: reportName || 'Daily Institutional Client Valuation',
      report_code: (reportName || 'report').toLowerCase().replace(/\s+/g, '_'),
      report_id: reportId || 'rep-custom-001',
      slice_key: simSliceKey,
      client_id: simSliceKey,
      seq: simSeq,
      seq_raw: parseInt(simSeq, 10) || 1,
    });
  }, [simTenantCode, simIsCore, simSliceKey, simSeq, tenantId, reportName, reportId]);

  // Evaluated values
  const folderEval = useMemo(() => evaluatePathExpression(currentFolderPath, testContext), [currentFolderPath, testContext]);
  const fileEval = useMemo(() => evaluatePathExpression(currentFileNamePattern, testContext), [currentFileNamePattern, testContext]);

  // Multi-Slice Simulation (3 simulated partitions)
  const simulatedSlices = useMemo(() => {
    const ext = exportFormat === 'PDF' ? 'pdf' : exportFormat === 'EXCEL' ? 'xlsx' : 'zip';
    const slices = [
      { key: 'client-001', name: 'Apex Capital Alpha Fund', seq: '001' },
      { key: 'client-002', name: 'Beacon Global Wealth Portfolio', seq: '002' },
      { key: 'client-003', name: 'Crestview Sovereign Trust', seq: '003' },
    ];

    return slices.map((s) => {
      const sliceCtx = {
        ...testContext,
        slice_key: s.key,
        client_id: s.key,
        slice_name: s.name,
        client_name: s.name,
        seq: s.seq,
        seq_raw: parseInt(s.seq, 10),
      };

      const evalFolder = evaluatePathExpression(currentFolderPath, sliceCtx).result;
      const cleanFolder = evalFolder.endsWith('/') ? evalFolder : evalFolder + '/';
      const evalFile = evaluatePathExpression(currentFileNamePattern, sliceCtx).result;

      return {
        key: s.key,
        name: s.name,
        seq: s.seq,
        folder: cleanFolder,
        filename: `${evalFile}.${ext}`,
        fullPath: `${cleanFolder}${evalFile}.${ext}`,
      };
    });
  }, [testContext, currentFolderPath, currentFileNamePattern, exportFormat]);

  // Filtered variables
  const filteredVars = useMemo(() => {
    return SYSTEM_VARIABLES.filter((v) => {
      const matchesCategory = selectedCategory === 'all' || v.category === selectedCategory;
      const matchesSearch =
        v.variable.toLowerCase().includes(searchQuery.toLowerCase()) ||
        v.displayName.toLowerCase().includes(searchQuery.toLowerCase()) ||
        v.description.toLowerCase().includes(searchQuery.toLowerCase());
      return matchesCategory && matchesSearch;
    });
  }, [selectedCategory, searchQuery]);

  const handleInsertVariable = (variable: string) => {
    if (activeTab === 'folder') {
      setCurrentFolderPath((prev) => {
        if (prev.startsWith('=')) {
          return `${prev} + ${variable}`;
        }
        return `${prev}${prev.endsWith('/') || prev === '' ? '' : '/'}${variable}`;
      });
    } else {
      setCurrentFileNamePattern((prev) => {
        if (prev.startsWith('=')) {
          return `${prev} + "_" + ${variable}`;
        }
        return `${prev}${prev === '' ? '' : '_'}${variable}`;
      });
    }
  };

  const handleApplyPreset = (preset: (typeof PATH_EXPRESSION_PRESETS)[0]) => {
    setCurrentFolderPath(preset.folderExpr);
    setCurrentFileNamePattern(preset.fileExpr);
  };

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedText(text);
    setTimeout(() => setCopiedText(null), 1500);
  };

  const handleSave = () => {
    onApply(currentFolderPath, currentFileNamePattern);
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth PaperProps={{ sx: { bgcolor: '#0B132B', color: '#E2E8F0', borderRadius: 2.5, border: '1px solid rgba(255,255,255,0.1)' } }}>
      {/* Title */}
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.08)', pb: 1.5 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Code2 size={22} color="#2DD4BF" />
          <Box>
            <Typography variant="subtitle1" sx={{ fontWeight: 800, color: '#F8FAFC' }}>
              Dynamic Path &amp; Name Expression Builder
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              SSRS/Crystal Formula &amp; System Variable Routing Engine for Multi-Tenant Bursting
            </Typography>
          </Box>
        </Box>
        <Chip
          size="small"
          label="Multi-Tenant Router"
          sx={{ bgcolor: 'rgba(13, 148, 136, 0.15)', color: '#2DD4BF', border: '1px solid rgba(13, 148, 136, 0.3)', fontWeight: 700 }}
        />
      </DialogTitle>

      <DialogContent sx={{ p: 3 }}>
        <Grid container spacing={3}>
          
          {/* Left Column: Expression Inputs & Simulation */}
          <Grid size={{ xs: 12, md: 7 }}>
            {/* Navigation Tabs */}
            <Tabs
              value={activeTab}
              onChange={(_, val) => setActiveTab(val)}
              sx={{
                mb: 2,
                minHeight: 38,
                '& .MuiTab-root': {
                  minHeight: 38,
                  textTransform: 'none',
                  fontWeight: 700,
                  fontSize: '0.82rem',
                  color: '#94A3B8',
                  '&.Mui-selected': { color: '#2DD4BF' },
                },
                '& .MuiTabs-indicator': { bgcolor: '#2DD4BF' },
              }}
            >
              <Tab icon={<Folder size={15} />} iconPosition="start" value="folder" label="Destination Folder Expression" />
              <Tab icon={<FileText size={15} />} iconPosition="start" value="filename" label="File Name Expression" />
              <Tab icon={<Layers size={15} />} iconPosition="start" value="sandbox" label="Multi-Slice Sandbox" />
            </Tabs>

            {/* Folder Expression Tab */}
            {activeTab === 'folder' && (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Box>
                  <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8', mb: 0.5, display: 'block' }}>
                    Destination Folder Path Expression:
                  </Typography>
                  <TextField
                    fullWidth
                    size="small"
                    multiline
                    minRows={2}
                    value={currentFolderPath}
                    onChange={(e) => setCurrentFolderPath(e.target.value)}
                    placeholder="/tenants/@tenant_code/@year/@month/ or =IIF(@is_core, ...)"
                    InputProps={{
                      sx: {
                        fontFamily: 'monospace',
                        fontSize: '0.85rem',
                        bgcolor: 'rgba(15, 23, 42, 0.6)',
                        color: '#38BDF8',
                        fontWeight: 700,
                      },
                    }}
                  />
                  <Typography variant="caption" sx={{ color: '#64748B', display: 'block', mt: 0.5 }}>
                    Prefix with <code>=</code> for formula mode (e.g. <code>=IIF(@is_core, &apos;/core_reports/&apos;, &apos;/tenants/&apos; + @tenant_code + &apos;/&apos;)</code>) or use standard variable interpolation.
                  </Typography>
                </Box>

                {/* Live Evaluated Preview */}
                <Paper sx={{ p: 2, bgcolor: 'rgba(15, 23, 42, 0.7)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 2 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                    <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8', display: 'flex', alignItems: 'center', gap: 0.5 }}>
                      <Sparkles size={14} color="#2DD4BF" /> Evaluated Folder Path:
                    </Typography>
                    {folderEval.error ? (
                      <Chip size="small" label="Formula Error" color="error" sx={{ height: 18, fontSize: '0.65rem' }} />
                    ) : (
                      <Chip size="small" label={folderEval.isFormula ? 'Formula Evaluated' : 'Interpolated'} sx={{ height: 18, fontSize: '0.65rem', bgcolor: 'rgba(13, 148, 136, 0.2)', color: '#2DD4BF' }} />
                    )}
                  </Box>
                  <Typography variant="body2" sx={{ fontFamily: 'monospace', color: folderEval.error ? '#EF4444' : '#F8FAFC', fontWeight: 700, wordBreak: 'break-all' }}>
                    {folderEval.error ? folderEval.error : (folderEval.result.endsWith('/') ? folderEval.result : `${folderEval.result}/`)}
                  </Typography>
                </Paper>
              </Box>
            )}

            {/* File Name Expression Tab */}
            {activeTab === 'filename' && (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Box>
                  <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8', mb: 0.5, display: 'block' }}>
                    Report File Name Expression:
                  </Typography>
                  <TextField
                    fullWidth
                    size="small"
                    multiline
                    minRows={2}
                    value={currentFileNamePattern}
                    onChange={(e) => setCurrentFileNamePattern(e.target.value)}
                    placeholder="@report_code_@slice_key_@date_@seq"
                    InputProps={{
                      sx: {
                        fontFamily: 'monospace',
                        fontSize: '0.85rem',
                        bgcolor: 'rgba(15, 23, 42, 0.6)',
                        color: '#A78BFA',
                        fontWeight: 700,
                      },
                    }}
                  />
                  <Typography variant="caption" sx={{ color: '#64748B', display: 'block', mt: 0.5 }}>
                    Supports <code>@seq</code> (001), <code>@slice_key</code> (client-001), <code>@date</code>, or formulas like <code>=Concat(@report_code, &apos;_&apos;, @slice_key)</code>.
                  </Typography>
                </Box>

                {/* Live Evaluated Preview */}
                <Paper sx={{ p: 2, bgcolor: 'rgba(15, 23, 42, 0.7)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 2 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                    <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8', display: 'flex', alignItems: 'center', gap: 0.5 }}>
                      <Sparkles size={14} color="#A78BFA" /> Evaluated File Name:
                    </Typography>
                    {fileEval.error ? (
                      <Chip size="small" label="Formula Error" color="error" sx={{ height: 18, fontSize: '0.65rem' }} />
                    ) : (
                      <Chip size="small" label={fileEval.isFormula ? 'Formula Evaluated' : 'Interpolated'} sx={{ height: 18, fontSize: '0.65rem', bgcolor: 'rgba(167, 139, 250, 0.2)', color: '#A78BFA' }} />
                    )}
                  </Box>
                  <Typography variant="body2" sx={{ fontFamily: 'monospace', color: fileEval.error ? '#EF4444' : '#F8FAFC', fontWeight: 700, wordBreak: 'break-all' }}>
                    {fileEval.error ? fileEval.error : `${fileEval.result}.${exportFormat === 'PDF' ? 'pdf' : exportFormat === 'EXCEL' ? 'xlsx' : 'zip'}`}
                  </Typography>
                </Paper>
              </Box>
            )}

            {/* Sandbox & Full Resolved View */}
            <Box sx={{ mt: 2.5 }}>
              <Typography variant="caption" sx={{ fontWeight: 700, color: '#94A3B8', mb: 1, display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <Split size={14} /> Multi-Slice Burst Output Simulation (3 Sample Client Slices):
              </Typography>
              <Paper sx={{ bgcolor: 'rgba(15, 23, 42, 0.5)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2, overflow: 'hidden' }}>
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'rgba(0,0,0,0.2)' }}>
                      <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem', fontWeight: 700 }}>Client Slice</TableCell>
                      <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem', fontWeight: 700 }}>Seq #</TableCell>
                      <TableCell sx={{ color: '#94A3B8', fontSize: '0.7rem', fontWeight: 700 }}>Resolved Output Path</TableCell>
                      <TableCell sx={{ width: 40 }}></TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {simulatedSlices.map((s) => (
                      <TableRow key={s.key} sx={{ '&:hover': { bgcolor: 'rgba(255,255,255,0.03)' } }}>
                        <TableCell sx={{ color: '#2DD4BF', fontSize: '0.72rem', fontWeight: 700 }}>
                          {s.key}
                          <Typography variant="caption" sx={{ display: 'block', color: '#64748B', fontSize: '0.62rem' }}>
                            {s.name}
                          </Typography>
                        </TableCell>
                        <TableCell sx={{ color: '#A78BFA', fontSize: '0.72rem', fontFamily: 'monospace', fontWeight: 700 }}>
                          #{s.seq}
                        </TableCell>
                        <TableCell sx={{ color: '#F8FAFC', fontSize: '0.72rem', fontFamily: 'monospace', wordBreak: 'break-all' }}>
                          {s.fullPath}
                        </TableCell>
                        <TableCell>
                          <Tooltip title="Copy Path">
                            <IconButton size="small" onClick={() => handleCopy(s.fullPath)} sx={{ color: '#64748B', '&:hover': { color: '#2DD4BF' } }}>
                              {copiedText === s.fullPath ? <Check size={13} color="#2DD4BF" /> : <Copy size={13} />}
                            </IconButton>
                          </Tooltip>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </Paper>
            </Box>

            {/* Simulation Parameter Controls */}
            <Box sx={{ mt: 2, p: 1.5, bgcolor: 'rgba(0,0,0,0.2)', borderRadius: 2, border: '1px solid rgba(255,255,255,0.05)', display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
              <Typography variant="caption" sx={{ color: '#64748B', fontWeight: 700 }}>
                Sandbox Test Parameters:
              </Typography>
              <TextField
                size="small"
                label="Simulated Tenant"
                value={simTenantCode}
                onChange={(e) => setSimTenantCode(e.target.value)}
                sx={{ width: 140, '& .MuiInputBase-input': { fontSize: '0.75rem', color: '#E2E8F0' }, '& label': { color: '#64748B', fontSize: '0.75rem' } }}
              />
              <Button
                size="small"
                variant={simIsCore ? 'contained' : 'outlined'}
                onClick={() => setSimIsCore(!simIsCore)}
                sx={{ fontSize: '0.7rem', textTransform: 'none', borderRadius: 1.5, bgcolor: simIsCore ? '#0D9488' : 'transparent', color: simIsCore ? '#FFF' : '#94A3B8' }}
              >
                @is_core = {simIsCore ? 'TRUE' : 'FALSE'}
              </Button>
            </Box>
          </Grid>

          {/* Right Column: Variable Palette & Quick Presets */}
          <Grid size={{ xs: 12, md: 5 }}>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, height: '100%' }}>
              
              {/* Presets Header */}
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 800, color: '#F8FAFC', mb: 1, display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Sparkles size={16} color="#F59E0B" /> Routing Presets
                </Typography>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  {PATH_EXPRESSION_PRESETS.map((p) => (
                    <Paper
                      key={p.name}
                      onClick={() => handleApplyPreset(p)}
                      sx={{
                        p: 1.2,
                        bgcolor: 'rgba(15, 23, 42, 0.5)',
                        border: '1px solid rgba(255,255,255,0.08)',
                        borderRadius: 1.5,
                        cursor: 'pointer',
                        transition: 'all 0.15s',
                        '&:hover': {
                          borderColor: '#2DD4BF',
                          bgcolor: 'rgba(13, 148, 136, 0.1)',
                        },
                      }}
                    >
                      <Typography variant="body2" sx={{ fontWeight: 700, color: '#F8FAFC', fontSize: '0.78rem' }}>
                        {p.name}
                      </Typography>
                      <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: '0.68rem', display: 'block' }}>
                        {p.description}
                      </Typography>
                    </Paper>
                  ))}
                </Box>
              </Box>

              <Divider sx={{ borderColor: 'rgba(255,255,255,0.08)' }} />

              {/* Variables Palette Header */}
              <Box>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="subtitle2" sx={{ fontWeight: 800, color: '#F8FAFC', display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Code2 size={16} color="#2DD4BF" /> System &amp; Burst Variables
                  </Typography>
                  <Typography variant="caption" sx={{ color: '#64748B' }}>
                    Click to insert
                  </Typography>
                </Box>

                {/* Filter categories */}
                <Box sx={{ display: 'flex', gap: 0.5, mb: 1.5, flexWrap: 'wrap' }}>
                  {[
                    { id: 'all', label: 'All' },
                    { id: 'system', label: '@system' },
                    { id: 'burst', label: '@burst' },
                    { id: 'datetime', label: '@date' },
                  ].map((cat) => (
                    <Chip
                      key={cat.id}
                      size="small"
                      label={cat.label}
                      onClick={() => setSelectedCategory(cat.id as any)}
                      sx={{
                        fontSize: '0.68rem',
                        fontWeight: 700,
                        cursor: 'pointer',
                        bgcolor: selectedCategory === cat.id ? '#0D9488' : 'rgba(255,255,255,0.06)',
                        color: selectedCategory === cat.id ? '#FFF' : '#94A3B8',
                      }}
                    />
                  ))}
                </Box>

                {/* Variable List */}
                <Box sx={{ maxHeight: 260, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 0.8, pr: 0.5 }}>
                  {filteredVars.map((v) => (
                    <Box
                      key={v.variable}
                      onClick={() => handleInsertVariable(v.variable)}
                      sx={{
                        p: 1,
                        bgcolor: 'rgba(15, 23, 42, 0.6)',
                        border: '1px solid rgba(255,255,255,0.06)',
                        borderRadius: 1.5,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        '&:hover': {
                          bgcolor: 'rgba(13, 148, 136, 0.15)',
                          borderColor: '#2DD4BF',
                        },
                      }}
                    >
                      <Box>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography variant="caption" sx={{ fontFamily: 'monospace', fontWeight: 800, color: '#2DD4BF', fontSize: '0.78rem' }}>
                            {v.variable}
                          </Typography>
                          <Chip
                            size="small"
                            label={v.category}
                            sx={{ height: 16, fontSize: '0.6rem', bgcolor: 'rgba(0,0,0,0.3)', color: '#94A3B8' }}
                          />
                        </Box>
                        <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: '0.68rem', display: 'block' }}>
                          {v.description}
                        </Typography>
                      </Box>
                      <Chip
                        size="small"
                        label={`e.g. ${v.example}`}
                        sx={{ height: 18, fontSize: '0.62rem', fontFamily: 'monospace', bgcolor: 'rgba(255,255,255,0.05)', color: '#CBD5E1' }}
                      />
                    </Box>
                  ))}
                </Box>
              </Box>

            </Box>
          </Grid>
        </Grid>
      </DialogContent>

      <DialogActions sx={{ p: 2, borderTop: '1px solid rgba(255,255,255,0.08)', justifyContent: 'space-between' }}>
        <Button onClick={onClose} sx={{ color: '#94A3B8', textTransform: 'none', fontWeight: 700 }}>
          Cancel
        </Button>
        <Box sx={{ display: 'flex', gap: 1.5 }}>
          <Button
            variant="contained"
            onClick={handleSave}
            startIcon={<Check size={16} />}
            sx={{
              bgcolor: '#0D9488',
              color: '#FFF',
              fontWeight: 800,
              textTransform: 'none',
              px: 3,
              '&:hover': { bgcolor: '#0F766E' },
            }}
          >
            Apply Expressions &amp; Return
          </Button>
        </Box>
      </DialogActions>
    </Dialog>
  );
};

export default ReportPathExpressionBuilderModal;
