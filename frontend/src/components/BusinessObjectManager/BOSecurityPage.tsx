import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Tabs,
  Tab,
  Card,
  CardContent,
  Alert,
  Stack,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Switch,
  Chip,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Security as SecurityIcon,
  Visibility as VisibilityIcon,
  VisibilityOff as VisibilityOffIcon,
} from '@mui/icons-material';

interface BOSecurityTabProps {
  boId: string;
}

export const BOSecurityPage: React.FC<BOSecurityTabProps> = ({ boId }) => {
  const [activeTab, setActiveTab] = useState(0);

  // Partial Implementation / Mock State for RLS
  const [rlsPolicies, setRlsPolicies] = useState([
    { id: '1', name: 'US Region Access', role: 'Sales_US', condition: "region = 'US'", enabled: true },
    { id: '2', name: 'Manager View', role: 'Manager', condition: "1=1", enabled: true }
  ]);

  // Partial Implementation / Mock State for CLS
  const [clsRules, setClsRules] = useState([
    { id: '1', field: 'salary_amount', role: 'HR_Admin', access: 'FULL', masked: false },
    { id: '2', field: 'salary_amount', role: 'Manager', access: 'READ', masked: true },
    { id: '3', field: 'ssn', role: 'Admin', access: 'FULL', masked: true }
  ]);

  return (
    <Box sx={{ p: 3 }}>
        <Box sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: 1 }}>
                <SecurityIcon color="primary" /> Security Configuration
            </Typography>
            <Typography variant="body2" color="text.secondary">
                Configure Row Level Service (RLS) and Column Level Security (CLS) policies.
            </Typography>
        </Box>

      <Paper sx={{ mb: 3 }}>
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tab label="Row Level Security (RLS)" />
          <Tab label="Column Level Security (CLS)" />
        </Tabs>

        {/* RLS Tab */}
        {activeTab === 0 && (
          <Box sx={{ p: 3 }}>
             <Alert severity="info" sx={{ mb: 3 }}>
                Row Level Security filters data rows based on user roles and attributes.
                Policies are additive (OR logic between policies for same user).
             </Alert>
             
             <Stack direction="row" justifyContent="flex-end" sx={{ mb: 2 }}>
                <Button variant="contained" startIcon={<AddIcon />} size="small">
                    Add Policy
                </Button>
             </Stack>

             <Table size="small">
                <TableHead>
                    <TableRow sx={{ bgcolor: 'action.hover' }}>
                        <TableCell sx={{ fontWeight: 600 }}>Policy Name</TableCell>
                        <TableCell sx={{ fontWeight: 600 }}>Role / Group</TableCell>
                        <TableCell sx={{ fontWeight: 600 }}>Filter Condition (SQL)</TableCell>
                         <TableCell sx={{ fontWeight: 600 }} align="center">Status</TableCell>
                        <TableCell align="right">Actions</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {rlsPolicies.map(policy => (
                        <TableRow key={policy.id}>
                            <TableCell>{policy.name}</TableCell>
                            <TableCell><Chip label={policy.role} size="small" /></TableCell>
                            <TableCell><code>{policy.condition}</code></TableCell>
                             <TableCell align="center">
                                <Switch size="small" checked={policy.enabled} />
                             </TableCell>
                            <TableCell align="right">
                                <IconButton size="small" color="error">
                                    <DeleteIcon fontSize="small" />
                                </IconButton>
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
             </Table>
          </Box>
        )}

        {/* CLS Tab */}
        {activeTab === 1 && (
             <Box sx={{ p: 3 }}>
                <Alert severity="info" sx={{ mb: 3 }}>
                    Column Level Security restricts access to specific fields or masks sensitive data.
                </Alert>

                <Stack direction="row" justifyContent="flex-end" sx={{ mb: 2 }}>
                    <Button variant="contained" startIcon={<AddIcon />} size="small">
                        Add Rule
                    </Button>
                </Stack>

                <Table size="small">
                    <TableHead>
                         <TableRow sx={{ bgcolor: 'action.hover' }}>
                            <TableCell sx={{ fontWeight: 600 }}>Field</TableCell>
                            <TableCell sx={{ fontWeight: 600 }}>Role / Group</TableCell>
                            <TableCell sx={{ fontWeight: 600 }}>Access Level</TableCell>
                             <TableCell sx={{ fontWeight: 600 }} align="center">Masked</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                     <TableBody>
                        {clsRules.map(rule => (
                            <TableRow key={rule.id}>
                                <TableCell sx={{ fontWeight: 500 }}>{rule.field}</TableCell>
                                <TableCell><Chip label={rule.role} size="small" /></TableCell>
                                <TableCell>
                                    <Chip 
                                        label={rule.access} 
                                        size="small" 
                                        color={rule.access === 'FULL' ? 'success' : 'warning'} 
                                        variant="outlined"
                                    />
                                </TableCell>
                                 <TableCell align="center">
                                    {rule.masked ? (
                                        <Tooltip title="Data is masked">
                                            <VisibilityOffIcon fontSize="small" color="action" />
                                        </Tooltip>
                                    ) : (
                                         <Tooltip title="Data is visible">
                                            <VisibilityIcon fontSize="small" color="disabled" />
                                        </Tooltip>
                                    )}
                                 </TableCell>
                                <TableCell align="right">
                                    <IconButton size="small" color="error">
                                        <DeleteIcon fontSize="small" />
                                    </IconButton>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
             </Box>
        )}
      </Paper>
    </Box>
  );
};
