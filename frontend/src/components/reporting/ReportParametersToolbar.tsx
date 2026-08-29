import React from 'react';
import { Box, Button, CircularProgress, TextField, MenuItem, Select, Typography, Paper } from '@mui/material';
import { Play, RefreshCw } from 'lucide-react';

type ReportParameter = {
  id: string;
  name: string;
  type: 'string' | 'number' | 'date' | 'boolean';
  prompt: string;
  defaultValue?: string;
  allowBlank?: boolean;
  allowMultiple?: boolean;
};

interface ReportParametersToolbarProps {
  parameters: ReportParameter[];
  values: Record<string, any>;
  onChange: (paramName: string, value: any) => void;
  onRun: (params: Record<string, any>) => void;
  currentUserProfile: {
    id: string;
    tenantId: string;
    tenantCode: string;
    accountId: string;
    clientId: string;
    branchId: string;
    region: string;
  };
  loading: boolean;
  reportId: string;
  reportKey: string;
}

const ReportParametersToolbar: React.FC<ReportParametersToolbarProps> = ({
  parameters,
  values,
  onChange,
  onRun,
  loading,
}) => {
  const handleRun = () => {
    onRun(values);
  };

  return (
    <Paper
      elevation={0}
      sx={{
        display: 'flex',
        alignItems: 'center',
        flexWrap: 'wrap',
        gap: 2,
        p: 1.5,
        bgcolor: 'rgba(255, 255, 255, 0.04)',
        border: '1px solid rgba(255, 255, 255, 0.08)',
        borderRadius: 1.5,
      }}
    >
      <Typography variant="subtitle2" sx={{ color: 'text.secondary', mr: 1 }}>
        Parameters
      </Typography>

      {parameters.map((param) => (
        <Box key={param.id} sx={{ minWidth: 150 }}>
          {param.type === 'boolean' ? (
            <Select
              size="small"
              fullWidth
              value={values[param.name] ?? param.defaultValue ?? false}
              onChange={(e) => onChange(param.name, e.target.value)}
              displayEmpty
            >
              <MenuItem value={true}>True</MenuItem>
              <MenuItem value={false}>False</MenuItem>
            </Select>
          ) : (
            <TextField
              size="small"
              fullWidth
              type={param.type === 'number' ? 'number' : param.type === 'date' ? 'date' : 'text'}
              label={param.prompt || param.name}
              value={values[param.name] ?? param.defaultValue ?? ''}
              onChange={(e) => onChange(param.name, e.target.value)}
              variant="outlined"
            />
          )}
        </Box>
      ))}

      <Box sx={{ display: 'flex', gap: 1, ml: 'auto' }}>
        <Button
          variant="contained"
          size="small"
          startIcon={loading ? <CircularProgress size={16} /> : <Play size={16} />}
          onClick={handleRun}
          disabled={loading}
        >
          Run
        </Button>
        <Button
          variant="outlined"
          size="small"
          startIcon={<RefreshCw size={16} />}
          onClick={() => onRun({})}
          disabled={loading}
        >
          Reset
        </Button>
      </Box>
    </Paper>
  );
};

export default ReportParametersToolbar;
