import React, { useEffect, useState } from 'react';
import {
  Alert, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, FormControl, IconButton, InputLabel, MenuItem,
  Select, Stack, TextField, Tooltip, Typography,
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import { NavigationMenuApi, type NavigationMenuNode } from '../../api/navigationMenu';
import type { CorePageDefinition } from '../../types/pageStudio';

interface Props {
  page: CorePageDefinition | null;
  onClose: () => void;
  /** After any change, so the list can reload placements. */
  onChanged: () => void;
}

/** Containers only (entries without a page), depth-first, with their path. */
export function sections(nodes: NavigationMenuNode[], path: string[] = []): { node: NavigationMenuNode; path: string[] }[] {
  return nodes.flatMap((n) => {
    const here = [...path, n.label];
    return n.targetPageKey ? [] : [{ node: n, path: here }, ...sections(n.children || [], here)];
  });
}

/**
 * Where a page sits on the navigation menu (the one Menu Designer edits and
 * Browse Pages shows): its current entries, removable when they are the
 * tenant's own, and a new entry under any section - including a section
 * inherited from the gold copy.
 */
const PlaceOnMenuDialog: React.FC<Props> = ({ page, onClose, onChanged }) => {
  const [tree, setTree] = useState<NavigationMenuNode[]>([]);
  const [parentId, setParentId] = useState('');
  const [label, setLabel] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!page) return;
    setError(null);
    setParentId('');
    setLabel(page.name);
    NavigationMenuApi.listTree().then(setTree).catch((e) => setError(e instanceof Error ? e.message : 'Failed to load the menu'));
  }, [page]);

  const add = async () => {
    if (!page) return;
    setBusy(true);
    setError(null);
    const node = { parentId: parentId || null, label: label.trim() || page.name, targetPageKey: page.slug, displayOrder: 100 };
    try {
      try {
        await NavigationMenuApi.create({ ...node, nodeKey: page.slug });
      } catch (e) {
        // Menu keys are unique per tenant; a page placed twice needs a second key.
        if (!(e instanceof Error) || !/409/.test(e.message)) throw e;
        await NavigationMenuApi.create({ ...node, nodeKey: `${page.slug}-${Date.now().toString(36)}` });
      }
      onChanged();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to place the page');
    } finally {
      setBusy(false);
    }
  };

  const remove = async (nodeId: string) => {
    setBusy(true);
    setError(null);
    try {
      await NavigationMenuApi.remove(nodeId);
      onChanged();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to remove the entry');
    } finally {
      setBusy(false);
    }
  };

  const placements = page?.menuPlacements ?? [];
  return (
    <Dialog open={!!page} onClose={() => !busy && onClose()} maxWidth="sm" fullWidth>
      <DialogTitle>Place “{page?.name}” on the menu</DialogTitle>
      <DialogContent dividers>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        <Typography variant="subtitle2" fontWeight={700} gutterBottom>On the menu now</Typography>
        {placements.length === 0 && <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>Not on the menu.</Typography>}
        <Stack spacing={1} sx={{ mb: 2 }}>
          {placements.map((p) => (
            <Stack key={p.nodeId} direction="row" spacing={1} alignItems="center">
              <Typography variant="body2" sx={{ flex: 1 }}>{p.path.join(' › ')}</Typography>
              {p.inherited ? (
                <Tooltip title="From the gold copy - every tenant has it. Change it in the gold copy's Menu Designer.">
                  <Chip size="small" variant="outlined" color="primary" label="Core" />
                </Tooltip>
              ) : (
                <Tooltip title="Remove this entry">
                  <IconButton size="small" onClick={() => remove(p.nodeId)} disabled={busy} aria-label="Remove from menu">
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              )}
            </Stack>
          ))}
        </Stack>
        <Typography variant="subtitle2" fontWeight={700} gutterBottom>Add an entry</Typography>
        <FormControl fullWidth size="small" sx={{ mb: 2 }}>
          <InputLabel id="place-parent">Under</InputLabel>
          <Select labelId="place-parent" label="Under" value={parentId} onChange={(e) => setParentId(e.target.value as string)}>
            <MenuItem value="">Top level</MenuItem>
            {sections(tree).map(({ node, path }) => (
              <MenuItem key={node.id} value={node.id}>
                {path.join(' › ')}{node.inherited ? ' (core)' : ''}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <TextField fullWidth size="small" label="Menu label" value={label} onChange={(e) => setLabel(e.target.value)} />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>Close</Button>
        <Button variant="contained" onClick={add} disabled={busy || !label.trim()}>Add to menu</Button>
      </DialogActions>
    </Dialog>
  );
};

export default PlaceOnMenuDialog;
