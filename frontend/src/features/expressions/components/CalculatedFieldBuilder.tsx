import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Grid,
  Chip,
  Alert,
  Paper,
  Tabs,
  Tab,
  IconButton,
  List,
  ListItem,
  ListItemText,
  CircularProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  PlayArrow as RunIcon,
  Save as SaveIcon,
  Code as CodeIcon,
  Add as AddIcon,
  Search as SearchIcon,
} from '@mui/icons-material';
import Editor from '@monaco-editor/react';

// Types
interface SemanticTerm {
  id: string;
  node_name: string;
  type: string;
  expression?: string;
}

interface CalculatedFieldBuilderProps {
  onSave?: (term: any) => void;
  initialTerm?: SemanticTerm;
}

export const CalculatedFieldBuilder: React.FC<CalculatedFieldBuilderProps> = ({
  onSave,
  initialTerm,
}) => {
  const [name, setName] = useState(initialTerm?.node_name || '');
  const [termType, setTermType] = useState(initialTerm?.type || 'calculated');
  const [expression, setExpression] = useState(initialTerm?.expression || '');
  const [availableTerms, setAvailableTerms] = useState<SemanticTerm[]>([]);
  const [resolveResult, setResolveResult] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [resolving, setResolving] = useState(false);

  // AI Assistant State
  const [aiDialogOpen, setAiDialogOpen] = useState(false);
  const [schemaInput, setSchemaInput] = useState('');
  const [aiLoading, setAiLoading] = useState(false);
  const [aiSuggestions, setAiSuggestions] = useState<any[]>([]);

  // Mock fetching terms (replace with real API call)
  useEffect(() => {
    const fetchTerms = async () => {
      setLoading(true);
      try {
        const res = await fetch('/api/semantic-terms?tenant_instance_id=default');
        const data = await res.json();
        // Handle GraphQL-like wrapping if present based on previous files
        const terms = data.data?.catalog_node || data.data || [];
        setAvailableTerms(terms);
      } catch (err) {
        console.error("Failed to load terms", err);
      } finally {
        setLoading(false);
      }
    };

    fetchTerms();
  }, []);

  const handleResolve = async () => {
    setResolving(true);
    try {
      const res = await fetch('/api/semantic-terms/resolve-expression', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ expression })
      });
      
      if (!res.ok) {
          throw new Error(await res.text());
      }
      
      const data = await res.json();
      setResolveResult(data);
    } catch (err) {
      setResolveResult({ error: String(err) });
    } finally {
      setResolving(false);
    }
  };

  const handleAiSuggest = async () => {
    setAiLoading(true);
    try {
        const res = await fetch('/api/semantic-terms/suggest', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ schema_context: schemaInput })
        });
        const data = await res.json();
        setAiSuggestions(data.suggestions || []);
    } catch (err) {
        console.error("AI Suggestion failed", err);
        // Could show toast error here
    } finally {
        setAiLoading(false);
    }
  };

  const handleInsertTerm = (term: string) => {
    setExpression(prev => prev + ` ${term} `);
  };

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="h5" gutterBottom>
        Calculated Field Builder (Workday Style)
      </Typography>
      
      <Grid container spacing={3}>
        {/* Left Panel: Configuration */}
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Grid container spacing={2}>
                <Grid item xs={12} md={6}>
                  <TextField 
                    fullWidth 
                    label="Field Name" 
                    value={name} 
                    onChange={(e) => setName(e.target.value)} 
                    placeholder="e.g. client.risk_drift"
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <FormControl fullWidth>
                    <InputLabel>Type</InputLabel>
                    <Select
                      value={termType}
                      onChange={(e) => setTermType(e.target.value)}
                      label="Type"
                    >
                      <MenuItem value="calculated">Calculated Field</MenuItem>
                      <MenuItem value="llm">LLM Derived</MenuItem>
                      <MenuItem value="relationship">Relationship</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                
                <Grid item xs={12}>
                   <Typography variant="subtitle2" gutterBottom>
                     Expression Logic
                   </Typography>
                   <Paper variant="outlined">
                     <Editor
                       height="200px"
                       defaultLanguage="sql" // Using SQL syntax highlighting as proxy
                       value={expression}
                       onChange={(val) => setExpression(val || '')}
                       theme="vs-light"
                       options={{
                         minimap: { enabled: false },
                         lineNumbers: 'off',
                         folding: false,
                         fontSize: 14
                       }}
                     />
                   </Paper>
                   <Typography variant="caption" color="text.secondary">
                     Supports: Arithmetic (+ - * /), Functions (SUM, AVG, IF), and Term References (client.risk_score)
                   </Typography>
                </Grid>
                
                <Grid item xs={12}>
                  <Box sx={{ display: 'flex', gap: 2, justifyContent: 'flex-end' }}>
                     <Button 
                       variant="outlined" 
                       startIcon={<RunIcon />} 
                       onClick={handleResolve}
                       disabled={resolving || !expression}
                     >
                       {resolving ? 'Resolving...' : 'Test Resolution'}
                     </Button>
                     <Button 
                       variant="outlined" 
                       startIcon={<CodeIcon />}
                       onClick={() => setAiDialogOpen(true)}
                     >
                       AI Suggest
                     </Button>
                     <Button 
                       variant="contained" 
                       startIcon={<SaveIcon />}
                       onClick={() => onSave?.({ node_name: name, type: termType, expression })}
                     >
                       Save Field
                     </Button>
                  </Box>
                </Grid>
              </Grid>
              
              {resolveResult && (
                <Box sx={{ mt: 3 }}>
                   <Alert severity={resolveResult.error ? "error" : "success"}>
                     <Typography variant="subtitle2">
                       {resolveResult.error ? "Resolution Failed" : "Resolution Successful"}
                     </Typography>
                     {!resolveResult.error && (
                        <Box sx={{ mt: 1 }}>
                          <Typography variant="caption" display="block">Generated SQL:</Typography>
                          <Paper sx={{ p: 1, bgcolor: '#f5f5f5', fontFamily: 'monospace' }}>
                            {resolveResult.sql}
                          </Paper>
                          
                          <Typography variant="caption" display="block" sx={{ mt: 1 }}>Dependencies:</Typography>
                          <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                            {resolveResult.lineage?.map((d: string) => (
                              <Chip key={d} label={d} size="small" variant="outlined" />
                            ))}
                          </Box>
                        </Box>
                     )}
                   </Alert>
                </Box>
              )}
              
            </CardContent>
          </Card>
        </Grid>
        
        {/* Right Panel: Term Browser */}
        <Grid item xs={12} md={4}>
           <Card sx={{ height: '100%' }}>
             <CardContent>
               <Typography variant="subtitle2" gutterBottom>
                 Available Terms
               </Typography>
               <TextField 
                 fullWidth 
                 size="small" 
                 placeholder="Search terms..." 
                 InputProps={{ startAdornment: <SearchIcon fontSize="small" color="action" /> }}
                 sx={{ mb: 2 }}
               />
               
               <Paper variant="outlined" sx={{ maxHeight: '400px', overflow: 'auto' }}>
                 {loading ? (
                   <Box sx={{ p: 2, textAlign: 'center' }}><CircularProgress size={20} /></Box>
                 ) : (
                   <List dense>
                     {availableTerms.map(term => (
                       <ListItem 
                         key={term.id} 
                         button 
                         onClick={() => handleInsertTerm(term.node_name)}
                       >
                         <ListItemText 
                           primary={term.node_name} 
                           secondary={term.type} 
                         />
                         <IconButton size="small" edge="end">
                           <AddIcon fontSize="small" />
                         </IconButton>
                       </ListItem>
                     ))}
                   </List>
                 )}
               </Paper>
             </CardContent>
           </Card>
        </Grid>
      </Grid>

      {/* AI Suggestion Dialog */}
      <Dialog open={aiDialogOpen} onClose={() => setAiDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>AI Semantic Assistant</DialogTitle>
        <DialogContent>
           <Typography variant="body2" sx={{ mb: 2 }}>
             Paste your table schema (DDL or column list) below, and the AI will suggest valuable calculated fields.
           </Typography>
           <TextField
             fullWidth
             multiline
             rows={6}
             placeholder="CREATE TABLE client_risk (client_id INT, current_risk DECIMAL, target_risk DECIMAL...)"
             value={schemaInput}
             onChange={(e) => setSchemaInput(e.target.value)}
           />
           
           {aiSuggestions.length > 0 && (
             <Box sx={{ mt: 2 }}>
                <Typography variant="subtitle2">Suggestions:</Typography>
                <List>
                  {aiSuggestions.map((s, idx) => (
                    <ListItem key={idx} button onClick={() => {
                        setName(s.node_name);
                        setExpression(s.expression);
                        setTermType(s.type);
                        setAiDialogOpen(false);
                    }}>
                      <ListItemText 
                        primary={s.node_name} 
                        secondary={`${s.description} | ${s.expression}`} 
                      />
                      <Chip label="Apply" size="small" color="primary" onClick={() => {
                          setName(s.node_name);
                          setExpression(s.expression);
                          setTermType(s.type);
                          setAiDialogOpen(false);
                      }} />
                    </ListItem>
                  ))}
                </List>
             </Box>
           )}

           {aiLoading && <Box sx={{ mt: 2, textAlign: 'center' }}><CircularProgress /></Box>}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAiDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleAiSuggest} disabled={aiLoading || !schemaInput}>
            Generate Suggestions
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default CalculatedFieldBuilder;
