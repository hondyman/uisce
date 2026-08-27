import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  IconButton,
  Chip,
  Switch,
  FormControlLabel,
  Stack,
  Divider,
} from '@mui/material';
import AutoStoriesIcon from '@mui/icons-material/AutoStories';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import MenuBookIcon from '@mui/icons-material/MenuBook';
import PictureAsPdfIcon from '@mui/icons-material/PictureAsPdf';

export interface BookSectionItem {
  id: string;
  reportDefinitionId: string;
  reportName: string;
  chapterTitle: string;
  insertDividerTab: boolean;
  estimatedPages: number;
}

export const ReportBookStudio: React.FC<{ tenantId?: string }> = () => {
  const [includeTOC, setIncludeTOC] = useState(true);
  const [sections, setSections] = useState<BookSectionItem[]>([
    {
      id: 'sec_1',
      reportDefinitionId: 'rep_fact_sheet',
      reportName: 'Flagship Fund Fact Sheet (1-Page)',
      chapterTitle: 'Executive Summary & Key Metrics',
      insertDividerTab: false,
      estimatedPages: 1,
    },
    {
      id: 'sec_2',
      reportDefinitionId: 'rep_holdings_hierarchical',
      reportName: 'Multi-Level Hierarchical Portfolio Rollup',
      chapterTitle: 'Portfolio Holdings & Valuations',
      insertDividerTab: true,
      estimatedPages: 4,
    },
    {
      id: 'sec_3',
      reportDefinitionId: 'rep_brinson_attribution',
      reportName: 'Brinson-Fachler Performance Attribution',
      chapterTitle: 'Performance Attribution & Alpha Decomposition',
      insertDividerTab: true,
      estimatedPages: 2,
    },
    {
      id: 'sec_4',
      reportDefinitionId: 'rep_compliance_exceptions',
      reportName: 'Pre-Trade & Post-Trade Compliance Ledger',
      chapterTitle: 'Regulatory Disclosures & Mandate Compliance',
      insertDividerTab: false,
      estimatedPages: 2,
    },
  ]);

  const totalCalculatedPages = (includeTOC ? 2 : 1) + sections.reduce(
    (acc, s) => acc + s.estimatedPages + (s.insertDividerTab ? 1 : 0),
    0
  );

  return (
    <Box sx={{ width: '100%', minHeight: '100vh', bgcolor: '#050D1A', color: '#fff', p: 3, fontFamily: 'sans-serif' }}>
      
      {/* Top Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 2, borderBottom: '1px solid #1E293B', mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <AutoStoriesIcon sx={{ color: '#00D4FF', fontSize: 26 }} />
          <Box>
            <Typography variant="h6" fontWeight="700">
              Report Book & Booklet Studio
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Compile multi-report executive pitchbooks, fact sheets, and regulatory packages into a single publication.
            </Typography>
          </Box>
        </Box>

        <Stack direction="row" spacing={2} alignItems="center">
          <Chip
            size="small"
            label={`${totalCalculatedPages} Pages Total`}
            sx={{ bgcolor: 'rgba(0, 212, 255, 0.1)', color: '#00D4FF', fontWeight: 700, fontSize: '11px', border: '1px solid rgba(0, 212, 255, 0.3)' }}
          />
          <Button
            variant="contained"
            startIcon={<PictureAsPdfIcon />}
            sx={{ bgcolor: '#00D4FF', color: '#050D1A', fontWeight: 700, fontSize: '11px', textTransform: 'none', '&:hover': { bgcolor: '#38BDF8' } }}
          >
            Compile & Preview Booklet
          </Button>
        </Stack>
      </Box>

      {/* Main Studio Split */}
      <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 340px', gap: 3 }}>
        
        {/* Left: Reorderable Section Deck */}
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="caption" fontWeight="700" sx={{ color: '#00D4FF', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Booklet Chapters & Report Sequence
            </Typography>
            <Button
              size="small"
              startIcon={<AddIcon />}
              sx={{ color: '#00D4FF', fontSize: '11px', textTransform: 'none' }}
            >
              Add Report Section
            </Button>
          </Box>

          <Stack spacing={1.5}>
            {sections.map((sec, idx) => (
              <Paper
                key={sec.id}
                sx={{
                  p: 2,
                  bgcolor: '#071526',
                  border: '1px solid #1E293B',
                  borderRadius: 2,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  '&:hover': { borderColor: 'rgba(0, 212, 255, 0.4)' },
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <DragIndicatorIcon sx={{ color: '#475569', cursor: 'grab' }} />
                  <Box>
                    <Typography variant="caption" sx={{ color: '#00D4FF', fontWeight: 700, display: 'block' }}>
                      Chapter {idx + 1}: {sec.chapterTitle}
                    </Typography>
                    <Typography variant="body2" fontWeight="600" sx={{ color: '#E2E8F0', fontSize: '12px' }}>
                      {sec.reportName}
                    </Typography>
                  </Box>
                </Box>

                <Stack direction="row" spacing={2} alignItems="center">
                  <FormControlLabel
                    control={
                      <Switch
                        size="small"
                        checked={sec.insertDividerTab}
                        onChange={(e) => {
                          const next = [...sections];
                          next[idx].insertDividerTab = e.target.checked;
                          setSections(next);
                        }}
                      />
                    }
                    label={<Typography variant="caption" sx={{ color: '#94A3B8', fontSize: '10px' }}>Divider Tab</Typography>}
                  />

                  <Chip
                    size="small"
                    label={`${sec.estimatedPages} pgs`}
                    sx={{ bgcolor: '#0B1E36', color: '#94A3B8', fontSize: '10px', height: 20 }}
                  />

                  <IconButton size="small" sx={{ color: '#64748B', '&:hover': { color: '#EF4444' } }}>
                    <DeleteOutlineIcon fontSize="small" />
                  </IconButton>
                </Stack>
              </Paper>
            ))}
          </Stack>
        </Box>

        {/* Right: Dynamic Table of Contents (TOC) Preview */}
        <Paper sx={{ p: 2.5, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 2, height: 'fit-content' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
            <MenuBookIcon sx={{ color: '#00D4FF', fontSize: 20 }} />
            <Typography variant="subtitle2" fontWeight="700">
              Live Table of Contents
            </Typography>
          </Box>

          <FormControlLabel
            control={
              <Switch
                size="small"
                checked={includeTOC}
                onChange={(e) => setIncludeTOC(e.target.checked)}
              />
            }
            label={<Typography variant="caption" sx={{ color: '#E2E8F0', fontSize: '11px' }}>Generate Page 2 Table of Contents</Typography>}
            sx={{ mb: 2 }}
          />

          <Divider sx={{ borderColor: '#1E293B', mb: 2 }} />

          {includeTOC ? (
            <Box sx={{ fontFamily: 'monospace', fontSize: '11px' }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', color: '#94A3B8', mb: 1 }}>
                <span>Cover Page</span>
                <span style={{ color: '#22D3EE' }}>Page 1</span>
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', color: '#94A3B8', mb: 1.5 }}>
                <span>Table of Contents</span>
                <span style={{ color: '#22D3EE' }}>Page 2</span>
              </Box>

              {sections.map((sec, idx) => {
                let startPage = 3;
                for (let i = 0; i < idx; i++) {
                  startPage += sections[i].estimatedPages + (sections[i].insertDividerTab ? 1 : 0);
                }
                return (
                  <Box key={sec.id} sx={{ display: 'flex', justifyContent: 'space-between', color: '#E2E8F0', mb: 1 }}>
                    <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', paddingRight: '8px' }}>
                      {idx + 1}. {sec.chapterTitle}
                    </span>
                    <span style={{ color: '#22D3EE', fontWeight: 700 }}>Pg {startPage}</span>
                  </Box>
                );
              })}
            </Box>
          ) : (
            <Typography variant="caption" sx={{ color: '#64748B', fontStyle: 'italic' }}>
              Table of Contents disabled. Booklet will begin directly on Page 2.
            </Typography>
          )}
        </Paper>

      </Box>

    </Box>
  );
};

export default ReportBookStudio;
