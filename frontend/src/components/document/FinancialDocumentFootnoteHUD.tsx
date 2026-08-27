import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  Grid,
  Divider,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow
} from '@mui/material';
import {
  Description as DocIcon,
  MenuBook as FootnoteIcon,
  CropFree as BBoxIcon,
  AccountTree as GraphIcon,
  CheckCircle as ValidIcon
} from '@mui/icons-material';

interface CellNode {
  cellId: string;
  rowHeader: string;
  colHeader: string;
  numericValue: number;
  rawText: string;
  bbox: { x1: number; y1: number; x2: number; y2: number };
  footnoteNumber?: string;
  footnoteText?: string;
  valuationHierarchy?: string;
}

interface FinancialDocumentFootnoteHUDProps {
  documentTitle?: string;
  documentKey?: string;
  pageNumber?: number;
  cells?: CellNode[];
}

export const FinancialDocumentFootnoteHUD: React.FC<FinancialDocumentFootnoteHUDProps> = ({
  documentTitle = 'Global Equity Fund - 2026 Q2 Statement of Investments.pdf',
  documentKey = 'doc.fund.2026_q2_investments',
  pageNumber = 14,
  cells = [
    {
      cellId: 'cell-1',
      rowHeader: 'Level 3 Private Equity Assets',
      colHeader: 'Fair Market Value',
      numericValue: 450000000,
      rawText: '$450,000,000 (Note 4)',
      bbox: { x1: 45, y1: 180, x2: 210, y2: 202 },
      footnoteNumber: 'Note 4',
      footnoteText: 'Valued using unobservable market inputs including a weighted 14.5% discount rate and revenue multiple of 6.2x.',
      valuationHierarchy: 'LEVEL_3_UNOBSERVABLE'
    },
    {
      cellId: 'cell-2',
      rowHeader: 'US Treasury Bills (3-Month)',
      colHeader: 'Fair Market Value',
      numericValue: 120000000,
      rawText: '$120,000,000',
      bbox: { x1: 45, y1: 210, x2: 210, y2: 232 },
      valuationHierarchy: 'LEVEL_1_QUOTED'
    }
  ]
}) => {
  const [activeHoverCell, setActiveHoverCell] = useState<CellNode | null>(cells[0] || null);

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
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={2} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <DocIcon sx={{ color: '#00D4FF', fontSize: 24 }} />
          <Box>
            <Typography variant="subtitle1" fontWeight={700} sx={{ color: '#F8FAFC' }}>
              Multimodal Document & Footnote Graph Inspector
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              {documentTitle} | Page {pageNumber} | Key: <span style={{ color: '#38bdf8', fontFamily: 'monospace' }}>{documentKey}</span>
            </Typography>
          </Box>
        </Stack>
        <Chip
          icon={<ValidIcon sx={{ fontSize: 14 }} />}
          label="AST Mapped"
          size="small"
          sx={{ bgcolor: '#064E3B', color: '#34D399', fontWeight: 700, fontSize: 11 }}
        />
      </Box>

      <Grid container spacing={3}>
        <Grid   size={{ xs: 12, md: 7 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 600, textTransform: 'uppercase', mb: 1, display: 'block' }}>
            Structured Statement Grid (Hover to inspect coordinates & footnotes)
          </Typography>
          <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B' }}>
            <Table size="small">
              <TableHead>
                <TableRow sx={{ '& th': { color: '#94A3B8', borderColor: '#1E293B', fontSize: 11 } }}>
                  <TableCell>Line Item Description</TableCell>
                  <TableCell align="right">Fair Value (USD)</TableCell>
                  <TableCell align="center">Hierarchy Tier</TableCell>
                  <TableCell align="center">Footnote Link</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {cells.map((c) => {
                  const isSelected = activeHoverCell?.cellId === c.cellId;
                  return (
                    <TableRow
                      key={c.cellId}
                      onMouseEnter={() => setActiveHoverCell(c)}
                      sx={{
                        cursor: 'pointer',
                        bgcolor: isSelected ? 'rgba(0, 212, 255, 0.08)' : 'transparent',
                        '& td': { borderColor: '#1E293B', color: '#F8FAFC' }
                      }}
                    >
                      <TableCell sx={{ fontWeight: 600, fontSize: 12 }}>{c.rowHeader}</TableCell>
                      <TableCell align="right" sx={{ fontFamily: 'monospace', color: '#34D399', fontSize: 12 }}>
                        ${c.numericValue.toLocaleString()}
                      </TableCell>
                      <TableCell align="center">
                        <Chip
                          label={c.valuationHierarchy === 'LEVEL_3_UNOBSERVABLE' ? 'Level 3' : 'Level 1'}
                          size="small"
                          sx={{
                            bgcolor: c.valuationHierarchy === 'LEVEL_3_UNOBSERVABLE' ? '#7C2D12' : '#064E3B',
                            color: c.valuationHierarchy === 'LEVEL_3_UNOBSERVABLE' ? '#FDBA74' : '#6EE7B7',
                            fontSize: 10,
                            fontWeight: 700
                          }}
                        />
                      </TableCell>
                      <TableCell align="center">
                        {c.footnoteNumber ? (
                          <Chip
                            icon={<FootnoteIcon sx={{ fontSize: 12 }} />}
                            label={c.footnoteNumber}
                            size="small"
                            sx={{ bgcolor: '#1E293B', color: '#38BDF8', fontSize: 10, border: '1px solid #0284C7' }}
                          />
                        ) : (
                          <Typography variant="caption" sx={{ color: '#64748B' }}>—</Typography>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
        </Grid>

        <Grid   size={{ xs: 12, md: 5 }}>
          {activeHoverCell ? (
            <Stack spacing={2}>
              <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
                <Box display="flex" alignItems="center" gap={1} mb={1}>
                  <BBoxIcon sx={{ color: '#00D4FF', fontSize: 18 }} />
                  <Typography variant="subtitle2" sx={{ fontWeight: 700, fontSize: 12 }}>
                    Spatial Bounding Coordinates
                  </Typography>
                </Box>
                <Grid container spacing={1} sx={{ textAlign: 'center', fontFamily: 'monospace', fontSize: 12 }}>
                  <Grid  size={{ xs: 3 }}>
                    <Box sx={{ p: 0.5, bgcolor: '#071526', borderRadius: 1, border: '1px solid #1E293B' }}>
                      <Typography variant="caption" sx={{ color: '#64748b', display: 'block', fontSize: 10 }}>X1</Typography>
                      <Typography sx={{ color: '#00d4ff', fontSize: 12 }}>{activeHoverCell.bbox.x1}</Typography>
                    </Box>
                  </Grid>
                  <Grid  size={{ xs: 3 }}>
                    <Box sx={{ p: 0.5, bgcolor: '#071526', borderRadius: 1, border: '1px solid #1E293B' }}>
                      <Typography variant="caption" sx={{ color: '#64748b', display: 'block', fontSize: 10 }}>Y1</Typography>
                      <Typography sx={{ color: '#00d4ff', fontSize: 12 }}>{activeHoverCell.bbox.y1}</Typography>
                    </Box>
                  </Grid>
                  <Grid  size={{ xs: 3 }}>
                    <Box sx={{ p: 0.5, bgcolor: '#071526', borderRadius: 1, border: '1px solid #1E293B' }}>
                      <Typography variant="caption" sx={{ color: '#64748b', display: 'block', fontSize: 10 }}>X2</Typography>
                      <Typography sx={{ color: '#00d4ff', fontSize: 12 }}>{activeHoverCell.bbox.x2}</Typography>
                    </Box>
                  </Grid>
                  <Grid  size={{ xs: 3 }}>
                    <Box sx={{ p: 0.5, bgcolor: '#071526', borderRadius: 1, border: '1px solid #1E293B' }}>
                      <Typography variant="caption" sx={{ color: '#64748b', display: 'block', fontSize: 10 }}>Y2</Typography>
                      <Typography sx={{ color: '#00d4ff', fontSize: 12 }}>{activeHoverCell.bbox.y2}</Typography>
                    </Box>
                  </Grid>
                </Grid>
              </Paper>

              {activeHoverCell.footnoteNumber && activeHoverCell.footnoteText ? (
                <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
                  <Box display="flex" alignItems="center" gap={1} mb={1}>
                    <GraphIcon sx={{ color: '#38BDF8', fontSize: 18 }} />
                    <Typography variant="subtitle2" sx={{ fontWeight: 700, fontSize: 12, color: '#38BDF8' }}>
                      Linked Graph Disclosure ({activeHoverCell.footnoteNumber})
                    </Typography>
                  </Box>
                  <Typography variant="body2" sx={{ color: '#CBD5E1', fontSize: 11, lineHeight: 1.5 }}>
                    {activeHoverCell.footnoteText}
                  </Typography>
                  <Divider sx={{ my: 1.5, borderColor: '#1E293B' }} />
                  <Stack direction="row" spacing={1}>
                    <Chip label="EXPLAINED_BY_FOOTNOTE" size="small" sx={{ bgcolor: '#071526', color: '#94A3B8', fontSize: 9, fontFamily: 'monospace' }} />
                    <Chip label="US_GAAP_ASC_820" size="small" sx={{ bgcolor: '#071526', color: '#00D4FF', fontSize: 9, fontFamily: 'monospace' }} />
                  </Stack>
                </Paper>
              ) : (
                <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5, textAlign: 'center' }}>
                  <Typography variant="caption" sx={{ color: '#64748B' }}>
                    No explicit footnote callouts attached to this line item.
                  </Typography>
                </Paper>
              )}
            </Stack>
          ) : (
            <Paper sx={{ p: 4, bgcolor: '#0B1E36', border: '1px solid #1E293B', textAlign: 'center' }}>
              <Typography variant="caption" sx={{ color: '#64748B' }}>
                Select a line item from the grid to inspect spatial bounding boxes and resolved footnotes.
              </Typography>
            </Paper>
          )}
        </Grid>
      </Grid>
    </Paper>
  );
};

export default FinancialDocumentFootnoteHUD;
