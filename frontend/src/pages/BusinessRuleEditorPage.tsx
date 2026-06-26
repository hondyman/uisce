
import React, { useState } from 'react';
import { 
  Box, 
  Typography, 
  Paper, 
  List, 
  ListItem, 
  ListItemText, 
  TextField, 
  Button, 
  Grid,
  IconButton,
  Chip,
  AppBar,
  Toolbar,
  Divider,
  CircularProgress
} from '@mui/material';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import SaveIcon from '@mui/icons-material/Save';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import EditIcon from '@mui/icons-material/Edit';
import axios from 'axios';

// Syntax Highlighting for Rego (simplified)
import Editor from 'react-simple-code-editor';
import { highlight, languages } from 'prismjs';
import 'prismjs/components/prism-clike';
import 'prismjs/components/prism-go'; // Rego is close enough to Go for basic highlighting
import 'prismjs/themes/prism-tomorrow.css';
import HelpCenter from '../components/HelpCenter/HelpCenter';

interface Policy {
  id: string;
  name: string;
  active: boolean;
  description: string;
}

const mockPolicies: Policy[] = [
  { id: '1', name: 'Block Sanctioned Trades', active: true, description: 'Deny trades if counterparty is sanctioned' },
  { id: '2', name: 'Limit High-Value FX', active: true, description: 'Review FX trades over $10M' },
  { id: '3', name: 'GDPR Data Compliance', active: true, description: 'Ensure user data residency' },
];

