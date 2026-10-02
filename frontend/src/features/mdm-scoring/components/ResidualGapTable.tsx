import React from 'react';
import {
  Box, Card, CardContent, Typography, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, Paper, Chip, GridLegacy as Grid, Alert
} from '@mui/material';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import HandshakeIcon from '@mui/icons-material/Handshake';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import LightbulbOutlinedIcon from '@mui/icons-material/LightbulbOutlined';
import type { DisplacementGapReport, ResidualGapAttribute } from '../api';

export interface ResidualGapTableProps {
  gapReport?: DisplacementGapReport;
  universeSize?: number;
  fullBundleCost?: number;
}

export const ResidualGapTable: React.FC<ResidualGapTableProps> = ({
  gapReport,
  fullBundleCost = 0,
}) => {
  if (!gapReport) {
    return null;
  }

  const {
    total_entities_affected,
    tier1_gaps = [],
    tier2_gaps = [],
    tier3_gaps = [],
    recommended_sub_licenses = [],
    estimated_gap_remediation_cost,
    net_negotiation_leverage,
  } = gapReport;

  const totalGapsCount = tier1_gaps.length + tier2_gaps.length + tier3_gaps.length;

  if (totalGapsCount === 0 && recommended_sub_licenses.length === 0) {
    return (
      <Alert severity="success" sx={{ mb: 3, borderRadius: 2 }}>
        <strong>Zero Orphaned Residual Gaps:</strong> The proposed displacement achieves complete attribute coverage with surviving/replacement feeds. No sub-license carve-outs required.
      </Alert>
    );
  }

  const fmtCurrency = (val: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(val);

  // Leverage Categorization
  let leverageLabel = 'Minimal Vendor Dependency';
  let leverageColor: 'success' | 'info' | 'warning' | 'error' = 'success';
  if (net_negotiation_leverage < 0.50) {
    leverageLabel = 'Heavy Dependency - High Carve-Out Risk';
    leverageColor = 'error';
  } else if (net_negotiation_leverage < 0.75) {
    leverageLabel = 'Moderate Dependency - Carve-Out Required';
    leverageColor = 'warning';
  } else if (net_negotiation_leverage < 0.90) {
    leverageLabel = 'Minor Carve-Out Needed';
    leverageColor = 'info';
  }

  const renderAttributeRow = (attr: ResidualGapAttribute) => (
    <TableRow key={attr.attribute_code} hover>
      <TableCell>
        <code><strong>{attr.attribute_code}</strong></code>
      </TableCell>
      <TableCell>
        <Chip size="small" variant="outlined" label={attr.entity_domain} />
      </TableCell>
      <TableCell align="right">
        {attr.entities_affected.toLocaleString()}
      </TableCell>
      <TableCell>
        <strong>{attr.sole_source_vendor_id}</strong>
      </TableCell>
      <TableCell>
        {attr.suggested_substitute_id ? (
          <Chip
            size="small"
            color="success"
            icon={<CheckCircleOutlineIcon />}
            label={`Substitute: ${attr.suggested_substitute_id}`}
          />
        ) : (
          <Chip size="small" color="error" variant="outlined" label="Sole Source (No Substitute)" />
        )}
      </TableCell>
      <TableCell align="right" sx={{ fontWeight: 600, color: attr.estimated_sub_license_cost > 0 ? 'warning.main' : 'text.secondary' }}>
        {attr.estimated_sub_license_cost > 0 ? fmtCurrency(attr.estimated_sub_license_cost) : '$0 (Covered)'}
      </TableCell>
    </TableRow>
  );

  return (
    <Box sx={{ mt: 3, mb: 3 }}>
      {/* 1. Negotiation Leverage & Intelligence Banner */}
      <Card
        variant="outlined"
        sx={{
          mb: 3,
          borderRadius: 2,
          borderColor: `${leverageColor}.main`,
          bgcolor: `${leverageColor}.50`,
          borderWidth: 2,
        }}
      >
        <CardContent sx={{ p: 2.5 }}>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} md={7}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1 }}>
                <HandshakeIcon color={leverageColor} sx={{ fontSize: 32 }} />
                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 700, lineHeight: 1.2 }}>
                    Negotiation Leverage Score: {(net_negotiation_leverage * 100).toFixed(1)}%
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Formula: 1 - (Targeted Sub-Licenses / Full Exit Bundle Spend)
                  </Typography>
                </Box>
                <Chip
                  label={leverageLabel}
                  color={leverageColor}
                  size="small"
                  sx={{ fontWeight: 700, ml: 'auto' }}
                />
              </Box>
              <Typography variant="body2" sx={{ color: 'text.primary', mt: 1 }}>
                Rather than renewing the complete <strong>{fmtCurrency(fullBundleCost)}</strong> vendor bundle,
                procurement can propose targeted data carve-outs for only <strong>{total_entities_affected.toLocaleString()}</strong> orphaned entities,
                projected at <strong>{fmtCurrency(estimated_gap_remediation_cost)}</strong>/yr (capped at 60% of annual spend).
              </Typography>
            </Grid>

            <Grid item xs={12} md={5}>
              <Box
                sx={{
                  p: 2,
                  bgcolor: 'background.paper',
                  borderRadius: 2,
                  border: 1,
                  borderColor: 'divider',
                  textAlign: 'right',
                }}
              >
                <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'uppercase', fontWeight: 600 }}>
                  Estimated Carve-Out Remediation
                </Typography>
                <Typography variant="h4" sx={{ fontWeight: 800, color: 'primary.main', my: 0.5 }}>
                  {fmtCurrency(estimated_gap_remediation_cost)}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Across {recommended_sub_licenses.length} unbundled vendor proposal(s)
                </Typography>
              </Box>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* 2. Recommended Sub-License Carve-Out Proposals (The Negotiation Weapon) */}
      {recommended_sub_licenses.length > 0 && (
        <Card variant="outlined" sx={{ mb: 3, borderRadius: 2 }}>
          <CardContent sx={{ p: 2.5 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
              <LightbulbOutlinedIcon color="primary" />
              <Typography variant="h6" sx={{ fontWeight: 700 }}>
                Procurement Negotiation Tearsheet: Targeted Sub-Licenses
              </Typography>
            </Box>
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ bgcolor: 'action.hover' }}>
                    <TableCell sx={{ fontWeight: 700 }}>Target Vendor</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Priority</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Orphaned Attributes to Carve Out</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 700 }}>Entities In Scope</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 700 }}>Est. Sub-License Fee</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Procurement Strategy</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {recommended_sub_licenses.map((prop) => (
                    <TableRow key={prop.vendor_id} hover>
                      <TableCell>
                        <strong>{prop.vendor_name}</strong> ({prop.vendor_id})
                      </TableCell>
                      <TableCell>
                        <Chip
                          size="small"
                          label={`Tier ${prop.tier_priority}`}
                          color={prop.tier_priority === 1 ? 'error' : prop.tier_priority === 2 ? 'warning' : 'default'}
                        />
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                          {prop.attributes_covered.map((attr) => (
                            <Chip key={attr} size="small" variant="outlined" label={attr} />
                          ))}
                        </Box>
                      </TableCell>
                      <TableCell align="right">
                        {prop.entities_covered.toLocaleString()}
                      </TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700, color: 'success.main' }}>
                        {fmtCurrency(prop.estimated_annual_cost)}
                      </TableCell>
                      <TableCell>
                        <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block' }}>
                          Contract strictly for unbundled data carve-out (capped at 60% of original contract) instead of full bundle renewal.
                        </Typography>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </CardContent>
        </Card>
      )}

      {/* 3. Detailed Orphaned Attribute Inventory by Tier */}
      <Card variant="outlined" sx={{ borderRadius: 2 }}>
        <CardContent sx={{ p: 2.5 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
            <WarningAmberIcon color="warning" />
            <Typography variant="h6" sx={{ fontWeight: 700 }}>
              Residual Attribute Coverage Gaps ({totalGapsCount} total orphaned)
            </Typography>
          </Box>

          {/* Tier 1 Critical Gaps */}
          {tier1_gaps.length > 0 && (
            <Box sx={{ mb: 3 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <ErrorOutlineIcon color="error" fontSize="small" />
                <Typography variant="subtitle1" sx={{ fontWeight: 700, color: 'error.main' }}>
                  Tier 1 Critical / Regulatory Gaps ({tier1_gaps.length})
                </Typography>
              </Box>
              <TableContainer component={Paper} variant="outlined" sx={{ mb: 2 }}>
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'error.50' }}>
                      <TableCell sx={{ fontWeight: 700 }}>Attribute</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Domain</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Entities Affected</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Sole Source</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Substitute Status</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Sub-License Est.</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {tier1_gaps.map(renderAttributeRow)}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>
          )}

          {/* Tier 2 Core Reporting Gaps */}
          {tier2_gaps.length > 0 && (
            <Box sx={{ mb: 3 }}>
              <Typography variant="subtitle1" sx={{ fontWeight: 700, color: 'warning.main', mb: 1 }}>
                Tier 2 Reporting & Descriptive Gaps ({tier2_gaps.length})
              </Typography>
              <TableContainer component={Paper} variant="outlined" sx={{ mb: 2 }}>
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'warning.50' }}>
                      <TableCell sx={{ fontWeight: 700 }}>Attribute</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Domain</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Entities Affected</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Sole Source</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Substitute Status</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Sub-License Est.</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {tier2_gaps.map(renderAttributeRow)}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>
          )}

          {/* Tier 3 Enrichment Gaps */}
          {tier3_gaps.length > 0 && (
            <Box>
              <Typography variant="subtitle1" sx={{ fontWeight: 700, color: 'text.secondary', mb: 1 }}>
                Tier 3 Enrichment & Extended Reference Gaps ({tier3_gaps.length})
              </Typography>
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ bgcolor: 'action.hover' }}>
                      <TableCell sx={{ fontWeight: 700 }}>Attribute</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Domain</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Entities Affected</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Sole Source</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Substitute Status</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>Sub-License Est.</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {tier3_gaps.map(renderAttributeRow)}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>
          )}
        </CardContent>
      </Card>
    </Box>
  );
};
