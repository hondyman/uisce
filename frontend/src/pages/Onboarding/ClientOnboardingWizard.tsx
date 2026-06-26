import React, { useState, useEffect } from 'react';
import {
  Box,
  Stepper,
  Step,
  StepLabel,
  Button,
  Paper,
  Typography,
  TextField,
  Grid,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Radio,
  RadioGroup,
  FormControlLabel,
  FormLabel,
  Slider,
  Alert,
  LinearProgress,
  Chip,
  IconButton,
} from '@mui/material';
import {
  CloudUpload as UploadIcon,
  Delete as DeleteIcon,
  CheckCircle as CheckIcon,
} from '@mui/icons-material';
import { useDropzone } from 'react-dropzone';
import { format } from 'date-fns';
import { useTenant } from '../../contexts/TenantContext';
import { getSelectedRegion } from '../../lib/region';

const steps = [
  'Personal Information',
  'Financial Goals',
  'Risk Assessment',
  'Document Upload',
  'Account Selection',
];

interface OnboardingData {
  personalInfo: {
    firstName: string;
    lastName: string;
    email: string;
    phone: string;
    dob: string;
    ssn: string;
    address: {
      street: string;
      city: string;
      state: string;
      zip: string;
    };
  };
  financialGoals: Array<{
    goalType: string;
    targetAmount: number;
    targetDate: string;
    priority: number;
  }>;
  riskAssessment: {
    answers: Record<string, any>;
    riskScore: number;
    riskCategory: string;
  };
  documents: Array<{
    documentId: string;
    documentType: string;
    fileName: string;
    status: string;
  }>;
  accountSelection: {
    accountType: string;
    fundingMethod: string;
    initialInvestment: number;
  };
}

