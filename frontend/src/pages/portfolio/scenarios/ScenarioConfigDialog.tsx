/**
 * ScenarioConfigDialog Component
 * 
 * Interactive modal for configuring stress test scenarios
 * with real-time slider inputs and portfolio selection.
 * 
 * Features:
 * - ✅ Material UI Dialog with form validation
 * - ✅ Responsive sliders for market factors
 * - ✅ Portfolio selection toggle
 * - ✅ Dark mode support
 * - ✅ Full TypeScript typing
 * - ✅ Error state handling
 * 
 * @example
 * <ScenarioConfigDialog
 *   open={open}
 *   onClose={handleClose}
 *   onSubmit={handleCreate}
 *   portfolios={portfolioList}
 * />
 */

import React, { useState, useCallback } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Box,
  TextField,
  Slider,
  Typography,
  Button,
  ToggleButton,
  ToggleButtonGroup,
  FormHelperText,
  Alert,
  Divider,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import { StressScenario, ScenarioFormErrors, SCENARIO_CONFIG_CONSTRAINTS } from '../../../types/scenarios';

interface ScenarioConfigDialogProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (scenario: StressScenario) => Promise<void>;
  isLoading?: boolean;
  portfolios?: Array<{ id: string; name: string; aum: number }>;
}

/**
 * Default values for a new scenario
 */
const DEFAULT_SCENARIO: Partial<StressScenario> = {
  name: '',
  description: '',
  equityMarketMove: 0,
  interestRateShift: 0,
  volatilityChange: 0,
  creditSpreadWidening: 0,
  scope: 'all-portfolios',
};

