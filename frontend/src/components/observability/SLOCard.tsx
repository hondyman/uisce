import React from 'react';
import { Card, CardContent, Typography, Box, Chip, IconButton } from '@mui/material';
import { MoreVert as MoreVertIcon, CheckCircle, Warning, Error as ErrorIcon } from '@mui/icons-material';
import { SLODefinition } from '../../api/sloApi';

interface SLOCardProps {
  slo: SLODefinition;
  status?: 'met' | 'violated' | 'warning' | 'unknown';
  onEdit?: (slo: SLODefinition) => void;
  onDelete?: (slo: SLODefinition) => void;
}

const SLOCard: React.FC<SLOCardProps> = ({ slo, status = 'unknown', onEdit, onDelete }) => {
  const getStatusColor = () => {
    switch (status) {
      case 'met': return 'success';
      case 'violated': return 'error';
      case 'warning': return 'warning';
      default: return 'default';
    }
  };

  const getStatusIcon = () => {
    switch (status) {
      case 'met': return <CheckCircle color="success" fontSize="small" />;
      case 'violated': return <ErrorIcon color="error" fontSize="small" />;
      case 'warning': return <Warning color="warning" fontSize="small" />;
      default: return null;
    }
  };

  return (
    <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent>
        <Box display="flex" justifyContent="space-between" alignItems="flex-start">
          <Box display="flex" alignItems="center" gap={1}>
            {getStatusIcon()}
            <Typography variant="h6" component="div">
              {slo.slo_type.toUpperCase()}
            </Typography>
          </Box>
          <IconButton size="small">
            <MoreVertIcon />
          </IconButton>
        </Box>
        
        <Typography color="text.secondary" variant="body2" sx={{ mb: 1.5 }}>
          {slo.scope_type}: {slo.scope_id}
        </Typography>

        <Box display="flex" gap={1} flexWrap="wrap" mb={2}>
            <Chip 
              label={`Target: ${slo.target}`} 
              size="small" 
              color="primary" 
              variant="outlined" 
            />
            <Chip 
              label={`Window: ${slo.time_window}`} 
              size="small" 
              variant="outlined" 
            />
        </Box>

        <Typography variant="body2">
          Environment: {slo.env}
        </Typography>
      </CardContent>
    </Card>
  );
};

export default SLOCard;
