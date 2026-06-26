import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Typography,
  Box,
  CircularProgress,
  Paper,
  Chip,
  Stepper,
  Step,
  StepLabel,
  Divider,
  Alert
} from '@mui/material';
import { AutoAwesome, CheckCircle, Warning } from '@mui/icons-material';

interface EnrichmentProposal {
  semantic_term_name: string;
  semantic_term_type: string;
  business_term_name: string;
  domain_hierarchy: string[];
  confidence: number;
  reasoning: string;
}

interface SemanticEnrichmentWizardProps {
  open: boolean;
  onClose: () => void;
  tenantId: string;
  datasourceId: string;
  onSuccess?: () => void;
}

export const SemanticEnrichmentWizard: React.FC<SemanticEnrichmentWizardProps> = ({
  open,
  onClose,
  tenantId,
  datasourceId,
  onSuccess
}) => {
  const [activeStep, setActiveStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Input State
  const [columnName, setColumnName] = useState('');
  const [tableName, setTableName] = useState('');
  const [schemaName, setSchemaName] = useState('');
  const [dataType, setDataType] = useState('');

  // Proposal State
  const [proposal, setProposal] = useState<EnrichmentProposal | null>(null);

  const handleAnalyze = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/semantic-mapping/enrich/suggest', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId
        },
        body: JSON.stringify({
          column: {
            column: columnName,
            table: tableName,
            schema: schemaName,
            data_type: dataType
          }
        })
      });

      if (!response.ok) {
        throw new Error('Failed to analyze column');
      }

      const data = await response.json();
      setProposal(data);
      setActiveStep(1);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleApply = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/semantic-mapping/enrich/apply', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId
        },
        body: JSON.stringify({
          proposal: proposal,
          column_id: `${schemaName}.${tableName}.${columnName}`, // Mock ID
          tenant_id: tenantId,
          tenant_instance_id: datasourceId
        })
      });

      if (!response.ok) {
        // Handle 501 Not Implemented gracefully
        if (response.status === 501) {
            alert("Apply feature is not yet fully implemented on the backend.");
            onClose();
            return;
        }
        throw new Error('Failed to apply enrichment');
      }

      onClose();
      if (onSuccess) {
        onSuccess();
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const steps = ['Input Column Details', 'Review Suggestion', 'Complete'];

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        <Box display="flex" alignItems="center" gap={1}>
          <AutoAwesome color="primary" />
          Semantic Enrichment Wizard
        </Box>
      </DialogTitle>
      <DialogContent>
        <Stepper activeStep={activeStep} sx={{ mb: 4, mt: 2 }}>
          {steps.map((label) => (
            <Step key={label}>
              <StepLabel>{label}</StepLabel>
            </Step>
          ))}
        </Stepper>

        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        {activeStep === 0 && (
          <Box display="flex" flexDirection="column" gap={2}>
            <Typography variant="body1" color="textSecondary">
              Enter the technical details of the database column you want to enrich.
              The wizard will analyze the name and type to suggest semantic terms.
            </Typography>
            <TextField
              label="Schema Name"
              value={schemaName}
              onChange={(e) => setSchemaName(e.target.value)}
              fullWidth
              placeholder="e.g., SALES"
            />
            <TextField
              label="Table Name"
              value={tableName}
              onChange={(e) => setTableName(e.target.value)}
              fullWidth
              placeholder="e.g., ORDERS"
            />
            <TextField
              label="Column Name"
              value={columnName}
              onChange={(e) => setColumnName(e.target.value)}
              fullWidth
              placeholder="e.g., ORDER_DT"
            />
            <TextField
              label="Data Type"
              value={dataType}
              onChange={(e) => setDataType(e.target.value)}
              fullWidth
              placeholder="e.g., TIMESTAMP"
            />
          </Box>
        )}

        {activeStep === 1 && proposal && (
          <Box display="flex" flexDirection="column" gap={3}>
            <Paper variant="outlined" sx={{ p: 2, bgcolor: '#f5f9ff' }}>
              <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                <Typography variant="h6" color="primary">AI Analysis Result</Typography>
                <Chip 
                  label={`${(proposal.confidence * 100).toFixed(0)}% Confidence`} 
                  color={proposal.confidence > 0.8 ? "success" : "warning"} 
                  icon={proposal.confidence > 0.8 ? <CheckCircle /> : <Warning />}
                />
              </Box>
              <Typography variant="body2" sx={{ fontStyle: 'italic' }}>
                {proposal.reasoning}
              </Typography>
            </Paper>

            <Box display="grid" gridTemplateColumns="1fr 1fr" gap={2}>
              <Box>
                <Typography variant="subtitle2" color="textSecondary">Suggested Semantic Term</Typography>
                <Typography variant="h6">{proposal.semantic_term_name}</Typography>
                <Chip label={proposal.semantic_term_type} size="small" sx={{ mt: 0.5 }} />
              </Box>
              <Box>
                <Typography variant="subtitle2" color="textSecondary">Suggested Business Term</Typography>
                <Typography variant="h6">{proposal.business_term_name}</Typography>
              </Box>
            </Box>

            <Divider />

            <Box>
              <Typography variant="subtitle2" color="textSecondary" gutterBottom>Suggested Data Domain Hierarchy</Typography>
              <Box display="flex" alignItems="center" gap={1}>
                {proposal.domain_hierarchy.map((level, index) => (
                  <React.Fragment key={index}>
                    <Chip label={level} variant="outlined" />
                    {index < proposal.domain_hierarchy.length - 1 && <Typography color="textSecondary">&gt;</Typography>}
                  </React.Fragment>
                ))}
              </Box>
            </Box>
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={loading}>Cancel</Button>
        {activeStep === 0 && (
          <Button 
            variant="contained" 
            onClick={handleAnalyze} 
            disabled={loading || !columnName || !tableName}
            startIcon={loading ? <CircularProgress size={20} /> : <AutoAwesome />}
          >
            Analyze
          </Button>
        )}
        {activeStep === 1 && (
          <Button 
            variant="contained" 
            onClick={handleApply} 
            disabled={loading}
          >
            Approve & Create
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};
