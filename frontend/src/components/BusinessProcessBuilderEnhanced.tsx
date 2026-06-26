import React, { useState, useCallback } from 'react';
import { Plus, Trash2, Clock, User, CheckCircle, FileText, Send, GitBranch, Settings, Play, Save } from 'lucide-react';
import {
  Box,
  TextField,
  Select,
  MenuItem,
  Button,
  Card,
  CardContent,
  FormControlLabel,
  Checkbox,
  Grid,
  Typography,
  Divider,
  FormControl,
  InputLabel,
  Paper,
  Container,
  Stack,
  Chip,
  FormGroup,
} from '@mui/material';
import SaveIcon from '@mui/icons-material/Save';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import DeleteIcon from '@mui/icons-material/Delete';
import { devDebug } from '../utils/devLogger';
import { useNotification } from '../hooks/useNotification';
import styles from './BusinessProcessBuilderEnhanced.module.css';

// Type definitions
interface BPStep {
  id: string;
  stepOrder: number;
  stepType: 'validate' | 'approve' | 'notify' | 'condition' | 'integrate';
  stepName: string;
  description?: string;
  durationHours: number;
  validationRules?: string[];
  assigneeRole?: string;
  assigneeUser?: string;
  notificationTemplate?: string;
  conditionLogic?: { condition: string; truePath?: string; falsePath?: string };
  apiEndpoint?: string;
}

interface BusinessProcess {
  id: string;
  processName: string;
  entity: string;
  description: string;
  steps: BPStep[];
  isActive: boolean;
  createdBy: string;
  createdAt: string;
}

// Step type configurations
const STEP_TYPES = [
  {
    type: 'validate',
    label: 'Validation',
    icon: CheckCircle,
    color: 'success',
    bgColor: '#4caf50',
    lightBg: '#e8f5e9',
    description: 'Verify data against validation rules',
  },
  {
    type: 'approve',
    label: 'Approval',
    icon: CheckCircle,
    color: 'info',
    bgColor: '#2196f3',
    lightBg: '#e3f2fd',
    description: 'Requires manual approval',
  },
  {
    type: 'notify',
    label: 'Notification',
    icon: Send,
    color: 'warning',
    bgColor: '#ff9800',
    lightBg: '#fff3e0',
    description: 'Send email or notification',
  },
  {
    type: 'integrate',
    label: 'Integration',
    icon: Settings,
    color: 'secondary',
    bgColor: '#9c27b0',
    lightBg: '#f3e5f5',
    description: 'Call external API/system',
  },
  {
    type: 'condition',
    label: 'Conditional Branch',
    icon: GitBranch,
    color: 'warning',
    bgColor: '#fbc02d',
    lightBg: '#fffde7',
    description: 'Branch based on conditions',
  },
];

const AVAILABLE_ROLES = [
  'Manager',
  'HR Admin',
  'Department Head',
  'Finance Approver',
  'System Admin',
  'Compliance Officer',
];

