import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  CircularProgress,
  Alert,
  Collapse,
  IconButton
} from '@mui/material';
import { AutoFixHigh, ExpandMore, ExpandLess } from '@mui/icons-material';
import { useGenerateDAG } from '../../api/aiDag';

interface AICoPilotPanelProps {
  onDAGGenerated: (dag: object) => void;
  currentDefinition?: object;
}

/**
 * AICoPilotPanel - Provides natural language to DAG generation
 * 
 * This component allows users to describe a workflow in plain English,
 * and uses GenAI to convert it into a valid Titan DAG configuration.
 * 
 * TODO:
 * - Add history of previous prompts
 * - Add "Edit" mode for refining existing DAGs
 * - Add validation before applying generated DAG
 * - Add preview/diff view before applying
 */
export const AICoPilotPanel: React.FC<AICoPilotPanelProps> = ({ 
  onDAGGenerated, 
  currentDefinition 
}) => {
  const [prompt, setPrompt] = useState('');
  const [expanded, setExpanded] = useState(true);
  const [lastExplanation, setLastExplanation] = useState<string | null>(null);
  
  const { mutate: generateDAG, isPending, error, reset } = useGenerateDAG();

  const handleGenerate = () => {
    if (!prompt.trim()) return;
    
    reset();
    setLastExplanation(null);
    
    generateDAG(
      { 
        prompt, 
        existingDefinition: currentDefinition 
      },
      {
        onSuccess: (response) => {
          setLastExplanation(response.explanation);
          onDAGGenerated(response.dagDefinition);
          // Optionally clear prompt on success
          // setPrompt('');
        },
      }
    );
  };

  const examplePrompts = [
    "Create a trade approval workflow with compliance check and manager approval for amounts over $100k",
    "Build a customer onboarding flow: KYC check → Risk assessment → Account creation → Welcome email",
    "Design an order processing workflow with high-value order approval for orders over $10,000",
  ];

  return (
    <Paper 
      elevation={2} 
      sx={{ 
        p: 2, 
        mb: 2, 
        backgroundColor: 'background.paper',
        border: '1px solid',
        borderColor: 'divider'
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: expanded ? 2 : 0 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <AutoFixHigh color="primary" />
          <Typography variant="h6">AI Co-Pilot</Typography>
        </Box>
        <IconButton size="small" onClick={() => setExpanded(!expanded)}>
          {expanded ? <ExpandLess /> : <ExpandMore />}
        </IconButton>
      </Box>

      <Collapse in={expanded}>
        <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
          Describe your workflow in plain English and let AI generate the configuration.
        </Typography>

        <TextField
          fullWidth
          multiline
          rows={3}
          placeholder="e.g., Create a workflow that checks compliance, then if the trade amount is over $50k, requires manager approval, otherwise auto-approves"
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          sx={{ mb: 2 }}
        />

        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', mb: 2 }}>
          <Button
            variant="contained"
            startIcon={isPending ? <CircularProgress size={16} /> : <AutoFixHigh />}
            onClick={handleGenerate}
            disabled={isPending || !prompt.trim()}
          >
            {isPending ? 'Generating...' : 'Generate Workflow'}
          </Button>
          
          {currentDefinition && (
            <Typography variant="caption" color="textSecondary">
              Will modify current workflow
            </Typography>
          )}
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error.message}
          </Alert>
        )}

        {lastExplanation && (
          <Alert severity="success" sx={{ mb: 2 }}>
            <Typography variant="subtitle2">Generated Successfully</Typography>
            <Typography variant="body2">{lastExplanation}</Typography>
          </Alert>
        )}

        <Box sx={{ mt: 2 }}>
          <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mb: 1 }}>
            Example prompts:
          </Typography>
          {examplePrompts.map((example, idx) => (
            <Button 
              key={idx} 
              size="small" 
              variant="text"
              onClick={() => setPrompt(example)}
              sx={{ 
                display: 'block', 
                textAlign: 'left', 
                textTransform: 'none',
                fontSize: '0.75rem',
                color: 'text.secondary',
                '&:hover': { color: 'primary.main' }
              }}
            >
              • {example}
            </Button>
          ))}
        </Box>
      </Collapse>
    </Paper>
  );
};

export default AICoPilotPanel;
