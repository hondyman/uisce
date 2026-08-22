import * as React from 'react';
import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Box,
  Typography,
  Button,
  Tabs,
  Tab,
  IconButton,
  Chip,
  Tooltip,
  CircularProgress,
  Snackbar,
  Alert
} from '@mui/material';
import {
  Close as CloseIcon,
  Psychology as PsychologyIcon,
  ContentCopy as ContentCopyIcon,
  Download as DownloadIcon,
  Hub as HubIcon,
  Code as CodeIcon,
  Check as CheckIcon
} from '@mui/icons-material';
import apiClient from '../../../utils/apiClient';
import { devError } from '../../../utils/devLogger';

interface AIContextPayload {
  version: string;
  generated_at: string;
  tenant_id: string;
  domain?: string;
  term_count: number;
  terms: Array<{
    term_name: string;
    qualified_path: string;
    domain: string;
    category: string;
    data_type: string;
    standard?: string;
    format_pattern?: string;
    definition: string;
    differentiator_notes: string;
    parent_term?: string;
    peer_identifiers?: string[];
    specialized_sub_terms?: string[];
    disambiguation_guidance?: string;
  }>;
  taxonomy_edges: Array<{
    source_term: string;
    predicate: string;
    target_term: string;
    explanation?: string;
  }>;
  active_directives?: string[];
  prompt_context_block: string;
  json_ld_schema: Record<string, any>;
}

interface AIContextFeedModalProps {
  open: boolean;
  onClose: () => void;
  termNames?: string[];
  domain?: string;
}

