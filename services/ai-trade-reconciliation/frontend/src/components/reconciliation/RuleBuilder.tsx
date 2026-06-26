import React, { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  TextField,
  Button,
  Stack,
  Typography,
  Chip,
  Paper,
  FormControl,
  InputLabel,
  Select,
  MenuItem
} from '@mui/material';
import { useTheme } from '@mui/material/styles';

interface Rule {
  id: string;
  name: string;
  rule_type: string;
  rule_expr: string;
  enabled: boolean;
}

const RuleBuilder: React.FC = () => {
  const theme = useTheme();
  const [rules, setRules] = useState<Rule[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    rule_type: 'share_tolerance',
    rule_expr: '',
  });

  const handleSaveRule = async () => {
    try {
      const res = await fetch('/api/reconciliation/rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      });

      if (res.ok) {
        const newRule = await res.json();
        setRules([...rules, newRule]);
        setFormData({ name: '', rule_type: 'share_tolerance', rule_expr: '' });
        setShowForm(false);
      }
    } catch (err) {
      console.error('Failed to save rule', err);
    }
  };

  return (
    <Card sx={{ bgcolor: 'grey.900', color: 'white', boxShadow: 3 }}>
      <CardHeader
        title="Low-Code Rules Engine"
        titleTypographyProps={{ variant: 'h6', sx: { fontWeight: 'bold' } }}
        sx={{ pb: 2 }}
      />
      <CardContent>
        {showForm && (
          <Paper
            sx={{
              mb: 3,
              p: 2,
              bgcolor: 'grey.800',
              border: '1px solid',
              borderColor: 'grey.700'
            }}
          >
            <Typography variant="subtitle1" sx={{ fontWeight: 'bold', mb: 2, color: 'white' }}>
              Create New Rule
            </Typography>

            <Stack spacing={2}>
              <TextField
                fullWidth
                label="Rule Name"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="e.g., strict_price_tolerance"
                variant="outlined"
                sx={{
                  '& .MuiOutlinedInput-root': {
                    color: 'white',
                    '& fieldset': {
                      borderColor: 'grey.700',
                    },
                    '&:hover fieldset': {
                      borderColor: 'grey.600',
                    },
                  },
                  '& .MuiInputBase-input::placeholder': {
                    color: 'grey.500',
                    opacity: 1,
                  },
                }}
              />

              <FormControl fullWidth>
                <InputLabel sx={{ color: 'grey.300' }}>Rule Type</InputLabel>
                <Select
                  aria-label="Rule Type"
                  value={formData.rule_type}
                  label="Rule Type"
                  onChange={(e) => setFormData({ ...formData, rule_type: e.target.value })}
                  sx={{
                    color: 'white',
                    '& .MuiOutlinedInput-notchedOutline': {
                      borderColor: 'grey.700',
                    },
                    '&:hover .MuiOutlinedInput-notchedOutline': {
                      borderColor: 'grey.600',
                    },
                  }}
                >
                  <MenuItem value="share_tolerance">Share Tolerance</MenuItem>
                  <MenuItem value="price_tolerance">Price Tolerance</MenuItem>
                  <MenuItem value="date_tolerance">Date Tolerance</MenuItem>
                  <MenuItem value="custom">Custom</MenuItem>
                </Select>
              </FormControl>

              <TextField
                fullWidth
                multiline
                rows={3}
                label="JSONata Expression"
                value={formData.rule_expr}
                onChange={(e) => setFormData({ ...formData, rule_expr: e.target.value })}
                placeholder="e.g., $abs(($trade.shares - $confirm.shares) / $trade.shares) <= 0.001"
                variant="outlined"
                sx={{
                  '& .MuiOutlinedInput-root': {
                    color: 'white',
                    fontFamily: 'monospace',
                    '& fieldset': {
                      borderColor: 'grey.700',
                    },
                    '&:hover fieldset': {
                      borderColor: 'grey.600',
                    },
                  },
                }}
              />

              <Stack direction="row" gap={1}>
                <Button
                  variant="contained"
                  color="primary"
                  onClick={handleSaveRule}
                  sx={{ fontWeight: 'bold' }}
                >
                  Save Rule
                </Button>
                <Button
                  variant="outlined"
                  onClick={() => setShowForm(false)}
                  sx={{ color: 'grey.400', borderColor: 'grey.600' }}
                >
                  Cancel
                </Button>
              </Stack>
            </Stack>
          </Paper>
        )}

        <Button
          fullWidth
          variant={showForm ? 'outlined' : 'contained'}
          color="success"
          onClick={() => setShowForm(!showForm)}
          sx={{ mb: 2, fontWeight: 'bold' }}
        >
          {showForm ? 'Hide Form' : '+ Add Rule'}
        </Button>

        <Stack spacing={1}>
          {rules.length > 0 ? (
            rules.map((rule) => (
              <Paper
                key={rule.id}
                sx={{
                  p: 2,
                  bgcolor: 'grey.800',
                  border: '1px solid',
                  borderColor: 'grey.700',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'flex-start'
                }}
              >
                <Box>
                  <Typography variant="body2" sx={{ fontWeight: 'bold', color: 'white' }}>
                    {rule.name}
                  </Typography>
                  <Typography variant="caption" sx={{ color: 'grey.400' }}>
                    {rule.rule_type}
                  </Typography>
                  <Typography
                    variant="caption"
                    sx={{
                      color: 'grey.500',
                      fontFamily: 'monospace',
                      display: 'block',
                      mt: 1
                    }}
                  >
                    {rule.rule_expr}
                  </Typography>
                </Box>
                <Chip
                  label={rule.enabled ? 'Enabled' : 'Disabled'}
                  color={rule.enabled ? 'success' : 'default'}
                  size="small"
                  variant={rule.enabled ? 'filled' : 'outlined'}
                />
              </Paper>
            ))
          ) : (
            <Typography variant="body2" sx={{ color: 'grey.500', textAlign: 'center', py: 2 }}>
              No rules configured
            </Typography>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
};

export default RuleBuilder;
