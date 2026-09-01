import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import { FactorExposureChart } from '../../../components/analytics/FactorExposureChart';
import { AttributionTable } from '../../../components/analytics/AttributionTable';

const fetchExposure = async (portfolioID: string) => {
  await new Promise(resolve => setTimeout(resolve, 500));
  return {
    portfolio_id: portfolioID,
    betas: {
      "Market": 1.1,
      "Size": 0.4,
      "Value": -0.2,
    },
    r_squared: 0.85,
  };
};

const fetchAttribution = async (portfolioID: string) => {
  await new Promise(resolve => setTimeout(resolve, 500));
  return {
    TotalReturn: 0.045,
    AlphaContribution: 0.012,
    FactorContributions: {
      "Market": 0.025,
      "Size": 0.005,
      "Value": 0.003,
    },
    Residual: 0.000,
  };
};

export const FactorAnalysisPage: React.FC = () => {
  const theme = useTheme();
  const { portfolioID } = useParams<{ portfolioID: string }>();
  const [exposure, setExposure] = useState<any>(null);
  const [attribution, setAttribution] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const pid = portfolioID || "demo-portfolio-001";
    
    Promise.all([
      fetchExposure(pid),
      fetchAttribution(pid)
    ]).then(([expData, attrData]) => {
      setExposure(expData);
      setAttribution(attrData);
      setLoading(false);
    });
  }, [portfolioID]);

  if (loading) return <Box sx={{ p: 4, color: theme.palette.text.primary }}>Loading analytics...</Box>;

  return (
    <Box sx={{ p: 4, bgcolor: theme.palette.background.paper, minHeight: '100vh' }}>
      <Typography variant="h5" sx={{ fontWeight: 600, mb: 3 }}>
        Factor Analytics: {portfolioID || "Demo Portfolio"}
      </Typography>
      
      <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 4 }}>
        <Paper sx={{ p: 3 }}>
          <FactorExposureChart betas={exposure.betas} />
          <Box sx={{ mt: 2, textAlign: 'center', fontSize: '0.875rem', color: theme.palette.text.secondary }}>
            R-Squared: {(exposure.r_squared * 100).toFixed(1)}%
          </Box>
        </Paper>

        <Paper sx={{ p: 3 }}>
          <AttributionTable data={attribution} />
        </Paper>
      </Box>
    </Box>
  );
};
