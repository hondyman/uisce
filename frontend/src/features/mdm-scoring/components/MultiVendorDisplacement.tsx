import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Card, CardContent, Typography, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, Paper, Chip, Button, Stack, CircularProgress, Alert,
  FormGroup, FormControlLabel, Checkbox, Slider, GridLegacy as Grid
} from '@mui/material';
import PrintIcon from '@mui/icons-material/Print';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import CalculateIcon from '@mui/icons-material/Calculate';
import SwapHorizIcon from '@mui/icons-material/SwapHoriz';
import PictureAsPdfIcon from '@mui/icons-material/PictureAsPdf';
import {
  generatePixelPerfectPDF,
  ELEMENT_TYPES,
  type ReportElement,
  type LayoutSettings
} from '../../../components/reporting/reportingUtils';
import {
  mdmScoringApi,
  type MultiVendorDisplacementRequest,
  type MultiVendorDisplacementResult
} from '../api';
import { ResidualGapTable } from './ResidualGapTable';
import '../styles/print.css';

export interface MultiVendorDisplacementProps {
  initialDroppedVendors?: string[];
  initialReplacementVendors?: string[];
  universeSize?: number;
}

const ALL_CANDIDATES = [
  { id: 'BBG', name: 'Bloomberg', defaultSpend: 2140000 },
  { id: 'RFT', name: 'Refinitiv (LSEG)', defaultSpend: 1180000 },
  { id: 'FDS', name: 'FactSet', defaultSpend: 720000 },
  { id: 'ICE', name: 'ICE Data Services', defaultSpend: 540000 },
  { id: 'SPG', name: 'S&P Global MI', defaultSpend: 610000 },
];

