import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert, Box, Button, Chip, CircularProgress, Collapse, Dialog, DialogActions, DialogContent, DialogTitle,
  Divider, IconButton, Stack, ToggleButton, ToggleButtonGroup, Tooltip, Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { PageStudioApi, type CoreChangeGroup, type CoreComparison } from '../../api/pageStudio';
import type { CorePageDefinition } from '../../types/pageStudio';

interface Props {
  page: CorePageDefinition | null;
  onClose: () => void;
  onChanged: (page: CorePageDefinition) => void;
}

const SUMMARY_COLOR: Record<string, 'success' | 'error' | 'info'> = { added: 'success', removed: 'error', changed: 'info' };

const brief = (v: unknown): string => {
  if (v === undefined) return '';
  const s = typeof v === 'string' ? v : JSON.stringify(v);
  return s.length > 80 ? `${s.slice(0, 77)}…` : s;
};

const GroupRow: React.FC<{
  group: CoreChangeGroup;
  decision?: 'keep' | 'remove';
  onDecide?: (d: 'keep' | 'remove') => void;
}> = ({ group, decision, onDecide }) => {
  const [open, setOpen] = useState(false);
  return (
    <Box sx={{ py: 1 }}>
      <Stack direction="row" spacing={1} alignItems="center">
        <IconButton size="small" onClick={() => setOpen((o) => !o)} aria-label={open ? 'Hide changes' : 'Show changes'}>
          {open ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
        </IconButton>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography variant="body2" fontWeight={600} noWrap>{group.label}</Typography>
          <Stack direction="row" spacing={0.5} sx={{ mt: 0.25 }}>
            <Chip size="small" variant="outlined" color={SUMMARY_COLOR[group.summary] || 'default'} label={group.summary} />
            {group.conflict && (
              <Tooltip title="The core also changed this. Keep = your version wins; Remove = take the core's.">
                <Chip size="small" color="warning" label="Core also changed this" />
              </Tooltip>
            )}
            {group.inCore && (
              <Tooltip title="The core now ships exactly this change, so it stays either way.">
                <Chip size="small" color="success" variant="outlined" label="Now in core" />
              </Tooltip>
            )}
          </Stack>
        </Box>
        {onDecide && (
          <ToggleButtonGroup
            exclusive
            size="small"
            value={decision}
            onChange={(_, v: 'keep' | 'remove' | null) => v && onDecide(v)}
          >
            <ToggleButton value="keep" sx={{ textTransform: 'none', px: 1.5 }}>Keep</ToggleButton>
            <ToggleButton value="remove" sx={{ textTransform: 'none', px: 1.5 }}>Remove</ToggleButton>
          </ToggleButtonGroup>
        )}
      </Stack>
      <Collapse in={open}>
        <Box component="ul" sx={{ m: 0, mt: 0.5, pl: 7, fontFamily: 'monospace', fontSize: 12, color: 'text.secondary' }}>
          {group.changes.map((c, i) => (
            <li key={i}>
              {c.op} {c.path || '(page)'}
              {c.op === 'change' && <> : {brief(c.old)} → {brief(c.new)}</>}
              {c.op === 'reorder' && <> : {brief(c.new)}</>}
            </li>
          ))}
        </Box>
      </Collapse>
    </Box>
  );
};

/**
 * Compare a tenant's extension of a core page with the core, and upgrade:
 * each customization is kept (carried onto the new core version) or
 * removed; everything the core changed is taken. With no new core version
 * this just removes the customizations marked Remove.
 */
const CoreCompareDialog: React.FC<Props> = ({ page, onClose, onChanged }) => {
  const [data, setData] = useState<CoreComparison | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [decisions, setDecisions] = useState<Record<string, 'keep' | 'remove'>>({});

  useEffect(() => {
    if (!page) return;
    setData(null);
    setError(null);
    setDecisions({});
    PageStudioApi.compareWithCore(page.id)
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to compare with core'));
  }, [page]);

  const removed = useMemo(
    () => (data?.customizations || []).filter((g) => decisions[g.id] === 'remove').map((g) => g.id),
    [data, decisions],
  );

  const apply = async () => {
    if (!page) return;
    setSaving(true);
    setError(null);
    try {
      onChanged(await PageStudioApi.upgradeExtension(page.id, removed));
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upgrade failed');
    } finally {
      setSaving(false);
    }
  };

  const conflicts = data?.customizations.filter((g) => g.conflict).length ?? 0;
  const upgrade = !!data?.upgradeAvailable;

  return (
    <Dialog open={!!page} onClose={() => !saving && onClose()} maxWidth="md" fullWidth>
      <DialogTitle>
        {upgrade ? 'Upgrade' : 'Compare with core'}: {page?.name}
        {data && (
          <Typography variant="body2" color="text.secondary">
            Your customizations are based on core v{data.baseVersion}
            {upgrade ? `; core is now v${data.coreVersion}.` : ', the current core version.'}
          </Typography>
        )}
      </DialogTitle>
      <DialogContent dividers>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        {!data && !error && <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}><CircularProgress /></Box>}
        {data && (
          <>
            <Typography variant="subtitle2" fontWeight={700}>
              Your customizations ({data.customizations.length})
            </Typography>
            {data.customizations.length === 0 ? (
              <Typography variant="body2" color="text.secondary" sx={{ py: 1 }}>None - this page matches core v{data.baseVersion}.</Typography>
            ) : (
              <>
                {conflicts > 0 && (
                  <Alert severity="warning" sx={{ my: 1 }}>
                    {conflicts} customization{conflicts === 1 ? '' : 's'} touch{conflicts === 1 ? 'es' : ''} something the core also changed. Keep = your version wins; Remove = take the core's.
                  </Alert>
                )}
                {data.customizations.map((g) => (
                  <GroupRow
                    key={g.id}
                    group={g}
                    decision={decisions[g.id] || 'keep'}
                    onDecide={(d) => setDecisions((prev) => ({ ...prev, [g.id]: d }))}
                  />
                ))}
              </>
            )}
            {upgrade && (
              <>
                <Divider sx={{ my: 2 }} />
                <Typography variant="subtitle2" fontWeight={700}>
                  Core changes v{data.baseVersion} → v{data.coreVersion} ({data.coreUpdates.length})
                </Typography>
                <Typography variant="caption" color="text.secondary">Taken on upgrade, except where a customization you keep conflicts.</Typography>
                {data.coreUpdates.map((g) => <GroupRow key={g.id} group={g} />)}
              </>
            )}
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={saving}>Cancel</Button>
        <Button
          variant="contained"
          onClick={apply}
          disabled={!data || saving || (!upgrade && removed.length === 0)}
        >
          {saving
            ? 'Applying…'
            : upgrade
              ? `Upgrade to v${data?.coreVersion}${removed.length ? ` and remove ${removed.length}` : ''}`
              : `Remove ${removed.length} customization${removed.length === 1 ? '' : 's'}`}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default CoreCompareDialog;