export const ScenarioConfigDialog: React.FC<ScenarioConfigDialogProps> = ({
  open,
  onClose,
  onSubmit,
  isLoading = false,
  portfolios = [],
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  // Form state
  const [formData, setFormData] = useState<Partial<StressScenario>>(DEFAULT_SCENARIO);
  const [errors, setErrors] = useState<ScenarioFormErrors>({});
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [selectedPortfolios, setSelectedPortfolios] = useState<string[]>(
    portfolios.map((p) => p.id)
  );

  /**
   * Validate form before submission
   */
  const validateForm = useCallback((): boolean => {
    const newErrors: ScenarioFormErrors = {};

    if (!formData.name?.trim()) {
      newErrors.scenarioName = 'Scenario name is required';
    }

    if (formData.name && formData.name.length > 100) {
      newErrors.scenarioName = 'Scenario name must be less than 100 characters';
    }

    if (formData.scope === 'selected' && selectedPortfolios.length === 0) {
      newErrors.portfoliosIncluded = 'At least one portfolio must be selected';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [formData.name, formData.scope, selectedPortfolios]);

  /**
   * Handle form submission
   */
  const handleSubmit = useCallback(async () => {
    if (!validateForm()) {
      return;
    }

    try {
      setSubmitError(null);

      const scenario: StressScenario = {
        id: `scenario_${Date.now()}`,
        name: formData.name || '',
        description: formData.description,
        equityMarketMove: formData.equityMarketMove || 0,
        interestRateShift: formData.interestRateShift || 0,
        volatilityChange: formData.volatilityChange || 0,
        creditSpreadWidening: formData.creditSpreadWidening || 0,
        portfoliosIncluded:
          formData.scope === 'all-portfolios'
            ? portfolios.map((p) => p.id)
            : selectedPortfolios,
        scope: formData.scope as 'all-portfolios' | 'selected' | 'comparison-pair',
        createdAt: new Date(),
        createdBy: 'current-user', // TODO: Get from auth context
        isHistorical: false,
      };

      await onSubmit(scenario);

      // Reset form on success
      setFormData(DEFAULT_SCENARIO);
      setSelectedPortfolios(portfolios.map((p) => p.id));
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create scenario';
      setSubmitError(message);
    }
  }, [validateForm, formData, portfolios, selectedPortfolios, onSubmit, onClose]);

  /**
   * Handle dialog close (with cleanup)
   */
  const handleClose = useCallback(() => {
    if (!isLoading) {
      setFormData(DEFAULT_SCENARIO);
      setErrors({});
      setSubmitError(null);
      setSelectedPortfolios(portfolios.map((p) => p.id));
      onClose();
    }
  }, [isLoading, portfolios, onClose]);

  const constraint = SCENARIO_CONFIG_CONSTRAINTS;

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth={isMobile}>
      {/* Header */}
      <DialogTitle sx={{ pb: 1 }}>
        <Typography variant="h6" sx={{ fontWeight: 600 }}>
          Configure Stress Test Scenario
        </Typography>
        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
          Define market stress parameters for stress testing
        </Typography>
      </DialogTitle>

      <Divider />

      {/* Content */}
      <DialogContent sx={{ py: 3 }}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
          {/* Error Alert */}
          {submitError && (
            <Alert severity="error" onClose={() => setSubmitError(null)}>
              {submitError}
            </Alert>
          )}

          {/* Scenario Name */}
          <Box>
            <TextField
              fullWidth
              label="Scenario Name"
              placeholder="e.g., 2008 Financial Crisis Simulation"
              value={formData.name || ''}
              onChange={(e) =>
                setFormData((prev) => ({ ...prev, name: e.target.value }))
              }
              error={!!errors.scenarioName}
              helperText={errors.scenarioName}
              disabled={isLoading}
              size="small"
            />
          </Box>

          {/* Description (optional) */}
          <Box>
            <TextField
              fullWidth
              label="Description (Optional)"
              placeholder="Add details about this scenario..."
              multiline
              rows={2}
              value={formData.description || ''}
              onChange={(e) =>
                setFormData((prev) => ({ ...prev, description: e.target.value }))
              }
              disabled={isLoading}
              size="small"
            />
          </Box>

          <Divider sx={{ my: 1 }} />

          {/* Market Factors Section */}
          <Typography variant="subtitle2" sx={{ fontWeight: 600, mt: 2 }}>
            Market Factors
          </Typography>

          {/* Equity Market Move */}
          <Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="body2" sx={{ fontWeight: 500 }}>
                Equity Market Move
              </Typography>
              <Typography
                variant="body2"
                sx={{
                  fontWeight: 600,
                  color: formData.equityMarketMove! < 0 ? 'error.main' : 'success.main',
                }}
              >
                {formData.equityMarketMove}%
              </Typography>
            </Box>
            <Slider
              value={formData.equityMarketMove || 0}
              onChange={(e, value) =>
                setFormData((prev) => ({
                  ...prev,
                  equityMarketMove: typeof value === 'number' ? value : value[0],
                }))
              }
              min={constraint.equityMarketMove.min}
              max={constraint.equityMarketMove.max}
              step={constraint.equityMarketMove.step}
              marks={[
                { value: -100, label: '-100%' },
                { value: 0, label: '0%' },
                { value: 100, label: '100%' },
              ]}
              disabled={isLoading}
              sx={{ mt: 2 }}
            />
            <FormHelperText sx={{ mt: 1, fontSize: '0.75rem' }}>
              Range: {constraint.equityMarketMove.min}% to {constraint.equityMarketMove.max}%
            </FormHelperText>
          </Box>

          {/* Interest Rate Shift */}
          <Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="body2" sx={{ fontWeight: 500 }}>
                Interest Rate Shift
              </Typography>
              <Typography
                variant="body2"
                sx={{
                  fontWeight: 600,
                  color: formData.interestRateShift! > 0 ? 'error.main' : 'success.main',
                }}
              >
                {formData.interestRateShift} bps
              </Typography>
            </Box>
            <Slider
              value={formData.interestRateShift || 0}
              onChange={(e, value) =>
                setFormData((prev) => ({
                  ...prev,
                  interestRateShift: typeof value === 'number' ? value : value[0],
                }))
              }
              min={constraint.interestRateShift.min}
              max={constraint.interestRateShift.max}
              step={constraint.interestRateShift.step}
              marks={[
                { value: -500, label: '-500 bps' },
                { value: 0, label: '0' },
                { value: 500, label: '500 bps' },
              ]}
              disabled={isLoading}
              sx={{ mt: 2 }}
            />
            <FormHelperText sx={{ mt: 1, fontSize: '0.75rem' }}>
              Range: {constraint.interestRateShift.min} to {constraint.interestRateShift.max} basis
              points
            </FormHelperText>
          </Box>

          {/* Volatility Change */}
          <Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="body2" sx={{ fontWeight: 500 }}>
                Volatility (VIX)
              </Typography>
              <Typography variant="body2" sx={{ fontWeight: 600, color: 'warning.main' }}>
                {formData.volatilityChange}%
              </Typography>
            </Box>
            <Slider
              value={formData.volatilityChange || 0}
              onChange={(e, value) =>
                setFormData((prev) => ({
                  ...prev,
                  volatilityChange: typeof value === 'number' ? value : value[0],
                }))
              }
              min={constraint.volatilityChange.min}
              max={constraint.volatilityChange.max}
              step={constraint.volatilityChange.step}
              marks={[
                { value: -100, label: '-100%' },
                { value: 0, label: '0%' },
                { value: 200, label: '200%' },
              ]}
              disabled={isLoading}
              sx={{ mt: 2 }}
            />
            <FormHelperText sx={{ mt: 1, fontSize: '0.75rem' }}>
              Range: {constraint.volatilityChange.min}% to {constraint.volatilityChange.max}%
            </FormHelperText>
          </Box>

          {/* Credit Spread Widening */}
          <Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="body2" sx={{ fontWeight: 500 }}>
                Credit Spread Widening
              </Typography>
              <Typography variant="body2" sx={{ fontWeight: 600, color: 'error.main' }}>
                {formData.creditSpreadWidening} bps
              </Typography>
            </Box>
            <Slider
              value={formData.creditSpreadWidening || 0}
              onChange={(e, value) =>
                setFormData((prev) => ({
                  ...prev,
                  creditSpreadWidening: typeof value === 'number' ? value : value[0],
                }))
              }
              min={constraint.creditSpreadWidening.min}
              max={constraint.creditSpreadWidening.max}
              step={constraint.creditSpreadWidening.step}
              marks={[
                { value: -100, label: '-100 bps' },
                { value: 0, label: '0' },
                { value: 500, label: '500 bps' },
              ]}
              disabled={isLoading}
              sx={{ mt: 2 }}
            />
            <FormHelperText sx={{ mt: 1, fontSize: '0.75rem' }}>
              Range: {constraint.creditSpreadWidening.min} to {constraint.creditSpreadWidening.max}{' '}
              basis points
            </FormHelperText>
          </Box>

          <Divider sx={{ my: 2 }} />

          {/* Portfolio Selection */}
          <Box>
            <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 2 }}>
              Portfolio Selection
            </Typography>
            <ToggleButtonGroup
              exclusive
              value={formData.scope || 'all-portfolios'}
              onChange={(e, newValue) => {
                if (newValue) {
                  setFormData((prev) => ({
                    ...prev,
                    scope: newValue as 'all-portfolios' | 'selected' | 'comparison-pair',
                  }));
                }
              }}
              fullWidth
              disabled={isLoading}
            >
              <ToggleButton value="all-portfolios">
                <Typography variant="caption" sx={{ fontWeight: 500 }}>
                  All Portfolios
                </Typography>
              </ToggleButton>
              <ToggleButton value="selected">
                <Typography variant="caption" sx={{ fontWeight: 500 }}>
                  Selected Only
                </Typography>
              </ToggleButton>
            </ToggleButtonGroup>

            {errors.portfoliosIncluded && (
              <FormHelperText error sx={{ mt: 1 }}>
                {errors.portfoliosIncluded}
              </FormHelperText>
            )}

            {formData.scope === 'selected' && portfolios.length > 0 && (
              <Box
                sx={{
                  mt: 2,
                  p: 2,
                  border: '1px solid',
                  borderColor: 'divider',
                  borderRadius: 1,
                  maxHeight: 200,
                  overflow: 'auto',
                }}
              >
                {portfolios.map((portfolio) => (
                  <Box
                    key={portfolio.id}
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      p: 1,
                      cursor: 'pointer',
                      '&:hover': { bgcolor: 'action.hover' },
                    }}
                    onClick={() => {
                      setSelectedPortfolios((prev) =>
                        prev.includes(portfolio.id)
                          ? prev.filter((id) => id !== portfolio.id)
                          : [...prev, portfolio.id]
                      );
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={selectedPortfolios.includes(portfolio.id)}
                      readOnly
                      style={{ marginRight: 8 }}
                    />
                    <Box>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {portfolio.name}
                      </Typography>
                      <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        AUM: ${portfolio.aum.toFixed(1)}M
                      </Typography>
                    </Box>
                  </Box>
                ))}
              </Box>
            )}
          </Box>
        </Box>
      </DialogContent>

      <Divider />

      {/* Actions */}
      <DialogActions sx={{ p: 2 }}>
        <Button onClick={handleClose} disabled={isLoading}>
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={isLoading}
          sx={{
            textTransform: 'uppercase',
            fontWeight: 600,
            fontSize: '0.75rem',
            letterSpacing: 0.5,
          }}
        >
          {isLoading ? 'Creating...' : 'Run Simulation'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default ScenarioConfigDialog;
