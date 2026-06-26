// React import not required directly in this file (JSX runtime handles it)
import {
  Box,
  Paper,
  Typography,
  Grid,
  Card,
  CardContent
} from '@mui/material';
import {
  AttachMoney,
  TrendingUp,
  AccountBalance
} from '@mui/icons-material';

interface FeeCarryPanelProps {
  fundId?: string;
  selectedFunds?: string[];
  excelResults?: Record<string, Record<string, any>> | null;
  bundle?: any;
}

export const FeeCarryPanel: React.FC<FeeCarryPanelProps> = ({ selectedFunds: _selectedFunds = [] }) => {
  // Mock fee and carry data
  const feeData = {
    managementFee: 2.0,
    performanceFee: 20.0,
    carriedInterest: 15.0,
    totalFees: 3.2,
    carryAccrued: 12.5
  };

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom>
        Fee & Carry Analysis
      </Typography>

      <Grid container spacing={2}>
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1} mb={1}>
                <AttachMoney color="primary" />
                <Typography variant="subtitle2">Management Fee</Typography>
              </Box>
              <Typography variant="h5" color="primary">
                {feeData.managementFee}%
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Annual management fee
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1} mb={1}>
                <TrendingUp color="secondary" />
                <Typography variant="subtitle2">Performance Fee</Typography>
              </Box>
              <Typography variant="h5" color="secondary">
                {feeData.performanceFee}%
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Carried interest hurdle
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1} mb={1}>
                <AccountBalance color="success" />
                <Typography variant="subtitle2">Carry Accrued</Typography>
              </Box>
              <Typography variant="h5" color="success.main">
                ${feeData.carryAccrued}M
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Accumulated carried interest
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center" gap={1} mb={1}>
                <AttachMoney color="warning" />
                <Typography variant="subtitle2">Total Fees</Typography>
              </Box>
              <Typography variant="h5" color="warning.main">
                ${feeData.totalFees}M
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Total fees collected YTD
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Paper>
  );
};
