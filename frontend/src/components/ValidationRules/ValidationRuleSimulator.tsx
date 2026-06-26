import React, { useState } from 'react';
import { Box, Button, Stack, Typography, Paper, Alert, CircularProgress, Chip } from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import RuleIcon from '@mui/icons-material/Rule';
import LightbulbIcon from '@mui/icons-material/Lightbulb';
import ErrorIcon from '@mui/icons-material/Error';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import ValidationRuleScriptEditor from './ValidationRuleScriptEditor'; // Reused for input

interface ValidationRuleSimulatorProps {
  scriptContent: string;
}

export const ValidationRuleSimulator: React.FC<ValidationRuleSimulatorProps> = ({ scriptContent }) => {
  const [testData, setTestData] = useState('{\n  "status": "active",\n  "salary": 50000\n}');
  const [result, setResult] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSimulate = async () => {
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      let parsedData;
      try {
        parsedData = JSON.parse(testData);
      } catch (e) {
        throw new Error("Invalid JSON in Test Data. Please ensure keys are quoted.");
      }

      const response = await fetch('/api/validation-rules/simulate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ script: scriptContent, data: parsedData }),
      });
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || data.message || "Simulation failed");
      }
      setResult(data);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Stack spacing={2} sx={{ height: '100%' }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="subtitle2" fontWeight={600}>Test Data (JSON)</Typography>
        <Button 
          variant="contained" 
          color="primary" 
          startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <PlayArrowIcon />}
          onClick={handleSimulate}
          disabled={loading}
          size="small"
        >
          Run Simulation
        </Button>
      </Box>

      <Box sx={{ display: 'flex', gap: 2, height: '400px' }}>
        {/* Left: Data Input */}
        <Paper variant="outlined" sx={{ flex: 1, borderRadius: 2, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
           <ValidationRuleScriptEditor 
             value={testData} 
             onChange={(val) => setTestData(val || '')} 
             height="100%"
           />
        </Paper>

        {/* Right: Results */}
        <Paper variant="outlined" sx={{ flex: 1, borderRadius: 2, overflow: 'hidden', p: 2, overflowY: 'auto', bgcolor: '#f5f5f5' }}>
           <Typography variant="subtitle2" sx={{ mb: 1 }}>Simulation Results</Typography>
           
           {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
           
           {result && (
             <Stack spacing={2}>
               <Box sx={{ display: 'flex', gap: 1 }}>
                 <Chip 
                   label={result.success ? "Passed (Unified)" : "Failed"} 
                   color={result.success ? "success" : "error"} 
                   icon={result.success ? <PlayArrowIcon /> : undefined}
                 />
               </Box>

               {result.errors && result.errors.length > 0 && (
                 <Alert severity="warning">
                   <Typography variant="subtitle2">Validation Errors:</Typography>
                   <ul style={{ margin: 0, paddingLeft: '1rem' }}>
                     {result.errors.map((e: string, i: number) => (
                       <li key={i}>{e}</li>
                     ))}
                   </ul>
                 </Alert>
               )}


               <Box>
               {/* Unified Outcomes Visualizer */}
               {(result.messages && result.messages.length > 0) && (
                 <Box sx={{ mb: 2 }}>
                   <Typography variant="subtitle2" sx={{ mb: 1, display: 'flex', alignItems: 'center', gap: 1 }}>
                     <RuleIcon fontSize="small" color="error" /> Validation Messages
                   </Typography>
                   <Stack spacing={1}>
                     {result.messages.map((msg: any, i: number) => (
                       <Alert 
                         key={i} 
                         severity={msg.severity === 'error' ? 'error' : msg.severity === 'warning' ? 'warning' : 'info'}
                         sx={{ '& .MuiAlert-message': { width: '100%' } }}
                         icon={msg.severity === 'error' ? <ErrorIcon /> : undefined}
                       >
                         <Box>
                           <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                             {msg.message}
                           </Typography>
                           {msg.field && msg.field !== "" && (
                             <Typography variant="caption" display="block" color="text.secondary">
                               Field: {msg.field}
                             </Typography>
                           )}
                           {msg.why && (
                             <Typography variant="body2" sx={{ mt: 0.5 }}>
                               <strong>Why:</strong> {msg.why}
                             </Typography>
                           )}
                           {msg.fix && (
                             <Box sx={{ mt: 1, p: 1, bgcolor: 'rgba(0,0,0,0.05)', borderRadius: 1 }}>
                               <Typography variant="body2" sx={{ fontWeight: 600, color: 'text.primary' }}>
                                 🔧 Fix: {msg.fix}
                               </Typography>
                             </Box>
                           )}
                         </Box>
                       </Alert>
                     ))}
                   </Stack>
                 </Box>
               )}

                 {/* Unified Workflow Outcomes (Legacy/Direct check) */}
                 {(result.result_data?.requiresApproval || result.result_data?.suggestedAction || result.result_data?.riskLevel) && (
                   <Paper variant="outlined" sx={{ p: 2, mb: 2, borderColor: '#2196f3', bgcolor: '#e3f2fd' }}>
                     <Typography variant="subtitle2" color="primary" sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                       <LightbulbIcon fontSize="small" /> Workflow Actions Triggered
                     </Typography>
                     <Stack spacing={1}>
                       {result.result_data.requiresApproval && (
                         <Chip 
                            label="VP Approval Required" 
                            color="warning" 
                            size="small" 
                            sx={{ width: 'fit-content' }} 
                         />
                       )}
                       {result.result_data.riskLevel && (
                         <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Typography variant="body2" fontWeight={600}>Risk Level:</Typography>
                            <Chip 
                                label={result.result_data.riskLevel.toUpperCase()} 
                                color={result.result_data.riskLevel === 'high' ? 'error' : 'default'} 
                                size="small" 
                            />
                         </Box>
                       )}
                       {result.result_data.suggestedAction && (
                         <Alert severity="info" icon={<LightbulbIcon />}>
                           {result.result_data.suggestedAction}
                         </Alert>
                       )}
                       {result.result_data.uiHints?.warning && (
                         <Alert severity="warning">
                           {result.result_data.uiHints.warning}
                         </Alert>
                       )}
                     </Stack>
                   </Paper>
                 )}

                 <Typography variant="caption" sx={{ display: 'block', mb: 0.5, fontWeight: 600 }}>Resulting Object:</Typography>
                 <SyntaxHighlighter
                    language="json"
                    style={vscDarkPlus}
                    customStyle={{
                      margin: 0,
                      borderRadius: '8px',
                      fontSize: '12px',
                    }}
                    wrapLines
                  >
                   {JSON.stringify(result.result_data || {}, null, 2)}
                 </SyntaxHighlighter>
               </Box>
             </Stack>
           )}
           
           {!result && !error && !loading && (
             <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic', mt: 4, textAlign: 'center' }}>
               Enter test data and click "Run Simulation" to see the outcome.
             </Typography>
           )}
        </Paper>
      </Box>
    </Stack>
  );
};

export default ValidationRuleSimulator;

