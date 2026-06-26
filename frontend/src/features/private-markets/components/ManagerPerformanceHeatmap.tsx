import type { FC } from 'react';
import { Bundle } from '../types';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
} from '@mui/material';

interface ManagerPerformanceHeatmapProps {
  data?: Array<{
    manager: string;
    funds: Array<{
      name: string;
      irr: number;
      tvpi: number;
      vintage: number;
    }>;
  }>;
  title?: string;
  compact?: boolean;
  selectedFunds?: string[];
  excelResults?: any;
  bundle?: Bundle;
}

const DEFAULT_DATA = [
  {
    manager: 'TechVentures Capital',
    funds: [
      { name: 'Tech Growth III', irr: 22.1, tvpi: 2.05, vintage: 2020 },
      { name: 'AI Innovation II', irr: 18.5, tvpi: 1.89, vintage: 2021 },
    ],
  },
  {
    manager: 'Global Infra Investments',
    funds: [
      { name: 'Infrastructure IV', irr: 15.6, tvpi: 1.85, vintage: 2019 },
      { name: 'Green Energy I', irr: 12.3, tvpi: 1.65, vintage: 2022 },
    ],
  },
  {
    manager: 'Urban Property Group',
    funds: [
      { name: 'Real Estate VI', irr: 18.9, tvpi: 1.92, vintage: 2020 },
      { name: 'Commercial Prop III', irr: 14.2, tvpi: 1.78, vintage: 2021 },
    ],
  },
];

export const ManagerPerformanceHeatmap: FC<ManagerPerformanceHeatmapProps> = ({
  data = DEFAULT_DATA,
  title = 'Manager Performance Heatmap',
  compact = false,
}) => {
  const getIrrColor = (irr: number) => {
    if (irr >= 20) return '#4caf50'; // green
    if (irr >= 15) return '#8bc34a'; // light green
    if (irr >= 10) return '#ffc107'; // yellow
    if (irr >= 5) return '#ff9800'; // orange
    return '#f44336'; // red
  };

  const getTvpiColor = (tvpi: number) => {
    if (tvpi >= 2.0) return '#4caf50'; // green
    if (tvpi >= 1.5) return '#8bc34a'; // light green
    if (tvpi >= 1.2) return '#ffc107'; // yellow
    if (tvpi >= 1.0) return '#ff9800'; // orange
    return '#f44336'; // red
  };

  if (compact) {
    return (
      <Box>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          {title}
        </Typography>
        <Grid container spacing={1}>
          {data.map((manager, managerIndex) => (
            <Grid item xs={12} key={managerIndex}>
              <Typography variant="caption" fontWeight="bold">
                {manager.manager}
              </Typography>
              <Box display="flex" gap={1} mt={0.5}>
                {manager.funds.map((fund, fundIndex) => (
                  <Box
                    key={fundIndex}
                    sx={{
                      width: 60,
                      height: 40,
                      backgroundColor: getIrrColor(fund.irr),
                      borderRadius: 1,
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      justifyContent: 'center',
                      color: 'white',
                      fontSize: '0.7rem',
                      fontWeight: 'bold',
                    }}
                  >
                    <Box>{fund.irr.toFixed(1)}%</Box>
                    <Box sx={{ fontSize: '0.6rem' }}>{fund.tvpi.toFixed(1)}x</Box>
                  </Box>
                ))}
              </Box>
            </Grid>
          ))}
        </Grid>
      </Box>
    );
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          {title}
        </Typography>

        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Manager</TableCell>
                <TableCell>Fund</TableCell>
                <TableCell align="center">Vintage</TableCell>
                <TableCell align="center">IRR</TableCell>
                <TableCell align="center">TVPI</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {data.map((manager, managerIndex) =>
                manager.funds.map((fund, fundIndex) => (
                  <TableRow key={`${managerIndex}-${fundIndex}`}>
                    {fundIndex === 0 && (
                      <TableCell
                        rowSpan={manager.funds.length}
                        sx={{ fontWeight: 'bold', verticalAlign: 'top' }}
                      >
                        {manager.manager}
                      </TableCell>
                    )}
                    <TableCell>{fund.name}</TableCell>
                    <TableCell align="center">{fund.vintage}</TableCell>
                    <TableCell align="center">
                      <Box
                        sx={{
                          backgroundColor: getIrrColor(fund.irr),
                          color: 'white',
                          px: 1,
                          py: 0.5,
                          borderRadius: 1,
                          fontWeight: 'bold',
                          display: 'inline-block',
                        }}
                      >
                        {fund.irr.toFixed(1)}%
                      </Box>
                    </TableCell>
                    <TableCell align="center">
                      <Box
                        sx={{
                          backgroundColor: getTvpiColor(fund.tvpi),
                          color: 'white',
                          px: 1,
                          py: 0.5,
                          borderRadius: 1,
                          fontWeight: 'bold',
                          display: 'inline-block',
                        }}
                      >
                        {fund.tvpi.toFixed(1)}x
                      </Box>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>

        <Box sx={{ mt: 2 }}>
          <Typography variant="body2" color="text.secondary" gutterBottom>
            Performance Legend:
          </Typography>
          <Grid container spacing={1}>
            <Grid item xs={6} sm={3}>
              <Box display="flex" alignItems="center" gap={1}>
                <Box sx={{ width: 16, height: 16, backgroundColor: '#4caf50', borderRadius: 1 }} />
                <Typography variant="caption">Excellent (≥20% IRR, ≥2.0x TVPI)</Typography>
              </Box>
            </Grid>
            <Grid item xs={6} sm={3}>
              <Box display="flex" alignItems="center" gap={1}>
                <Box sx={{ width: 16, height: 16, backgroundColor: '#8bc34a', borderRadius: 1 }} />
                <Typography variant="caption">Good (≥15% IRR, ≥1.5x TVPI)</Typography>
              </Box>
            </Grid>
            <Grid item xs={6} sm={3}>
              <Box display="flex" alignItems="center" gap={1}>
                <Box sx={{ width: 16, height: 16, backgroundColor: '#ffc107', borderRadius: 1 }} />
                <Typography variant="caption">Average (≥10% IRR, ≥1.2x TVPI)</Typography>
              </Box>
            </Grid>
            <Grid item xs={6} sm={3}>
              <Box display="flex" alignItems="center" gap={1}>
                <Box sx={{ width: 16, height: 16, backgroundColor: '#f44336', borderRadius: 1 }} />
                <Typography variant="caption">Below Average (&lt;10% IRR, &lt;1.2x TVPI)</Typography>
              </Box>
            </Grid>
          </Grid>
        </Box>
      </CardContent>
    </Card>
  );
};
