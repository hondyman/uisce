import React, { useState, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  CircularProgress,
  Alert,
  Chip,
  Divider,
  IconButton,
  Collapse,
  Tooltip
} from '@mui/material';
import {
  Psychology as AIIcon,
  AutoAwesome as SparkleIcon,
  Add as AddIcon,
  ExpandMore as ExpandIcon,
  ExpandLess as CollapseIcon,
  Code as CodeIcon
} from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext';

interface GeneratedNode {
  id: string;
  type: string;
  config: Record<string, unknown>;
  next?: string;
}

interface CopilotPanelProps {
  onNodeGenerated: (node: GeneratedNode) => void;
  existingNodeIds?: string[];
  businessObjects?: string[];
}

const examplePrompts = [
  "If order total is greater than 1000, send for manager approval",
  "If customer country is USA, apply domestic tax rules",
  "When trade value exceeds 100000, require compliance check",
  "Send email notification to the submitter",
  "If risk score is high, escalate to senior reviewer"
];

const CopilotPanel: React.FC<CopilotPanelProps> = ({
  onNodeGenerated,
  existingNodeIds = [],
  businessObjects = []
}) => {
  const { tenant } = useTenant();
  const [prompt, setPrompt] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<{ node: GeneratedNode; explanation: string } | null>(null);
  const [expanded, setExpanded] = useState(true);
  const [showCode, setShowCode] = useState(false);

  const handleGenerate = useCallback(async () => {
    if (!prompt.trim()) return;

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const response = await fetch('/api/v1/copilot/generate-node', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
        },
        body: JSON.stringify({
          text: prompt,
          businessObjects,
          existingNodeIds,
        }),
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || 'Generation failed');
      }

      const data = await response.json();
      
      let parsedNode: GeneratedNode;
      if (typeof data.nodeJson === 'string') {
        parsedNode = JSON.parse(data.nodeJson);
      } else {
        parsedNode = data.nodeJson;
      }

      // Ensure node has unique ID
      if (!parsedNode.id) {
        parsedNode.id = `node_${Date.now()}`;
      }

      setResult({ node: parsedNode, explanation: data.explanation });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [prompt, tenant?.id, businessObjects, existingNodeIds]);

  const handleAddToCanvas = useCallback(() => {
    if (result?.node) {
      onNodeGenerated(result.node);
      setResult(null);
      setPrompt('');
    }
  }, [result, onNodeGenerated]);

  const handleExampleClick = (example: string) => {
    setPrompt(example);
    setResult(null);
  };

  return (
    <Paper
      elevation={2}
      sx={{
        p: 2,
        background: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 100%)',
        color: 'white',
        borderRadius: 2,
      }}
    >
      {/* Header */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          mb: expanded ? 2 : 0,
          cursor: 'pointer',
        }}
        onClick={() => setExpanded(!expanded)}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <AIIcon sx={{ color: '#7c3aed' }} />
          <Typography variant="subtitle1" fontWeight="bold">
            AI Co-pilot
          </Typography>
          <Chip
            label="Beta"
            size="small"
            sx={{
              backgroundColor: 'rgba(124, 58, 237, 0.3)',
              color: '#a78bfa',
              fontSize: '0.7rem',
            }}
          />
        </Box>
        <IconButton size="small" sx={{ color: 'white' }}>
          {expanded ? <CollapseIcon /> : <ExpandIcon />}
        </IconButton>
      </Box>

      <Collapse in={expanded}>
        {/* Prompt Input */}
        <TextField
          fullWidth
          multiline
          rows={2}
          placeholder="Describe what this workflow step should do..."
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          sx={{
            mb: 1,
            '& .MuiOutlinedInput-root': {
              backgroundColor: 'rgba(255,255,255,0.05)',
              color: 'white',
              '& fieldset': { borderColor: 'rgba(255,255,255,0.2)' },
              '&:hover fieldset': { borderColor: 'rgba(124, 58, 237, 0.5)' },
              '&.Mui-focused fieldset': { borderColor: '#7c3aed' },
            },
            '& .MuiInputBase-input::placeholder': {
              color: 'rgba(255,255,255,0.5)',
            },
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && e.ctrlKey) {
              handleGenerate();
            }
          }}
        />

        {/* Generate Button */}
        <Button
          fullWidth
          variant="contained"
          startIcon={loading ? <CircularProgress size={16} color="inherit" /> : <SparkleIcon />}
          onClick={handleGenerate}
          disabled={loading || !prompt.trim()}
          sx={{
            mb: 2,
            background: 'linear-gradient(135deg, #7c3aed 0%, #4f46e5 100%)',
            '&:hover': {
              background: 'linear-gradient(135deg, #6d28d9 0%, #4338ca 100%)',
            },
          }}
        >
          {loading ? 'Generating...' : 'Generate Node (Ctrl+Enter)'}
        </Button>

        {/* Example Prompts */}
        <Box sx={{ mb: 2 }}>
          <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.6)', mb: 1, display: 'block' }}>
            Try an example:
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
            {examplePrompts.slice(0, 3).map((example, i) => (
              <Chip
                key={i}
                label={example.length > 30 ? example.substring(0, 30) + '...' : example}
                size="small"
                onClick={() => handleExampleClick(example)}
                sx={{
                  backgroundColor: 'rgba(255,255,255,0.1)',
                  color: 'rgba(255,255,255,0.8)',
                  '&:hover': { backgroundColor: 'rgba(124, 58, 237, 0.3)' },
                  cursor: 'pointer',
                }}
              />
            ))}
          </Box>
        </Box>

        {/* Error */}
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {/* Result */}
        {result && (
          <Box
            sx={{
              p: 2,
              borderRadius: 1,
              backgroundColor: 'rgba(16, 185, 129, 0.1)',
              border: '1px solid rgba(16, 185, 129, 0.3)',
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="subtitle2" sx={{ color: '#10b981' }}>
                Generated Node
              </Typography>
              <Tooltip title={showCode ? 'Hide JSON' : 'Show JSON'}>
                <IconButton size="small" onClick={() => setShowCode(!showCode)} sx={{ color: 'white' }}>
                  <CodeIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Box>

            <Typography variant="body2" sx={{ mb: 1, color: 'rgba(255,255,255,0.8)' }}>
              {result.explanation}
            </Typography>

            <Collapse in={showCode}>
              <Box
                component="pre"
                sx={{
                  p: 1,
                  borderRadius: 1,
                  backgroundColor: 'rgba(0,0,0,0.3)',
                  fontSize: '0.75rem',
                  overflow: 'auto',
                  maxHeight: 150,
                  color: '#a78bfa',
                }}
              >
                {JSON.stringify(result.node, null, 2)}
              </Box>
            </Collapse>

            <Divider sx={{ my: 1, borderColor: 'rgba(255,255,255,0.1)' }} />

            <Button
              fullWidth
              variant="outlined"
              startIcon={<AddIcon />}
              onClick={handleAddToCanvas}
              sx={{
                borderColor: '#10b981',
                color: '#10b981',
                '&:hover': {
                  borderColor: '#059669',
                  backgroundColor: 'rgba(16, 185, 129, 0.1)',
                },
              }}
            >
              Add to Canvas
            </Button>
          </Box>
        )}
      </Collapse>
    </Paper>
  );
};

export default CopilotPanel;
