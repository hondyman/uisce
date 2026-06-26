import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineOppositeContent,
  Card,
  CardContent,
  Grid,
  Chip,
  LinearProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
  Alert,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Description as FormIcon,
  CheckCircle as CheckIcon,
  Warning as WarningIcon,
  TrendingUp as TrendingIcon,
} from '@mui/icons-material';
import { format } from 'date-fns';

interface Gift {
  gift_id: string;
  donor_member_id: string;
  donor_name: string;
  recipient_member_id: string;
  recipient_name: string;
  gift_date: string;
  gift_type: string;
  asset_description: string;
  fair_market_value: number;
  annual_exclusion_utilized: number;
  lifetime_exemption_utilized: number;
  gst_exemption_utilized: number;
  requires_form_709: boolean;
  form_709_filed: boolean;
  form_709_due_date?: string;
}

interface ExemptionSummary {
  annual_exclusion_used_this_year: number;
  annual_exclusion_remaining: number;
  lifetime_exemption_used: number;
  lifetime_exemption_remaining: number;
  gst_exemption_used: number;
  gst_exemption_remaining: number;
}

interface GiftTrackingDashboardProps {
  familyId: string;
}

export const GiftTrackingDashboard: React.FC<GiftTrackingDashboardProps> = ({ familyId }) => {
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [exemptionSummary, setExemptionSummary] = useState<ExemptionSummary | null>(null);
  const [selectedMemberId, setSelectedMemberId] = useState<string>('');
  const [newGiftDialog, setNewGiftDialog] = useState(false);
  const [loading, setLoading] = useState(false);

  // Form state
  const [formData, setFormData] = useState({
    donor_member_id: '',
    recipient_member_id: '',
    gift_date: format(new Date(), 'yyyy-MM-dd'),
    gift_type: 'ANNUAL_EXCLUSION',
    asset_description: '',
    fair_market_value: '',
    valuation_method: 'MARKET_PRICE',
    spousal_split_election: false,
    is_generation_skipping: false,
  });

  useEffect(() => {
    loadGifts();
  }, [familyId]);

  const loadGifts = async () => {
    try {
      const response = await fetch(`/api/wealth-transfer/families/${familyId}/gifts`);
      const data = await response.json();
      setGifts(data || []);
    } catch (error) {
      console.error('Failed to load gifts:', error);
    }
  };

  const loadExemptionSummary = async (memberId: string) => {
    try {
      const response = await fetch(`/api/wealth-transfer/families/${familyId}/members/${memberId}/exemptions`);
      const data = await response.json();
      setExemptionSummary(data);
    } catch (error) {
      console.error('Failed to load exemption summary:', error);
    }
  };

  const handleRecordGift = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/wealth-transfer/gifts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...formData,
          family_id: familyId,
          fair_market_value: parseFloat(formData.fair_market_value),
        }),
      });

      if (response.ok) {
        setNewGiftDialog(false);
        loadGifts();
        // Reset form
        setFormData({
          donor_member_id: '',
          recipient_member_id: '',
          gift_date: format(new Date(), 'yyyy-MM-dd'),
          gift_type: 'ANNUAL_EXCLUSION',
          asset_description: '',
          fair_market_value: '',
          valuation_method: 'MARKET_PRICE',
          spousal_split_election: false,
          is_generation_skipping: false,
        });
      }
    } catch (error) {
      console.error('Failed to record gift:', error);
    } finally {
      setLoading(false);
    }
  };

  const formatCurrency = (value: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const pendingForm709 = gifts.filter(g => g.requires_form_709 && !g.form_709_filed);

  return (
    <Box>
      {/* Header */}
      <Box sx={{ display: 'flex', gap: 2, mb: 3, alignItems: 'center' }}>
        <Typography variant="h6">Gift Tracking & Exemption Management</Typography>
        <Box sx={{ flexGrow: 1 }} />
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setNewGiftDialog(true)}
        >
          Record New Gift
        </Button>
      </Box>

      {/* Exemption Summary Cards */}
      {exemptionSummary && (
        <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid item xs={12} md={4}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="text.secondary" gutterBottom variant="body2">
                  Annual Exclusion (2025)
                </Typography>
                <Typography variant="h4">
                  {formatCurrency(exemptionSummary.annual_exclusion_remaining)}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Remaining this year
                </Typography>
                <LinearProgress
                  variant="determinate"
                  value={(exemptionSummary.annual_exclusion_used_this_year / 18500) * 100}
                  sx={{ mt: 1 }}
                />
                <Typography variant="caption">
                  Used: {formatCurrency(exemptionSummary.annual_exclusion_used_this_year)} / $18,500
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={4}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="text.secondary" gutterBottom variant="body2">
                  Lifetime Exemption
                </Typography>
                <Typography variant="h4" color={exemptionSummary.lifetime_exemption_remaining < 1000000 ? 'error' : 'inherit'}>
                  {formatCurrency(exemptionSummary.lifetime_exemption_remaining)}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Remaining of $13.99M
                </Typography>
                <LinearProgress
                  variant="determinate"
                  value={(exemptionSummary.lifetime_exemption_used / 13990000) * 100}
                  color={exemptionSummary.lifetime_exemption_remaining < 1000000 ? 'error' : 'primary'}
                  sx={{ mt: 1 }}
                />
                <Typography variant="caption">
                  Used: {formatCurrency(exemptionSummary.lifetime_exemption_used)}
                </Typography>
              </CardContent>
            </Card>
          </Grid>

          <Grid item xs={12} md={4}>
            <Card elevation={2}>
              <CardContent>
                <Typography color="text.secondary" gutterBottom variant="body2">
                  GST Exemption
                </Typography>
                <Typography variant="h4">
                  {formatCurrency(exemptionSummary.gst_exemption_remaining)}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Remaining of $13.99M
                </Typography>
                <LinearProgress
                  variant="determinate"
                  value={(exemptionSummary.gst_exemption_used / 13990000) * 100}
                  sx={{ mt: 1 }}
                />
                <Typography variant="caption">
                  Used: {formatCurrency(exemptionSummary.gst_exemption_used)}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* Form 709 Alert */}
      {pendingForm709.length > 0 && (
        <Alert severity="warning" sx={{ mb: 3 }} icon={<FormIcon />}>
          <Typography variant="subtitle2">
            {pendingForm709.length} gift{pendingForm709.length > 1 ? 's' : ''} requiring Form 709 filing
          </Typography>
          <Typography variant="body2">
            Due date: April 15, {new Date().getFullYear() + 1}
          </Typography>
          <Button size="small" sx={{ mt: 1 }} variant="outlined">
            Prepare Forms
          </Button>
        </Alert>
      )}

      {/* Gift Timeline */}
      <Paper elevation={2} sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>
          Gift History Timeline
        </Typography>

        {gifts.length === 0 ? (
          <Alert severity="info">No gifts recorded yet.</Alert>
        ) : (
          <Timeline position="alternate">
            {gifts.map((gift, index) => (
              <TimelineItem key={gift.gift_id}>
                <TimelineOppositeContent color="text.secondary">
                  {format(new Date(gift.gift_date), 'MMM d, yyyy')}
                </TimelineOppositeContent>
                <TimelineSeparator>
                  <TimelineDot color={gift.requires_form_709 ? 'warning' : 'success'}>
                    {gift.form_709_filed ? <CheckIcon /> : <TrendingIcon />}
                  </TimelineDot>
                  {index < gifts.length - 1 && <TimelineConnector />}
                </TimelineSeparator>
                <TimelineContent>
                  <Card variant="outlined">
                    <CardContent>
                      <Typography variant="subtitle2">
                        {gift.donor_name} → {gift.recipient_name}
                      </Typography>
                      <Typography variant="h6" color="primary">
                        {formatCurrency(gift.fair_market_value)}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" gutterBottom>
                        {gift.asset_description}
                      </Typography>
                      <Box sx={{ display: 'flex', gap: 1, mt: 1, flexWrap: 'wrap' }}>
                        <Chip
                          label={gift.gift_type.replace(/_/g, ' ')}
                          size="small"
                          variant="outlined"
                        />
                        {gift.annual_exclusion_utilized > 0 && (
                          <Chip
                            label={`Annual: ${formatCurrency(gift.annual_exclusion_utilized)}`}
                            size="small"
                            color="success"
                          />
                        )}
                        {gift.lifetime_exemption_utilized > 0 && (
                          <Chip
                            label={`Lifetime: ${formatCurrency(gift.lifetime_exemption_utilized)}`}
                            size="small"
                            color="primary"
                          />
                        )}
                        {gift.requires_form_709 && (
                          <Chip
                            label={gift.form_709_filed ? 'Form 709 Filed' : 'Form 709 Required'}
                            size="small"
                            color={gift.form_709_filed ? 'success' : 'warning'}
                            icon={gift.form_709_filed ? <CheckIcon /> : <WarningIcon />}
                          />
                        )}
                      </Box>
                    </CardContent>
                  </Card>
                </TimelineContent>
              </TimelineItem>
            ))}
          </Timeline>
        )}
      </Paper>

      {/* New Gift Dialog */}
      <Dialog open={newGiftDialog} onClose={() => setNewGiftDialog(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Record New Gift</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
            <TextField
              label="Donor"
              select
              value={formData.donor_member_id}
              onChange={(e) => {
                setFormData({ ...formData, donor_member_id: e.target.value });
                loadExemptionSummary(e.target.value);
              }}
              fullWidth
            >
              <MenuItem value="donor1">John Smith (Patriarch)</MenuItem>
              <MenuItem value="donor2">Mary Smith (Matriarch)</MenuItem>
            </TextField>

            <TextField
              label="Recipient"
              select
              value={formData.recipient_member_id}
              onChange={(e) => setFormData({ ...formData, recipient_member_id: e.target.value })}
              fullWidth
            >
              <MenuItem value="recipient1">Child 1</MenuItem>
              <MenuItem value="recipient2">Child 2</MenuItem>
              <MenuItem value="recipient3">Grandchild 1</MenuItem>
            </TextField>

            <TextField
              label="Gift Date"
              type="date"
              value={formData.gift_date}
              onChange={(e) => setFormData({ ...formData, gift_date: e.target.value })}
              fullWidth
              InputLabelProps={{ shrink: true }}
            />

            <TextField
              label="Gift Amount"
              type="number"
              value={formData.fair_market_value}
              onChange={(e) => setFormData({ ...formData, fair_market_value: e.target.value })}
              fullWidth
              InputProps={{
                startAdornment: <Typography sx={{ mr: 1 }}>$</Typography>,
              }}
            />

            <TextField
              label="Asset Description"
              value={formData.asset_description}
              onChange={(e) => setFormData({ ...formData, asset_description: e.target.value })}
              fullWidth
              placeholder="e.g., Cash, Stock (100 shares AAPL), Real estate"
            />

            <TextField
              label="Gift Type"
              select
              value={formData.gift_type}
              onChange={(e) => setFormData({ ...formData, gift_type: e.target.value })}
              fullWidth
            >
              <MenuItem value="ANNUAL_EXCLUSION">Annual Exclusion</MenuItem>
              <MenuItem value="LIFETIME_EXEMPTION">Lifetime Exemption</MenuItem>
              <MenuItem value="CHARITABLE">Charitable</MenuItem>
              <MenuItem value="QUALIFIED_TUITION">Qualified Tuition/Medical</MenuItem>
            </TextField>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setNewGiftDialog(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleRecordGift}
            disabled={loading || !formData.donor_member_id || !formData.fair_market_value}
          >
            {loading ? 'Recording...' : 'Record Gift'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