export const AIContextFeedModal: React.FC<AIContextFeedModalProps> = ({
  open,
  onClose,
  termNames,
  domain,
}) => {
  const [tab, setTab] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(false);
  const [data, setData] = useState<AIContextPayload | null>(null);
  const [copied, setCopied] = useState<boolean>(false);

  useEffect(() => {
    if (!open) return;

    const fetchAIContext = async () => {
      setLoading(true);
      try {
        const res = await apiClient<AIContextPayload>('/api/semantic-mapper/ai-context', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            domain: domain || 'Capital Markets & Enterprise Accounting',
          })
        });
        const val = (res as any)?.data ?? res;
        if (val) {
          setData(val);
        }
      } catch (err) {
        devError('[AIContextFeedModal] Failed to fetch AI context:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchAIContext();
  }, [open, termNames, domain]);

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
  };

  const handleDownload = () => {
    if (!data) return;
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `semantic_ai_context_${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <>
      <Dialog
        open={open}
        onClose={onClose}
        maxWidth="md"
        fullWidth
        PaperProps={{
          sx: {
            backgroundColor: '#0F172A',
            color: '#F8FAFC',
            border: '1px solid rgba(99, 102, 241, 0.3)',
            borderRadius: 3,
            boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.7)',
          }
        }}
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 1, borderBottom: '1px solid rgba(148, 163, 184, 0.15)' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Box
              sx={{
                width: 36,
                height: 36,
                borderRadius: 2,
                backgroundColor: 'rgba(99, 102, 241, 0.15)',
                border: '1px solid rgba(99, 102, 241, 0.4)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <PsychologyIcon sx={{ color: '#818CF8' }} />
            </Box>
            <Box>
              <Typography variant="h6" sx={{ fontSize: 16, fontWeight: 700, color: '#F1F5F9' }}>
                AI Semantic Context & Differentiation Feed
              </Typography>
              <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 11 }}>
                High-density graph context to feed LLMs & AI agents for zero-hallucination disambiguation
              </Typography>
            </Box>
          </Box>

          <IconButton onClick={onClose} sx={{ color: '#94A3B8' }} size="small">
            <CloseIcon fontSize="small" />
          </IconButton>
        </DialogTitle>

        <Box sx={{ borderBottom: 1, borderColor: 'rgba(148, 163, 184, 0.15)', px: 3, pt: 1, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Tabs
            value={tab}
            onChange={(_, val) => setTab(val)}
            sx={{
              '& .MuiTab-root': { color: '#94A3B8', textTransform: 'none', fontWeight: 600, fontSize: 12 },
              '& .Mui-selected': { color: '#818CF8' },
              '& .MuiTabs-indicator': { backgroundColor: '#818CF8' }
            }}
          >
            <Tab icon={<PsychologyIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="Prompt Context (LLM Ready)" />
            <Tab icon={<HubIcon sx={{ fontSize: 16 }} />} iconPosition="start" label={`Taxonomy Graph (${data?.taxonomy_edges?.length || 0} Edges)`} />
            <Tab icon={<CodeIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="JSON-LD / Schema" />
            <Tab icon={<CheckIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="Active Directives" />
          </Tabs>

          {data && (
            <Chip
              size="small"
              label={`${data.term_count} terms indexed`}
              sx={{ backgroundColor: 'rgba(99, 102, 241, 0.2)', color: '#C7D2FE', fontSize: 11, fontWeight: 700 }}
            />
          )}
        </Box>

        <DialogContent sx={{ p: 3, minHeight: 380, maxHeight: 520 }}>
          {loading ? (
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: 320, gap: 2 }}>
              <CircularProgress size={36} sx={{ color: '#818CF8' }} />
              <Typography sx={{ color: '#94A3B8', fontSize: 13 }}>
                Generating structured AI context from semantic ontology graph...
              </Typography>
            </Box>
          ) : !data ? (
            <Alert severity="warning" sx={{ backgroundColor: 'rgba(245, 158, 11, 0.1)', color: '#FDE68A' }}>
              No semantic context payload available. Ensure terms and relationship edges are seeded.
            </Alert>
          ) : (
            <>
              {/* Tab 0: Prompt Context Block */}
              {tab === 0 && (
                <Box>
                  <Typography sx={{ fontSize: 12, color: '#94A3B8', mb: 1 }}>
                    Copy and prepend this block directly into your LLM System Prompt or Agent Knowledge Context:
                  </Typography>
                  <Box
                    component="pre"
                    sx={{
                      p: 2,
                      borderRadius: 2,
                      backgroundColor: 'rgba(2, 6, 23, 0.8)',
                      border: '1px solid rgba(148, 163, 184, 0.2)',
                      color: '#E2E8F0',
                      fontFamily: 'monospace',
                      fontSize: 11.5,
                      lineHeight: 1.6,
                      whiteSpace: 'pre-wrap',
                      overflowY: 'auto',
                      maxHeight: 360,
                    }}
                  >
                    {data.prompt_context_block}
                  </Box>
                </Box>
              )}

              {/* Tab 1: Taxonomy Edges */}
              {tab === 1 && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <Typography sx={{ fontSize: 12, color: '#94A3B8', mb: 0.5 }}>
                    Semantic relationship predicates connecting parent/child concepts, peer symbologies, and differentiators:
                  </Typography>
                  {data.taxonomy_edges.map((edge, idx) => (
                    <Box
                      key={idx}
                      sx={{
                        p: 1.5,
                        borderRadius: 1.5,
                        backgroundColor: 'rgba(30, 41, 59, 0.5)',
                        border: '1px solid rgba(148, 163, 184, 0.1)',
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 0.5,
                      }}
                    >
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Chip
                          size="small"
                          label={edge.source_term}
                          sx={{ backgroundColor: '#6366F122', color: '#818CF8', fontWeight: 700, fontSize: 11 }}
                        />
                        <Typography sx={{ fontSize: 11, fontWeight: 700, color: '#F59E0B', fontFamily: 'monospace' }}>
                          ── [{edge.predicate}] ──▶
                        </Typography>
                        <Chip
                          size="small"
                          label={edge.target_term}
                          sx={{ backgroundColor: '#10B98122', color: '#34D399', fontWeight: 700, fontSize: 11 }}
                        />
                      </Box>
                      {edge.explanation && (
                        <Typography sx={{ fontSize: 11, color: '#CBD5E1', mt: 0.5 }}>
                          💡 {edge.explanation}
                        </Typography>
                      )}
                    </Box>
                  ))}
                </Box>
              )}

              {/* Tab 2: JSON-LD */}
              {tab === 2 && (
                <Box
                  component="pre"
                  sx={{
                    p: 2,
                    borderRadius: 2,
                    backgroundColor: 'rgba(2, 6, 23, 0.8)',
                    border: '1px solid rgba(148, 163, 184, 0.2)',
                    color: '#34D399',
                    fontFamily: 'monospace',
                    fontSize: 11,
                    lineHeight: 1.5,
                    overflowY: 'auto',
                    maxHeight: 360,
                  }}
                >
                  {JSON.stringify(data.json_ld_schema, null, 2)}
                </Box>
              )}

              {/* Tab 3: Active Directives */}
              {tab === 3 && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                  <Typography sx={{ fontSize: 12, color: '#94A3B8', mb: 0.5 }}>
                    Deterministic boundaries and disambiguation directives enforced across AI query planners & mappers:
                  </Typography>
                  {(data.active_directives && data.active_directives.length > 0 ? data.active_directives : [
                    'Do NOT substitute custodial_account_code when allocation_account_code is requested in trade allocation contexts.',
                    'When security symbology is ambiguous, inspect IS_PEER_IDENTIFIER_OF edges to resolve CUSIP/SEDOL cross-references.',
                    'Enforce strict distinction between trade_date (market agreement) and settlement_date (cash delivery).'
                  ]).map((dir, idx) => (
                    <Box
                      key={idx}
                      sx={{
                        p: 1.5,
                        borderRadius: 1.5,
                        backgroundColor: 'rgba(99, 102, 241, 0.08)',
                        border: '1px solid rgba(99, 102, 241, 0.25)',
                        display: 'flex',
                        alignItems: 'start',
                        gap: 1.5,
                      }}
                    >
                      <Chip
                        size="small"
                        label={`Directive ${idx + 1}`}
                        sx={{ backgroundColor: '#6366F122', color: '#A5B4FC', fontWeight: 700, fontSize: 10 }}
                      />
                      <Typography sx={{ fontSize: 12, color: '#E2E8F0', fontWeight: 500, lineHeight: 1.5 }}>
                        {dir}
                      </Typography>
                    </Box>
                  ))}
                </Box>
              )}
            </>
          )}
        </DialogContent>

        <DialogActions sx={{ p: 2.5, borderTop: '1px solid rgba(148, 163, 184, 0.15)', justifyContent: 'space-between' }}>
          <Button
            startIcon={<DownloadIcon />}
            onClick={handleDownload}
            disabled={!data}
            sx={{ color: '#94A3B8', textTransform: 'none', fontSize: 12 }}
          >
            Export JSON
          </Button>

          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="outlined"
              startIcon={copied ? <CheckIcon sx={{ color: '#34D399' }} /> : <ContentCopyIcon />}
              onClick={() => handleCopy(tab === 0 ? (data?.prompt_context_block || '') : JSON.stringify(data, null, 2))}
              disabled={!data}
              sx={{
                borderColor: '#6366F166',
                color: '#C7D2FE',
                textTransform: 'none',
                fontSize: 12,
                '&:hover': { borderColor: '#818CF8', backgroundColor: '#6366F115' }
              }}
            >
              {copied ? 'Copied to Clipboard!' : 'Copy Context for AI'}
            </Button>
            <Button
              variant="contained"
              onClick={onClose}
              sx={{
                backgroundColor: '#6366F1',
                color: '#FFFFFF',
                textTransform: 'none',
                fontSize: 12,
                fontWeight: 600,
                '&:hover': { backgroundColor: '#4F46E5' }
              }}
            >
              Close
            </Button>
          </Box>
        </DialogActions>
      </Dialog>

      <Snackbar
        open={copied}
        autoHideDuration={2500}
        onClose={() => setCopied(false)}
        message="AI Semantic Context copied to clipboard!"
      />
    </>
  );
};
