import * as React from 'react';
import { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Chip,
  Tooltip,
  Paper,
  Button,
  Collapse,
  IconButton,
  CircularProgress
} from '@mui/material';
import {
  Hub as HubIcon,
  CompareArrows as CompareArrowsIcon,
  AutoAwesome as AutoAwesomeIcon,
  InfoOutlined as InfoIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Check as CheckIcon,
  Psychology as PsychologyIcon
} from '@mui/icons-material';
import apiClient from '../../../utils/apiClient';
import { devDebug, devError } from '../../../utils/devLogger';

export interface RelatedTermInfo {
  term_id: string;
  term_name: string;
  qualified_path?: string;
  category?: string;
  data_type?: string;
  domain?: string;
  role?: string;
  relationship_type: string;
  differentiation_notes?: string;
  format_pattern?: string;
  standard?: string;
  confidence: number;
  is_gold_copy?: boolean;
}

export interface TermDisambiguation {
  primary_term: RelatedTermInfo;
  related_terms: RelatedTermInfo[];
  differentiator_summary: string;
  domain_scope?: string;
}

interface RelatedTermsDisambiguationProps {
  termName?: string;
  columnName?: string;
  tableName?: string;
  onSelectTerm?: (termName: string, termInfo?: RelatedTermInfo) => void;
  onOpenAIFeed?: (termNames?: string[]) => void;
  compact?: boolean;
}

