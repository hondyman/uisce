import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Grid,
  Divider,
  Alert,
  Card,
  CardContent
} from '@mui/material';
import {
  Chat as ChatIcon,
  ShieldCheck as ShieldCheckIcon,
  Public as PublicIcon,
  Lock as LockIcon,
} from '@mui/icons-material';

interface AmbientProposal {
  proposalId: string;
  sourceChannel: string;
  originator: string;
  rawSnippet: string;
  proposedKey: string;
  generatedSQL: string;
  sanityPass: boolean;
  graphResolved: boolean;
  contradictionScore: number;
}

export const AmbientKnowledgeInbox: React.FC<{ tenantId: string }> = ({ tenantId: _tenantId }) => {
  const [proposal, _setProposal] = useState<AmbientProposal>({
    proposalId: 'a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d',
    sourceChannel: 'SLACK (#wealth-data-ops)',
    originator: 'steward_pat',
    rawSnippet: 'For CRM data, use Affinity for North American deals from 2025 onwards, but Salesforce for all global leads before that.',
    proposedKey: 'rule.routing.crm_affinity_vs_salesforce',
    generatedSQL: "CASE WHEN region = 'USCAN' AND EXTRACT(YEAR FROM deal_date) >= 2025 THEN 'affinity.deals' ELSE 'sfdc.leads' END",
    sanityPass: true,
    graphResolved: true,
    contradictionScore: 0.0
  });

  const [isProcessing, setIsProcessing] = useState(false);
  const [actionNotice, setActionNotice] = useState<string | null>(null);

  const handleAction = async (nominateCore: boolean) => {
    setIsProcessing(true);
    setTimeout(() => {
      setActionNotice(
        nominateCore
          ? 'Nomination submitted to Uisce Global Product Management for Gold-Copy inclusion.'
          : 'Rule approved and committed to your private tenant knowledge base.'
      );
      setIsProcessing(false);
    }, 400);
  };

  return (
    <Paper
      elevation={0}
      sx={{
        p: 3,
        bgcolor: '#071526',
        color: '#F8FAFC',
        border: '1px solid #1E293B',
        borderRadius: 2
      }}
    >
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <ChatIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Ambient Knowledge & Tribal Ingestion Inbox
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Autonomous extraction from informal conversations with multi-tenant governance gating
            </Typography>
          </Box>
        </Stack>
        <Chip
          icon={<ShieldCheckIcon sx={{ fontSize: 16, color: '#10B981 !important' }} />}
          label="Sanity Gate: Passed"
          size="small"
          sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
        />
      </Box>

      {actionNotice && (
        <Alert severity="success" sx={{ mb: 3, bgcolor: '#064E3B', color: '#F8FAFC' }}>
          {actionNotice}
        </Alert>
      )}

      <Grid container spacing={3}>
        <Grid   size={{ xs: 12, md: 6 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
            Inbound Context Payload
          </Typography>
          <Card sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', color: '#F8FAFC' }}>
            <CardContent>
              <Stack direction="row" spacing={1} alignItems="center" mb={1.5}>
                <Chip label={proposal.sourceChannel} size="small" sx={{ bgcolor: '#1E293B', color: '#38BDF8', fontSize: 10 }} />
                <Typography variant="caption" sx={{ color: '#64748B' }}>
                  Author: {proposal.originator}
                </Typography>
              </Stack>
              <Typography variant="body2" sx={{ fontStyle: 'italic', color: '#CBD5E1', mb: 2 }}>
                "{proposal.rawSnippet}"
              </Typography>
              <Divider sx={{ my: 1.5, borderColor: '#1E293B' }} />
              <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 0.5 }}>
                Parsed OKF Concept Target:
              </Typography>
              <Typography variant="subtitle2" sx={{ fontFamily: 'monospace', color: '#00D4FF' }}>
                {proposal.proposedKey}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid   size={{ xs: 12, md: 6 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
            Compiled AST Routing & Sanity Verification
          </Typography>
          <Card sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', color: '#F8FAFC' }}>
            <CardContent>
              <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 0.5 }}>
                Synthesized Base SQL Expression:
              </Typography>
              <Box sx={{ p: 1.5, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 1, fontFamily: 'monospace', fontSize: 11, color: '#34D399', mb: 2 }}>
                {proposal.generatedSQL}
              </Box>

              <Grid container spacing={1}>
                <Grid  size={{ xs: 6 }}>
                  <Paper sx={{ p: 1, bgcolor: '#071526', border: '1px solid #1E293B', textAlign: 'center' }}>
                    <Typography variant="caption" sx={{ color: '#64748B' }}>Catalog Graph</Typography>
                    <Typography variant="subtitle2" sx={{ color: '#38BDF8', fontWeight: 700, fontSize: 11 }}>100% Bound</Typography>
                  </Paper>
                </Grid>
                <Grid  size={{ xs: 6 }}>
                  <Paper sx={{ p: 1, bgcolor: '#071526', border: '1px solid #1E293B', textAlign: 'center' }}>
                    <Typography variant="caption" sx={{ color: '#64748B' }}>Contradiction Proof</Typography>
                    <Typography variant="subtitle2" sx={{ color: '#10B981', fontWeight: 700, fontSize: 11 }}>0.00 (Zero Conflicts)</Typography>
                  </Paper>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Box display="flex" justifyContent="flex-end" gap={2} mt={3} pt={2} borderTop="1px solid #1E293B">
        <Button
          variant="outlined"
          startIcon={<LockIcon />}
          onClick={() => handleAction(false)}
          disabled={isProcessing}
          sx={{ borderColor: '#334155', color: '#CBD5E1', textTransform: 'none', '&:hover': { borderColor: '#64748B' } }}
        >
          Accept for Tenant Knowledgebase (Private)
        </Button>
        <Button
          variant="contained"
          startIcon={<PublicIcon />}
          onClick={() => handleAction(true)}
          disabled={isProcessing}
          sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 600, '&:hover': { bgcolor: '#0369A1' } }}
        >
          Nominate for Uisce Global Core (Community)
        </Button>
      </Box>
    </Paper>
  );
};

export default AmbientKnowledgeInbox;
