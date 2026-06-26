import React, { useState, useEffect } from 'react';
import { Box, Button, Card, Typography, IconButton, Table, TableBody, TableCell, TableHead, TableRow, Dialog, DialogTitle, DialogContent, DialogActions, TextField, MenuItem, Select, FormControl, InputLabel } from '@mui/material';
import { Add as AddIcon, Delete as DeleteIcon, Refresh as RefreshIcon } from '@mui/icons-material';
import { useTenant } from '../../../contexts/TenantContext';
import { addCalculation, removeCalculation, getCalculations, SemanticModelCalculation } from '../../../api/semantic_models';
import { useCalculations } from '../../../hooks/useCalculations'; // Assuming this hook exists or I need to create it/use API directly
import { toast } from 'sonner';

interface Props {
  modelId: string;
}

export const SemanticModelCalculations: React.FC<Props> = ({ modelId }) => {
  const [calculations, setCalculations] = useState<SemanticModelCalculation[]>([]);
  const [loading, setLoading] = useState(false);
  const [addModalOpen, setAddModalOpen] = useState(false);
  
  // Add Calculation State
  const [selectedCalcId, setSelectedCalcId] = useState('');
  const [outputName, setOutputName] = useState('');
  const [argumentMapping, setArgumentMapping] = useState<Record<string, string>>({});
  
  // Library Calculations
  const { calculations: libraryCalculations, loading: libraryLoading } = useCalculations(); 

  const fetchCalculations = async () => {
    setLoading(true);
    try {
      const data = await getCalculations(modelId);
      setCalculations(data);
    } catch (error) {
      console.error('Failed to fetch calculations:', error);
      toast.error('Failed to fetch calculations');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (modelId) {
      fetchCalculations();
    }
  }, [modelId]);

  const handleAdd = async () => {
    try {
      await addCalculation(modelId, {
        calculation_id: selectedCalcId,
        argument_mapping: argumentMapping,
        output_name: outputName,
        is_public: true
      });
      toast.success('Calculation added successfully');
      setAddModalOpen(false);
      fetchCalculations();
      // Reset form
      setSelectedCalcId('');
      setOutputName('');
      setArgumentMapping({});
    } catch (error) {
      console.error('Failed to add calculation:', error);
      toast.error('Failed to add calculation');
    }
  };

  const handleRemove = async (calcId: string) => {
    if (!confirm('Are you sure you want to remove this calculation?')) return;
    try {
      await removeCalculation(modelId, calcId);
      toast.success('Calculation removed successfully');
      fetchCalculations();
    } catch (error) {
      console.error('Failed to remove calculation:', error);
      toast.error('Failed to remove calculation');
    }
  };

  const selectedLibraryCalc = libraryCalculations.find(c => c.id === selectedCalcId);

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h6">Calculations</Typography>
        <Button startIcon={<AddIcon />} variant="contained" onClick={() => setAddModalOpen(true)}>
          Add Calculation
        </Button>
      </Box>

      <Card>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Output Name</TableCell>
              <TableCell>Calculation</TableCell>
              <TableCell>Arguments</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {calculations.map((calc) => (
              <TableRow key={calc.id}>
                <TableCell>{calc.output_name}</TableCell>
                <TableCell>{calc.calculation_name || calc.calculation_id}</TableCell>
                <TableCell>
                  {Object.entries(calc.argument_mapping).map(([arg, val]) => (
                    <div key={arg}>{arg}: {val}</div>
                  ))}
                </TableCell>
                <TableCell align="right">
                  <IconButton onClick={() => handleRemove(calc.id)} color="error">
                    <DeleteIcon />
                  </IconButton>
                </TableCell>
              </TableRow>
            ))}
            {calculations.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} align="center">No calculations associated with this model.</TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </Card>

      <Dialog open={addModalOpen} onClose={() => setAddModalOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Add Calculation</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <FormControl fullWidth>
              <InputLabel>Calculation</InputLabel>
              <Select
                value={selectedCalcId}
                label="Calculation"
                onChange={(e) => setSelectedCalcId(e.target.value)}
              >
                {libraryCalculations.map((c: any) => (
                  <MenuItem key={c.id} value={c.id}>{c.name}</MenuItem>
                ))}
              </Select>
            </FormControl>

            <TextField
              label="Output Name"
              value={outputName}
              onChange={(e) => setOutputName(e.target.value)}
              fullWidth
            />

            {selectedLibraryCalc && (
              <Box>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>Argument Mapping</Typography>
                {/* Assuming arguments is a JSON array of objects with name property */}
                {(selectedLibraryCalc.arguments as any[])?.map((arg: any) => (
                  <TextField
                    key={arg.name}
                    label={`Map ${arg.name} to...`}
                    value={argumentMapping[arg.name] || ''}
                    onChange={(e) => setArgumentMapping({ ...argumentMapping, [arg.name]: e.target.value })}
                    fullWidth
                    sx={{ mb: 1 }}
                    helperText={`Type: ${arg.type}`}
                  />
                ))}
              </Box>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAddModalOpen(false)}>Cancel</Button>
          <Button onClick={handleAdd} variant="contained" disabled={!selectedCalcId || !outputName}>
            Add
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
