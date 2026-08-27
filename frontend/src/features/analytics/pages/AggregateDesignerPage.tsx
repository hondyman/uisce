import React, { useState } from 'react';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import TextField from '@mui/material/TextField';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormControl from '@mui/material/FormControl';
import FormLabel from '@mui/material/FormLabel';
import { devDebug } from '../../../utils/devLogger';

interface AggregateDefinition {
  name: string;
  sourceTable: string;
  dimensions: string[];
  measures: string[];
  filter: string;
  target: 'StarRocks' | 'Cube' | 'Both';
}

export const AggregateDesignerPage: React.FC = () => {
  const theme = useTheme();
  const [definition, setDefinition] = useState<AggregateDefinition>({
    name: '',
    sourceTable: 'trades',
    dimensions: [],
    measures: [],
    filter: '',
    target: 'Both',
  });

  const [previewData, setPreviewData] = useState<any[]>([]);
  const [generatedSQL, setGeneratedSQL] = useState('');
  const [generatedCube, setGeneratedCube] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    setLoading(true);
    setError(null);
    try {
      devDebug('Saving Aggregate:', definition);
      
      const response = await fetch('/api/analytics/aggregates', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(definition),
      });

      if (!response.ok) {
        throw new Error(`Failed to save aggregate: ${response.statusText}`);
      }

      const result = await response.json();
      
      if (result.starrocks_sql) setGeneratedSQL(result.starrocks_sql);
      if (result.cube_schema) setGeneratedCube(result.cube_schema);

      alert('Aggregate Saved Successfully! Audit Record Created.');
    } catch (err: any) {
      setError(err.message);
      console.error('Save failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const handlePreview = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/analytics/preview', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(definition),
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch preview: ${response.statusText}`);
      }

      const data = await response.json();
      setPreviewData(data);
    } catch (err: any) {
      setError(err.message);
      console.error('Preview failed:', err);
      setPreviewData([
          { desk_id: 'DESK-A', total_pnl: 1234.56, note: 'Mock Data (API Failed)' },
          { desk_id: 'DESK-B', total_pnl: 987.65, note: 'Mock Data (API Failed)' },
      ]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ p: 3, bgcolor: theme.palette.background.default, minHeight: '100vh' }}>
      <Typography variant="h5" sx={{ fontWeight: 600, mb: 3, color: theme.palette.text.primary }}>
        Aggregate Designer (Dual-Mode)
      </Typography>
      
      {error && (
        <Paper sx={{ mb: 3, p: 2, bgcolor: theme.palette.error.light, color: theme.palette.error.dark }}>
          Error: {error}
        </Paper>
      )}

      <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', lg: '1fr 1fr' }, gap: 3, bgcolor: theme.palette.background.paper }}>
        <Paper sx={{ p: 3 }}>
          <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
            Definition
          </Typography>
          
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              label="Aggregate Name"
              value={definition.name}
              onChange={(e) => setDefinition({...definition, name: e.target.value})}
              placeholder="e.g., daily_pnl_by_desk"
              fullWidth
              size="small"
            />

            <FormControl fullWidth size="small">
              <FormLabel>Source Table</FormLabel>
              <select 
                value={definition.sourceTable}
                onChange={(e) => setDefinition({...definition, sourceTable: e.target.value})}
                sx={{ 
                  padding: '8px', 
                  borderRadius: 4, 
                  border: `1px solid ${theme.palette.divider}`,
                  width: '100%'
                }}
              >
                <option value="trades">trades</option>
                <option value="compliance_events">compliance_events</option>
              </select>
            </FormControl>

            <TextField
              label="Dimensions (Comma separated)"
              placeholder="desk_id, trade_date"
              onChange={(e) => setDefinition({...definition, dimensions: e.target.value.split(',').map(s => s.trim())})}
              fullWidth
              size="small"
            />

            <TextField
              label="Measures (Comma separated)"
              placeholder="SUM(pnl), AVG(price)"
              onChange={(e) => setDefinition({...definition, measures: e.target.value.split(',').map(s => s.trim())})}
              fullWidth
              size="small"
            />

            <FormControl>
              <FormLabel>Deployment Target</FormLabel>
              <RadioGroup
                row
                value={definition.target}
                onChange={(e) => setDefinition({...definition, target: e.target.value as any})}
              >
                <FormControlLabel value="StarRocks" control={<Radio size="small" />} label="StarRocks (Lakehouse)" />
                <FormControlLabel value="Cube" control={<Radio size="small" />} label="Cube (Semantic)" />
                <FormControlLabel value="Both" control={<Radio size="small" />} label="Both" />
              </RadioGroup>
            </FormControl>

            <Box sx={{ display: 'flex', gap: 2, pt: 2 }}>
                <Button 
                  variant="outlined"
                  onClick={handlePreview} 
                  disabled={loading}
                  sx={{ 
                    bgcolor: loading ? theme.palette.action.hover : theme.palette.primary.light,
                    color: loading ? theme.palette.action.disabled : theme.palette.primary.main,
                    '&:hover': { bgcolor: theme.palette.primary.main }
                  }}
                >
                  {loading ? 'Loading...' : 'Preview Results'}
                </Button>
                <Button 
                  variant="contained"
                  onClick={handleSave} 
                  disabled={loading}
                  sx={{ 
                    bgcolor: loading ? theme.palette.action.hover : theme.palette.primary.main,
                    '&:hover': { bgcolor: theme.palette.primary.dark }
                  }}
                >
                  {loading ? 'Saving...' : 'Save Aggregate'}
                </Button>
            </Box>
          </Box>
        </Paper>

        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            <Paper sx={{ p: 3 }}>
                <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                    Preview
                </Typography>
                {previewData.length > 0 ? (
                    <TableContainer>
                        <Table size="small">
                            <TableHead>
                                <TableRow sx={{ bgcolor: theme.palette.action.hover }}>
                                    {Object.keys(previewData[0]).map(k => (
                                        <TableCell key={k} sx={{ fontWeight: 600, color: theme.palette.text.secondary, fontSize: '0.75rem' }}>{k}</TableCell>
                                    ))}
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {previewData.map((row, i) => (
                                    <TableRow key={i}>
                                        {Object.values(row).map((v: any, j) => (
                                            <TableCell key={j} sx={{ fontSize: '0.875rem', color: theme.palette.text.secondary }}>{v}</TableCell>
                                        ))}
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                ) : (
                    <Typography sx={{ color: theme.palette.text.secondary, fontStyle: 'italic' }}>
                        Run preview to see sample data.
                    </Typography>
                )}
            </Paper>

            {(generatedSQL || generatedCube) && (
                <Paper sx={{ p: 3, bgcolor: theme.palette.background.paper, color: theme.palette.text.primary, fontFamily: 'monospace', fontSize: '0.875rem', overflow: 'auto' }}>
                    <Typography variant="h6" sx={{ fontWeight: 600, mb: 2, color: theme.palette.text.primary }}>
                        Generated Artifacts
                    </Typography>
                    {generatedSQL && (
                        <Box sx={{ mb: 2 }}>
                            <Typography sx={{ color: theme.palette.success.main, mb: 1 }}>StarRocks SQL (Iceberg)</Typography>
                            <pre>{generatedSQL}</pre>
                        </Box>
                    )}
                    {generatedCube && (
                        <Box>
                            <Typography sx={{ color: theme.palette.error.main, mb: 1 }}>Cube Schema</Typography>
                            <pre>{generatedCube}</pre>
                        </Box>
                    )}
                </Paper>
            )}
        </Box>
      </Box>
    </Box>
  );
};
