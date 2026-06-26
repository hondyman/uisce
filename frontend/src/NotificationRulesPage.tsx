import { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  CircularProgress,
  Alert,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Tooltip,
  Chip,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import { listNotificationRules } from './api';
import type { NotificationRoutingRule } from './types';
import RoutingRuleEditor from './RoutingRuleEditor';

const RuleRow: React.FC<{ rule: NotificationRoutingRule; onEdit: (rule: NotificationRoutingRule) => void }> = ({ rule, onEdit }) => {
  const logic = rule.routing_logic ?? { notify: [] };

  return (
    <TableRow hover>
      <TableCell sx={{ fontFamily: 'monospace' }}>{rule.rule_id}</TableCell>
      <TableCell>{rule.trigger}</TableCell>
      <TableCell>{rule.scope}</TableCell>
      <TableCell>{rule.asset_type}</TableCell>
      <TableCell>
        {(logic.notify || []).map((r: string) => <Chip key={r} label={r} size="small" sx={{ mr: 0.5 }} color="primary" variant="outlined" />)}
        {(logic.escalate_to || []).map((r: string) => <Chip key={r} label={r} size="small" sx={{ mr: 0.5 }} color="error" />)}
      </TableCell>
      <TableCell>{new Date(rule.updated_at).toLocaleDateString()}</TableCell>
      <TableCell>{rule.updated_by}</TableCell>
      <TableCell>
        <Tooltip title="Edit Rule">
          <IconButton onClick={() => onEdit(rule)} size="small">
            <EditIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </TableCell>
    </TableRow>
  );
};

export default function NotificationRulesPage() {
  const [rules, setRules] = useState<NotificationRoutingRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingRule, setEditingRule] = useState<NotificationRoutingRule | null>(null);
  const [isEditorOpen, setIsEditorOpen] = useState(false);

  const fetchRules = useCallback(async () => {
    try {
      setLoading(true);
      const data = await listNotificationRules();
      setRules(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch notification rules');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const handleEdit = (rule: NotificationRoutingRule) => {
    setEditingRule(rule);
    setIsEditorOpen(true);
  };

  const handleCloseEditor = () => {
    setIsEditorOpen(false);
    setEditingRule(null);
    fetchRules(); // Refresh list on close
  };

  if (loading) return <CircularProgress />;
  if (error) return <Alert severity="error">{error}</Alert>;

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>Notification Routing Rules</Typography>
      <Typography color="text.secondary" sx={{ mb: 3 }}>
        Define how governance alerts are routed to different users and roles based on event triggers and asset context.
      </Typography>

      <Paper>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Rule ID</TableCell>
                <TableCell>Trigger</TableCell>
                <TableCell>Scope</TableCell>
                <TableCell>Asset Type</TableCell>
                <TableCell>Recipients</TableCell>
                <TableCell>Last Updated</TableCell>
                <TableCell>By</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {rules.map((rule) => <RuleRow key={rule.id} rule={rule} onEdit={handleEdit} />)}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {isEditorOpen && <RoutingRuleEditor rule={editingRule} onClose={handleCloseEditor} />}
    </Box>
  );
}