export const MultiVendorDisplacement: React.FC<MultiVendorDisplacementProps> = ({
  initialDroppedVendors = ['BBG'],
  initialReplacementVendors = ['ICE'],
  universeSize = 42000,
}) => {
  const [droppedVendors, setDroppedVendors] = useState<string[]>(initialDroppedVendors);
  const [replacementVendors, setReplacementVendors] = useState<string[]>(initialReplacementVendors);
  const [targetT1, setTargetT1] = useState<number>(0.995);
  const [targetT2, setTargetT2] = useState<number>(0.950);
  const [targetT3, setTargetT3] = useState<number>(0.850);

  const [data, setData] = useState<MultiVendorDisplacementResult | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const evaluate = useCallback(async () => {
    if (droppedVendors.length === 0) {
      setError('Please select at least one vendor to exit/drop.');
      setData(null);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const req: MultiVendorDisplacementRequest = {
        dropped_vendor_ids: droppedVendors,
        replacement_vendor_ids: replacementVendors,
        target_t1_coverage: targetT1,
        target_t2_coverage: targetT2,
        target_t3_coverage: targetT3,
        universe_size: universeSize,
      };
      const res = await mdmScoringApi.displacementMulti(req);
      setData(res);
    } catch (err: any) {
      setError(err?.message || 'Failed to simulate multi-vendor displacement');
    } finally {
      setLoading(false);
    }
  }, [droppedVendors, replacementVendors, targetT1, targetT2, targetT3, universeSize]);

  useEffect(() => {
    evaluate();
  }, [evaluate]);

  const toggleDropped = (vid: string) => {
    setDroppedVendors((prev) => {
      const next = prev.includes(vid) ? prev.filter((id) => id !== vid) : [...prev, vid];
      // A vendor cannot be both dropped and a replacement
      setReplacementVendors((rep) => rep.filter((id) => id !== vid));
      return next;
    });
  };

  const toggleReplacement = (vid: string) => {
    setReplacementVendors((prev) => {
      const next = prev.includes(vid) ? prev.filter((id) => id !== vid) : [...prev, vid];
      // A vendor cannot be both dropped and a replacement
      setDroppedVendors((drop) => drop.filter((id) => id !== vid));
      return next;
    });
  };

  const fmtCurrency = (val: number) => {
    const abs = Math.abs(val);
    const sign = val < 0 ? '-' : '';
    if (abs >= 1000000) return `${sign}$${(abs / 1000000).toFixed(2)}M`;
    if (abs >= 1000) return `${sign}$${(abs / 1000).toFixed(0)}k`;
    return `${sign}$${abs.toFixed(0)}`;
  };

  const handleExportPDF = () => {
    if (!data) return;

    const licenseSavings = data.combined_tco?.license_savings_total ?? 0;
    const replacementCost = data.combined_tco?.replacement_cost_delta ?? 0;
    const netAnnualTCO = data.combined_tco?.net_annual_tco_benefit ?? 0;
    const sublicenseCost = data.gap_report?.estimated_gap_remediation_cost ?? 0;
    const leverage = data.gap_report?.net_negotiation_leverage ?? 0;

    const elements: ReportElement[] = [
      {
        id: 'header-title',
        type: ELEMENT_TYPES.TEXTBOX,
        section: 'reportHeader',
        position: { x: 40, y: 40 },
        size: { width: 500, height: 30 },
        properties: {
          text: 'MDM Vendor Displacement & Sublicense Analysis Tearsheet',
          textAlign: 'left',
          fontSize: 16,
          fontWeight: 'bold',
        },
      },
      {
        id: 'header-subtitle',
        type: ELEMENT_TYPES.TEXTBOX,
        section: 'reportHeader',
        position: { x: 40, y: 70 },
        size: { width: 500, height: 20 },
        properties: {
          text: `Dropped: ${data.dropped_vendor_ids.join(', ')} | Replacement: ${data.replacement_vendor_ids.join(', ')} | Universe: ${universeSize.toLocaleString()} | Date: ${new Date().toISOString().split('T')[0]}`,
          textAlign: 'left',
          fontSize: 10,
        },
      },
      {
        id: 'financial-summary-table',
        type: ELEMENT_TYPES.TABLE,
        section: 'body',
        position: { x: 40, y: 100 },
        size: { width: 520, height: 80 },
        properties: {
          columns: ['License Savings', 'Replacement Cost', 'Est Sub-license Cost', 'Net Annual TCO Benefit', 'Negotiation Leverage'],
          previewData: [
            [
              `$${Math.round(licenseSavings).toLocaleString()}`,
              `$${Math.round(replacementCost).toLocaleString()}`,
              `$${Math.round(sublicenseCost).toLocaleString()}`,
              `$${Math.round(netAnnualTCO).toLocaleString()}`,
              `${(leverage * 100).toFixed(1)}%`,
            ],
          ],
        },
      },
      {
        id: 'tier-compliance-table',
        type: ELEMENT_TYPES.TABLE,
        section: 'body',
        position: { x: 40, y: 200 },
        size: { width: 520, height: 100 },
        properties: {
          columns: ['Tier', 'Coverage Before', 'Coverage After', 'Target Threshold', 'Status'],
          previewData: (data.tier_coverage_deltas || []).map((t) => [
            `Tier ${t.tier}`,
            `${(t.coverage_before * 100).toFixed(1)}%`,
            `${(t.coverage_after * 100).toFixed(1)}%`,
            `${(t.threshold * 100).toFixed(1)}%`,
            t.meets_threshold ? 'Compliant' : 'Breach',
          ]),
        },
      },
    ];

    if (data.residual_gaps && data.residual_gaps.length > 0) {
      elements.push({
        id: 'residual-gaps-table',
        type: ELEMENT_TYPES.TABLE,
        section: 'body',
        position: { x: 40, y: 320 },
        size: { width: 520, height: 150 },
        properties: {
          columns: ['Attribute Code', 'Tier', 'Records Lost', 'Solo Rate %'],
          previewData: data.residual_gaps.map((g) => [
            g.attribute_code,
            `Tier ${g.tier}`,
            g.records_lost.toLocaleString(),
            `${(g.solo_rate_pct * 100).toFixed(1)}%`,
          ]),
        },
      });
    }

    if (data.gap_report?.recommended_sub_licenses && data.gap_report.recommended_sub_licenses.length > 0) {
      elements.push({
        id: 'sublicense-proposals-table',
        type: ELEMENT_TYPES.TABLE,
        section: 'body',
        position: { x: 40, y: 490 },
        size: { width: 520, height: 150 },
        properties: {
          columns: ['Target Vendor', 'Attributes Covered', 'Entities Covered', 'Est. Sub-license Cost'],
          previewData: data.gap_report.recommended_sub_licenses.map((p) => [
            p.vendor_name || p.vendor_id,
            (p.attributes_covered || []).join(', '),
            p.entities_covered.toLocaleString(),
            `$${Math.round(p.estimated_annual_cost).toLocaleString()}`,
          ]),
        },
      });
    }

    const layoutSettings: LayoutSettings = {
      pageBreakBeforeGroup: false,
      pageBreakAfterGroup: false,
      pageBreakBetweenRegions: false,
      fixedPageSize: false,
      columns: 1,
      columnSpacing: 10,
      headerTokens: ['CONFIDENTIAL - PROCUREMENT TEARSHEET'],
      footerTokens: ['Generated by Ivy MDM Intelligence Platform'],
      includeExecutionTime: true,
      includeUserName: false,
    };

    generatePixelPerfectPDF(elements, layoutSettings);
  };

  return (
    <Box sx={{ width: '100%' }} className="displacement-tearsheet">
      {/* Print-only Executive Header */}
      <div className="print-header">
        <h1>Institutional MDM: Multi-Vendor Displacement Tearsheet</h1>
        <p>
          Generated: {data?.generated_at ? new Date(data.generated_at).toLocaleDateString() : new Date().toLocaleDateString()} | Universe Size: {universeSize.toLocaleString()} Golden Master Entities | Precision SLA: 99.5%
        </p>
      </div>

      {/* Interactive Controls (hidden when printing) */}
      <Card variant="outlined" sx={{ mb: 3, borderRadius: 2 }} className="no-print">
        <CardContent sx={{ p: 2.5 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2, flexWrap: 'wrap', gap: 1.5 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <SwapHorizIcon color="primary" />
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                Multi-Vendor Displacement Scenario Builder
              </Typography>
            </Box>
            <Stack direction="row" spacing={1}>
              <Button
                variant="outlined"
                startIcon={<PictureAsPdfIcon />}
                onClick={handleExportPDF}
                size="small"
                color="secondary"
                disabled={!data || loading}
              >
                Export Tearsheet (PDF)
              </Button>
              <Button
                variant="outlined"
                startIcon={<PrintIcon />}
                onClick={() => window.print()}
                size="small"
              >
                Print Browser View
              </Button>
              <Button
                variant="contained"
                startIcon={<CalculateIcon />}
                onClick={evaluate}
                size="small"
              >
                Recalculate TCO
              </Button>
            </Stack>
          </Box>

          <Grid container spacing={3}>
            {/* Vendors to Drop */}
            <Grid item xs={12} md={6}>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, color: 'error.main', mb: 1 }}>
                1. Incumbents to Exit / Drop (License Savings)
              </Typography>
              <FormGroup row>
                {ALL_CANDIDATES.map((c) => (
                  <FormControlLabel
                    key={c.id}
                    control={
                      <Checkbox
                        checked={droppedVendors.includes(c.id)}
                        onChange={() => toggleDropped(c.id)}
                        color="error"
                      />
                    }
                    label={`${c.name} (${c.id})`}
                  />
                ))}
              </FormGroup>
            </Grid>

            {/* Replacement Vendors */}
            <Grid item xs={12} md={6}>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, color: 'success.main', mb: 1 }}>
                2. Target Replacement Vendors to Onboard
              </Typography>
              <FormGroup row>
                {ALL_CANDIDATES.map((c) => (
                  <FormControlLabel
                    key={c.id}
                    control={
                      <Checkbox
                        checked={replacementVendors.includes(c.id)}
                        onChange={() => toggleReplacement(c.id)}
                        color="success"
                      />
                    }
                    label={`${c.name} (${c.id})`}
                  />
                ))}
              </FormGroup>
            </Grid>

            {/* Target Coverage Sliders */}
            <Grid item xs={12}>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                Coverage Thresholds: Tier 1 ({(targetT1 * 100).toFixed(1)}%), Tier 2 ({(targetT2 * 100).toFixed(1)}%), Tier 3 ({(targetT3 * 100).toFixed(1)}%)
              </Typography>
              <Grid container spacing={3}>
                <Grid item xs={12} sm={4}>
                  <Typography variant="caption" color="text.secondary">Tier 1 Target</Typography>
                  <Slider
                    value={targetT1}
                    min={0.90}
                    max={1.00}
                    step={0.005}
                    valueLabelDisplay="auto"
                    valueLabelFormat={(v) => `${(v * 100).toFixed(1)}%`}
                    onChange={(_, val) => setTargetT1(val as number)}
                  />
                </Grid>
                <Grid item xs={12} sm={4}>
                  <Typography variant="caption" color="text.secondary">Tier 2 Target</Typography>
                  <Slider
                    value={targetT2}
                    min={0.80}
                    max={0.99}
                    step={0.01}
                    valueLabelDisplay="auto"
                    valueLabelFormat={(v) => `${(v * 100).toFixed(1)}%`}
                    onChange={(_, val) => setTargetT2(val as number)}
                  />
                </Grid>
                <Grid item xs={12} sm={4}>
                  <Typography variant="caption" color="text.secondary">Tier 3 Target</Typography>
                  <Slider
                    value={targetT3}
                    min={0.60}
                    max={0.95}
                    step={0.02}
                    valueLabelDisplay="auto"
                    valueLabelFormat={(v) => `${(v * 100).toFixed(1)}%`}
                    onChange={(_, val) => setTargetT3(val as number)}
                  />
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Loading & Error Indicators */}
      {loading && (
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', p: 4 }}>
          <CircularProgress size={28} />
          <Typography variant="body2" sx={{ ml: 2 }}>Calculating combined displacement survivorship & TCO...</Typography>
        </Box>
      )}

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>
      )}

      {data && (
        <Box>
          {/* Executive Summary Cards */}
          <Grid container spacing={2} sx={{ mb: 3 }}>
            <Grid item xs={12} sm={6} md={3}>
              <Card variant="outlined" sx={{ borderRadius: 2, height: '100%', bgcolor: 'background.paper' }}>
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>
                    Gross License Savings
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: 'success.main', mt: 0.5 }}>
                    {fmtCurrency(data.combined_tco.license_savings_total)}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {data.dropped_vendor_ids.length} exit vendor(s)
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card variant="outlined" sx={{ borderRadius: 2, height: '100%', bgcolor: 'background.paper' }}>
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>
                    Replacement & Friction Delta
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: 'warning.main', mt: 0.5 }}>
                    -{fmtCurrency(data.combined_tco.replacement_cost_delta + data.combined_tco.friction_delta_total)}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {data.replacement_vendor_ids.length} replacement(s)
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card variant="outlined" sx={{ borderRadius: 2, height: '100%', bgcolor: 'background.paper' }}>
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>
                    Net Year-1 Benefit
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: data.combined_tco.net_first_year_tco_benefit > 0 ? 'success.main' : 'error.main', mt: 0.5 }}>
                    {fmtCurrency(data.combined_tco.net_first_year_tco_benefit)}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    After one-time remediation drag
                  </Typography>
                </CardContent>
              </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
              <Card variant="outlined" sx={{ borderRadius: 2, height: '100%', bgcolor: 'background.paper' }}>
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>
                    Payback Period
                  </Typography>
                  <Typography variant="h5" sx={{ fontWeight: 700, color: 'primary.main', mt: 0.5 }}>
                    {data.combined_tco.payback_months <= 12
                      ? `${data.combined_tco.payback_months.toFixed(1)} mos`
                      : `${(data.combined_tco.payback_months / 12).toFixed(1)} yrs`}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Recurring net: {fmtCurrency(data.combined_tco.net_annual_tco_benefit)}/yr
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          </Grid>

          {/* Tier Coverage Deltas */}
          <Card variant="outlined" sx={{ mb: 3, borderRadius: 2 }}>
            <CardContent sx={{ p: 2.5 }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 1.5 }}>
                Tier Coverage & Quality Preservation
              </Typography>
              <Grid container spacing={2}>
                {data.tier_coverage_deltas.map((t) => (
                  <Grid item xs={12} sm={4} key={t.tier}>
                    <Box sx={{ p: 2, border: 1, borderColor: 'divider', borderRadius: 2, bgcolor: t.meets_threshold ? 'success.50' : 'warning.50' }}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                        <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                          Tier {t.tier} ({t.tier === 1 ? 'Primary Identifiers' : t.tier === 2 ? 'Core Descriptors' : 'Extended Reference'})
                        </Typography>
                        <Chip
                          size="small"
                          label={t.meets_threshold ? 'Meets SLA' : 'Below Target'}
                          color={t.meets_threshold ? 'success' : 'warning'}
                        />
                      </Box>
                      <Typography variant="body2">
                        Before: <strong>{(t.coverage_before * 100).toFixed(2)}%</strong> → After: <strong>{(t.coverage_after * 100).toFixed(2)}%</strong>
                      </Typography>
                      <Typography variant="caption" color={t.delta_pct >= 0 ? 'success.main' : 'error.main'}>
                        Delta: {t.delta_pct >= 0 ? '+' : ''}{(t.delta_pct * 100).toFixed(2)}% (Target: {(t.threshold * 100).toFixed(1)}%)
                      </Typography>
                    </Box>
                  </Grid>
                ))}
              </Grid>
            </CardContent>
          </Card>

          {/* Per-Vendor Financial Decomposition Table */}
          <Card variant="outlined" sx={{ mb: 3, borderRadius: 2 }}>
            <CardContent sx={{ p: 2.5 }}>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 1.5 }}>
                Per-Vendor Net TCO Decomposition
              </Typography>
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'action.hover' }}>
                      <TableCell sx={{ fontWeight: 700 }}>Vendor</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>License Savings</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Friction Delta</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Forfeited SLA</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Remediation Drag</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Net First-Year Benefit</TableCell>
                      <TableCell align="center" sx={{ fontWeight: 700 }}>Readiness</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {data.per_vendor_breakdown.map((row) => (
                      <TableRow key={row.vendor_id} hover>
                        <TableCell>
                          <strong>{row.vendor_name}</strong> ({row.vendor_id})
                        </TableCell>
                        <TableCell align="right" sx={{ color: 'success.main', fontWeight: 600 }}>
                          +{fmtCurrency(row.annual_savings)}
                        </TableCell>
                        <TableCell align="right">
                          -{fmtCurrency(row.friction_change)}
                        </TableCell>
                        <TableCell align="right">
                          -{fmtCurrency(row.sla_credits_lost)}
                        </TableCell>
                        <TableCell align="right">
                          -{fmtCurrency(row.remediation_cost)}
                        </TableCell>
                        <TableCell align="right" sx={{ fontWeight: 700, color: row.net_tco_benefit > 0 ? 'success.main' : 'error.main' }}>
                          {fmtCurrency(row.net_tco_benefit)}
                        </TableCell>
                        <TableCell align="center">
                          <Chip
                            size="small"
                            label={`${row.displacement_readiness_pct.toFixed(1)}%`}
                            color={row.displacement_readiness_pct >= 90 ? 'success' : row.displacement_readiness_pct >= 70 ? 'warning' : 'error'}
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                    {/* Combined Totals Row */}
                    <TableRow sx={{ bgcolor: 'action.selected' }}>
                      <TableCell sx={{ fontWeight: 800 }}>COMBINED TOTALS</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 800, color: 'success.main' }}>
                        +{fmtCurrency(data.combined_tco.license_savings_total)}
                      </TableCell>
                      <TableCell align="right" sx={{ fontWeight: 800 }}>
                        -{fmtCurrency(data.combined_tco.friction_delta_total)}
                      </TableCell>
                      <TableCell align="right" sx={{ fontWeight: 800 }}>
                        -{fmtCurrency(data.combined_tco.forfeited_sla_credits_total)}
                      </TableCell>
                      <TableCell align="right" sx={{ fontWeight: 800 }}>
                        -{fmtCurrency(data.combined_tco.remediation_drag_total)}
                      </TableCell>
                      <TableCell align="right" sx={{ fontWeight: 800, color: data.combined_tco.net_first_year_tco_benefit > 0 ? 'success.main' : 'error.main' }}>
                        {fmtCurrency(data.combined_tco.net_first_year_tco_benefit)}
                      </TableCell>
                      <TableCell align="center">
                        <Chip
                          size="small"
                          icon={<CheckCircleOutlineIcon />}
                          label="VIABLE"
                          color="success"
                          sx={{ fontWeight: 700 }}
                        />
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>

          {/* Residual Coverage Gaps & Negotiation Tearsheet */}
          {data.gap_report ? (
            <ResidualGapTable
              gapReport={data.gap_report}
              universeSize={universeSize}
              fullBundleCost={data.combined_tco.license_savings_total}
            />
          ) : data.residual_gaps && data.residual_gaps.length > 0 ? (
            <Card variant="outlined" sx={{ borderRadius: 2, borderColor: 'warning.light' }}>
              <CardContent sx={{ p: 2.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
                  <WarningAmberIcon color="warning" />
                  <Typography variant="h6" sx={{ fontWeight: 600 }}>
                    Residual Coverage Gaps Requiring Remediation
                  </Typography>
                </Box>
                <TableContainer component={Paper} variant="outlined">
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell sx={{ fontWeight: 700 }}>Attribute</TableCell>
                        <TableCell sx={{ fontWeight: 700 }}>Tier</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 700 }}>Solo Rate</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 700 }}>Records Affected</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {data.residual_gaps.map((gap) => (
                        <TableRow key={gap.attribute_code}>
                          <TableCell><code>{gap.attribute_code}</code></TableCell>
                          <TableCell>Tier {gap.tier}</TableCell>
                          <TableCell align="right">{(gap.solo_rate_pct * 100).toFixed(2)}%</TableCell>
                          <TableCell align="right">{gap.records_lost.toLocaleString()}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </CardContent>
            </Card>
          ) : null}
        </Box>
      )}
    </Box>
  );
};
