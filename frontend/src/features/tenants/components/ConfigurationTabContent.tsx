import React, { useState, useMemo } from 'react';
import {
  Box,
  Button,
  CardContent,
  CardHeader,
  Checkbox,
  FormControl,
  FormControlLabel,
  FormHelperText,
  FormLabel,
  InputAdornment,
  Paper,
  RadioGroup,
  Radio,
  Select,
  MenuItem,
  TextField,
  Typography,
  Switch,
  Alert,
  CircularProgress,
} from '@mui/material';
import {
  Schedule,
  Security,
  Api,
  Save,
  Close,
} from '@mui/icons-material';
import { gql, useQuery, useMutation } from '@apollo/client';

export interface TenantConfiguration {
  retention: {
    enabled: boolean;
    auditLogDays: number;
    transactionalDataYears: number;
    backupFrequency: 'daily' | 'weekly' | 'monthly';
    archiveStorageClass: 'standard' | 'infrequent' | 'glacier';
  };
  security: {
    mfaRequired: boolean;
    ssoEnabled: boolean;
    ipWhitelist: string[];
  };
  api: {
    rateLimitPerMinute: number;
    webhookRetryAttempts: number;
    defaultWebhookUrl: string;
  };
}

interface ConfigurationTabContentProps {
  tenantId: string;
  datasourceId: string;
}

// GraphQL query to fetch tenant configuration
const GET_TENANT_CONFIGURATION = gql`
  query GetTenantConfiguration($tenantId: uuid!, $datasourceId: uuid!) {
    tenants_by_pk(id: $tenantId) {
      id
      configuration
      updated_at
    }
  }
`;

// GraphQL mutation to update tenant configuration
const UPDATE_TENANT_CONFIGURATION = gql`
  mutation UpdateTenantConfiguration($tenantId: uuid!, $configuration: jsonb!) {
    update_tenants_by_pk(pk_columns: {id: $tenantId}, _set: {configuration: $configuration}) {
      id
      configuration
      updated_at
    }
  }
`;

// Default configuration fallback
const defaultConfig: TenantConfiguration = {
  retention: {
    enabled: true,
    auditLogDays: 90,
    transactionalDataYears: 7,
    backupFrequency: 'weekly',
    archiveStorageClass: 'infrequent',
  },
  security: {
    mfaRequired: true,
    ssoEnabled: false,
    ipWhitelist: ['203.0.113.5', '198.51.100.0/24'],
  },
  api: {
    rateLimitPerMinute: 5000,
    webhookRetryAttempts: 3,
    defaultWebhookUrl: 'api.acmenorthamerica.com/hooks/listener',
  },
};

