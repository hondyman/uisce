import { useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Typography,
  Stack,
  Paper,
  Tabs,
  Tab,
  Alert,
  Stepper,
  Step,
  StepLabel,
  FormControlLabel,
  Checkbox,
  List,
  ListItem,
  ListItemText,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  GitCompare as CompareIcon,
  Check as CheckIcon,
  Schedule as ScheduleIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';

interface RuleVersion {
  version: number;
  createdAt: string;
  createdBy: string;
  status: 'draft' | 'testing' | 'staging' | 'production';
  description: string;
  changes: string;
  approvals?: {
    role: string;
    approved: boolean;
    approver?: string;
    timestamp?: string;
  }[];
}

interface RuleVersionControlProps {
  ruleId: string;
  versions: RuleVersion[];
  currentVersionId: number;
  onVersionSelect: (versionId: number) => void;
  onCompareVersions?: (v1: number, v2: number) => void;
}

/**
 * RuleVersionControl Component (Material-UI)
 * Governance tab for rule versions and promotions
 */
export const RuleVersionControl = ({
  ruleId,
  versions,
  currentVersionId,
  onVersionSelect,
  onCompareVersions,
}: RuleVersionControlProps) => {
  const [compareMode, setCompareMode] = useState(false);
  const [compareV1, setCompareV1] = useState<number | null>(null);
  const [compareV2, setCompareV2] = useState<number | null>(null);

  const statusConfig = {
    draft: { label: 'Draft', color: 'default' as const, description: 'In development' },
    testing: { label: 'Testing', color: 'warning' as const, description: 'Under review' },
    staging: { label: 'Staging', color: 'info' as const, description: 'Ready to promote' },
    production: { label: 'Production', color: 'success' as const, description: 'Live' },
  };

  const promotionPath = ['draft', 'testing', 'staging', 'production'];

  const toggleCompareMode = (version: RuleVersion) => {
    if (!compareMode) {
      setCompareMode(true);
      setCompareV1(version.version);
    } else if (compareV1 === version.version) {
      setCompareMode(false);
      setCompareV1(null);
      setCompareV2(null);
    } else if (!compareV2) {
      setCompareV2(version.version);
      if (onCompareVersions && compareV1 !== null) {
        onCompareVersions(compareV1, version.version);
      }
    } else {
      setCompareV2(version.version);
      if (onCompareVersions && compareV1 !== null) {
        onCompareVersions(compareV1, version.version);
      }
    }
  };

  const getApprovalStatus = (version: RuleVersion) => {
    if (!version.approvals || version.approvals.length === 0) return null;
    const approved = version.approvals.filter((a) => a.approved).length;
    return {
      approved,
      total: version.approvals.length,
    };
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Header */}
      <Paper sx={{ p: 2, borderRadius: 0 }} elevation={0}>
        <Typography variant="subtitle2" fontWeight="600">
          Version History & Approvals
        </Typography>
        <Typography variant="caption" color="textSecondary" sx={{ mt: 0.5, display: 'block' }}>
          Track changes and manage approvals
        </Typography>
      </Paper>

      {/* Promotion Path */}
      <Box sx={{ p: 2, backgroundColor: 'info.lighter', borderBottom: 1, borderColor: 'divider' }}>
        <Typography variant="caption" fontWeight="600" sx={{ mb: 1.5, display: 'block' }}>
          Promotion Workflow
        </Typography>
        <Stepper activeStep={promotionPath.indexOf(versions[0]?.status || 'draft')} sx={{ backgroundColor: 'transparent' }}>
          {promotionPath.map((status, idx) => (
            <Step key={status} completed={idx <= promotionPath.indexOf(versions[0]?.status || 'draft')}>
              <StepLabel>{status.charAt(0).toUpperCase() + status.slice(1)}</StepLabel>
            </Step>
          ))}
        </Stepper>
      </Box>

      {/* Compare Mode Toggle */}
      <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Button
          variant={compareMode ? 'contained' : 'outlined'}
          startIcon={<CompareIcon />}
          onClick={() => {
            setCompareMode(!compareMode);
            if (compareMode) {
              setCompareV1(null);
              setCompareV2(null);
            }
          }}
        >
          {compareMode ? 'Compare Mode' : 'Compare Versions'}
        </Button>
      </Box>

      {/* Content */}
      <Box sx={{ flex: 1, overflowY: 'auto', p: 2 }}>
        {compareMode && (compareV1 || compareV2) ? (
          <Stack spacing={2}>
            <Typography variant="caption" color="textSecondary" fontWeight="600">
              {compareV1 && compareV2
                ? `Comparing Version ${compareV1} with Version ${compareV2}`
                : 'Select two versions to compare'}
            </Typography>

            {compareV1 && compareV2 && (
              <>
                {/* Changes Summary */}
                <Paper sx={{ p: 2 }}>
                  <Typography variant="subtitle2" fontWeight="600" sx={{ mb: 1.5 }}>
                    Changes Summary
                  </Typography>
                  <Stack spacing={1}>
                    <Typography variant="caption">Priority steps changed: 2 modified</Typography>
                    <Typography variant="caption">Confidence thresholds: 1 increased to 80%</Typography>
                    <Typography variant="caption">New conditions: 1 added (Region filtering)</Typography>
                  </Stack>
                </Paper>

                {/* Side-by-side */}
                <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
                  {[
                    { title: `Version ${compareV1}`, version: compareV1 },
                    { title: `Version ${compareV2}`, version: compareV2 },
                  ].map((col) => (
                    <Card key={col.version}>
                      <CardContent>
                        <Typography variant="subtitle2" fontWeight="600" sx={{ mb: 1 }}>
                          {col.title}
                        </Typography>
                        <Chip label="Staging" size="small" color="info" sx={{ mb: 1.5 }} />
                        <Stack spacing={0.5} variant="caption">
                          <Typography variant="caption">Priority 1: IsBusinessDay = true</Typography>
                          <Typography variant="caption">Confidence: 85%</Typography>
                          <Typography variant="caption" sx={{ fontWeight: 600, mt: 1 }}>
                            Priority 2: RegionCode = US
                          </Typography>
                          <Typography variant="caption">Confidence: 75%</Typography>
                        </Stack>
                      </CardContent>
                    </Card>
                  ))}
                </Box>

                {/* Impact */}
                <Alert severity="warning">
                  Changing priorities may affect 127 calendar dates (2.3% change rate)
                </Alert>
              </>
            )}

            {/* Version Selection */}
            <Box sx={{ pt: 2, borderTop: 1, borderColor: 'divider' }}>
              <Typography variant="caption" fontWeight="600" sx={{ mb: 1, display: 'block' }}>
                Select versions to compare:
              </Typography>
              <Stack spacing={1}>
                {versions.slice(0, 5).map((version) => (
                  <Button
                    key={version.version}
                    onClick={() => toggleCompareMode(version)}
                    variant={
                      compareV1 === version.version || compareV2 === version.version
                        ? 'contained'
                        : 'outlined'
                    }
                    sx={{ justifyContent: 'flex-start', textTransform: 'none' }}
                  >
                    <Typography variant="body2" fontWeight="500">
                      v{version.version}
                    </Typography>
                    <Typography variant="caption" sx={{ ml: 1 }}>
                      {version.description}
                    </Typography>
                  </Button>
                ))}
              </Stack>
            </Box>
          </Stack>
        ) : (
          <Stack spacing={1.5}>
            {versions.map((version) => {
              const approvalStatus = getApprovalStatus(version);
              const config = statusConfig[version.status];

              return (
                <Accordion key={version.version} defaultExpanded={version.version === versions[0]?.version}>
                  <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, width: '100%' }}>
                      <Typography variant="subtitle2" fontWeight="600">
                        v{version.version}
                      </Typography>
                      <Chip label={config.label} color={config.color} size="small" />
                      {currentVersionId === version.version && (
                        <Chip label="Current" color="success" size="small" icon={<CheckIcon />} />
                      )}
                      {approvalStatus && version.status !== 'draft' && (
                        <Box sx={{ ml: 'auto', display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Chip
                            label={`${approvalStatus.approved}/${approvalStatus.total}`}
                            color="success"
                            size="small"
                          />
                        </Box>
                      )}
                    </Box>
                  </AccordionSummary>
                  <AccordionDetails>
                    <Stack spacing={2}>
                      {/* Description */}
                      <Box>
                        <Typography variant="caption" fontWeight="600" sx={{ mb: 1, display: 'block' }}>
                          Description
                        </Typography>
                        <Typography variant="caption">{version.description}</Typography>
                      </Box>

                      {/* Metadata */}
                      <Box>
                        <Typography variant="caption" fontWeight="600" sx={{ mb: 1, display: 'block' }}>
                          Metadata
                        </Typography>
                        <Stack spacing={0.5}>
                          <Typography variant="caption">Created: {version.createdAt}</Typography>
                          <Typography variant="caption">By: {version.createdBy}</Typography>
                        </Stack>
                      </Box>

                      {/* Changes */}
                      <Box>
                        <Typography variant="caption" fontWeight="600" sx={{ mb: 1, display: 'block' }}>
                          Changes
                        </Typography>
                        <Typography variant="caption">{version.changes}</Typography>
                      </Box>

                      {/* Approvals */}
                      {version.approvals && version.approvals.length > 0 && (
                        <Box>
                          <Typography variant="caption" fontWeight="600" sx={{ mb: 1, display: 'block' }}>
                            Approvals
                          </Typography>
                          <Stack spacing={1}>
                            {version.approvals.map((approval, idx) => (
                              <Paper
                                key={idx}
                                sx={{
                                  p: 1.5,
                                  backgroundColor: approval.approved ? 'success.lighter' : 'warning.lighter',
                                  display: 'flex',
                                  justifyContent: 'space-between',
                                  alignItems: 'center',
                                }}
                              >
                                <Typography variant="caption" fontWeight="600">
                                  {approval.role}
                                </Typography>
                                {approval.approved ? (
                                  <Typography variant="caption" color="success.main" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                    <CheckIcon sx={{ fontSize: '1rem' }} />
                                    {approval.approver}
                                  </Typography>
                                ) : (
                                  <Typography variant="caption" color="warning.main" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                    <ScheduleIcon sx={{ fontSize: '1rem' }} />
                                    Pending
                                  </Typography>
                                )}
                              </Paper>
                            ))}
                          </Stack>
                        </Box>
                      )}

                      {/* Actions */}
                      <Box sx={{ display: 'flex', gap: 1, pt: 1, borderTop: 1, borderColor: 'divider' }}>
                        <Button
                          size="small"
                          variant="outlined"
                          startIcon={<CompareIcon />}
                          onClick={() => toggleCompareMode(version)}
                        >
                          Compare
                        </Button>
                        {currentVersionId !== version.version && (
                          <Button size="small" variant="outlined" color="warning">
                            Rollback
                          </Button>
                        )}
                      </Box>
                    </Stack>
                  </AccordionDetails>
                </Accordion>
              );
            })}
          </Stack>
        )}
      </Box>

      {/* Footer */}
      {versions[0]?.status === 'draft' && (
        <Box sx={{ borderTop: 1, borderColor: 'divider', p: 2, backgroundColor: 'info.lighter' }}>
          <Typography variant="caption" fontWeight="600" sx={{ mb: 1, display: 'block' }}>
            Ready to test?
          </Typography>
          <Button variant="contained" color="primary" fullWidth>
            Request Testing Approval
          </Button>
        </Box>
      )}
    </Box>
  );
};

export default RuleVersionControl;