export const ClientOnboardingWizard: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const [activeStep, setActiveStep] = useState(0);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [resumeToken, setResumeToken] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  
  const [data, setData] = useState<OnboardingData>({
    personalInfo: {
      firstName: '',
      lastName: '',
      email: '',
      phone: '',
      dob: '',
      ssn: '',
      address: { street: '', city: '', state: '', zip: '' },
    },
    financialGoals: [],
    riskAssessment: {
      answers: {},
      riskScore: 0,
      riskCategory: '',
    },
    documents: [],
    accountSelection: {
      accountType: '',
      fundingMethod: '',
      initialInvestment: 0,
    },
  });

  // Check for resume token in URL
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get('resume_token');
    if (token) {
      resumeOnboarding(token);
    } else {
      startNewSession();
    }
  }, []);

  const startNewSession = async () => {
    try {
      const response = await fetch('/api/onboarding/sessions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
          'X-Tenant-Region': getSelectedRegion(),
        },
        body: JSON.stringify({ email: data.personalInfo.email }),
      });
      const session = await response.json();
      setSessionId(session.session_id);
      setResumeToken(session.resume_token);
    } catch (error) {
      console.error('Failed to start session:', error);
    }
  };

  const resumeOnboarding = async (token: string) => {
    try {
      const response = await fetch(`/api/onboarding/sessions/resume/${token}`, {
        headers: {
          'X-Tenant-ID': tenant?.id || '',
          'X-Tenant-Datasource-ID': datasource?.id || '',
          'X-Tenant-Region': getSelectedRegion(),
        },
      });
      const session = await response.json();
      setSessionId(session.session_id);
      setResumeToken(session.resume_token);
      setData({
        personalInfo: session.personal_info || data.personalInfo,
        financialGoals: session.financial_goals || [],
        riskAssessment: session.risk_assessment || data.riskAssessment,
        documents: [],
        accountSelection: session.account_selection || data.accountSelection,
      });
      setActiveStep(session.completed_steps?.length || 0);
    } catch (error) {
      console.error('Failed to resume session:', error);
    }
  };

  const saveProgress = async () => {
    if (!sessionId) return;

    setSaving(true);
    try {
      await fetch(`/api/onboarding/sessions/${sessionId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          current_step: steps[activeStep],
          personal_info: data.personalInfo,
          financial_goals: data.financialGoals,
          risk_assessment: data.riskAssessment,
          account_selection: data.accountSelection,
        }),
      });
    } catch (error) {
      console.error('Failed to save progress:', error);
    } finally {
      setSaving(false);
    }
  };

  const validateStep = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (activeStep === 0) {
      if (!data.personalInfo.firstName) newErrors.firstName = 'Required';
      if (!data.personalInfo.lastName) newErrors.lastName = 'Required';
      if (!data.personalInfo.email) newErrors.email = 'Required';
      if (!data.personalInfo.dob) newErrors.dob = 'Required';
      if (!data.personalInfo.ssn || data.personalInfo.ssn.length !== 9) {
        newErrors.ssn = 'Invalid SSN format';
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleNext = async () => {
    if (!validateStep()) return;

    await saveProgress();

    if (activeStep === steps.length - 1) {
      await completeOnboarding();
    } else {
      setActiveStep((prev) => prev + 1);
    }
  };

  const handleBack = () => {
    setActiveStep((prev) => prev - 1);
  };

  const completeOnboarding = async () => {
    if (!sessionId) return;

    try {
      await fetch(`/api/onboarding/sessions/${sessionId}/complete`, {
        method: 'POST',
      });
      // Redirect to success page or dashboard
      window.location.href = '/onboarding/success';
    } catch (error) {
      console.error('Failed to complete onboarding:', error);
    }
  };

  // Render step content
  const renderStepContent = (step: number) => {
    switch (step) {
      case 0:
        return renderPersonalInfo();
      case 1:
        return renderFinancialGoals();
      case 2:
        return renderRiskAssessment();
      case 3:
        return renderDocumentUpload();
      case 4:
        return renderAccountSelection();
      default:
        return null;
    }
  };

  const renderPersonalInfo = () => (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <Typography variant="h6" gutterBottom>
          Tell us about yourself
        </Typography>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          We need this information to comply with federal regulations and open your account.
        </Typography>
      </Grid>

      <Grid item xs={12} sm={6}>
        <TextField
          label="First Name"
          fullWidth
          required
          value={data.personalInfo.firstName}
          onChange={(e) => setData({
            ...data,
            personalInfo: { ...data.personalInfo, firstName: e.target.value },
          })}
          error={!!errors.firstName}
          helperText={errors.firstName}
        />
      </Grid>

      <Grid item xs={12} sm={6}>
        <TextField
          label="Last Name"
          fullWidth
          required
          value={data.personalInfo.lastName}
          onChange={(e) => setData({
            ...data,
            personalInfo: { ...data.personalInfo, lastName: e.target.value },
          })}
          error={!!errors.lastName}
          helperText={errors.lastName}
        />
      </Grid>

      <Grid item xs={12} sm={6}>
        <TextField
          label="Email"
          type="email"
          fullWidth
          required
          value={data.personalInfo.email}
          onChange={(e) => setData({
            ...data,
            personalInfo: { ...data.personalInfo, email: e.target.value },
          })}
          error={!!errors.email}
          helperText={errors.email}
        />
      </Grid>

      <Grid item xs={12} sm={6}>
        <TextField
          label="Phone"
          fullWidth
          value={data.personalInfo.phone}
          onChange={(e) => setData({
            ...data,
            personalInfo: { ...data.personalInfo, phone: e.target.value },
          })}
          placeholder="+1 (555) 123-4567"
        />
      </Grid>

      <Grid item xs={12} sm={6}>
        <TextField
          label="Date of Birth"
          type="date"
          fullWidth
          required
          InputLabelProps={{ shrink: true }}
          value={data.personalInfo.dob}
          onChange={(e) => setData({
            ...data,
            personalInfo: { ...data.personalInfo, dob: e.target.value },
          })}
          error={!!errors.dob}
          helperText={errors.dob}
        />
      </Grid>

      <Grid item xs={12} sm={6}>
        <TextField
          label="Social Security Number"
          fullWidth
          required
          value={data.personalInfo.ssn}
          onChange={(e) => {
            // Only allow digits, max 9
            const cleaned = e.target.value.replace(/\D/g, '').slice(0, 9);
            setData({
              ...data,
              personalInfo: { ...data.personalInfo, ssn: cleaned },
            });
          }}
          placeholder="123456789"
          error={!!errors.ssn}
          helperText={errors.ssn || 'Your SSN is encrypted and secure'}
          inputProps={{ maxLength: 9 }}
        />
      </Grid>

      <Grid item xs={12}>
        <Typography variant="subtitle1" gutterBottom sx={{ mt: 2 }}>
          Address
        </Typography>
      </Grid>

      <Grid item xs={12}>
        <TextField
          label="Street Address"
          fullWidth
          value={data.personalInfo.address.street}
          onChange={(e) => setData({
            ...data,
            personalInfo: {
              ...data.personalInfo,
              address: { ...data.personalInfo.address, street: e.target.value },
            },
          })}
        />
      </Grid>

      <Grid item xs={12} sm={4}>
        <TextField
          label="City"
          fullWidth
          value={data.personalInfo.address.city}
          onChange={(e) => setData({
            ...data,
            personalInfo: {
              ...data.personalInfo,
              address: { ...data.personalInfo.address, city: e.target.value },
            },
          })}
        />
      </Grid>

      <Grid item xs={12} sm={4}>
        <FormControl fullWidth>
          <InputLabel>State</InputLabel>
          <Select
            value={data.personalInfo.address.state}
            onChange={(e) => setData({
              ...data,
              personalInfo: {
                ...data.personalInfo,
                address: { ...data.personalInfo.address, state: e.target.value },
              },
            })}
          >
            <MenuItem value="CA">California</MenuItem>
            <MenuItem value="NY">New York</MenuItem>
            <MenuItem value="TX">Texas</MenuItem>
            {/* Add more states */}
          </Select>
        </FormControl>
      </Grid>

      <Grid item xs={12} sm={4}>
        <TextField
          label="ZIP Code"
          fullWidth
          value={data.personalInfo.address.zip}
          onChange={(e) => {
            const cleaned = e.target.value.replace(/\D/g, '').slice(0, 5);
            setData({
              ...data,
              personalInfo: {
                ...data.personalInfo,
                address: { ...data.personalInfo.address, zip: cleaned },
              },
            });
          }}
          inputProps={{ maxLength: 5 }}
        />
      </Grid>
    </Grid>
  );

  const renderFinancialGoals = () => (
    <Box>
      <Typography variant="h6" gutterBottom>
        What are your financial goals?
      </Typography>
      <Typography variant="body2" color="text.secondary" gutterBottom>
        Help us understand what you're saving for so we can build the right portfolio.
      </Typography>

      <Button
        variant="outlined"
        onClick={() => {
          setData({
            ...data,
            financialGoals: [
              ...data.financialGoals,
              { goalType: '', targetAmount: 0, targetDate: '', priority: data.financialGoals.length + 1 },
            ],
          });
        }}
        sx={{ mt: 2, mb: 3 }}
      >
        + Add Goal
      </Button>

      {data.financialGoals.map((goal, index) => (
        <Paper key={index} sx={{ p: 2, mb: 2 }}>
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth>
                <InputLabel>Goal Type</InputLabel>
                <Select
                  value={goal.goalType}
                  onChange={(e) => {
                    const newGoals = [...data.financialGoals];
                    newGoals[index].goalType = e.target.value;
                    setData({ ...data, financialGoals: newGoals });
                  }}
                >
                  <MenuItem value="RETIREMENT">Retirement</MenuItem>
                  <MenuItem value="EDUCATION">Education</MenuItem>
                  <MenuItem value="HOME_PURCHASE">Home Purchase</MenuItem>
                  <MenuItem value="WEALTH_BUILDING">Wealth Building</MenuItem>
                  <MenuItem value="OTHER">Other</MenuItem>
                </Select>
              </FormControl>
            </Grid>

            <Grid item xs={12} sm={6}>
              <TextField
                label="Target Amount"
                fullWidth
                type="number"
                value={goal.targetAmount}
                onChange={(e) => {
                  const newGoals = [...data.financialGoals];
                  newGoals[index].targetAmount = parseFloat(e.target.value);
                  setData({ ...data, financialGoals: newGoals });
                }}
                InputProps={{ startAdornment: '$' }}
              />
            </Grid>

            <Grid item xs={12} sm={6}>
              <TextField
                label="Target Date"
                type="date"
                fullWidth
                InputLabelProps={{ shrink: true }}
                value={goal.targetDate}
                onChange={(e) => {
                  const newGoals = [...data.financialGoals];
                  newGoals[index].targetDate = e.target.value;
                  setData({ ...data, financialGoals: newGoals });
                }}
              />
            </Grid>

            <Grid item xs={12} sm={6}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Chip label={`Priority ${goal.priority}`} color="primary" size="small" />
                <IconButton
                  size="small"
                  onClick={() => {
                    const newGoals = data.financialGoals.filter((_, i) => i !== index);
                    setData({ ...data, financialGoals: newGoals });
                  }}
                >
                  <DeleteIcon />
                </IconButton>
              </Box>
            </Grid>
          </Grid>
        </Paper>
      ))}
    </Box>
  );

  const renderRiskAssessment = () => (
    <Box>
      <Typography variant="h6" gutterBottom>
        Risk Assessment
      </Typography>
      <Typography variant="body2" color="text.secondary" gutterBottom sx={{ mb: 3 }}>
        Answer a few questions to help us understand your investment comfort level.
      </Typography>

      <FormControl component="fieldset" fullWidth sx={{ mb: 3 }}>
        <FormLabel component="legend">
          If your portfolio dropped 20% in value, what would you do?
        </FormLabel>
        <RadioGroup
          value={data.riskAssessment.answers.marketDrop || ''}
          onChange={(e) => setData({
            ...data,
            riskAssessment: {
              ...data.riskAssessment,
              answers: { ...data.riskAssessment.answers, marketDrop: e.target.value },
            },
          })}
        >
          <FormControlLabel value="sell" control={<Radio />} label="Sell everything to avoid further losses" />
          <FormControlLabel value="hold" control={<Radio />} label="Hold and wait for recovery" />
          <FormControlLabel value="buy" control={<Radio />} label="Buy more at lower prices" />
        </RadioGroup>
      </FormControl>

      <FormControl component="fieldset" fullWidth sx={{ mb: 3 }}>
        <FormLabel component="legend">
          What is your investment time horizon?
        </FormLabel>
        <RadioGroup
          value={data.riskAssessment.answers.timeHorizon || ''}
          onChange={(e) => setData({
            ...data,
            riskAssessment: {
              ...data.riskAssessment,
              answers: { ...data.riskAssessment.answers, timeHorizon: e.target.value },
            },
          })}
        >
          <FormControlLabel value="<3" control={<Radio />} label="Less than 3 years" />
          <FormControlLabel value="3-7" control={<Radio />} label="3-7 years" />
          <FormControlLabel value="7-15" control={<Radio />} label="7-15 years" />
          <FormControlLabel value=">15" control={<Radio />} label="More than 15 years" />
        </RadioGroup>
      </FormControl>

      <Alert severity="info" sx={{ mt: 3 }}>
        <Typography variant="subtitle2">Your Risk Profile</Typography>
        <Typography variant="body2">
          Based on your answers, we'll recommend a portfolio allocation that matches your comfort level.
        </Typography>
      </Alert>
    </Box>
  );

  const renderDocumentUpload = () => {
    const { getRootProps, getInputProps } = useDropzone({
      accept: {
        'image/*': ['.png', '.jpg', '.jpeg'],
        'application/pdf': ['.pdf'],
      },
      maxSize: 10485760, // 10MB
      onDrop: async (acceptedFiles) => {
        for (const file of acceptedFiles) {
          await uploadDocument(file);
        }
      },
    });

    return (
      <Box>
        <Typography variant="h6" gutterBottom>
          Upload Identity Documents
        </Typography>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          We need to verify your identity per federal regulations (KYC/AML).
        </Typography>

        <Box
          {...getRootProps()}
          sx={{
            border: '2px dashed #ccc',
            borderRadius: 2,
            p: 4,
            textAlign: 'center',
            cursor: 'pointer',
            mt: 3,
            mb: 3,
            '&:hover': { borderColor: 'primary.main' },
          }}
        >
          <input {...getInputProps()} />
          <UploadIcon sx={{ fontSize: 48, color: 'primary.main', mb: 2 }} />
          <Typography variant="h6">Drag & drop files here</Typography>
          <Typography variant="body2" color="text.secondary">
            or click to select files
          </Typography>
          <Typography variant="caption" color="text.secondary" display="block" sx={{ mt: 1 }}>
            Accepted: Driver's License, Passport, Utility Bill (PDF, JPG, PNG, max 10MB)
          </Typography>
        </Box>

        {data.documents.map((doc) => (
          <Paper key={doc.documentId} sx={{ p: 2, mb: 1, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Box>
              <Typography variant="body1">{doc.fileName}</Typography>
              <Chip label={doc.documentType} size="small" sx={{ mr: 1 }} />
              <Chip label={doc.status} color={doc.status === 'VERIFIED' ? 'success' : 'default'} size="small" />
            </Box>
            <CheckIcon color="success" />
          </Paper>
        ))}
      </Box>
    );
  };

  const uploadDocument = async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('session_id', sessionId!);
    formData.append('document_type', 'DRIVERS_LICENSE'); // Auto-detect via OCR

    try {
      const response = await fetch('/api/onboarding/documents', {
        method: 'POST',
        body: formData,
      });
      const doc = await response.json();
      setData({
        ...data,
        documents: [...data.documents, doc],
      });
    } catch (error) {
      console.error('Upload failed:', error);
    }
  };

  const renderAccountSelection = () => (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <Typography variant="h6" gutterBottom>
          Choose Your Account
        </Typography>
      </Grid>

      <Grid item xs={12}>
        <FormControl fullWidth>
          <InputLabel>Account Type</InputLabel>
          <Select
            value={data.accountSelection.accountType}
            onChange={(e) => setData({
              ...data,
              accountSelection: { ...data.accountSelection, accountType: e.target.value },
            })}
          >
            <MenuItem value="INDIVIDUAL">Individual Brokerage</MenuItem>
            <MenuItem value="JOINT">Joint Account</MenuItem>
            <MenuItem value="IRA">Traditional IRA</MenuItem>
            <MenuItem value="ROTH_IRA">Roth IRA</MenuItem>
          </Select>
        </FormControl>
      </Grid>

      <Grid item xs={12}>
        <FormControl fullWidth>
          <InputLabel>Funding Method</InputLabel>
          <Select
            value={data.accountSelection.fundingMethod}
            onChange={(e) => setData({
              ...data,
              accountSelection: { ...data.accountSelection, fundingMethod: e.target.value },
            })}
          >
            <MenuItem value="ACH">Bank Transfer (ACH)</MenuItem>
            <MenuItem value="WIRE">Wire Transfer</MenuItem>
            <MenuItem value="CHECK">Mail a Check</MenuItem>
          </Select>
        </FormControl>
      </Grid>

      <Grid item xs={12}>
        <TextField
          label="Initial Investment"
          fullWidth
          type="number"
          value={data.accountSelection.initialInvestment}
          onChange={(e) => setData({
            ...data,
            accountSelection: { ...data.accountSelection, initialInvestment: parseFloat(e.target.value) },
          })}
          InputProps={{ startAdornment: '$' }}
          helperText="Minimum: $1,000"
        />
      </Grid>
    </Grid>
  );

  return (
    <Box sx={{ maxWidth: 900, mx: 'auto', p: 3 }}>
      <Typography variant="h4" gutterBottom align="center">
        Welcome to Your Financial Future
      </Typography>

      <Stepper activeStep={activeStep} sx={{ mt: 4, mb: 4 }}>
        {steps.map((label) => (
          <Step key={label}>
            <StepLabel>{label}</StepLabel>
          </Step>
        ))}
      </Stepper>

      {saving && <LinearProgress sx={{ mb: 2 }} />}

      {resumeToken && (
        <Alert severity="info" sx={{ mb: 3 }}>
          Your progress is automatically saved. You can resume anytime using this link:
          <Typography variant="body2" component="div" sx={{ mt: 1, fontFamily: 'monospace', fontSize: '0.875rem' }}>
            {window.location.origin}/onboarding?resume_token={resumeToken}
          </Typography>
        </Alert>
      )}

      <Paper sx={{ p: 4, mb: 3 }}>
        {renderStepContent(activeStep)}
      </Paper>

      <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
        <Button
          disabled={activeStep === 0}
          onClick={handleBack}
        >
          Back
        </Button>
        <Button
          variant="contained"
          onClick={handleNext}
          disabled={saving}
        >
          {activeStep === steps.length - 1 ? 'Complete' : 'Next'}
        </Button>
      </Box>

      <Typography variant="caption" color="text.secondary" align="center" display="block" sx={{ mt: 3 }}>
        Step {activeStep + 1} of {steps.length} • {Math.round(((activeStep + 1) / steps.length) * 100)}% Complete
      </Typography>
    </Box>
  );
};
