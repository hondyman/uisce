import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Button,
  Container,
  Paper,
  Step,
  StepLabel,
  Stepper,
  Typography,
  Stack,
  Alert,
} from '@mui/material';
import {
  ArrowBack as BackIcon,
  ArrowForward as NextIcon,
  Save as SaveIcon,
} from '@mui/icons-material';
import { accessRulesApi, AccessRuleInput } from '../../../api/accessRules';
import { WhoStep } from '../components/wizard/WhoStep';
import { WhatStep } from '../components/wizard/WhatStep';
import { AccessLevelStep } from '../components/wizard/AccessLevelStep';
import { RowFilterStep } from '../components/wizard/RowFilterStep';
import { ColumnMaskStep } from '../components/wizard/ColumnMaskStep';
import { ReviewStep } from '../components/wizard/ReviewStep';

const steps = [
  { label: 'Who', description: 'Select team or user group' },
  { label: 'What', description: 'Choose data type' },
  { label: 'Access Level', description: 'Set permissions' },
  { label: 'Row Filters', description: 'Optional data filtering' },
  { label: 'Field Security', description: 'Optional field masking' },
  { label: 'Review', description: 'Confirm and save' },
];

export const AccessRuleWizardPage: React.FC = () => {
  const navigate = useNavigate();
  const [activeStep, setActiveStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [ruleData, setRuleData] = useState<AccessRuleInput>({
    ruleId: '',
    tenantId: '',
    businessObjectId: '',
    groupDn: '',
    accessLevel: 'READ',
    status: 'DRAFT',
    rowFilterDsl: '',
    columnMasks: [],
    scope: { appliesToApis: true, appliesToBi: true, appliesToAi: true },
  });

  const handleNext = () => {
    setError(null);
    setActiveStep((prev) => prev + 1);
  };

  const handleBack = () => {
    setError(null);
    setActiveStep((prev) => prev - 1);
  };

  const handleSave = async () => {
    setLoading(true);
    setError(null);
    try {
      await accessRulesApi.create(ruleData);
      navigate('/security/access-rules');
    } catch (e: any) {
      setError(e?.message || 'Failed to create access rule');
    } finally {
      setLoading(false);
    }
  };

  const updateRuleData = (updates: Partial<AccessRuleInput>) => {
    setRuleData((prev) => ({ ...prev, ...updates }));
  };

  const canProceed = () => {
    switch (activeStep) {
      case 0: // Who
        return !!ruleData.groupDn;
      case 1: // What
        return !!ruleData.businessObjectId;
      case 2: // Access Level
        return !!ruleData.accessLevel;
      case 3: // Row Filters (optional)
        return true;
      case 4: // Column Masks (optional)
        return true;
      case 5: // Review
        return true;
      default:
        return false;
    }
  };

  const renderStepContent = () => {
    switch (activeStep) {
      case 0:
        return <WhoStep ruleData={ruleData} updateRuleData={updateRuleData} />;
      case 1:
        return <WhatStep ruleData={ruleData} updateRuleData={updateRuleData} />;
      case 2:
        return <AccessLevelStep ruleData={ruleData} updateRuleData={updateRuleData} />;
      case 3:
        return <RowFilterStep ruleData={ruleData} updateRuleData={updateRuleData} />;
      case 4:
        return <ColumnMaskStep ruleData={ruleData} updateRuleData={updateRuleData} />;
      case 5:
        return <ReviewStep ruleData={ruleData} />;
      default:
        return null;
    }
  };

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      <Paper elevation={3} sx={{ p: 4 }}>
        {/* Header */}
        <Box sx={{ mb: 4 }}>
          <Typography variant="h4" sx={{ fontWeight: 700, mb: 1 }}>
            Create Access Rule
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Follow the steps below to create a new data access rule
          </Typography>
        </Box>

        {/* Stepper */}
        <Stepper activeStep={activeStep} sx={{ mb: 4 }}>
          {steps.map((step, index) => (
            <Step key={step.label}>
              <StepLabel>
                <Typography variant="body2" sx={{ fontWeight: activeStep === index ? 700 : 400 }}>
                  {step.label}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {step.description}
                </Typography>
              </StepLabel>
            </Step>
          ))}
        </Stepper>

        {/* Error Alert */}
        {error && (
          <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        {/* Step Content */}
        <Box sx={{ minHeight: 400, mb: 4 }}>
          {renderStepContent()}
        </Box>

        {/* Navigation Buttons */}
        <Stack direction="row" spacing={2} justifyContent="space-between">
          <Button
            variant="outlined"
            onClick={() => navigate('/security/access-rules')}
            disabled={loading}
          >
            Cancel
          </Button>
          <Stack direction="row" spacing={2}>
            <Button
              variant="outlined"
              startIcon={<BackIcon />}
              onClick={handleBack}
              disabled={activeStep === 0 || loading}
            >
              Back
            </Button>
            {activeStep < steps.length - 1 ? (
              <Button
                variant="contained"
                endIcon={<NextIcon />}
                onClick={handleNext}
                disabled={!canProceed() || loading}
              >
                Next
              </Button>
            ) : (
              <Button
                variant="contained"
                color="success"
                startIcon={<SaveIcon />}
                onClick={handleSave}
                disabled={!canProceed() || loading}
              >
                {loading ? 'Saving...' : 'Create Rule'}
              </Button>
            )}
          </Stack>
        </Stack>
      </Paper>
    </Container>
  );
};

export default AccessRuleWizardPage;
