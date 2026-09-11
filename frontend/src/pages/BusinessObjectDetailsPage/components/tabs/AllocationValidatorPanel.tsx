import { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, Paper, Stack, Chip, Button, TextField, IconButton,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  CircularProgress, Alert, Divider,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import InfoIcon from '@mui/icons-material/Info';
import SaveIcon from '@mui/icons-material/Save';
import apiClient from '../../../../utils/apiClient';
import { evaluateRuleWasm } from '../../../../rules/wasmRuntime';
import { useTenant } from '../../../../contexts/TenantContext';

interface AllocationRow {
  id?: string;
  order_id?: string;
  account_id: string;
  target_qty: number;
  allocated_qty: number;
}

interface RuleDescriptor {
  id: string;
  name: string;
  bo_name: string;
  severity: string;
  timing: string;
  category?: string;
  domain: string;
  rule_ast: unknown;
  is_active?: boolean;
}

interface AllocationValidatorPanelProps {
  order: any;
  businessObjectKey: string;
}

type EvalResult = 'pass' | 'fail' | 'error' | 'untriaged';

export function AllocationValidatorPanel({ order, businessObjectKey }: AllocationValidatorPanelProps) {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || '';
  const [allocations, setAllocations] = useState<AllocationRow[]>([]);
  const [rules, setRules] = useState<RuleDescriptor[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [evalResult, setEvalResult] = useState<EvalResult>('untriaged');
  const [evalMessage, setEvalMessage] = useState<string>('');
  const [addingRow, setAddingRow] = useState(false);
  const [newAccountId, setNewAccountId] = useState('ACCT-DISC-ACTIVE');
  const [newTargetQty, setNewTargetQty] = useState('');
  const [saving, setSaving] = useState(false);
  const [dirtyAllocations, setDirtyAllocations] = useState<AllocationRow[]>([]);
  const [isDirty, setIsDirty] = useState(false);

  const orderId = order?.id;
  const targetQty: number = order?.target_qty ?? 0;
  const totalAllocated = (isDirty ? dirtyAllocations : allocations)
    .reduce((sum: number, a: AllocationRow) => sum + (Number(a.target_qty) || 0), 0);

  const load = useCallback(async () => {
    if (!orderId || !tenantId) return;
    setLoading(true);
    setFetchError(null);
    try {
      const [allocRes, ruleRes] = await Promise.all([
        apiClient<any>(`/business-objects/${encodeURIComponent('order_allocation')}/data?limit=100`, {
          headers: { 'X-Tenant-ID': tenantId },
        }),
        apiClient<{ validationRules: RuleDescriptor[] }>(
          `/validation-rule-nodes?bo_name=${encodeURIComponent('order')}`, {
            headers: { 'X-Tenant-ID': tenantId },
          }
        ),
      ]);

      const allAllocs: AllocationRow[] = allocRes.rows || [];
      const forThisOrder = allAllocs.filter((a: any) => String(a.order_id) === String(orderId));
      setAllocations(forThisOrder);
      setDirtyAllocations(forThisOrder);
      setRules(ruleRes.validationRules || []);
    } catch (err: any) {
      setFetchError(err?.message || 'Failed to load allocation data');
    } finally {
      setLoading(false);
    }
  }, [orderId, tenantId]);

  useEffect(() => { load(); }, [load]);

  const reevaluate = useCallback(async (allocRows: AllocationRow[]) => {
    if (!rules.length || !targetQty) return;
    const sumRule = rules.find(
      (r: RuleDescriptor) =>
        r.is_active !== false &&
        JSON.stringify(r.rule_ast)?.includes('SUM') &&
        JSON.stringify(r.rule_ast)?.includes('OrderAllocations')
    );
    if (!sumRule) {
      setEvalResult('untriaged');
      setEvalMessage('No active SUM collection rule found for this BO');
      return;
    }
    try {
      const ctx = {
        TargetQuantity: targetQty,
        OrderAllocations: allocRows.map((a: AllocationRow) => ({
          target_qty: Number(a.target_qty) || 0,
        })),
      };
      const result = await evaluateRuleWasm(sumRule.rule_ast, ctx);
      setEvalResult(result ? 'pass' : 'fail');
      setEvalMessage(
        result
          ? `${totalAllocated} of ${targetQty} allocated — rule satisfied`
          : `${totalAllocated} of ${targetQty} allocated — rule violated`
      );
    } catch (err: any) {
      setEvalResult('error');
      setEvalMessage(`WASM eval error: ${err?.message || 'unknown'}`);
    }
  }, [rules, targetQty, totalAllocated]);

  useEffect(() => {
    if (rules.length && allocations.length >= 0) {
      reevaluate(isDirty ? dirtyAllocations : allocations);
    }
  }, [rules, allocations, dirtyAllocations, isDirty, reevaluate]);

  const handleAddAllocation = async () => {
    if (!newTargetQty || isNaN(Number(newTargetQty))) return;
    setSaving(true);
    try {
      await apiClient(`/business-objects/${encodeURIComponent('order_allocation')}/data`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId },
        body: JSON.stringify({
          record: {
            order_id: orderId,
            account_id: newAccountId,
            target_qty: Number(newTargetQty),
            allocated_qty: 0,
          },
        }),
      });
      setNewTargetQty('');
      setAddingRow(false);
      await load();
    } catch (err: any) {
      setFetchError(err?.message || 'Failed to add allocation');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteAllocation = async (alloc: AllocationRow) => {
    if (!alloc.id) return;
    try {
      await apiClient(
        `/business-objects/${encodeURIComponent('order_allocation')}/data/${encodeURIComponent(alloc.id)}`,
        { method: 'DELETE', headers: { 'X-Tenant-ID': tenantId } }
      );
      await load();
    } catch (err: any) {
      setFetchError(err?.message || 'Failed to delete allocation');
    }
  };

  const sumRule = rules.find(
    (r: RuleDescriptor) =>
      r.is_active !== false &&
      JSON.stringify(r.rule_ast)?.includes('SUM') &&
      JSON.stringify(r.rule_ast)?.includes('OrderAllocations')
  );

  return (
    <Paper variant="outlined" sx={{ mt: 2 }}>
      <Box sx={{ p: 2 }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1.5 }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
            Allocation Validator
          </Typography>
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="body2" color="text.secondary">
              {totalAllocated} / {targetQty} allocated
            </Typography>
            {evalResult === 'pass' && (
              <Chip
                icon={<CheckCircleIcon sx={{ fontSize: '1rem !important' }} />}
                label={sumRule?.name || 'Allocation rule'}
                size="small"
                color="success"
                variant="outlined"
              />
            )}
            {evalResult === 'fail' && (
              <Chip
                icon={<ErrorIcon sx={{ fontSize: '1rem !important' }} />}
                label={sumRule?.name || 'Allocation rule'}
                size="small"
                color="error"
                variant="outlined"
              />
            )}
            {evalResult === 'error' && (
              <Chip
                icon={<InfoIcon sx={{ fontSize: '1rem !important' }} />}
                label="WASM error"
                size="small"
                color="warning"
                variant="outlined"
              />
            )}
            {evalResult === 'untriaged' && (
              <Chip
                label="evaluated on save"
                size="small"
                color="default"
                variant="outlined"
              />
            )}
          </Stack>
        </Stack>

        {evalMessage && (
          <Typography variant="body2" color={evalResult === 'pass' ? 'success.main' : evalResult === 'fail' ? 'error.main' : 'text.secondary'} sx={{ mb: 1.5 }}>
            {evalMessage}
          </Typography>
        )}

        <Divider sx={{ mb: 1.5 }} />

        {loading ? (
          <Box sx={{ textAlign: 'center', py: 2 }}>
            <CircularProgress size={20} />
          </Box>
        ) : fetchError ? (
          <Alert severity="error" sx={{ mb: 1 }}>{fetchError}</Alert>
        ) : (
          <>
            <TableContainer sx={{ maxHeight: 240 }}>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 700 }}>Account</TableCell>
                    <TableCell sx={{ fontWeight: 700 }} align="right">Target Qty</TableCell>
                    <TableCell sx={{ fontWeight: 700 }} align="right">Allocated Qty</TableCell>
                    <TableCell align="right" sx={{ width: 48 }} />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(isDirty ? dirtyAllocations : allocations).map((alloc: AllocationRow, idx: number) => (
                    <TableRow key={alloc.id || idx}>
                      <TableCell>{alloc.account_id}</TableCell>
                      <TableCell align="right">{alloc.target_qty}</TableCell>
                      <TableCell align="right">{alloc.allocated_qty}</TableCell>
                      <TableCell align="right">
                        <IconButton size="small" onClick={() => handleDeleteAllocation(alloc)}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                  {(isDirty ? dirtyAllocations : allocations).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={4} align="center">
                        <Typography variant="body2" color="text.disabled">No allocations yet</Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>

            {addingRow ? (
              <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 1.5 }}>
                <TextField
                  size="small"
                  label="Account"
                  value={newAccountId}
                  onChange={(e) => setNewAccountId(e.target.value)}
                  sx={{ width: 180 }}
                />
                <TextField
                  size="small"
                  label="Target Qty"
                  type="number"
                  value={newTargetQty}
                  onChange={(e) => setNewTargetQty(e.target.value)}
                  sx={{ width: 120 }}
                />
                <Button
                  variant="contained"
                  size="small"
                  startIcon={<SaveIcon sx={{ fontSize: '1rem !important' }} />}
                  onClick={handleAddAllocation}
                  disabled={saving || !newTargetQty}
                >
                  {saving ? 'Saving…' : 'Save'}
                </Button>
                <Button size="small" onClick={() => { setAddingRow(false); setNewTargetQty(''); }}>
                  Cancel
                </Button>
              </Stack>
            ) : (
              <Button
                size="small"
                startIcon={<AddIcon sx={{ fontSize: '1rem !important' }} />}
                onClick={() => setAddingRow(true)}
                sx={{ mt: 1.5 }}
              >
                Add Allocation Row
              </Button>
            )}
          </>
        )}
      </Box>
    </Paper>
  );
}
