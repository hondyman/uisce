import React, { useState } from 'react';
import { 
  Box, Button, TextField, Typography, Paper, Grid, 
  MenuItem, Select, FormControl, InputLabel, Chip, Stack 
} from '@mui/material';
import { startModelChangeBP } from '../api/modelChangeApi';

// Mock Data
const MOCK_MODELS = [
  { id: 'MOD-GROWTH-2025', name: 'Growth 2025 (Aggressive)' },
  { id: 'MOD-BALANCED-2025', name: 'Balanced 2025' },
  { id: 'MOD-CONSERVATIVE', name: 'Preservation Core' },
];

const MOCK_ACCOUNTS = [
  { id: 'ACC-101', name: 'Joint Tenant', value: '$1.2M', model: 'MOD-BALANCED-2025' },
  { id: 'ACC-102', name: 'IRA', value: '$450k', model: 'MOD-BALANCED-2025' },
  { id: 'ACC-103', name: 'Trust', value: '$2.1M', model: 'MOD-GROWTH-2025' },
];

export const ModelChangeProposalForm: React.FC = () => {
  const [fromModel, setFromModel] = useState('');
  const [toModel, setToModel] = useState('');
  const [selectedAccounts, setSelectedAccounts] = useState<string[]>([]);
  const [rationale, setRationale] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async () => {
    setIsSubmitting(true);
    try {
      await startModelChangeBP({
        clientId: 'CLI-JaneDoe', // hardcoded for demo
        initiatorUserId: 'ADV-AlexSmith',
        fromModelId: fromModel,
        toModelId: toModel,
        accountIds: selectedAccounts,
        rationale: rationale
      });
      alert('Model Change Process Started via Temporal!');
      // In real app: navigate to status page
    } catch (e) {
      console.error(e);
      alert('Failed to start process');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Box p={3} maxWidth={1200} margin="auto">
      <Typography variant="h4" gutterBottom>Propose Model Change</Typography>
      
      <Grid container spacing={3}>
        {/* Left Panel: Context */}
        <Grid item xs={12} md={4}>
           <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
             <Typography variant="h6">Client: Jane Doe</Typography>
             <Chip label="Risk Profile: Moderate" color="primary" variant="outlined" size="small" sx={{ mt: 1 }} />
             <Chip label="Discretionary: False" color="warning" variant="outlined" size="small" sx={{ mt: 1, ml: 1 }} />
           </Paper>
           
           <Paper variant="outlined" sx={{ p: 2 }}>
             <Typography variant="subtitle1" gutterBottom>Current Accounts</Typography>
             {MOCK_ACCOUNTS.map(acc => (
               <Box key={acc.id} mb={1} p={1} bgcolor="action.hover" borderRadius={1}>
                 <Typography variant="body2" fontWeight="bold">{acc.name}</Typography>
                 <Typography variant="caption" color="text.secondary">{acc.model} • {acc.value}</Typography>
               </Box>
             ))}
           </Paper>
        </Grid>

        {/* Right Panel: Proposal */}
        <Grid item xs={12} md={8}>
          <Paper sx={{ p: 3 }}>
             <Typography variant="h6" gutterBottom>Change Details</Typography>
             
             <FormControl fullWidth margin="normal">
               <InputLabel>Target Model</InputLabel>
               <Select value={toModel} label="Target Model" onChange={(e) => setToModel(e.target.value)}>
                  {MOCK_MODELS.map(m => <MenuItem key={m.id} value={m.id}>{m.name}</MenuItem>)}
               </Select>
             </FormControl>

             <FormControl fullWidth margin="normal">
               <InputLabel>Accounts in Scope</InputLabel>
               <Select 
                 multiple 
                 value={selectedAccounts} 
                 label="Accounts in Scope" 
                 onChange={(e) => setSelectedAccounts(typeof e.target.value === 'string' ? e.target.value.split(',') : e.target.value)}
                 renderValue={(selected) => (
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                      {selected.map((value) => (
                        <Chip key={value} label={MOCK_ACCOUNTS.find(a => a.id === value)?.name || value} />
                      ))}
                    </Box>
                  )}
               >
                 {MOCK_ACCOUNTS.map(a => <MenuItem key={a.id} value={a.id}>{a.name}</MenuItem>)}
               </Select>
             </FormControl>

             <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>Rationale (LLM Input)</Typography>
             <Typography variant="body2" color="text.secondary" paragraph>
                This text will be analyzed by the "Interpretation" LLM step (S1) to structure the intent.
             </Typography>
             <TextField
               fullWidth
               multiline
               rows={4}
               label="Why are you proposing this change?"
               placeholder="e.g., Client requested more growth exposure..."
               value={rationale}
               onChange={(e) => setRationale(e.target.value)}
             />

             <Box mt={3} display="flex" justifyContent="flex-end">
               <Button 
                 variant="contained" 
                 size="large" 
                 onClick={handleSubmit} 
                 disabled={isSubmitting || !toModel}
               >
                 {isSubmitting ? 'Starting Workflow...' : 'Submit for Suitability Review'}
               </Button>
             </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};
