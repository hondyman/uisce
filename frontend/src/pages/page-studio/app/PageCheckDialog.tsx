import React from 'react';
import {
  Alert, Badge, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, List, ListItemButton, ListItemText, Stack, Tooltip, Typography,
} from '@mui/material';
import FactCheckIcon from '@mui/icons-material/FactCheck';
import type { PageIssue } from './pageChecker';

/** The toolbar button: how many problems the page has now, errors in red. */
export function PageCheckButton({ issues, onClick }: { issues: PageIssue[]; onClick: () => void }) {
  const errors = issues.filter((i) => i.severity === 'error').length;
  const warnings = issues.length - errors;
  const tip = errors ? `${errors} error${errors === 1 ? '' : 's'} - fix before publishing` : warnings ? `${warnings} warning${warnings === 1 ? '' : 's'}` : 'No problems found';
  return (
    <Tooltip title={tip}>
      <Badge color={errors ? 'error' : 'warning'} badgeContent={errors || warnings} invisible={!issues.length}>
        <Button size="small" variant="outlined" color={errors ? 'error' : 'inherit'} startIcon={<FactCheckIcon />} onClick={onClick}>Check</Button>
      </Badge>
    </Tooltip>
  );
}

/** Every problem the checker found; picking one selects the widget it is on. */
export function PageCheckDialog({ open, issues, blockedPublish, onClose, onPick }: {
  open: boolean; issues: PageIssue[]; blockedPublish?: boolean; onClose: () => void; onPick: (componentId: string) => void;
}) {
  const errors = issues.filter((i) => i.severity === 'error').length;
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle>Page check</DialogTitle>
      <DialogContent dividers>
        {blockedPublish && <Alert severity="error" sx={{ mb: 2 }}>Not published: fix the errors below first.</Alert>}
        {issues.length === 0 ? (
          <Alert severity="success">No problems found. Bindings, queries, operations, layout and overlays all check out.</Alert>
        ) : (
          <>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              {errors ? `${errors} error${errors === 1 ? '' : 's'} (these block publishing)` : 'No errors'}
              {issues.length - errors ? ` · ${issues.length - errors} warning${issues.length - errors === 1 ? '' : 's'}` : ''}
            </Typography>
            <List dense>
              {issues.map((i, k) => (
                <ListItemButton key={k} disabled={!i.componentId} onClick={() => { if (i.componentId) { onPick(i.componentId); onClose(); } }}
                  sx={{ alignItems: 'flex-start', '&.Mui-disabled': { opacity: 1 } }}>
                  <ListItemText
                    primary={
                      <Stack direction="row" spacing={1} alignItems="center">
                        <Chip size="small" color={i.severity === 'error' ? 'error' : 'warning'} label={i.severity === 'error' ? 'Error' : 'Warning'} />
                        <Typography variant="body2">{i.message}</Typography>
                      </Stack>
                    }
                    secondary={<Typography variant="caption" color="text.secondary" fontFamily="monospace">{i.where}</Typography>} />
                </ListItemButton>
              ))}
            </List>
          </>
        )}
      </DialogContent>
      <DialogActions><Button onClick={onClose}>Close</Button></DialogActions>
    </Dialog>
  );
}
