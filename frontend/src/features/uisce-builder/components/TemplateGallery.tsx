import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Box,
  Card,
  CardContent,
  CardActions,
  Grid,
  Chip,
  IconButton,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import TimelineIcon from '@mui/icons-material/Timeline';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import AccountBalanceIcon from '@mui/icons-material/AccountBalance';
import PaymentsIcon from '@mui/icons-material/Payments';
import StorefrontIcon from '@mui/icons-material/Storefront';

interface PipelineTemplate {
  id: string;
  name: string;
  description: string;
  icon: any;
  category: string;
  nodeCount: number;
  nodes: Array<{
    type: string;
    label: string;
    position: { x: number; y: number };
  }>;
}

const templates: PipelineTemplate[] = [
  {
    id: 'northwind-customer',
    name: 'Northwind Customer Validation',
    description: 'Complete customer validation pipeline for Northwind database with credit limit, country, and contact checks.',
    icon: StorefrontIcon,
    category: 'Northwind',
    nodeCount: 6,
    nodes: [
      { type: 'List_Lookup', label: 'Valid Country Check', position: { x: 100, y: 100 } },
      { type: 'Cross_Reference', label: 'Company Exists', position: { x: 300, y: 100 } },
      { type: 'List_Lookup', label: 'Contact Title Valid', position: { x: 500, y: 100 } },
      { type: 'Limit', label: 'Credit Limit Check', position: { x: 700, y: 100 } },
      { type: 'Formula', label: 'Phone Format', position: { x: 900, y: 100 } },
      { type: 'AI_Anomaly', label: 'Duplicate Detection', position: { x: 1100, y: 100 } },
    ]
  },
  {
    id: 'trade-approval-standard',
    name: 'Standard Trade Approval',
    description: 'Comprehensive trade flow with Policy Checks, Sanctions, and Risk Scoring.',
    icon: TimelineIcon,
    category: 'Trading',
    nodeCount: 6,
    nodes: [
      { type: 'Sanctions', label: 'Sanctions Screening', position: { x: 100, y: 100 } },
      { type: 'Policy_Check', label: 'Block High-Value Risky Trades', position: { x: 300, y: 100 } },
      { type: 'AI_Prediction', label: 'Settlement Risk Score', position: { x: 500, y: 100 } },
      { type: 'Limit', label: 'Desk Limit Check', position: { x: 700, y: 100 } },
      { type: 'Approval_Gate', label: 'Supervisor Approval', position: { x: 900, y: 100 } },
      { type: 'Durable_Ledger', label: 'Record Transaction', position: { x: 1100, y: 100 } },
    ]
  },
  {
    id: 'client-onboarding',
    name: 'Client Onboarding',
    description: 'KYC verification, documentation checks, and account setup workflow.',
    icon: PersonAddIcon,
    category: 'Operations',
    nodeCount: 4,
    nodes: [
      { type: 'Sanctions', label: 'Sanctions Screening', position: { x: 100, y: 100 } },
      { type: 'External_API', label: 'KYC Provider', position: { x: 300, y: 100 } },
      { type: 'List_Lookup', label: 'Document Checklist', position: { x: 500, y: 100 } },
      { type: 'Approval_Gate', label: 'Compliance Approval', position: { x: 700, y: 100 } },
    ]
  },
  {
    id: 'investment-compliance',
    name: 'Investment Compliance',
    description: 'Concentration limits, restricted securities, and regulatory checks.',
    icon: AccountBalanceIcon,
    category: 'Compliance',
    nodeCount: 4,
    nodes: [
      { type: 'Formula', label: 'Concentration Check', position: { x: 100, y: 100 } },
      { type: 'List_Lookup', label: 'Restricted Securities', position: { x: 300, y: 100 } },
      { type: 'Aggregation', label: 'Sector Exposure', position: { x: 500, y: 100 } },
      { type: 'Conditional', label: 'Threshold Alert', position: { x: 700, y: 100 } },
    ]
  },
  {
    id: 'payment-processing',
    name: 'Payment Processing',
    description: 'Wire transfer validation with fraud detection and approval workflow.',
    icon: PaymentsIcon,
    category: 'Payments',
    nodeCount: 5,
    nodes: [
      { type: 'Sanctions', label: 'OFAC Check', position: { x: 100, y: 100 } },
      { type: 'Limit', label: 'Daily Limit', position: { x: 300, y: 100 } },
      { type: 'AI_Anomaly', label: 'Fraud Detection', position: { x: 500, y: 100 } },
      { type: 'Conditional', label: 'Amount Router', position: { x: 700, y: 100 } },
      { type: 'Approval_Gate', label: 'Treasury Approval', position: { x: 900, y: 100 } },
    ]
  },
];

interface TemplateGalleryProps {
  open: boolean;
  onClose: () => void;
  onApplyTemplate: (template: PipelineTemplate) => void;
}

const TemplateGallery: React.FC<TemplateGalleryProps> = ({ open, onClose, onApplyTemplate }) => {
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null);

  const handleApply = () => {
    const template = templates.find(t => t.id === selectedTemplate);
    if (template) {
      onApplyTemplate(template);
      onClose();
    }
  };

  return (
    <Dialog 
      open={open} 
      onClose={onClose} 
      maxWidth="md" 
      fullWidth
      PaperProps={{
        sx: { borderRadius: 3 }
      }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box>
          <Typography variant="h6" fontWeight="bold">Pipeline Templates</Typography>
          <Typography variant="caption" color="text.secondary">
            Start with a pre-built pipeline and customize it for your needs
          </Typography>
        </Box>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers>
        <Grid container spacing={2}>
          {templates.map((template) => {
            const Icon = template.icon;
            const isSelected = selectedTemplate === template.id;
            return (
              <Grid item xs={12} sm={6} key={template.id}>
                <Card 
                  elevation={isSelected ? 8 : 1}
                  sx={{ 
                    cursor: 'pointer',
                    border: isSelected ? '2px solid' : '1px solid',
                    borderColor: isSelected ? 'primary.main' : 'divider',
                    transition: 'all 0.2s',
                    '&:hover': {
                      borderColor: 'primary.light',
                      transform: 'translateY(-2px)'
                    }
                  }}
                  onClick={() => setSelectedTemplate(template.id)}
                >
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1.5 }}>
                      <Box sx={{ 
                        p: 1, 
                        borderRadius: 2, 
                        bgcolor: 'primary.light',
                        color: 'primary.main',
                        display: 'flex'
                      }}>
                        <Icon />
                      </Box>
                      <Box sx={{ flex: 1 }}>
                        <Typography variant="subtitle1" fontWeight="bold">
                          {template.name}
                        </Typography>
                        <Chip 
                          size="small" 
                          label={template.category} 
                          sx={{ fontSize: '0.65rem', height: 18 }} 
                        />
                      </Box>
                    </Box>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
                      {template.description}
                    </Typography>
                    <Typography variant="caption" color="text.disabled">
                      {template.nodeCount} filters pre-configured
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
            );
          })}
        </Grid>
      </DialogContent>
      <DialogActions sx={{ p: 2 }}>
        <Button onClick={onClose}>Cancel</Button>
        <Button 
          variant="contained" 
          disabled={!selectedTemplate}
          onClick={handleApply}
        >
          Apply Template
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default TemplateGallery;
export type { PipelineTemplate };
