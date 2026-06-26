import React, { useState } from 'react';
import {
  Box,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Checkbox,
  Button,
  Chip,
  Typography,
  LinearProgress,
  IconButton,
  Collapse,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import {
  KeyboardArrowDown,
  KeyboardArrowUp,
  CompareArrows,
  Visibility,
  Download,
} from '@mui/icons-material';

interface Scenario {
  scenario_id: string;
  scenario_name: string;
  strategy_type: string;
  tax_savings: number;
  tax_savings_pct: number;
  complexity_score: number;
  implementation_weeks: number;
  annual_cost: number;
  overall_score: number;
  narrative_explanation?: string;
  structures_used: string[];
}

interface ScenarioComparisonTableProps {
  scenarios: Scenario[];
  familyId: string;
}

export const ScenarioComparisonTable: React.FC<ScenarioComparisonTableProps> = ({ scenarios, familyId }) => {
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [expandedRow, setExpandedRow] = useState<string | null>(null);
  const [narrativeDialog, setNarrativeDialog] = useState<Scenario | null>(null);

  const handleSelectAll = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      setSelected(new Set(scenarios.map(s => s.scenario_id)));
    } else {
      setSelected(new Set());
    }
  };

  const handleSelect = (id: string) => {
    const newSelected = new Set(selected);
    if (newSelected.has(id)) {
      newSelected.delete(id);
    } else {
      newSelected.add(id);
    }
    setSelected(newSelected);
  };

  const handleToggleRow = (id: string) => {
    setExpandedRow(expandedRow === id ? null : id);
  };

  const getComplexityColor = (score: number): 'success' | 'warning' | 'error' => {
    if (score <= 3) return 'success';
    if (score <=  6) return 'warning';
    return 'error';
  };

  const getComplexityLabel = (score: number): string => {
    if (score <= 3) return 'Simple';
    if (score <= 6) return 'Moderate';
    return 'Complex';
  };

  const formatCurrency = (value: number): string => {
    return `$${(value / 1000000).toFixed(1)}M`;
  };

  const selectedScenarios = scenarios.filter(s => selected.has(s.scenario_id));

  return (
    <Box>
      {/* Header Actions */}
      <Box sx={{ mb: 3, display: 'flex', gap: 2, alignItems: 'center' }}>
        <Typography variant="h6">Estate Planning Scenarios</Typography>
        <Box sx={{ flexGrow: 1 }} />
        {selected.size > 0 && (
          <>
            <Chip label={`${selected.size} selected`} color="primary" />
            <Button
              variant="outlined"
              startIcon={<CompareArrows />}
              disabled={selected.size < 2}
            >
              Compare Selected
            </Button>
            <Button
              variant="outlined"
              startIcon={<Download />}
            >
              Export PDF
            </Button>
          </>
        )}
      </Box>

      {scenarios.length === 0 ? (
        <Alert severity="info">
          No scenarios generated yet. Click "Generate Estate Plan" to create optimized scenarios.
        </Alert>
      ) : (
        <TableContainer component={Paper} elevation={2}>
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: 'grey.100' }}>
                <TableCell padding="checkbox">
                  <Checkbox
                    indeterminate={selected.size > 0 && selected.size < scenarios.length}
                    checked={scenarios.length > 0 && selected.size === scenarios.length}
                    onChange={handleSelectAll}
                  />
                </TableCell>
                <TableCell />
                <TableCell><strong>Strategy</strong></TableCell>
                <TableCell align="right"><strong>Tax Savings</strong></TableCell>
                <TableCell align="right"><strong>Savings %</strong></TableCell>
                <TableCell><strong>Complexity</strong></TableCell>
                <TableCell align="right"><strong>Implementation</strong></TableCell>
                <TableCell align="right"><strong>Annual Cost</strong></TableCell>
                <TableCell align="right"><strong>Score</strong></TableCell>
                <TableCell align="center"><strong>Actions</strong></TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {scenarios.map((scenario, index) => (
                <React.Fragment key={scenario.scenario_id}>
                  <TableRow
                    hover
                    sx={{
                      bgcolor: index === 0 ? 'success.50' : 'inherit',
                      borderLeft: index === 0 ? 3 : 0,
                      borderColor: 'success.main',
                    }}
                  >
                    <TableCell padding="checkbox">
                      <Checkbox
                        checked={selected.has(scenario.scenario_id)}
                        onChange={() => handleSelect(scenario.scenario_id)}
                      />
                    </TableCell>
                    <TableCell>
                      <IconButton
                        size="small"
                        onClick={() => handleToggleRow(scenario.scenario_id)}
                      >
                        {expandedRow === scenario.scenario_id ? <KeyboardArrowUp /> : <KeyboardArrowDown />}
                      </IconButton>
                    </TableCell>
                    <TableCell>
                      <Box>
                        <Typography variant="body2" fontWeight={index === 0 ? 'bold' : 'normal'}>
                          {scenario.scenario_name}
                          {index === 0 && <Chip label="Recommended" color="success" size="small" sx={{ ml: 1 }} />}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {scenario.strategy_type}
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2" color="success.main" fontWeight="medium">
                        {formatCurrency(scenario.tax_savings)}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {scenario.tax_savings_pct.toFixed(1)}%
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={`${scenario.complexity_score}/10 - ${getComplexityLabel(scenario.complexity_score)}`}
                        color={getComplexityColor(scenario.complexity_score)}
                        size="small"
                      />
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {scenario.implementation_weeks} weeks
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        ${(scenario.annual_cost / 1000).toFixed(0)}K/yr
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Box sx={{ minWidth: 60 }}>
                        <Typography variant="caption" display="block">
                          {(scenario.overall_score * 100).toFixed(0)}
                        </Typography>
                        <LinearProgress
                          variant="determinate"
                          value={scenario.overall_score * 100}
                          color={scenario.overall_score > 0.7 ? 'success' : 'primary'}
                          sx={{ mt: 0.5 }}
                        />
                      </Box>
                    </TableCell>
                    <TableCell align="center">
                      <IconButton
                        size="small"
                        onClick={() => setNarrativeDialog(scenario)}
                        disabled={!scenario.narrative_explanation}
                      >
                        <Visibility fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>

                  {/* Expanded Row */}
                  <TableRow>
                    <TableCell style={{ paddingBottom: 0, paddingTop: 0 }} colSpan={10}>
                      <Collapse in={expandedRow === scenario.scenario_id} timeout="auto" unmountOnExit>
                        <Box sx={{ py: 2, px: 3, bgcolor: 'grey.50' }}>
                          <Typography variant="subtitle2" gutterBottom>
                            Structures Used:
                          </Typography>
                          <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                            {scenario.structures_used.map((structure, i) => (
                              <Chip key={i} label={structure} size="small" variant="outlined" />
                            ))}
                          </Box>

                          {scenario.narrative_explanation && (
                            <Box sx={{ mt: 2 }}>
                              <Typography variant="subtitle2" gutterBottom>
                                Summary:
                              </Typography>
                              <Typography variant="body2" color="text.secondary" sx={{ whiteSpace: 'pre-line' }}>
                                {scenario.narrative_explanation.substring(0, 300)}...
                              </Typography>
                              <Button size="small" sx={{ mt: 1 }} onClick={() => setNarrativeDialog(scenario)}>
                                Read Full Explanation
                              </Button>
                            </Box>
                          )}
                        </Box>
                      </Collapse>
                    </TableCell>
                  </TableRow>
                </React.Fragment>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Narrative Dialog */}
      <Dialog
        open={narrativeDialog !== null}
        onClose={() => setNarrativeDialog(null)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          {narrativeDialog?.scenario_name} - Detailed Explanation
        </DialogTitle>
        <DialogContent dividers>
          <Typography variant="body1" sx={{ whiteSpace: 'pre-line', lineHeight: 1.8 }}>
            {narrativeDialog?.narrative_explanation}
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setNarrativeDialog(null)}>Close</Button>
          <Button variant="contained" startIcon={<Download />}>
            Download PDF
          </Button>
        </DialogActions>
      </Dialog>

      {/* Comparison Summary (when multiple selected) */}
      {selectedScenarios.length >= 2 && (
        <Paper elevation={3} sx={{ mt: 3, p: 3 }}>
          <Typography variant="h6" gutterBottom>
            Comparison Summary
          </Typography>
          <Box sx={{ display: 'flex', gap: 4 }}>
            <Box>
              <Typography variant="caption" color="text.secondary">
                Best Tax Savings
              </Typography>
              <Typography variant="h6" color="success.main">
                {formatCurrency(Math.max(...selectedScenarios.map(s => s.tax_savings)))}
              </Typography>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary">
                Simplest Option
              </Typography>
              <Typography variant="h6">
                {Math.min(...selectedScenarios.map(s => s.complexity_score))}/10
              </Typography>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary">
                Lowest Annual Cost
              </Typography>
              <Typography variant="h6">
                ${(Math.min(...selectedScenarios.map(s => s.annual_cost)) / 1000).toFixed(0)}K/yr
              </Typography>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary">
                Fastest Implementation
              </Typography>
              <Typography variant="h6">
                {Math.min(...selectedScenarios.map(s => s.implementation_weeks))} weeks
              </Typography>
            </Box>
          </Box>
        </Paper>
      )}
    </Box>
  );
};
