import React, { useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Typography,
  Stack,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  FormControlLabel,
  Switch,
  Alert,
  Chip,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
} from '@mui/material';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import FileUploadIcon from '@mui/icons-material/FileUpload';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { validationRulesApi, type RuleImportReport } from '../api';

interface RuleBundleManagerProps {
  boName?: string;
  domain?: string;
  onRefresh?: () => void;
}

export const RuleBundleManager: React.FC<RuleBundleManagerProps> = ({
  boName,
  domain,
  onRefresh,
}) => {
  const [exportFormat, setExportFormat] = useState<'json' | 'yaml'>('json');
  const [isExporting, setIsExporting] = useState(false);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [bundlePayload, setBundlePayload] = useState('');
  const [overwritePolicy, setOverwritePolicy] = useState('fail');
  const [preserveStatus, setPreserveStatus] = useState(true);
  const [pruneMissing, setPruneMissing] = useState(false);
  const [preflightLoading, setPreflightLoading] = useState(false);
  const [preflightReport, setPreflightReport] = useState<RuleImportReport | null>(null);
  const [importing, setImporting] = useState(false);
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const handleExport = async (format: 'json' | 'yaml') => {
    setIsExporting(true);
    setActionError(null);
    try {
      const data = await validationRulesApi.exportBundle(boName, domain, undefined, format);
      const content = typeof data === 'string' ? data : JSON.stringify(data, null, 2);
      const mime = format === 'yaml' ? 'application/x-yaml' : 'application/json';
      const blob = new Blob([content], { type: mime });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `rules_${boName || 'all'}_${domain || 'default'}_${new Date().toISOString().slice(0, 10)}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      setActionSuccess(`Exported ${format.toUpperCase()} bundle successfully.`);
    } catch (err: any) {
      setActionError(err.message || 'Export failed');
    } finally {
      setIsExporting(false);
    }
  };

  const handlePreflight = async () => {
    if (!bundlePayload.trim()) {
      setActionError('Bundle payload is empty');
      return;
    }
    setPreflightLoading(true);
    setActionError(null);
    try {
      let payloadObj: any = bundlePayload;
      if (bundlePayload.trim().startsWith('{')) {
        payloadObj = JSON.parse(bundlePayload);
        if (payloadObj.rules) payloadObj = payloadObj;
      }
      const report = await validationRulesApi.preflightImport(
        payloadObj,
        overwritePolicy,
        preserveStatus,
        pruneMissing
      );
      setPreflightReport(report);
    } catch (err: any) {
      setActionError(err.message || 'Preflight check failed');
    } finally {
      setPreflightLoading(false);
    }
  };

  const handleExecuteImport = async () => {
    setImporting(true);
    setActionError(null);
    try {
      let payloadObj: any = bundlePayload;
      if (bundlePayload.trim().startsWith('{')) {
        payloadObj = JSON.parse(bundlePayload);
      }
      const report = await validationRulesApi.importBundle(
        payloadObj,
        overwritePolicy,
        preserveStatus,
        pruneMissing
      );
      if (report.success) {
        setActionSuccess(
          `Import successful! Created: ${report.created.length}, Updated: ${report.updated.length}, Deleted: ${report.deleted.length}`
        );
        setImportModalOpen(false);
        setBundlePayload('');
        setPreflightReport(null);
        if (onRefresh) onRefresh();
      } else {
        setActionError(`Import rejected with ${report.errors.length} validation errors.`);
      }
    } catch (err: any) {
      setActionError(err.message || 'Import failed');
    } finally {
      setImporting(false);
    }
  };

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (event) => {
      const text = event.target?.result as string;
      setBundlePayload(text || '');
      setPreflightReport(null);
    };
    reader.readAsText(file);
  };

  return (
    <Card variant="outlined" sx={{ borderRadius: 2, mb: 3 }}>
      <CardContent>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="center" justifyContent="space-between">
          <Box>
            <Typography variant="h6" fontWeight="bold">
              Rule Bundle Management & GitOps Portability
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Export and import portable AST rule bundles across tenant environments without UUID leakage.
            </Typography>
          </Box>
          <Stack direction="row" spacing={1}>
            <Button
              variant="outlined"
              startIcon={<FileDownloadIcon />}
              onClick={() => handleExport('json')}
              disabled={isExporting}
            >
              Export JSON
            </Button>
            <Button
              variant="outlined"
              startIcon={<FileDownloadIcon />}
              onClick={() => handleExport('yaml')}
              disabled={isExporting}
            >
              Export YAML
            </Button>
            <Button
              variant="contained"
              startIcon={<FileUploadIcon />}
              onClick={() => setImportModalOpen(true)}
            >
              Import Bundle
            </Button>
          </Stack>
        </Stack>

        {actionSuccess && (
          <Alert severity="success" sx={{ mt: 2 }} onClose={() => setActionSuccess(null)}>
            {actionSuccess}
          </Alert>
        )}
        {actionError && (
          <Alert severity="error" sx={{ mt: 2 }} onClose={() => setActionError(null)}>
            {actionError}
          </Alert>
        )}

        {/* Import & Preflight Modal */}
        <Dialog
          open={importModalOpen}
          onClose={() => setImportModalOpen(false)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>Import Portable Rule Bundle</DialogTitle>
          <DialogContent dividers>
            <Stack spacing={2}>
              <Stack direction="row" spacing={2} alignItems="center">
                <Button variant="outlined" component="label">
                  Upload .json or .yaml File
                  <input type="file" accept=".json,.yaml,.yml" hidden onChange={handleFileUpload} />
                </Button>
                <Typography variant="caption" color="text.secondary">
                  Paste raw payload or upload bundle spec
                </Typography>
              </Stack>

              <TextField
                label="Rule Bundle JSON or YAML"
                multiline
                rows={8}
                value={bundlePayload}
                onChange={(e) => {
                  setBundlePayload(e.target.value);
                  setPreflightReport(null);
                }}
                placeholder='{"bundle_version":"1.0","rules":[{"rule_key":"party.min_age","bo_name":"party","severity":"BLOCK",...}]}'
                fullWidth
              />

              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                <FormControl size="small" sx={{ minWidth: 200 }}>
                  <InputLabel>Overwrite Policy</InputLabel>
                  <Select
                    value={overwritePolicy}
                    label="Overwrite Policy"
                    onChange={(e) => setOverwritePolicy(e.target.value)}
                  >
                    <MenuItem value="fail">Fail on conflict (Strict)</MenuItem>
                    <MenuItem value="overwrite">Overwrite existing rules</MenuItem>
                    <MenuItem value="skip">Skip existing (Keep current)</MenuItem>
                  </Select>
                </FormControl>

                <FormControlLabel
                  control={
                    <Switch
                      checked={preserveStatus}
                      onChange={(e) => setPreserveStatus(e.target.checked)}
                    />
                  }
                  label="Preserve Active Status"
                />

                <FormControlLabel
                  control={
                    <Switch
                      checked={pruneMissing}
                      onChange={(e) => setPruneMissing(e.target.checked)}
                    />
                  }
                  label="Prune Missing Rules"
                />
              </Stack>

              <Button
                variant="outlined"
                color="secondary"
                onClick={handlePreflight}
                disabled={preflightLoading || !bundlePayload.trim()}
              >
                {preflightLoading ? <CircularProgress size={20} /> : 'Run Preflight Dry-Run'}
              </Button>

              {/* Preflight Report Card */}
              {preflightReport && (
                <Paper variant="outlined" sx={{ p: 2, bgcolor: 'background.default' }}>
                  <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                    {preflightReport.success ? (
                      <Chip icon={<CheckCircleIcon />} label="Preflight Passed" color="success" size="small" />
                    ) : (
                      <Chip icon={<ErrorOutlineIcon />} label="Preflight Failed" color="error" size="small" />
                    )}
                    <Chip label={`Total Rules: ${preflightReport.total_rules}`} size="small" />
                    <Chip label={`Created: ${preflightReport.created.length}`} color="primary" size="small" />
                    <Chip label={`Updated: ${preflightReport.updated.length}`} color="warning" size="small" />
                    <Chip label={`Unchanged: ${preflightReport.unchanged.length}`} size="small" />
                  </Stack>

                  {preflightReport.errors.length > 0 && (
                    <Alert severity="error" sx={{ my: 1 }}>
                      <Typography variant="subtitle2" fontWeight="bold">Errors:</Typography>
                      {preflightReport.errors.map((err, idx) => (
                        <Typography key={idx} variant="caption" display="block">
                          • [{err.code}] {err.rule_key}: {err.reason}
                        </Typography>
                      ))}
                    </Alert>
                  )}

                  {preflightReport.diff_summary && preflightReport.diff_summary.length > 0 && (
                    <TableContainer sx={{ maxHeight: 200 }}>
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            <TableCell>Rule Key</TableCell>
                            <TableCell>Action</TableCell>
                            <TableCell>BO Name</TableCell>
                            <TableCell>Changes / Errors</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {preflightReport.diff_summary.map((diff, i) => (
                            <TableRow key={i}>
                              <TableCell sx={{ fontFamily: 'monospace' }}>{diff.rule_key}</TableCell>
                              <TableCell>
                                <Chip
                                  label={diff.action}
                                  size="small"
                                  color={
                                    diff.action === 'CREATE'
                                      ? 'success'
                                      : diff.action === 'UPDATE'
                                      ? 'warning'
                                      : diff.action === 'DELETE'
                                      ? 'error'
                                      : 'default'
                                  }
                                />
                              </TableCell>
                              <TableCell>{diff.bo_name}</TableCell>
                              <TableCell>
                                {diff.changes?.join(', ') || diff.rule_errors?.join(', ') || 'None'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  )}
                </Paper>
              )}
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setImportModalOpen(false)}>Cancel</Button>
            <Button
              variant="contained"
              color="primary"
              onClick={handleExecuteImport}
              disabled={
                importing ||
                !bundlePayload.trim() ||
                (preflightReport !== null && !preflightReport.success)
              }
            >
              {importing ? <CircularProgress size={20} /> : 'Apply Import'}
            </Button>
          </DialogActions>
        </Dialog>
      </CardContent>
    </Card>
  );
};
