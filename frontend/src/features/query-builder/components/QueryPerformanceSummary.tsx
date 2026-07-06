/**
 * Query Performance Summary — Governance Scorecard.
 *
 * Surfaces the FederatedExplainPlan metrics and warnings so developers can
 * optimize semantic models before execution. Flags high-complexity queries
 * based on configurable thresholds.
 */

import React, { useMemo } from 'react';
import {
  Alert,
  AlertTitle,
  Box,
  Chip,
  Divider,
  Paper,
  Stack,
  Typography,
} from '@mui/material';
import WarningIcon from '@mui/icons-material/Warning';
import SpeedIcon from '@mui/icons-material/Speed';
import StorageIcon from '@mui/icons-material/Storage';
import FunctionsIcon from '@mui/icons-material/Functions';
import SecurityIcon from '@mui/icons-material/Security';
import DialectIcon from './DialectIcon';
import type { FederatedPlan, PlanNode } from '../types/queryDef';

interface Props {
  plan: FederatedPlan;
}

// Conservative alpha thresholds. These can be promoted to tenant-level
// configuration once governance policies are modelled in the catalog.
const GOVERNANCE_THRESHOLDS = {
  maxLatencyMs: 500,
  maxDataScannedBytes: 100 * 1024 * 1024, // 100 MB
  maxPlanCost: 1000,
};

function humanReadableBytes(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${value.toFixed(1)} ${units[unitIndex]}`;
}

function aggregateCost(node: PlanNode): number {
  let total = node.cost || 0;
  node.children?.forEach((child) => {
    total += aggregateCost(child);
  });
  return total;
}

function countNodes(node: PlanNode): number {
  let count = 1;
  node.children?.forEach((child) => {
    count += countNodes(child);
  });
  return count;
}

function collectDataSources(node: PlanNode): Set<string> {
  const sources = new Set<string>();
  sources.add(node.dataSource);
  node.children?.forEach((child) => {
    collectDataSources(child).forEach((s) => sources.add(s));
  });
  return sources;
}

export const QueryPerformanceSummary: React.FC<Props> = ({ plan }) => {
  const totalCost = useMemo(() => aggregateCost(plan.root), [plan.root]);
  const nodeCount = useMemo(() => countNodes(plan.root), [plan.root]);
  const dataSources = useMemo(() => Array.from(collectDataSources(plan.root)), [plan.root]);
  const warnings = plan.warnings || [];

  const isHighComplexity =
    totalCost > GOVERNANCE_THRESHOLDS.maxPlanCost ||
    plan.metrics.totalLatencyMs > GOVERNANCE_THRESHOLDS.maxLatencyMs ||
    plan.metrics.dataScannedBytes > GOVERNANCE_THRESHOLDS.maxDataScannedBytes;

  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexWrap: 'wrap',
          gap: 1,
          mb: 1,
        }}
      >
        <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
          Execution Governance
        </Typography>
        <Stack direction="row" spacing={1}>
          {isHighComplexity && (
            <Chip
              icon={<WarningIcon />}
              label="High Complexity"
              color="warning"
              size="small"
            />
          )}
          {plan.root.isSecured && (
            <Chip
              icon={<SecurityIcon />}
              label="Tenant Isolated"
              color="success"
              size="small"
              variant="outlined"
            />
          )}
          {dataSources.map((source) => (
            <Chip
              key={source}
              icon={<DialectIcon dialect={source} size="small" />}
              label={source}
              size="small"
              variant="outlined"
            />
          ))}
        </Stack>
      </Box>

      <Divider sx={{ my: 1.5 }} />

      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        spacing={3}
        sx={{ mb: 2 }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <FunctionsIcon fontSize="small" color="action" />
          <Box>
            <Typography variant="caption" color="text.secondary" display="block">
              Total Cost
            </Typography>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>
              {totalCost.toFixed(2)}
            </Typography>
          </Box>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <SpeedIcon fontSize="small" color="action" />
          <Box>
            <Typography variant="caption" color="text.secondary" display="block">
              Est. Latency
            </Typography>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>
              {plan.metrics.totalLatencyMs} ms
            </Typography>
          </Box>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <StorageIcon fontSize="small" color="action" />
          <Box>
            <Typography variant="caption" color="text.secondary" display="block">
              Data Scanned
            </Typography>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>
              {humanReadableBytes(plan.metrics.dataScannedBytes)}
            </Typography>
          </Box>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box>
            <Typography variant="caption" color="text.secondary" display="block">
              Plan Nodes
            </Typography>
            <Typography variant="body2" sx={{ fontWeight: 600 }}>
              {nodeCount}
            </Typography>
          </Box>
        </Box>
      </Stack>

      {warnings.length > 0 && (
        <Box sx={{ mt: 1 }}>
          {warnings.map((warning, index) => (
            <Alert
              key={index}
              severity="warning"
              sx={{ mb: 1, '&:last-child': { mb: 0 } }}
              icon={<WarningIcon />}
            >
              {index === 0 && <AlertTitle>Governance Warning</AlertTitle>}
              {warning}
            </Alert>
          ))}
        </Box>
      )}

      {isHighComplexity && warnings.length === 0 && (
        <Alert severity="info" sx={{ mt: 1 }}>
          Query complexity is elevated. Consider reducing dimensions, adding filters, or
          verifying indexes on the driving table.
        </Alert>
      )}
    </Paper>
  );
};

export default QueryPerformanceSummary;
