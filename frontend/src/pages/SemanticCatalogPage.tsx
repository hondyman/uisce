import React, { useState, useEffect } from 'react';
import { 
  Box, Typography, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, 
  Chip, TextField, IconButton, Button, Dialog, DialogTitle, DialogContent, Grid, Card, CardContent
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import VisibilityIcon from '@mui/icons-material/Visibility';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningIcon from '@mui/icons-material/Warning';

interface SemanticTerm {
  id: string;
  node_name: string;
  type: 'physical' | 'calculated' | 'relationship' | 'llm';
  data_type: string;
  description: string;
  expression?: string;
  properties?: any;
}

export const SemanticCatalogPage: React.FC = () => {
  const [terms, setTerms] = useState<SemanticTerm[]>([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState('');
  const [selectedTerm, setSelectedTerm] = useState<SemanticTerm | null>(null);

  useEffect(() => {
    setLoading(true);
    fetch('/api/semantic-terms')
      .then(res => res.json())
      .then(data => setTerms(data.data || []))
      .catch(err => console.error(err))
      .finally(() => setLoading(false));
  }, []);

  const filteredTerms = terms.filter(t => 
    t.node_name.toLowerCase().includes(filter.toLowerCase()) || 
    t.description?.toLowerCase().includes(filter.toLowerCase())
  );

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'physical': return 'default';
      case 'calculated': return 'primary';
      case 'relationship': return 'secondary';
      case 'llm': return 'warning';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ p: 4 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4">Semantic Term Catalog</Typography>
        <Button variant="contained" color="primary">Create New Term</Button>
      </Box>

      <Paper sx={{ p: 2, mb: 3 }}>
        <TextField 
          fullWidth 
          variant="outlined" 
          placeholder="Search semantic terms..." 
          value={filter}
          onChange={e => setFilter(e.target.value)}
        />
      </Paper>

      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Data Type</TableCell>
              <TableCell>Description</TableCell>
              <TableCell>Governance</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredTerms.map(term => (
              <TableRow key={term.id} hover>
                <TableCell sx={{ fontWeight: 'medium' }}>{term.node_name}</TableCell>
                <TableCell>
                  <Chip label={term.type} size="small" color={getTypeColor(term.type) as any} variant="outlined" />
                </TableCell>
                <TableCell>{term.data_type}</TableCell>
                <TableCell>{term.description}</TableCell>
                <TableCell>
                  {/* Mock Status */}
                  <Chip icon={<CheckCircleIcon />} label="Approved" size="small" color="success" sx={{ mr: 1 }} />
                </TableCell>
                <TableCell align="right">
                  <IconButton size="small" onClick={() => setSelectedTerm(term)}>
                    <VisibilityIcon />
                  </IconButton>
                  <IconButton size="small">
                    <EditIcon />
                  </IconButton>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Detail Dialog */}
      <Dialog open={!!selectedTerm} onClose={() => setSelectedTerm(null)} maxWidth="md" fullWidth>
        <DialogTitle>Term Details: {selectedTerm?.node_name}</DialogTitle>
        <DialogContent>
          {selectedTerm && (
            <Grid container spacing={3} sx={{ mt: 1 }}>
              <Grid item xs={12}>
                <Typography variant="subtitle2">Description</Typography>
                <Typography variant="body1" paragraph>{selectedTerm.description}</Typography>
              </Grid>
              <Grid item xs={6}>
                 <Card variant="outlined">
                   <CardContent>
                     <Typography variant="subtitle2" color="text.secondary">Definition</Typography>
                     <Box sx={{ mt: 1, p: 1, bgcolor: '#f5f5f5', borderRadius: 1, fontFamily: 'monospace' }}>
                       {selectedTerm.type === 'calculated' ? selectedTerm.expression : 
                        selectedTerm.type === 'physical' ? `${selectedTerm.properties?.physical_mapping?.table}.${selectedTerm.properties?.physical_mapping?.column}` :
                        selectedTerm.type === 'relationship' ? selectedTerm.properties?.relationship?.join_expression : 
                        'N/A'}
                     </Box>
                   </CardContent>
                 </Card>
              </Grid>
              <Grid item xs={6}>
                 <Card variant="outlined">
                   <CardContent>
                     <Typography variant="subtitle2" color="text.secondary">Lineage</Typography>
                     <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mt: 1 }}>
                       {selectedTerm.properties?.lineage?.map((dep: string) => (
                         <Chip key={dep} label={dep} size="small" />
                       )) || <Typography variant="caption" color="text.secondary">No upstream dependencies</Typography>}
                     </Box>
                   </CardContent>
                 </Card>
              </Grid>
            </Grid>
          )}
        </DialogContent>
      </Dialog>
    </Box>
  );
};
