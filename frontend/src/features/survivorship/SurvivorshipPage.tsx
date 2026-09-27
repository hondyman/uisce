import React, { useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  FormControl,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  CreateRuleInput,
  SurvivorshipRule,
  survivorshipApi,
} from './api';

const ENTITY_TYPES = ['ACCOUNT', 'PRODUCT', 'SECURITY', 'PARTY'];
const STRATEGIES = [
  'SOURCE_PRIORITY',
  'MOST_RECENT',
  'CONSERVATIVE_MIN',
  'CONSERVATIVE_MAX',
  'WEIGHTED_CONFIDENCE',
];

const SurvivorshipPage: React.FC = () => {
  const qc = useQueryClient();
  const [entityType, setEntityType] = useState('ACCOUNT');
  const [termId, setTermId] = useState('');
  const [strategy, setStrategy] = useState('SOURCE_PRIORITY');
  const [priorityText, setPriorityText] = useState('GOLDENSOURCE,MARKET_EDM,ASSET_CONTROL,INTERNAL');
  const [error, setError] = useState<string | null>(null);

  const sourcesQ = useQuery({
    queryKey: ['survivorship-sources'],
    queryFn: () => survivorshipApi.listSources(),
  });
  const rulesQ = useQuery({
    queryKey: ['survivorship-rules', entityType],
    queryFn: () => survivorshipApi.list(entityType),
  });

  const createMut = useMutation({
    mutationFn: (body: CreateRuleInput) => survivorshipApi.create(body),
    onSuccess: () => {
      setTermId('');
      setError(null);
      qc.invalidateQueries({ queryKey: ['survivorship-rules', entityType] });
    },
    onError: (e: Error) => setError(e.message || 'Create failed'),
  });

  const deactivateMut = useMutation({
    mutationFn: (id: string) => survivorshipApi.deactivate(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['survivorship-rules', entityType] }),
    onError: (e: Error) => setError(e.message || 'Deactivate failed'),
  });

  const moveMut = useMutation({
    mutationFn: ({ rule, priority_order }: { rule: SurvivorshipRule; priority_order: string[] }) =>
      survivorshipApi.update(rule.id, { priority_order }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['survivorship-rules', entityType] }),
    onError: (e: Error) => setError(e.message || 'Update failed'),
  });

  const sources = sourcesQ.data?.sources ?? [];
  const rules = rulesQ.data?.rules ?? [];

  const sourceLabels = useMemo(() => {
    const m = new Map<string, string>();
    for (const s of sources) m.set(s.code, s.display_name);
    return m;
  }, [sources]);

  const onCreate = () => {
    const priority_order = priorityText
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
    if (!termId.trim()) {
      setError('semantic_term_id is required');
      return;
    }
    createMut.mutate({
      entity_type: entityType,
      semantic_term_id: termId.trim(),
      strategy,
      priority_order,
    });
  };

  const bumpSource = (rule: SurvivorshipRule, code: string, dir: -1 | 1) => {
    const order = [...(rule.priority_order || [])];
    const i = order.indexOf(code);
    if (i < 0) return;
    const j = i + dir;
    if (j < 0 || j >= order.length) return;
    [order[i], order[j]] = [order[j], order[i]];
    moveMut.mutate({ rule, priority_order: order });
  };

  return (
    <Box sx={{ p: 3, maxWidth: 1200, mx: 'auto' }}>
      <Typography variant="h4" gutterBottom>
        Survivorship Rules
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 2 }}>
        Source hierarchy for mastering (GoldenSource / Market EDM / Asset Control style). Rules are keyed by
        semantic term so validation and gold-copy mastering share the same contract.
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ mb: 3 }}>
        <FormControl size="small" sx={{ minWidth: 180 }}>
          <InputLabel>Entity</InputLabel>
          <Select
            label="Entity"
            value={entityType}
            onChange={(e) => setEntityType(e.target.value)}
          >
            {ENTITY_TYPES.map((e) => (
              <MenuItem key={e} value={e}>
                {e}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Stack>

      <Paper sx={{ p: 2, mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          Add rule
        </Typography>
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems="flex-start">
          <TextField
            size="small"
            label="Semantic term ID"
            value={termId}
            onChange={(e) => setTermId(e.target.value)}
            helperText="catalog_node UUID of type semantic_term"
            sx={{ minWidth: 320 }}
          />
          <FormControl size="small" sx={{ minWidth: 200 }}>
            <InputLabel>Strategy</InputLabel>
            <Select label="Strategy" value={strategy} onChange={(e) => setStrategy(e.target.value)}>
              {STRATEGIES.map((s) => (
                <MenuItem key={s} value={s}>
                  {s}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <TextField
            size="small"
            label="Priority order"
            value={priorityText}
            onChange={(e) => setPriorityText(e.target.value)}
            helperText="Comma-separated source codes (first wins)"
            sx={{ flex: 1, minWidth: 280 }}
          />
          <Button variant="contained" onClick={onCreate} disabled={createMut.isPending}>
            Create
          </Button>
        </Stack>
        {sources.length > 0 && (
          <Stack direction="row" spacing={1} sx={{ mt: 2, flexWrap: 'wrap', gap: 1 }}>
            {sources.map((s) => (
              <Chip
                key={s.code}
                size="small"
                label={`${s.default_rank}. ${s.display_name} (${s.code})`}
                variant="outlined"
              />
            ))}
          </Stack>
        )}
      </Paper>

      <Paper sx={{ p: 2 }}>
        <Typography variant="h6" gutterBottom>
          {entityType} rules
        </Typography>
        {rulesQ.isLoading ? (
          <CircularProgress size={28} />
        ) : rules.length === 0 ? (
          <Typography color="text.secondary">No rules for this entity yet.</Typography>
        ) : (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Term</TableCell>
                <TableCell>Strategy</TableCell>
                <TableCell>Source hierarchy</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {rules.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell>
                    <Typography variant="body2" fontWeight={600}>
                      {rule.semantic_term_name || rule.semantic_term_id}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {rule.semantic_term_path}
                    </Typography>
                  </TableCell>
                  <TableCell>{rule.strategy}</TableCell>
                  <TableCell>
                    <Stack direction="row" spacing={0.5} flexWrap="wrap" useFlexGap>
                      {(rule.priority_order || []).map((code, idx) => (
                        <Chip
                          key={`${rule.id}-${code}`}
                          size="small"
                          label={`${idx + 1}. ${sourceLabels.get(code) || code}`}
                          onClick={() => bumpSource(rule, code, -1)}
                          onDelete={
                            idx < (rule.priority_order?.length || 0) - 1
                              ? () => bumpSource(rule, code, 1)
                              : undefined
                          }
                          deleteIcon={<span style={{ fontSize: 11, padding: '0 4px' }}>↓</span>}
                        />
                      ))}
                    </Stack>
                    <Typography variant="caption" color="text.secondary">
                      Click chip to move up; ↓ to move down
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Button
                      size="small"
                      color="warning"
                      onClick={() => deactivateMut.mutate(rule.id)}
                      disabled={deactivateMut.isPending}
                    >
                      Deactivate
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Paper>
    </Box>
  );
};

export default SurvivorshipPage;
