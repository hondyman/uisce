import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  IconButton,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  Tooltip,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import UpgradeIcon from '@mui/icons-material/Upgrade';
import RestoreIcon from '@mui/icons-material/Restore';
import {
  compareSavedQuery,
  upgradeSavedQuery,
  revertSavedQuery,
  type QueryComparisonResult,
  type CoreGroup,
} from '../services/savedQueryApi';
import type { SavedQuery } from '../types/queryDef';

interface Props {
  query: SavedQuery | null;
  onClose: () => void;
  onChanged: (updated: SavedQuery) => void;
}

const formatValue = (v: unknown): string => {
  if (v === undefined) return '';
  const s = typeof v === 'string' ? v : JSON.stringify(v);
  return s.length > 80 ? `${s.slice(0, 77)}…` : s;
};

const QueryGroupRow: React.FC<{
  group: CoreGroup;
  decision?: 'keep' | 'remove';
  onDecide?: (d: 'keep' | 'remove') => void;
}> = ({ group, decision, onDecide }) => {
  const [open, setOpen] = useState(false);
  return (
    <Box sx={{ py: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
      <Stack direction="row" spacing={1} alignItems="center">
        <IconButton
          size="small"
          onClick={() => setOpen((o) => !o)}
          aria-label={open ? 'Hide changes' : 'Show changes'}
        >
          {open ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
        </IconButton>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography variant="body2" fontWeight={600} noWrap>
            {group.label}
          </Typography>
          <Stack direction="row" spacing={0.5} sx={{ mt: 0.25 }}>
            <Chip size="small" variant="outlined" label={group.kind} />
          </Stack>
        </Box>
        {onDecide && (
          <ToggleButtonGroup
            exclusive
            size="small"
            value={decision}
            onChange={(_, v: 'keep' | 'remove' | null) => v && onDecide(v)}
          >
            <ToggleButton value="keep" sx={{ textTransform: 'none', px: 1.5 }}>
              Keep
            </ToggleButton>
            <ToggleButton value="remove" sx={{ textTransform: 'none', px: 1.5 }}>
              Remove
            </ToggleButton>
          </ToggleButtonGroup>
        )}
      </Stack>
      <Collapse in={open}>
        <Box
          component="ul"
          sx={{
            m: 0,
            mt: 0.5,
            pl: 7,
            fontFamily: 'monospace',
            fontSize: 12,
            color: 'text.secondary',
          }}
        >
          {group.changes.map((c, i) => (
            <li key={i}>
              {c.op}{' '}
              {c.path
                .map((p) => (p.value !== undefined ? `${p.kind}[${p.value}]` : p.kind))
                .join('.')}
              {c.op === 'change' && (
                <> : {formatValue(c.old)} → {formatValue(c.new)}</>
              )}
              {c.op === 'add' && <> : {formatValue(c.new)}</>}
              {c.op === 'remove' && <> : {formatValue(c.old)}</>}
            </li>
          ))}
        </Box>
      </Collapse>
    </Box>
  );
};

export default function CoreQueryCompareDialog({ query, onClose, onChanged }: Props) {
  const [data, setData] = useState<QueryComparisonResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [decisions, setDecisions] = useState<Record<string, 'keep' | 'remove'>>({});
  const [showRevertConfirm, setShowRevertConfirm] = useState(false);

  useEffect(() => {
    if (!query) return;
    setData(null);
    setError(null);
    setDecisions({});
    compareSavedQuery(query.id)
      .then(setData)
      .catch((err) =>
        setError(err instanceof Error ? err.message : 'Failed to compare query with core')
      );
  }, [query]);

  const removed = useMemo(
    () =>
      (data?.customizations || [])
        .filter((g) => decisions[g.id] === 'remove')
        .map((g) => g.id),
    [data, decisions]
  );

  const handleUpgrade = async () => {
    if (!query) return;
    setSaving(true);
    setError(null);
    try {
      const res = await upgradeSavedQuery(query.id, { remove: removed });
      onChanged(res as SavedQuery);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upgrade failed');
    } finally {
      setSaving(false);
    }
  };

  const handleRevert = async () => {
    if (!query) return;
    setSaving(true);
    setError(null);
    try {
      const res = await revertSavedQuery(query.id);
      onChanged(res);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Revert failed');
    } finally {
      setSaving(false);
      setShowRevertConfirm(false);
    }
  };

  const upgrade = !!data?.upgradeAvailable;

  return (
    <Dialog open={!!query} onClose={() => !saving && onClose()} maxWidth="md" fullWidth>
      <DialogTitle>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Box>
            <Typography variant="h6" fontWeight={700}>
              {upgrade ? 'Upgrade Core Query' : 'Compare with Core'}: {query?.name}
            </Typography>
            {data && (
              <Typography variant="body2" color="text.secondary">
                Your extensions are based on core v{data.baseVersion}
                {upgrade ? `; core is now v${data.coreVersion}.` : ' (current core version).'}
              </Typography>
            )}
          </Box>
          <Button
            size="small"
            color="warning"
            variant="outlined"
            startIcon={<RestoreIcon />}
            onClick={() => setShowRevertConfirm(true)}
            disabled={saving}
          >
            Revert to Core
          </Button>
        </Stack>
      </DialogTitle>

      <DialogContent dividers>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        {showRevertConfirm && (
          <Alert
            severity="warning"
            sx={{ mb: 2 }}
            action={
              <Stack direction="row" spacing={1}>
                <Button size="small" color="inherit" onClick={() => setShowRevertConfirm(false)}>
                  Cancel
                </Button>
                <Button size="small" color="warning" variant="contained" onClick={handleRevert}>
                  Confirm Revert
                </Button>
              </Stack>
            }
          >
            Reverting switches this query back to vanilla core. Your extensions will be kept dormant so you can re-extend anytime.
          </Alert>
        )}

        {data?.conflicts && data.conflicts.length > 0 && (
          <Alert severity="error" sx={{ mb: 2 }}>
            <Typography variant="subtitle2" fontWeight={700}>
              Alias Collisions Detected:
            </Typography>
            <ul style={{ margin: 0, paddingLeft: 20 }}>
              {data.conflicts.map((c, i) => (
                <li key={i}>{c}</li>
              ))}
            </ul>
          </Alert>
        )}

        {!data && !error && (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress />
          </Box>
        )}

        {data && (
          <>
            <Typography variant="subtitle2" fontWeight={700}>
              Your Additive Extensions ({data.customizations.length})
            </Typography>
            {data.customizations.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ py: 1 }}>
                None - this query is running pure vanilla core v{data.baseVersion}.
              </Typography>
            ) : (
              <Box sx={{ my: 1 }}>
                {data.customizations.map((g) => (
                  <QueryGroupRow
                    key={g.id}
                    group={g}
                    decision={decisions[g.id] || 'keep'}
                    onDecide={(d) => setDecisions((prev) => ({ ...prev, [g.id]: d }))}
                  />
                ))}
              </Box>
            )}

            {upgrade && (
              <>
                <Divider sx={{ my: 2 }} />
                <Typography variant="subtitle2" fontWeight={700}>
                  Core Updates v{data.baseVersion} → v{data.coreVersion} ({data.coreUpdates.length})
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  These core additions and improvements will be automatically merged into your query.
                </Typography>
                <Box sx={{ mt: 1 }}>
                  {data.coreUpdates.map((g) => (
                    <QueryGroupRow key={g.id} group={g} />
                  ))}
                </Box>
              </>
            )}
          </>
        )}
      </DialogContent>

      <DialogActions sx={{ p: 2, justifyContent: 'space-between' }}>
        <Button onClick={onClose} disabled={saving}>
          Cancel
        </Button>
        <Button
          variant="contained"
          startIcon={<UpgradeIcon />}
          onClick={handleUpgrade}
          disabled={!data || saving || (!upgrade && removed.length === 0) || Boolean(data?.conflicts && data.conflicts.length > 0)}
        >
          {saving
            ? 'Applying…'
            : upgrade
            ? `Upgrade to v${data?.coreVersion}${removed.length ? ` (removing ${removed.length})` : ''}`
            : `Remove ${removed.length} extension${removed.length === 1 ? '' : 's'}`}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
