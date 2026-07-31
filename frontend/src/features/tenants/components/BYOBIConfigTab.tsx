import React, { useState } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Grid,
  Button,
  Chip,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField
} from '@mui/material';
import DownloadIcon from '@mui/icons-material/Download';
import PowerIcon from '@mui/icons-material/Power';
import StorageIcon from '@mui/icons-material/Storage';
import VerifiedUserIcon from '@mui/icons-material/VerifiedUser';

interface BYOBITabProps {
  tenantId: string;
}

export const BYOBIConfigTab: React.FC<BYOBITabProps> = ({ tenantId }) => {
  const [downloadDialogOpen, setDownloadDialogOpen] = useState(false);
  const [selectedFormat, setSelectedFormat] = useState<string>('Power BI (.pbids)');
  const [boName, setBoName] = useState('Customer');
  const [generatedManifest, setGeneratedManifest] = useState<string>('');

  const handleGenerateManifest = (format: string) => {
    setSelectedFormat(format);
    if (format.includes('Power BI')) {
      setGeneratedManifest(JSON.stringify({
        version: "0.1",
        connections: [{
          details: {
            protocol: "postgresql",
            address: { server: "100.84.50.65", port: "5433", database: "uisce" }
          },
          options: { DirectQuery: true },
          mode: "DirectQuery"
        }]
      }, null, 2));
    } else if (format.includes('Tableau')) {
      setGeneratedManifest(`<?xml version='1.0' encoding='utf-8' ?>
<datasource formatted-name='UisceSemanticOS' inline='true' version='18.1'>
  <connection class='postgres' dbname='uisce' server='100.84.50.65' port='5433' username='postgres'>
    <relation name='${boName}' table='[${boName}]' type='table' />
  </connection>
</datasource>`);
    } else {
      setGeneratedManifest(`cube('${boName}', {
  sql: 'SELECT * FROM "${boName}"',
  measures: { total_count: { type: 'count' } },
  dimensions: { id: { sql: 'id', type: 'string' } }
});`);
    }
    setDownloadDialogOpen(true);
  };

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="h6" fontWeight="700" mb={1}>
        Bring Your Own BI (BYOBI) Connections & Data Contracts
      </Typography>
      <Typography variant="body2" color="textSecondary" mb={3}>
        Download pre-configured 1-click manifests for Power BI, Tableau, and Cube.dev connected to tenant {tenantId}.
      </Typography>

      <Grid container spacing={3} mb={4}>
        {/* Power BI */}
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1.5} mb={1}>
                <PowerIcon sx={{ color: '#facc15' }} />
                <Typography variant="h6" fontWeight="600">Power BI Desktop</Typography>
              </Box>
              <Typography variant="body2" color="#94a3b8" mb={2}>
                Generate DirectQuery `.pbids` manifest for instant Power BI model binding over PGWire (:5433).
              </Typography>
              <Button
                variant="contained"
                startIcon={<DownloadIcon />}
                onClick={() => handleGenerateManifest('Power BI (.pbids)')}
                sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}
              >
                Get PowerBI .pbids
              </Button>
            </CardContent>
          </Card>
        </Grid>

        {/* Tableau */}
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1.5} mb={1}>
                <StorageIcon sx={{ color: '#38bdf8' }} />
                <Typography variant="h6" fontWeight="600">Tableau Data Source</Typography>
              </Box>
              <Typography variant="body2" color="#94a3b8" mb={2}>
                Export formatted Tableau `.tds` data sources containing native semantic object table definitions.
              </Typography>
              <Button
                variant="contained"
                startIcon={<DownloadIcon />}
                onClick={() => handleGenerateManifest('Tableau (.tds)')}
                sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}
              >
                Get Tableau .tds
              </Button>
            </CardContent>
          </Card>
        </Grid>

        {/* Cube.dev */}
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1.5} mb={1}>
                <VerifiedUserIcon sx={{ color: '#c084fc' }} />
                <Typography variant="h6" fontWeight="600">Cube.dev Schema</Typography>
              </Box>
              <Typography variant="body2" color="#94a3b8" mb={2}>
                Export Cube JavaScript semantic schema file mapping measures & dimensions to Uisce OS.
              </Typography>
              <Button
                variant="contained"
                startIcon={<DownloadIcon />}
                onClick={() => handleGenerateManifest('Cube.dev Schema')}
                sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}
              >
                Get Cube.js Schema
              </Button>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Active Data Contracts Table */}
      <Typography variant="h6" fontWeight="700" mb={2}>
        Tenant Data Contracts (Active Lock Versions)
      </Typography>
      <Paper sx={{ bgcolor: '#1e293b', border: '1px solid #334155', overflow: 'hidden' }}>
        <Table>
          <TableHead sx={{ bgcolor: '#0f172a' }}>
            <TableRow>
              <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Business Object</TableCell>
              <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Contract Version</TableCell>
              <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Status</TableCell>
              <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Breaking Change Guard</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            <TableRow>
              <TableCell sx={{ color: '#f8fafc', fontWeight: 600 }}>Customer</TableCell>
              <TableCell sx={{ color: '#38bdf8' }}>v1.0.0</TableCell>
              <TableCell>
                <Chip label="ACTIVE" size="small" color="success" />
              </TableCell>
              <TableCell sx={{ color: '#4ade80' }}>Enforced (Zero Breaking Drift)</TableCell>
            </TableRow>
            <TableRow>
              <TableCell sx={{ color: '#f8fafc', fontWeight: 600 }}>Order</TableCell>
              <TableCell sx={{ color: '#38bdf8' }}>v1.2.0</TableCell>
              <TableCell>
                <Chip label="ACTIVE" size="small" color="success" />
              </TableCell>
              <TableCell sx={{ color: '#4ade80' }}>Enforced (Zero Breaking Drift)</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </Paper>

      {/* Download Dialog */}
      <Dialog open={downloadDialogOpen} onClose={() => setDownloadDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Download {selectedFormat}</DialogTitle>
        <DialogContent>
          <Box mb={2} mt={1}>
            <TextField
              label="Business Object Target"
              value={boName}
              onChange={(e) => setBoName(e.target.value)}
              size="small"
              fullWidth
            />
          </Box>
          <Typography variant="caption" color="textSecondary" mb={1} display="block">
            Manifest Payload:
          </Typography>
          <Paper sx={{ p: 2, bgcolor: '#0f172a', color: '#38bdf8', fontFamily: 'monospace', fontSize: '12px' }}>
            <pre style={{ margin: 0 }}>{generatedManifest}</pre>
          </Paper>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDownloadDialogOpen(false)}>Close</Button>
          <Button
            variant="contained"
            onClick={() => {
              const element = document.createElement("a");
              const file = new Blob([generatedManifest], { type: 'text/plain' });
              element.href = URL.createObjectURL(file);
              element.download = `uisce_byobi_${boName.toLowerCase()}`;
              document.body.appendChild(element);
              element.click();
              setDownloadDialogOpen(false);
            }}
          >
            Download File
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
