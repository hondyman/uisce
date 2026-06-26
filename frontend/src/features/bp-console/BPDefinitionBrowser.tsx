import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  TextField,
  InputAdornment,
  Grid,
  Card,
  CardContent,
  Divider,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import EditIcon from '@mui/icons-material/Edit';
import VisibilityIcon from '@mui/icons-material/Visibility';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import HistoryIcon from '@mui/icons-material/History';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';

// Types
interface BPDefinition {
  id: string;
  name: string;
  version: string;
  status: 'Active' | 'Draft' | 'Archived';
  category: string;
  lastUpdated: string;
  owner: string;
  stepsCount: number;
}

const BPDefinitionBrowser: React.FC = () => {
  const [definitions, setDefinitions] = useState<BPDefinition[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedBP, setSelectedBP] = useState<BPDefinition | null>(null);

  useEffect(() => {
    // Mock data
    const mockDefinitions: BPDefinition[] = [
      {
        id: 'bp-001',
        name: 'Model Change Approval',
        version: 'v2.1.0',
        status: 'Active',
        category: 'Wealth Management',
        lastUpdated: '2025-12-15T10:00:00Z',
        owner: 'Compliance Team',
        stepsCount: 12,
      },
      {
        id: 'bp-002',
        name: 'Client Onboarding',
        version: 'v1.5.2',
        status: 'Active',
        category: 'Operations',
        lastUpdated: '2025-11-20T14:30:00Z',
        owner: 'Ops Team',
        stepsCount: 24,
      },
      {
        id: 'bp-003',
        name: 'Trade Settlement Exception',
        version: 'v1.0.0',
        status: 'Draft',
        category: 'Back Office',
        lastUpdated: '2026-01-01T09:00:00Z',
        owner: 'Settlements',
        stepsCount: 8,
      },
    ];
    setDefinitions(mockDefinitions);
  }, []);

  const filteredDefs = definitions.filter(d => 
    d.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    d.category.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight={600}>
          <AccountTreeIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          Process Definitions
        </Typography>
        <TextField
          size="small"
          placeholder="Search processes..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
          sx={{ width: 300 }}
        />
      </Stack>

      <Grid container spacing={3}>
        {/* Left: Definition List */}
        <Grid item xs={12} md={7}>
          <TableContainer component={Paper} sx={{ borderRadius: 2 }}>
            <Table>
              <TableHead>
                <TableRow sx={{ bgcolor: 'grey.50' }}>
                  <TableCell>Name</TableCell>
                  <TableCell>Version</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Last Updated</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredDefs.map((def) => (
                  <TableRow 
                    key={def.id} 
                    hover 
                    onClick={() => setSelectedBP(def)}
                    selected={selectedBP?.id === def.id}
                    sx={{ cursor: 'pointer' }}
                  >
                    <TableCell>
                      <Typography fontWeight={500}>{def.name}</Typography>
                      <Typography variant="caption" color="text.secondary">{def.category}</Typography>
                    </TableCell>
                    <TableCell>
                      <Chip label={def.version} size="small" variant="outlined" />
                    </TableCell>
                    <TableCell>
                      <Chip 
                        label={def.status} 
                        size="small" 
                        color={def.status === 'Active' ? 'success' : 'default'}
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="caption">{new Date(def.lastUpdated).toLocaleDateString()}</Typography>
                    </TableCell>
                    <TableCell align="right">
                      <IconButton size="small"><EditIcon fontSize="small" /></IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Grid>

        {/* Right: Detail View */}
        <Grid item xs={12} md={5}>
          {selectedBP ? (
            <Card variant="outlined">
              <CardContent>
                <Stack direction="row" justifyContent="space-between" alignItems="start" sx={{ mb: 2 }}>
                  <Box>
                    <Typography variant="h6" gutterBottom>{selectedBP.name}</Typography>
                    <Chip label={selectedBP.category} size="small" sx={{ mr: 1 }} />
                    <Chip label={selectedBP.owner} size="small" variant="outlined" />
                  </Box>
                  <IconButton><VisibilityIcon /></IconButton>
                </Stack>
                
                <Divider sx={{ my: 2 }} />
                
                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <Typography variant="caption" color="text.secondary">Total Steps</Typography>
                    <Typography variant="h6">{selectedBP.stepsCount}</Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="caption" color="text.secondary">Active Instances</Typography>
                    <Typography variant="h6">14</Typography>
                  </Grid>
                </Grid>

                <Box sx={{ mt: 3, p: 2, bgcolor: 'grey.50', borderRadius: 2, textAlign: 'center' }}>
                  <AccountTreeIcon sx={{ fontSize: 40, color: 'text.disabled', mb: 1 }} />
                  <Typography variant="body2" color="text.secondary">
                    Process Graph Visualization Placeholder
                  </Typography>
                </Box>

                <Stack direction="row" alignItems="center" spacing={1} sx={{ mt: 2 }}>
                  <HistoryIcon fontSize="small" color="action" />
                  <Typography variant="caption">
                    Last deployed by <strong>System Admin</strong> on {new Date(selectedBP.lastUpdated).toLocaleString()}
                  </Typography>
                </Stack>
              </CardContent>
            </Card>
          ) : (
            <Box sx={{ 
              height: '100%', 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              color: 'text.secondary',
              p: 4,
              border: '1px dashed #e0e0e0',
              borderRadius: 2
            }}>
              <Typography>Select a process definition to view details</Typography>
            </Box>
          )}
        </Grid>
      </Grid>
    </Box>
  );
};

export default BPDefinitionBrowser;
