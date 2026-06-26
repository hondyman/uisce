import React, { useMemo } from 'react';
import { Box, Card, CardContent, CardHeader, Grid, TextField, Switch, FormControlLabel, Typography, Button, Chip } from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DownloadIcon from '@mui/icons-material/Download';
import { SemanticModelConfig } from './types';
import { validateCubeConfig, generateCompanionGovernanceJSON } from './governance';

type Props = {
  config: SemanticModelConfig;
  setConfig: React.Dispatch<React.SetStateAction<SemanticModelConfig>>;
  modelName: string;
  toast: (options: { title: string; description: string; variant?: string }) => void;
};

export default function GovernanceTab({ config, setConfig, modelName, toast }: Props) {
  const coreGov = config.core.options?.governance || {};
  const customGov = config.custom.options?.governance || {};

  const issues = useMemo(() => validateCubeConfig(config, modelName), [config, modelName]);

  const setCoreGov = (patch: any) => {
    setConfig(prev => ({
      ...prev,
      core: {
        ...prev.core,
        options: {
          ...prev.core.options,
          governance: { ...(prev.core.options?.governance || {}), ...patch }
        }
      }
    }));
  };

  const setCustomGov = (patch: any) => {
    setConfig(prev => ({
      ...prev,
      custom: {
        ...prev.custom,
        options: {
          ...prev.custom.options,
          governance: { ...(prev.custom.options?.governance || {}), ...patch }
        }
      }
    }));
  };

  const parseCsv = (val: string) => val.split(',').map(s => s.trim()).filter(Boolean);

  const governanceJson = useMemo(() => generateCompanionGovernanceJSON(config, modelName), [config, modelName]);

  const copy = (text: string) => {
    navigator.clipboard.writeText(text);
    toast({ title: 'Copied', description: 'Governance JSON copied to clipboard.' });
  };
  const download = (text: string) => {
    const blob = new Blob([text], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${modelName || 'model'}-governance.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast({ title: 'Downloaded', description: 'Governance JSON downloaded.' });
  };

  return (
    <Box sx={{ p: 2 }}>
      <Grid container spacing={2}>
        <Grid item xs={12} md={6}>
          <Card>
            <CardHeader title="Core Governance" />
            <CardContent>
              <TextField fullWidth label="Steward" sx={{ mb: 2 }} value={coreGov.steward || ''} onChange={e => setCoreGov({ steward: e.target.value })} />
              <FormControlLabel control={<Switch checked={!!coreGov.pii} onChange={e => setCoreGov({ pii: e.target.checked })} />} label="Contains PII" />
              <TextField fullWidth label="Lineage Note" sx={{ my: 2 }} value={coreGov.lineage || ''} onChange={e => setCoreGov({ lineage: e.target.value })} />
              <TextField fullWidth label="Primary Key Fields (comma-separated)" sx={{ mb: 2 }} value={(coreGov.pkFields || []).join(', ')} onChange={e => setCoreGov({ pkFields: parseCsv(e.target.value) })} />
              <TextField fullWidth label="Tenant Field" sx={{ mb: 2 }} value={coreGov.tenantField || ''} onChange={e => setCoreGov({ tenantField: e.target.value })} />
              <TextField fullWidth label="Audit Fields (comma-separated)" sx={{ mb: 2 }} value={(coreGov.audit_fields || []).join(', ')} onChange={e => setCoreGov({ audit_fields: parseCsv(e.target.value) })} />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardHeader title="Final Cube Governance (overrides)" />
            <CardContent>
              <TextField fullWidth label="Steward" sx={{ mb: 2 }} value={customGov.steward || ''} onChange={e => setCustomGov({ steward: e.target.value })} />
              <FormControlLabel control={<Switch checked={!!customGov.pii} onChange={e => setCustomGov({ pii: e.target.checked })} />} label="Contains PII" />
              <TextField fullWidth label="Lineage Note" sx={{ my: 2 }} value={customGov.lineage || ''} onChange={e => setCustomGov({ lineage: e.target.value })} />
              <TextField fullWidth label="Primary Key Fields (comma-separated)" sx={{ mb: 2 }} value={(customGov.pkFields || []).join(', ')} onChange={e => setCustomGov({ pkFields: parseCsv(e.target.value) })} />
              <TextField fullWidth label="Tenant Field" sx={{ mb: 2 }} value={customGov.tenantField || ''} onChange={e => setCustomGov({ tenantField: e.target.value })} />
              <TextField fullWidth label="Audit Fields (comma-separated)" sx={{ mb: 2 }} value={(customGov.audit_fields || []).join(', ')} onChange={e => setCustomGov({ audit_fields: parseCsv(e.target.value) })} />
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12}>
          <Card>
            <CardHeader title="Validation" />
            <CardContent>
              {issues.length === 0 ? (
                <Typography color="success.main">No issues detected.</Typography>
              ) : (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  {issues.map((i, idx) => (
                    <Box key={idx} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Chip label={i.level.toUpperCase()} color={i.level === 'error' ? 'error' : 'warning'} size="small" />
                      <Typography variant="body2">[{i.code}] {i.message}</Typography>
                    </Box>
                  ))}
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12}>
          <Card>
            <CardHeader title="Companion Governance JSON" action={
              <Box>
                <Button size="small" variant="outlined" startIcon={<ContentCopyIcon />} sx={{ mr: 1 }} onClick={() => copy(governanceJson)}>Copy</Button>
                <Button size="small" variant="contained" startIcon={<DownloadIcon />} onClick={() => download(governanceJson)}>Download</Button>
              </Box>
            } />
            <CardContent>
              <pre className="fabric-pre">{governanceJson}</pre>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