export const RelatedTermsDisambiguation: React.FC<RelatedTermsDisambiguationProps> = ({
  termName,
  columnName,
  tableName,
  onSelectTerm,
  onOpenAIFeed,
  compact = false,
}) => {
  const [data, setData] = useState<TermDisambiguation | null>(null);
  const [l3Classification, setL3Classification] = useState<{ name: string; breadcrumb: string; domain_name: string; category_name: string } | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [expanded, setExpanded] = useState<boolean>(false);
  const [activeComparison, setActiveComparison] = useState<RelatedTermInfo | null>(null);

  useEffect(() => {
    let isMounted = true;
    const fetchRelated = async () => {
      const queryTarget = termName || columnName;
      if (!queryTarget || queryTarget.trim().length < 2) return;

      setLoading(true);
      try {
        let endpoint = `/api/semantic-terms/${encodeURIComponent(queryTarget)}/related`;
        if (columnName && !termName) {
          endpoint = `/api/semantic-mapper/suggest-related?column=${encodeURIComponent(columnName)}&entity=${encodeURIComponent(tableName || '')}`;
        }

        const [res, l3Res] = await Promise.allSettled([
          apiClient<TermDisambiguation | { suggestions: RelatedTermInfo[] }>(endpoint),
          apiClient<{ name: string; breadcrumb: string; domain_name: string; category_name: string }>(
            `/api/taxonomy/suggest-l3?term=${encodeURIComponent(queryTarget)}&column=${encodeURIComponent(columnName || '')}`
          )
        ]);

        if (isMounted) {
          if (res.status === 'fulfilled' && res.value) {
            const raw = res.value;
            const val = (raw as any)?.data ?? raw;
            if (val && 'primary_term' in val) {
              setData(val);
            } else if (val && 'suggestions' in val && val.suggestions.length > 0) {
              const suggestions = val.suggestions;
              setData({
                primary_term: suggestions[0],
                related_terms: suggestions.slice(1),
                differentiator_summary: suggestions[0].differentiation_notes || 'Associated business term family.',
                domain_scope: suggestions[0].domain
              });
            }
          }
          if (l3Res.status === 'fulfilled' && l3Res.value) {
            const raw = l3Res.value;
            const val = (raw as any)?.data ?? raw;
            if (val && val.name) {
              setL3Classification(val);
            }
          }
        }
      } catch (err) {
        devDebug('[RelatedTermsDisambiguation] Fetch fallback for:', queryTarget);
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    fetchRelated();
    return () => {
      isMounted = false;
    };
  }, [termName, columnName, tableName]);

  if (loading) {
    return (
      <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 1, py: 0.5 }}>
        <CircularProgress size={12} sx={{ color: '#818CF8' }} />
        <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: 11 }}>
          Reasoning term relationships...
        </Typography>
      </Box>
    );
  }

  if (!data || (!data.related_terms || data.related_terms.length === 0)) {
    return null;
  }

  const getRoleColor = (role?: string) => {
    if (!role) return { color: '#60A5FA', bg: '#60A5FA18', border: '#60A5FA44' };
    const r = role.toLowerCase();
    if (r.includes('specialization') || r.includes('sub-type')) {
      return { color: '#F59E0B', bg: '#F59E0B18', border: '#F59E0B44' };
    }
    if (r.includes('peer') || r.includes('symbology')) {
      return { color: '#A855F7', bg: '#A855F718', border: '#A855F744' };
    }
    if (r.includes('parent') || r.includes('general')) {
      return { color: '#3B82F6', bg: '#3B82F618', border: '#3B82F644' };
    }
    return { color: '#10B981', bg: '#10B98118', border: '#10B98144' };
  };

  return (
    <Box
      sx={{
        mt: compact ? 0.5 : 1,
        p: compact ? 1 : 1.5,
        borderRadius: 2,
        background: 'rgba(15, 23, 42, 0.65)',
        border: '1px solid rgba(99, 102, 241, 0.25)',
        backdropFilter: 'blur(8px)',
      }}
    >
      {/* Header bar */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <HubIcon sx={{ fontSize: 16, color: '#818CF8' }} />
          <Typography
            sx={{
              fontSize: 11,
              fontWeight: 700,
              color: '#C7D2FE',
              letterSpacing: '0.04em',
              textTransform: 'uppercase',
              fontFamily: 'monospace'
            }}
          >
            Related Terms & Node Differentiators
          </Typography>
          <Chip
            size="small"
            label={`${data.related_terms.length} related`}
            sx={{
              height: 18,
              fontSize: 10,
              fontWeight: 700,
              backgroundColor: '#4338CA44',
              color: '#A5B4FC',
              border: '1px solid #6366F144'
            }}
          />
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          {onOpenAIFeed && (
            <Tooltip title="View exact context feed generated for AI / LLMs">
              <Button
                size="small"
                variant="outlined"
                onClick={() => {
                  const allTerms = [data.primary_term.term_name, ...data.related_terms.map(r => r.term_name)];
                  onOpenAIFeed(allTerms);
                }}
                startIcon={<PsychologyIcon sx={{ fontSize: 13 }} />}
                sx={{
                  py: 0.2,
                  px: 1,
                  fontSize: 10,
                  fontWeight: 700,
                  textTransform: 'none',
                  borderColor: '#6366F166',
                  color: '#C7D2FE',
                  '&:hover': {
                    borderColor: '#818CF8',
                    backgroundColor: '#6366F122'
                  }
                }}
              >
                Feed AI
              </Button>
            </Tooltip>
          )}

          <IconButton
            size="small"
            onClick={() => setExpanded(!expanded)}
            sx={{ color: '#94A3B8', p: 0.5 }}
          >
            {expanded ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
          </IconButton>
        </Box>
      </Box>

      {/* 3-Tier Taxonomy Breadcrumb */}
      {l3Classification && (
        <Box sx={{ mt: 0.75, display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
          <Typography sx={{ fontSize: 10, fontWeight: 700, color: '#94A3B8', textTransform: 'uppercase', letterSpacing: '0.03em' }}>
            Taxonomy (L3):
          </Typography>
          <Tooltip title="Strict 3-Tier Enterprise Classification (Domain > Category > Classification)">
            <Chip
              size="small"
              icon={<AutoAwesomeIcon sx={{ fontSize: '12px !important', color: '#34D399' }} />}
              label={l3Classification.breadcrumb || `${l3Classification.domain_name} > ${l3Classification.category_name} > ${l3Classification.name}`}
              sx={{
                height: 20,
                fontSize: 10.5,
                fontWeight: 600,
                backgroundColor: 'rgba(16, 185, 129, 0.12)',
                color: '#6EE7B7',
                border: '1px solid rgba(16, 185, 129, 0.3)',
                '& .MuiChip-label': { px: 0.75 }
              }}
            />
          </Tooltip>
        </Box>
      )}

      {/* Suggestion Chips */}
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mt: 1, alignItems: 'center' }}>
        {data.related_terms.map((term, idx) => {
          const style = getRoleColor(term.role);
          const isSelected = activeComparison?.term_name === term.term_name;

          return (
            <Tooltip
              key={idx}
              title={
                <Box sx={{ p: 0.5 }}>
                  <Typography sx={{ fontWeight: 700, fontSize: 12, color: '#F8FAFC' }}>
                    {term.term_name} ({term.role || 'Related'})
                  </Typography>
                  {term.differentiation_notes && (
                    <Typography sx={{ fontSize: 11, color: '#E2E8F0', mt: 0.5 }}>
                      {term.differentiation_notes}
                    </Typography>
                  )}
                  {term.standard && (
                    <Typography sx={{ fontSize: 10, color: '#A5B4FC', mt: 0.5 }}>
                      Standard: {term.standard} {term.format_pattern ? `(${term.format_pattern})` : ''}
                    </Typography>
                  )}
                  <Typography sx={{ fontSize: 10, color: '#6EE7B7', mt: 0.5 }}>
                    Click to compare or map to this term
                  </Typography>
                </Box>
              }
              arrow
            >
              <Chip
                clickable
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <span>{term.term_name}</span>
                    {term.role && (
                      <span
                        style={{
                          fontSize: 9,
                          opacity: 0.85,
                          fontFamily: 'monospace',
                          padding: '0 3px',
                          borderRadius: 3,
                          backgroundColor: `${style.color}22`
                        }}
                      >
                        {term.role.split(' ')[0]}
                      </span>
                    )}
                  </Box>
                }
                onClick={() => {
                  setActiveComparison(isSelected ? null : term);
                  if (!expanded) setExpanded(true);
                }}
                sx={{
                  height: 24,
                  fontSize: 11,
                  fontWeight: 600,
                  color: style.color,
                  backgroundColor: isSelected ? style.bg : `${style.color}10`,
                  border: `1px solid ${isSelected ? style.color : style.border}`,
                  '&:hover': {
                    backgroundColor: style.bg,
                    borderColor: style.color,
                  }
                }}
              />
            </Tooltip>
          );
        })}
      </Box>

      {/* Expandable Disambiguation & Comparison Card */}
      <Collapse in={expanded} timeout="auto" unmountOnExit>
        <Box
          sx={{
            mt: 1.5,
            p: 1.5,
            borderRadius: 1.5,
            backgroundColor: 'rgba(30, 41, 59, 0.8)',
            border: '1px solid rgba(148, 163, 184, 0.15)',
          }}
        >
          {activeComparison ? (
            // Comparison view for a selected related term
            <Box>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                <Typography sx={{ fontSize: 12, fontWeight: 700, color: '#F1F5F9', display: 'flex', alignItems: 'center', gap: 1 }}>
                  <CompareArrowsIcon sx={{ fontSize: 16, color: '#F59E0B' }} />
                  Comparison: <span style={{ color: '#818CF8' }}>{data.primary_term.term_name}</span> vs <span style={{ color: '#F59E0B' }}>{activeComparison.term_name}</span>
                </Typography>
                {onSelectTerm && (
                  <Button
                    size="small"
                    variant="contained"
                    startIcon={<CheckIcon sx={{ fontSize: 13 }} />}
                    onClick={() => onSelectTerm(activeComparison.term_name, activeComparison)}
                    sx={{
                      fontSize: 10,
                      py: 0.2,
                      px: 1.5,
                      textTransform: 'none',
                      backgroundColor: '#10B981',
                      '&:hover': { backgroundColor: '#059669' }
                    }}
                  >
                    Map to {activeComparison.term_name}
                  </Button>
                )}
              </Box>

              <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 1.5, mt: 1 }}>
                {/* Primary Card */}
                <Paper sx={{ p: 1, backgroundColor: 'rgba(15, 23, 42, 0.7)', border: '1px solid rgba(99, 102, 241, 0.3)' }}>
                  <Typography sx={{ fontSize: 11, fontWeight: 700, color: '#C7D2FE' }}>
                    {data.primary_term.term_name} (Current)
                  </Typography>
                  <Typography sx={{ fontSize: 10, color: '#94A3B8', mt: 0.5 }}>
                    {data.primary_term.differentiation_notes || 'Standard umbrella term in ' + (data.primary_term.domain || 'Domain')}
                  </Typography>
                  {data.primary_term.standard && (
                    <Typography sx={{ fontSize: 10, color: '#6EE7B7', mt: 0.5, fontFamily: 'monospace' }}>
                      Standard: {data.primary_term.standard}
                    </Typography>
                  )}
                </Paper>

                {/* Alternative Card */}
                <Paper sx={{ p: 1, backgroundColor: 'rgba(15, 23, 42, 0.7)', border: '1px solid rgba(245, 158, 11, 0.3)' }}>
                  <Typography sx={{ fontSize: 11, fontWeight: 700, color: '#FDE68A' }}>
                    {activeComparison.term_name} ({activeComparison.role || 'Specialization'})
                  </Typography>
                  <Typography sx={{ fontSize: 10, color: '#E2E8F0', mt: 0.5 }}>
                    {activeComparison.differentiation_notes || 'Specialized domain node with tailored semantics.'}
                  </Typography>
                  {activeComparison.standard && (
                    <Typography sx={{ fontSize: 10, color: '#FBBF24', mt: 0.5, fontFamily: 'monospace' }}>
                      Standard: {activeComparison.standard}
                    </Typography>
                  )}
                  {activeComparison.format_pattern && (
                    <Typography sx={{ fontSize: 10, color: '#94A3B8', mt: 0.2, fontFamily: 'monospace' }}>
                      Pattern: {activeComparison.format_pattern}
                    </Typography>
                  )}
                </Paper>
              </Box>
            </Box>
          ) : (
            // Summary view
            <Box>
              <Typography sx={{ fontSize: 11, fontWeight: 700, color: '#94A3B8', mb: 0.5, textTransform: 'uppercase', letterSpacing: '0.04em' }}>
                Disambiguation Guidance
              </Typography>
              <Typography sx={{ fontSize: 11, color: '#CBD5E1', whiteSpace: 'pre-line', lineHeight: 1.5 }}>
                {data.differentiator_summary}
              </Typography>
            </Box>
          )}
        </Box>
      </Collapse>
    </Box>
  );
};
