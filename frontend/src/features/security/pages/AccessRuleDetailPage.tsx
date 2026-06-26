import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Box,
  Button,
  Container,
  Paper,
  Tab,
  Tabs,
  Typography,
  Stack,
  Chip,
  IconButton,
  Alert,
  Card,
  CardContent,
  Grid,
} from '@mui/material';
import {
  ArrowBack as BackIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Security as SecurityIcon,
} from '@mui/icons-material';
import { accessRulesApi, AccessRule } from '../../../api/accessRules';
import { RuleImpactPanel } from '../components/RuleImpactPanel';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div role="tabpanel" hidden={value !== index} {...other}>
      {value === index && <Box sx={{ py: 3 }}>{children}</Box>}
    </div>
  );
}

export const AccessRuleDetailPage: React.FC = () => {
  const { ruleId } = useParams<{ ruleId: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState(0);
  const [rule, setRule] = useState<AccessRule | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadRule = async () => {
      if (!ruleId) return;
      setLoading(true);
      setError(null);
      try {
        const data = await accessRulesApi.get(ruleId);
        setRule(data);
      } catch (e: any) {
        setError(e?.message || 'Failed to load access rule');
      } finally {
        setLoading(false);
      }
    };
    void loadRule();
  }, [ruleId]);

  if (loading) {
    return (
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Typography>Loading...</Typography>
      </Container>
    );
  }

  if (error || !rule) {
    return (
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Alert severity="error">{error || 'Rule not found'}</Alert>
        <Button onClick={() => navigate('/security/access-rules')} sx={{ mt: 2 }}>
          Back to Rules
        </Button>
      </Container>
    );
  }

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Stack direction="row" spacing={2} alignItems="center">
          <IconButton onClick={() => navigate('/security/access-rules')}>
            <BackIcon />
          </IconButton>
          <Box>
            <Typography variant="h4" sx={{ fontWeight: 700 }}>
              Access Rule Details
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {rule.ruleId}
            </Typography>
          </Box>
        </Stack>
        <Stack direction="row" spacing={2}>
          <Chip label={rule.status} color={rule.status === 'APPROVED' ? 'success' : 'default'} />
          <Button variant="outlined" startIcon={<EditIcon />}>
            Edit
          </Button>
          <Button variant="outlined" color="error" startIcon={<DeleteIcon />}>
            Delete
          </Button>
        </Stack>
      </Stack>

      {/* Overview Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="caption" color="text.secondary">
                Team/User Group
              </Typography>
              <Typography variant="body1" sx={{ fontWeight: 600, mt: 1 }}>
                {rule.groupDn}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="caption" color="text.secondary">
                Data Type
              </Typography>
              <Typography variant="body1" sx={{ fontWeight: 600, mt: 1 }}>
                {rule.businessObjectId}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="caption" color="text.secondary">
                Access Level
              </Typography>
              <Chip
                label={rule.accessLevel}
                color={rule.accessLevel === 'WRITE' ? 'primary' : 'default'}
                sx={{ mt: 1 }}
              />
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Tabs */}
      <Paper elevation={2}>
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
          <Tab label="Overview" />
          <Tab label="Row Filters" />
          <Tab label="Field Masks" />
          <Tab label="Impact" />
        </Tabs>

        <TabPanel value={activeTab} index={0}>
          <Stack spacing={3}>
            <Box>
              <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
                Rule Configuration
              </Typography>
              <Grid container spacing={2}>
                <Grid item xs={12} md={6}>
                  <Typography variant="caption" color="text.secondary">
                    Applies to APIs
                  </Typography>
                  <Typography variant="body2">
                    {rule.scope?.appliesToApis ? 'Yes' : 'No'}
                  </Typography>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="caption" color="text.secondary">
                    Applies to BI
                  </Typography>
                  <Typography variant="body2">
                    {rule.scope?.appliesToBi ? 'Yes' : 'No'}
                  </Typography>
                </Grid>
                <Grid item xs={12} md={6}>
                  <Typography variant="caption" color="text.secondary">
                    Applies to AI
                  </Typography>
                  <Typography variant="body2">
                    {rule.scope?.appliesToAi ? 'Yes' : 'No'}
                  </Typography>
                </Grid>
              </Grid>
            </Box>
          </Stack>
        </TabPanel>

        <TabPanel value={activeTab} index={1}>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              Row Filter Expression
            </Typography>
            {rule.rowFilterDsl ? (
              <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50', fontFamily: 'monospace' }}>
                {rule.rowFilterDsl}
              </Paper>
            ) : (
              <Alert severity="info">No row filters configured. All rows are accessible.</Alert>
            )}
          </Box>
        </TabPanel>

        <TabPanel value={activeTab} index={2}>
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              Field Masks
            </Typography>
            {rule.columnMasks && rule.columnMasks.length > 0 ? (
              <Stack spacing={1}>
                {rule.columnMasks.map((mask, index) => (
                  <Paper key={index} elevation={0} sx={{ p: 2, bgcolor: 'grey.50' }}>
                    <Stack direction="row" spacing={2} alignItems="center">
                      <Chip label={mask.semanticTermId} variant="outlined" />
                      <Typography variant="body2">→</Typography>
                      <Chip label={mask.maskType} color="primary" />
                    </Stack>
                  </Paper>
                ))}
              </Stack>
            ) : (
              <Alert severity="info">No field masks configured. All fields are visible.</Alert>
            )}
          </Box>
        </TabPanel>

        <TabPanel value={activeTab} index={3}>
          <RuleImpactPanel rule={rule} />
        </TabPanel>
      </Paper>
    </Container>
  );
};

export default AccessRuleDetailPage;