// Step Configurator Component - Using MUI
const StepConfigurator: React.FC<{
  step: BPStep;
  onUpdate: (step: BPStep) => void;
  onDelete: () => void;
  availableRules: string[];
}> = ({ step, onUpdate, onDelete, availableRules }) => {
  const stepConfig = STEP_TYPES.find((t) => t.type === step.stepType);
  const Icon = stepConfig?.icon || FileText;

  return (
    <Card
      elevation={3}
      sx={{
        mb: 3,
        borderLeft: `6px solid ${stepConfig?.bgColor}`,
        '&:hover': {
          elevation: 6,
          boxShadow: '0 12px 20px rgba(0,0,0,0.15)',
        },
      }}
    >
      <CardContent sx={{ p: 3 }}>
        {/* Header */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'flex-start' }}>
            {/* Icon */}
            <Box
              sx={{
                width: 56,
                height: 56,
                borderRadius: 2,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                bgcolor: stepConfig?.lightBg,
              }}
            >
              <Icon size={28} color={stepConfig?.bgColor} />
            </Box>

            {/* Step name and info */}
            <Box sx={{ flex: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 1 }}>
                <Chip
                  label={step.stepOrder}
                  size="small"
                  sx={{ fontWeight: 'bold', minWidth: 40 }}
                />
                <TextField
                  value={step.stepName}
                  onChange={(e) =>
                    onUpdate({ ...step, stepName: e.target.value })
                  }
                  variant="standard"
                  placeholder="Step name"
                  sx={{
                    '& .MuiInput-root': { fontSize: '1.1rem', fontWeight: 600 },
                  }}
                  fullWidth
                />
              </Box>
              <Typography variant="caption" color="textSecondary">
                {stepConfig?.description}
              </Typography>
            </Box>
          </Box>

          {/* Delete button */}
          <Button
            variant="text"
            color="error"
            size="small"
            onClick={onDelete}
            startIcon={<DeleteIcon />}
            sx={{ mt: 1 }}
          >
            Delete
          </Button>
        </Box>

        <Divider sx={{ my: 2 }} />

        {/* Configuration fields */}
        <Grid container spacing={2}>
          {/* Duration */}
          <Grid item xs={12} sm={6}>
            <TextField
              fullWidth
              type="number"
              label="Duration (hours)"
              value={step.durationHours}
              onChange={(e) =>
                onUpdate({
                  ...step,
                  durationHours: parseInt(e.target.value) || 0,
                })
              }
              inputProps={{ min: 0 }}
              size="small"
              variant="outlined"
            />
          </Grid>

          {/* Description */}
          <Grid item xs={12} sm={6}>
            <TextField
              fullWidth
              label="Description"
              value={step.description || ''}
              onChange={(e) =>
                onUpdate({ ...step, description: e.target.value })
              }
              placeholder="What happens in this step..."
              size="small"
              variant="outlined"
            />
          </Grid>

          {/* Validation Step - Rules */}
          {step.stepType === 'validate' && (
            <Grid item xs={12}>
              <FormControl fullWidth size="small">
                <InputLabel>Validation Rules</InputLabel>
                <FormGroup sx={{ mt: 1, mb: 1 }}>
                  {availableRules.map((rule) => (
                    <FormControlLabel
                      key={rule}
                      control={
                        <Checkbox
                          checked={
                            step.validationRules?.includes(rule) || false
                          }
                          onChange={(e) => {
                            const rules = step.validationRules || [];
                            const updated = e.target.checked
                              ? [...rules, rule]
                              : rules.filter((r) => r !== rule);
                            onUpdate({
                              ...step,
                              validationRules: updated,
                            });
                          }}
                        />
                      }
                      label={rule}
                    />
                  ))}
                </FormGroup>
              </FormControl>
              {(!step.validationRules ||
                step.validationRules.length === 0) && (
                <Typography
                  variant="caption"
                  color="error"
                  sx={{ mt: 1, display: 'block' }}
                >
                  ⚠️ No validation rules selected
                </Typography>
              )}
            </Grid>
          )}

          {/* Approval Step - Role and User */}
          {step.stepType === 'approve' && (
            <>
              <Grid item xs={12} sm={6}>
                <FormControl fullWidth size="small">
                  <InputLabel>Assignee Role</InputLabel>
                  <Select
                    value={step.assigneeRole || ''}
                    onChange={(e) =>
                      onUpdate({ ...step, assigneeRole: e.target.value })
                    }
                    label="Assignee Role"
                  >
                    <MenuItem value="">Select role...</MenuItem>
                    {AVAILABLE_ROLES.map((role) => (
                      <MenuItem key={role} value={role}>
                        {role}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Or Specific User"
                  value={step.assigneeUser || ''}
                  onChange={(e) =>
                    onUpdate({ ...step, assigneeUser: e.target.value })
                  }
                  placeholder="user@example.com"
                  size="small"
                  variant="outlined"
                />
              </Grid>
            </>
          )}

          {/* Notification Step - Template */}
          {step.stepType === 'notify' && (
            <Grid item xs={12}>
              <TextField
                fullWidth
                multiline
                rows={3}
                label="Notification Template"
                value={step.notificationTemplate || ''}
                onChange={(e) =>
                  onUpdate({
                    ...step,
                    notificationTemplate: e.target.value,
                  })
                }
                placeholder={'Subject: {{subject}}\nBody: Hi {{name}}, ...'}
                size="small"
                variant="outlined"
              />
              <Typography variant="caption" color="textSecondary" sx={{ mt: 1 }}>
                Use {'{{variable}}'} for dynamic values
              </Typography>
            </Grid>
          )}

          {/* Condition Step - Logic */}
          {step.stepType === 'condition' && (
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Condition Logic"
                value={step.conditionLogic?.condition || ''}
                onChange={(e) =>
                  onUpdate({
                    ...step,
                    conditionLogic: {
                      ...step.conditionLogic,
                      condition: e.target.value,
                    },
                  })
                }
                placeholder="e.g., amount > 10000 OR vip_status = true"
                size="small"
                variant="outlined"
                multiline
                rows={2}
              />
              <Typography variant="caption" color="textSecondary" sx={{ mt: 1 }}>
                Define condition using field names and operators
              </Typography>
            </Grid>
          )}

          {/* Integration Step - API */}
          {step.stepType === 'integrate' && (
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="API Endpoint"
                value={step.apiEndpoint || ''}
                onChange={(e) =>
                  onUpdate({ ...step, apiEndpoint: e.target.value })
                }
                placeholder="https://api.example.com/webhook"
                size="small"
                variant="outlined"
              />
            </Grid>
          )}
        </Grid>
      </CardContent>
    </Card>
  );
};

// Main Component
const BusinessProcessBuilderEnhanced: React.FC = () => {
  const [process, setProcess] = useState<BusinessProcess>({
    id: 'bp_new',
    processName: 'New Business Process',
    entity: 'Employee',
    description: '',
    steps: [],
    isActive: false,
    createdBy: 'Current User',
    createdAt: new Date().toISOString(),
  });

  const [isSaving, setIsSaving] = useState(false);
  const [showPreview, setShowPreview] = useState(false);
  const notification = useNotification();

  const availableRules = [
    'Email Format Validation',
    'Age Verification (18+)',
    'Salary Range Check',
    'Duplicate Email Check',
    'Required Fields Validation',
    'Date Range Validation',
  ];

  const addStep = useCallback(
    (stepType: string) => {
      const newStep: BPStep = {
        id: `step_${Date.now()}`,
        stepOrder: process.steps.length + 1,
        stepType: stepType as any,
        stepName: `${STEP_TYPES.find((t) => t.type === stepType)?.label} Step`,
        durationHours: stepType === 'approve' ? 48 : 24,
        validationRules: [],
        description: '',
      };

      setProcess({
        ...process,
        steps: [...process.steps, newStep],
      });
    },
    [process]
  );

  const updateStep = useCallback(
    (stepId: string, updatedStep: BPStep) => {
      setProcess({
        ...process,
        steps: process.steps.map((s) => (s.id === stepId ? updatedStep : s)),
      });
    },
    [process]
  );

  const deleteStep = useCallback(
    (stepId: string) => {
      const filtered = process.steps.filter((s) => s.id !== stepId);
      const reordered = filtered.map((step, idx) => ({
        ...step,
        stepOrder: idx + 1,
      }));
      setProcess({ ...process, steps: reordered });
    },
    [process]
  );

  const saveBP = async () => {
    setIsSaving(true);
    try {
      await new Promise((resolve) => setTimeout(resolve, 1500));
      devDebug('Saving BP:', process);
      notification.success('Business Process saved successfully!');
    } finally {
      setIsSaving(false);
    }
  };

  const simulateBP = () => {
    notification.info('Simulating BP execution...\n\nThis would start a Temporal workflow and show real-time progress.');
  };

  const totalDuration = process.steps.reduce(
    (sum, step) => sum + step.durationHours,
    0
  );

  return (
    <Box className={styles.bpEnhancedContainer}>
      <Box className={styles.bpContentWrapper}>
        <Stack className={styles.bpStack}>
          {/* Header Card */}
          <Paper
            elevation={4}
            className={styles.bpHeader}
            sx={{
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
              color: 'white',
            }}
          >
            <Box>
              <Typography variant="h4" fontWeight="bold" className={styles.bpHeaderTitle}>
                Business Process Builder
              </Typography>
              <Typography variant="subtitle1" className={styles.bpHeaderSubtitle}>
                Create automated workflows with validation, approvals, and
                integrations
              </Typography>
            </Box>
            <Stack direction="row" spacing={2} className={styles.bpButtonGroup}>
              <Button
                variant="contained"
                color="inherit"
                onClick={() => setShowPreview(!showPreview)}
                className={styles.bpButtonPrimary}
                sx={{
                  bgcolor: 'white',
                  color: '#667eea',
                  '&:hover': { bgcolor: '#f0f0f0' },
                }}
              >
                {showPreview ? 'Hide' : 'Show'} JSON
              </Button>
              <Button
                variant="contained"
                disabled={process.steps.length === 0}
                onClick={simulateBP}
                startIcon={<PlayArrowIcon />}
                sx={{ bgcolor: '#4caf50' }}
              >
                Simulate
              </Button>
              <Button
                variant="contained"
                disabled={isSaving || process.steps.length === 0}
                onClick={saveBP}
                startIcon={<SaveIcon />}
                sx={{ bgcolor: 'white', color: '#667eea' }}
              >
                {isSaving ? 'Saving...' : 'Save'}
              </Button>
            </Stack>
          </Paper>

          {/* Process Information */}
          <Card elevation={2}>
            <CardContent sx={{ p: 3 }}>
              <Typography variant="h6" fontWeight="bold" sx={{ mb: 3 }}>
                Process Information
              </Typography>
              <Grid container spacing={2}>
                <Grid item xs={12} sm={6} md={4}>
                  <TextField
                    fullWidth
                    label="Process Name"
                    value={process.processName}
                    onChange={(e) =>
                      setProcess({ ...process, processName: e.target.value })
                    }
                    placeholder="e.g., Hire Employee"
                    variant="outlined"
                  />
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <FormControl fullWidth>
                    <InputLabel>Target Entity</InputLabel>
                    <Select
                      value={process.entity}
                      onChange={(e) =>
                        setProcess({ ...process, entity: e.target.value })
                      }
                      label="Target Entity"
                    >
                      <MenuItem value="Employee">Employee</MenuItem>
                      <MenuItem value="Order">Order</MenuItem>
                      <MenuItem value="Invoice">Invoice</MenuItem>
                      <MenuItem value="Request">Request</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={12} sm={6} md={4}>
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={process.isActive}
                        onChange={(e) =>
                          setProcess({
                            ...process,
                            isActive: e.target.checked,
                          })
                        }
                      />
                    }
                    label="Active"
                    sx={{ mt: 1 }}
                  />
                </Grid>
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    multiline
                    rows={2}
                    label="Description"
                    value={process.description}
                    onChange={(e) =>
                      setProcess({
                        ...process,
                        description: e.target.value,
                      })
                    }
                    placeholder="Describe what this business process does..."
                    variant="outlined"
                  />
                </Grid>
              </Grid>
            </CardContent>
          </Card>

          {/* Stats Cards */}
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6} md={3}>
              <Card elevation={1}>
                <CardContent sx={{ textAlign: 'center' }}>
                  <Typography
                    variant="h4"
                    fontWeight="bold"
                    color="primary"
                    sx={{ mb: 1 }}
                  >
                    {process.steps.length}
                  </Typography>
                  <Typography variant="caption" color="textSecondary">
                    Total Steps
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Card elevation={1}>
                <CardContent sx={{ textAlign: 'center' }}>
                  <Typography
                    variant="h4"
                    fontWeight="bold"
                    color="secondary"
                    sx={{ mb: 1 }}
                  >
                    {totalDuration}h
                  </Typography>
                  <Typography variant="caption" color="textSecondary">
                    Total Duration
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Card elevation={1}>
                <CardContent sx={{ textAlign: 'center' }}>
                  <Typography
                    variant="h4"
                    fontWeight="bold"
                    sx={{ color: '#4caf50', mb: 1 }}
                  >
                    {process.steps.filter((s) => s.stepType === 'validate')
                      .length}
                  </Typography>
                  <Typography variant="caption" color="textSecondary">
                    Validation Steps
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Card elevation={1}>
                <CardContent sx={{ textAlign: 'center' }}>
                  <Typography
                    variant="h4"
                    fontWeight="bold"
                    sx={{ color: '#2196f3', mb: 1 }}
                  >
                    {process.steps.filter((s) => s.stepType === 'approve')
                      .length}
                  </Typography>
                  <Typography variant="caption" color="textSecondary">
                    Approval Steps
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          </Grid>

          {/* Add Step Palette */}
          <Card elevation={2}>
            <CardContent sx={{ p: 3 }}>
              <Typography variant="h6" fontWeight="bold" sx={{ mb: 3 }}>
                Add Step
              </Typography>
              <Grid container spacing={2}>
                {STEP_TYPES.map((stepType) => (
                  <Grid item xs={12} sm={6} md={4} lg={2.4} key={stepType.type}>
                    <Card
                      sx={{
                        cursor: 'pointer',
                        height: '100%',
                        display: 'flex',
                        flexDirection: 'column',
                        transition: 'all 0.3s ease',
                        border: '2px solid #e0e0e0',
                        '&:hover': {
                          borderColor: stepType.bgColor,
                          boxShadow: `0 8px 16px ${stepType.bgColor}20`,
                          transform: 'translateY(-4px)',
                        },
                      }}
                      onClick={() => addStep(stepType.type)}
                    >
                      <CardContent
                        sx={{
                          display: 'flex',
                          flexDirection: 'column',
                          alignItems: 'center',
                          textAlign: 'center',
                          flex: 1,
                        }}
                      >
                        <Box
                          sx={{
                            width: 48,
                            height: 48,
                            borderRadius: '50%',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            bgcolor: stepType.lightBg,
                            mb: 2,
                          }}
                        >
                          <stepType.icon
                            size={24}
                            color={stepType.bgColor}
                          />
                        </Box>
                        <Typography variant="subtitle2" fontWeight="bold">
                          {stepType.label}
                        </Typography>
                        <Typography variant="caption" color="textSecondary">
                          {stepType.description}
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>
                ))}
              </Grid>
            </CardContent>
          </Card>

          {/* Process Steps */}
          <Box>
            <Typography variant="h6" fontWeight="bold" sx={{ mb: 3 }}>
              Process Steps
            </Typography>

            {process.steps.length === 0 ? (
              <Paper sx={{ p: 6, textAlign: 'center', bgcolor: '#f9f9f9' }}>
                <Plus size={64} color="#ccc" style={{ margin: '0 auto 1rem' }} />
                <Typography variant="h6" color="textSecondary" sx={{ mb: 1 }}>
                  No steps added yet
                </Typography>
                <Typography variant="body2" color="textSecondary">
                  Click on a step type above to start building your business
                  process
                </Typography>
              </Paper>
            ) : (
              <Stack spacing={0}>
                {process.steps.map((step) => (
                  <StepConfigurator
                    key={step.id}
                    step={step}
                    onUpdate={(updated) => updateStep(step.id, updated)}
                    onDelete={() => deleteStep(step.id)}
                    availableRules={availableRules}
                  />
                ))}
              </Stack>
            )}
          </Box>

          {/* JSON Preview */}
          {showPreview && (
            <Card elevation={2}>
              <CardContent sx={{ p: 3 }}>
                <Typography variant="h6" fontWeight="bold" sx={{ mb: 2 }}>
                  JSON Configuration
                </Typography>
                <Box
                  component="pre"
                  sx={{
                    bgcolor: '#1e1e1e',
                    color: '#4ec9b0',
                    p: 3,
                    borderRadius: 1,
                    overflowX: 'auto',
                    fontSize: '0.875rem',
                    fontFamily: 'monospace',
                  }}
                >
                  {JSON.stringify(process, null, 2)}
                </Box>
              </CardContent>
            </Card>
          )}
        </Stack>
      </Box>
    </Box>
  );
};

export default BusinessProcessBuilderEnhanced;
