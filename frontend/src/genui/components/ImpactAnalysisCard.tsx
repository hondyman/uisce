import React from 'react';
import { motion } from 'framer-motion';
import { Card, CardHeader, CardContent, Typography, Chip, Box, Grid, useTheme } from '@mui/material';
import { TrendingDown, TrendingUp } from 'lucide-react';

interface ImpactAnalysisCardProps {
  headline: string;
  affectedSector: string;
  impactScore: number; // 0-100
}

export function ImpactAnalysisCard({ headline, affectedSector, impactScore }: ImpactAnalysisCardProps) {
  const theme = useTheme();
  const isHighImpact = impactScore > 70;
  const isPositive = impactScore > 50; // Simplified logic

  return (
    <motion.div 
      initial={{ x: -20, opacity: 0 }}
      animate={{ x: 0, opacity: 1 }}
    >
      <Card sx={{ my: 2, borderRadius: 3, border: '1px solid', borderColor: 'divider', overflow: 'hidden', boxShadow: theme.shadows[2] }}>
        <Box sx={{ 
          px: 2, 
          py: 1, 
          borderBottom: '1px solid', 
          borderColor: 'divider', 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center', 
          backgroundColor: theme.palette.action.hover 
        }}>
          <Typography variant="caption" sx={{ fontWeight: 'bold', textTransform: 'uppercase', letterSpacing: 1.1, color: 'text.secondary' }}>
            Impact Analysis
          </Typography>
          <Chip 
            label={`Score: ${impactScore}/100`} 
            size="small" 
            color={isHighImpact ? 'error' : 'default'} 
            sx={{ fontWeight: 'bold' }} 
          />
        </Box>
        
        <CardContent sx={{ p: 3 }}>
          <Typography variant="h6" component="h3" sx={{ fontWeight: 'bold', color: 'text.primary', mb: 2 }}>
            {headline}
          </Typography>
          
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12} sm={6}>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
                Affected Sector
              </Typography>
              <Typography variant="body1" sx={{ fontWeight: 'medium', color: 'text.primary' }}>
                {affectedSector}
              </Typography>
            </Grid>
            
            <Grid item xs={12} sm={6}>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
                Projected Impact
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                {isPositive ? (
                  <TrendingUp size={20} color={theme.palette.success.main} />
                ) : (
                  <TrendingDown size={20} color={theme.palette.error.main} />
                )}
                <Typography variant="body1" sx={{ 
                  fontWeight: 'medium', 
                  color: isPositive ? 'success.main' : 'error.main' 
                }}>
                  {isPositive ? 'Positive' : 'Negative'} Volatility
                </Typography>
              </Box>
            </Grid>
          </Grid>
        </CardContent>
      </Card>
    </motion.div>
  );
}