export const ConfigurationTabContent: React.FC<ConfigurationTabContentProps> = ({
  tenantId,
  datasourceId,
}) => {
  // Fetch configuration from backend
  const { loading, error, data } = useQuery(GET_TENANT_CONFIGURATION, {
    variables: { tenantId, datasourceId },
    skip: !tenantId || !datasourceId,
  });

  // Mutation for saving configuration
  const [updateConfiguration, { loading: saving }] = useMutation(UPDATE_TENANT_CONFIGURATION);

  // Get configuration from fetched data or use default
  const initialConfig = useMemo(() => {
    if (data?.tenants_by_pk?.configuration) {
      return data.tenants_by_pk.configuration as TenantConfiguration;
    }
    return defaultConfig;
  }, [data]);

  const [formData, setFormData] = useState<TenantConfiguration>(initialConfig);
  const [hasChanges, setHasChanges] = useState(false);

  // Update form data when initial config loads
  React.useEffect(() => {
    if (initialConfig) {
      setFormData(initialConfig);
      setHasChanges(false);
    }
  }, [initialConfig]);

  const handleConfigChange = (
    section: keyof TenantConfiguration,
    field: string,
    value: any
  ) => {
    setFormData((prev) => ({
      ...prev,
      [section]: {
        ...prev[section],
        [field]: value,
      },
    }));
    setHasChanges(true);
  };

  const handleIpWhitelistChange = (value: string) => {
    const ips = value.split('\n').filter((ip) => ip.trim());
    handleConfigChange('security', 'ipWhitelist', ips);
  };

  const handleSave = async () => {
    try {
      await updateConfiguration({
        variables: {
          tenantId,
          configuration: formData,
        },
      });
      setHasChanges(false);
    } catch (error) {
      console.error('Failed to save configuration:', error);
    }
  };

  const handleDiscard = () => {
    setFormData(initialConfig);
    setHasChanges(false);
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, maxWidth: '900px' }}>
      {/* Loading State */}
      {loading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      )}

      {/* Error State */}
      {error && (
        <Alert severity="error">
          Failed to load configuration: {error.message}
        </Alert>
      )}

      {/* Configuration Content */}
      {!loading && !error && (
        <>
          {/* Header with Action Buttons */}
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              flexWrap: 'wrap',
              gap: 2,
            }}
          >
            <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
              Tenant Configuration
            </Typography>
            <Box sx={{ display: 'flex', gap: 2 }}>
              <Button
                variant="outlined"
                startIcon={<Close />}
                onClick={handleDiscard}
                disabled={!hasChanges || saving}
              >
                Discard Changes
              </Button>
              <Button
                variant="contained"
                startIcon={<Save />}
                onClick={handleSave}
                disabled={!hasChanges || saving}
                sx={{
                  backgroundColor: hasChanges ? '#0d7ff2' : '#ccc',
                  '&:hover': {
                    backgroundColor: hasChanges ? '#0a5fb8' : '#ccc',
                  },
                }}
              >
                {saving ? 'Saving...' : 'Save Changes'}
              </Button>
            </Box>
          </Box>

          {/* Data Retention Policy */}
      <Paper variant="outlined">
        <CardHeader
          avatar={<Schedule sx={{ color: '#3b82f6' }} />}
          title="Data Retention Policy"
          subheader="Define how long historical data is preserved for this tenant."
          sx={{
            backgroundColor: '#f9fafb',
            borderBottom: '1px solid #e5e7eb',
          }}
          action={
            <FormControlLabel
              control={
                <Switch
                  checked={formData.retention.enabled}
                  onChange={(e) =>
                    handleConfigChange('retention', 'enabled', e.target.checked)
                  }
                />
              }
              label=""
            />
          }
        />
        <CardContent>
          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 3 }}>
            {/* Audit Log Retention */}
            <FormControl fullWidth>
              <FormLabel>Audit Log Retention</FormLabel>
              <Select
                value={formData.retention.auditLogDays}
                onChange={(e) =>
                  handleConfigChange('retention', 'auditLogDays', e.target.value)
                }
                size="small"
              >
                <MenuItem value={30}>30 Days</MenuItem>
                <MenuItem value={60}>60 Days</MenuItem>
                <MenuItem value={90}>90 Days</MenuItem>
                <MenuItem value={180}>180 Days</MenuItem>
                <MenuItem value={365}>1 Year</MenuItem>
              </Select>
              <FormHelperText>Duration to keep system access logs.</FormHelperText>
            </FormControl>

            {/* Transactional Data Retention */}
            <FormControl fullWidth>
              <FormLabel>Transactional Data Retention</FormLabel>
              <Select
                value={formData.retention.transactionalDataYears}
                onChange={(e) =>
                  handleConfigChange('retention', 'transactionalDataYears', e.target.value)
                }
                size="small"
              >
                <MenuItem value={1}>1 Year</MenuItem>
                <MenuItem value={3}>3 Years</MenuItem>
                <MenuItem value={5}>5 Years</MenuItem>
                <MenuItem value={7}>7 Years</MenuItem>
                <MenuItem value={999}>Indefinite</MenuItem>
              </Select>
              <FormHelperText>Required for compliance in most regions.</FormHelperText>
            </FormControl>

            {/* Backup Frequency */}
            <FormControl fullWidth>
              <FormLabel sx={{ mb: 1 }}>Backup Frequency</FormLabel>
              <RadioGroup
                row
                value={formData.retention.backupFrequency}
                onChange={(e) =>
                  handleConfigChange('retention', 'backupFrequency', e.target.value)
                }
              >
                <FormControlLabel value="daily" control={<Radio />} label="Daily" />
                <FormControlLabel value="weekly" control={<Radio />} label="Weekly" />
                <FormControlLabel value="monthly" control={<Radio />} label="Monthly" />
              </RadioGroup>
            </FormControl>

            {/* Archive Storage Class */}
            <FormControl fullWidth>
              <FormLabel>Archive Storage Class</FormLabel>
              <Select
                value={formData.retention.archiveStorageClass}
                onChange={(e) =>
                  handleConfigChange('retention', 'archiveStorageClass', e.target.value)
                }
                size="small"
              >
                <MenuItem value="standard">Standard</MenuItem>
                <MenuItem value="infrequent">Infrequent Access</MenuItem>
                <MenuItem value="glacier">Glacier (Deep Archive)</MenuItem>
              </Select>
            </FormControl>
          </Box>
        </CardContent>
      </Paper>

      {/* Security & Access Control */}
      <Paper variant="outlined">
        <CardHeader
          avatar={<Security sx={{ color: '#10b981' }} />}
          title="Security & Access Control"
          subheader="Manage authentication and IP restriction settings."
          sx={{
            backgroundColor: '#f9fafb',
            borderBottom: '1px solid #e5e7eb',
          }}
        />
        <CardContent sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
          {/* MFA */}
          <FormControlLabel
            control={
              <Checkbox
                checked={formData.security.mfaRequired}
                onChange={(e) =>
                  handleConfigChange('security', 'mfaRequired', e.target.checked)
                }
              />
            }
            label={
              <Box>
                <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                  Enforce Multi-Factor Authentication (MFA)
                </Typography>
                <Typography variant="caption" color="textSecondary">
                  Require all users accessing this tenant to use 2FA.
                </Typography>
              </Box>
            }
          />

          {/* SSO */}
          <FormControlLabel
            control={
              <Checkbox
                checked={formData.security.ssoEnabled}
                onChange={(e) =>
                  handleConfigChange('security', 'ssoEnabled', e.target.checked)
                }
              />
            }
            label={
              <Box>
                <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                  Enable SAML Single Sign-On (SSO)
                </Typography>
                <Typography variant="caption" color="textSecondary">
                  Allow users to sign in using your organization's identity provider.
                </Typography>
              </Box>
            }
          />

          <Box sx={{ borderTop: '1px solid #e5e7eb', pt: 2 }}>
            {/* IP Whitelist */}
            <FormControl fullWidth>
              <FormLabel sx={{ mb: 1 }}>IP Whitelist</FormLabel>
              <Typography variant="caption" color="textSecondary" sx={{ mb: 1, display: 'block' }}>
                Only allow access from these IP addresses or ranges (CIDR).
              </Typography>
              <TextField
                multiline
                rows={3}
                value={formData.security.ipWhitelist.join('\n')}
                onChange={(e) => handleIpWhitelistChange(e.target.value)}
                placeholder="192.168.1.1&#10;10.0.0.0/24"
                variant="outlined"
                size="small"
                sx={{ fontFamily: 'monospace', fontSize: '0.85rem' }}
              />
            </FormControl>
          </Box>
        </CardContent>
      </Paper>

      {/* API Integration Preferences */}
      <Paper variant="outlined">
        <CardHeader
          avatar={<Api sx={{ color: '#a855f7' }} />}
          title="API Integration Preferences"
          subheader="Configure rate limits and webhook endpoints."
          sx={{
            backgroundColor: '#f9fafb',
            borderBottom: '1px solid #e5e7eb',
          }}
        />
        <CardContent>
          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 3 }}>
            {/* API Rate Limit */}
            <TextField
              label="API Rate Limit (Requests/Minute)"
              type="number"
              value={formData.api.rateLimitPerMinute}
              onChange={(e) =>
                handleConfigChange('api', 'rateLimitPerMinute', parseInt(e.target.value, 10))
              }
              size="small"
              fullWidth
            />

            {/* Webhook Retry Attempts */}
            <TextField
              label="Webhook Retry Attempts"
              type="number"
              value={formData.api.webhookRetryAttempts}
              onChange={(e) =>
                handleConfigChange(
                  'api',
                  'webhookRetryAttempts',
                  Math.min(10, parseInt(e.target.value, 10))
                )
              }
              size="small"
              fullWidth
              inputProps={{ min: 0, max: 10 }}
            />

            {/* Default Webhook URL */}
            <TextField
              label="Default Webhook Endpoint URL"
              value={formData.api.defaultWebhookUrl}
              onChange={(e) =>
                handleConfigChange('api', 'defaultWebhookUrl', e.target.value)
              }
              size="small"
              fullWidth
              sx={{ gridColumn: { xs: '1', md: '1 / -1' } }}
              InputProps={{
                startAdornment: <InputAdornment position="start">https://</InputAdornment>,
              }}
              placeholder="api.example.com/hooks"
            />
          </Box>
        </CardContent>
      </Paper>
        </>
      )}
    </Box>
  );
};