const BusinessRuleEditorPage: React.FC = () => {
    const [prompt, setPrompt] = useState<string>('Block trades over $1M if counterparty is strictly sanctioned');
    const [regoCode, setRegoCode] = useState<string>('');
    const [explanation, setExplanation] = useState<string>('');
    const [isGenerating, setIsGenerating] = useState<boolean>(false);
    const [selectedPolicyId, setSelectedPolicyId] = useState<string | null>('1');

    const handleGenerate = async () => {
        setIsGenerating(true);
        try {
            const response = await axios.post('/api/v1/policies/generate', {
                prompt: prompt,
                context: "Trade Authorization"
            });
            setRegoCode(response.data.regoCode);
            setExplanation(response.data.explanation);
        } catch (error) {
            console.error("Failed to generate policy", error);
        } finally {
            setIsGenerating(false);
        }
    };

    return (
        <Box sx={{ display: 'flex', height: '100vh', bgcolor: '#0f172a', color: 'white' }}>
            {/* Sidebar: Existing Policies */}
            <Paper 
                square 
                elevation={0} 
                sx={{ 
                    width: 280, 
                    borderRight: '1px solid rgba(255,255,255,0.1)', 
                    bgcolor: '#1e293b',
                    color: 'white'
                }}
            >
                <Box sx={{ p: 2 }}>
                    <Typography variant="h6" fontWeight="bold">Existing Policies</Typography>
                </Box>
                <List dense>
                    {mockPolicies.map(policy => (
                        <ListItem 
                            key={policy.id} 
                            button 
                            selected={selectedPolicyId === policy.id}
                            onClick={() => setSelectedPolicyId(policy.id)}
                            sx={{ 
                                '&.Mui-selected': { bgcolor: 'rgba(56, 189, 248, 0.15)', borderLeft: '4px solid #38bdf8' },
                                '&:hover': { bgcolor: 'rgba(255,255,255,0.05)' }
                            }}
                        >
                            <ListItemText 
                                primary={policy.name} 
                                secondary={
                                    <Box display="flex" alignItems="center" gap={1}>
                                       {policy.active && <CheckCircleOutlineIcon sx={{ fontSize: 14, color: '#4ade80' }} />}
                                       <Typography variant="caption" color="rgba(255,255,255,0.5)">Active</Typography>
                                    </Box>
                                } 
                                primaryTypographyProps={{ color: 'white' }}
                            />
                            <IconButton size="small" sx={{ color: 'rgba(255,255,255,0.3)' }}><EditIcon fontSize="small" /></IconButton>
                        </ListItem>
                    ))}
                </List>
            </Paper>

            {/* Main Content */}
            <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
                <AppBar position="static" color="transparent" elevation={0} sx={{ borderBottom: '1px solid rgba(255,255,255,0.1)', backdropFilter: 'blur(8px)' }}>
                    <Toolbar variant="dense">
                        <Typography variant="h6" component="div" sx={{ flexGrow: 1, fontWeight: 'bold' }}>
                            Business Rule Editor
                        </Typography>
                        <Button startIcon={<PlayArrowIcon />} sx={{ color: 'rgba(255,255,255,0.7)', mr: 1 }}>Test Rule</Button>
                        <Button startIcon={<SaveIcon />} variant="contained" color="primary">Save Policy</Button>
                    </Toolbar>
                </AppBar>

                <Box sx={{ p: 3, flexGrow: 1, overflowY: 'auto' }}>
                    
                    {/* Input Area */}
                    <Box sx={{ mb: 4 }}>
                        <Typography variant="subtitle2" sx={{ mb: 1, color: '#94a3b8' }}>DESCRIBE YOUR RULE IN PLAIN ENGLISH</Typography>
                        <Paper sx={{ p: 0.5, bgcolor: '#0f172a', border: '1px solid #334155' }}>
                            <TextField 
                                fullWidth 
                                multiline 
                                minRows={3} 
                                value={prompt}
                                onChange={(e) => setPrompt(e.target.value)}
                                placeholder="e.g. Reject any payment over $500 to a high-risk country..."
                                sx={{ 
                                    '& .MuiInputBase-root': { color: 'white', fontSize: '1.1rem' },
                                    '& fieldset': { border: 'none' }
                                }}
                            />
                        </Paper>
                        <Button 
                            fullWidth 
                            variant="contained" 
                            size="large"
                            onClick={handleGenerate}
                            disabled={isGenerating}
                            startIcon={isGenerating ? <CircularProgress size={20} color="inherit"/> : <AutoFixHighIcon />}
                            sx={{ mt: 2, background: 'linear-gradient(90deg, #3b82f6 0%, #8b5cf6 100%)', height: 48, fontSize: '1rem', fontWeight: 'bold' }}
                        >
                            {isGenerating ? 'Generating Policy Logic...' : 'Generate Policy'}
                        </Button>
                    </Box>

                    {/* Split View: Code & Explanation */}
                    <Grid container spacing={3} sx={{ height: 'calc(100% - 220px)' }}>
                        <Grid item xs={12} md={7}>
                            <Typography variant="subtitle2" sx={{ mb: 1, color: '#94a3b8' }}>GENERATED REGO CODE</Typography>
                            <Paper sx={{ 
                                height: '100%', 
                                bgcolor: '#0d1117', 
                                border: '1px solid #30363d',
                                overflow: 'auto',
                                fontFamily: 'monospace'
                            }}>
                                <Editor
                                    value={regoCode || "// Generated code will appear here..."}
                                    onValueChange={(code: string) => setRegoCode(code)}
                                    highlight={(code: string) => highlight(code, languages.go, 'go')}
                                    padding={20}
                                    style={{
                                        fontFamily: '"Fira Code", "Fira Mono", monospace',
                                        fontSize: 14,
                                        backgroundColor: '#0d1117',
                                        color: '#c9d1d9',
                                        minHeight: '100%'
                                    }}
                                />
                            </Paper>
                        </Grid>
                        <Grid item xs={12} md={5}>
                             <Typography variant="subtitle2" sx={{ mb: 1, color: '#94a3b8' }}>EXPLANATION</Typography>
                             <Paper sx={{ p: 3, height: '100%', bgcolor: '#1e293b', color: '#e2e8f0', border: '1px solid #334155' }}>
                                 {explanation ? (
                                     <Typography variant="body1" sx={{ lineHeight: 1.7 }}>
                                         {explanation}
                                     </Typography>
                                 ) : (
                                     <Typography variant="body2" sx={{ color: 'rgba(255,255,255,0.3)', fontStyle: 'italic' }}>
                                         Natural language explanation will appear here after generation.
                                     </Typography>
                                 )}
                             </Paper>
                        </Grid>
                    </Grid>
                </Box>
            </Box>
            <HelpCenter context="rules-editor" />
        </Box>
    );
};

export default BusinessRuleEditorPage;
