import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Alert,
  AlertTitle,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  CircularProgress,
  Box,
  Divider,
} from '@mui/material';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import LayersIcon from '@mui/icons-material/Layers';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import NavigationIcon from '@mui/icons-material/Navigation';
import BusinessIcon from '@mui/icons-material/Business';
import {
  SavedQuery,
  SavedQueryUsageReport,
  getSavedQueryUsage,
  deleteSavedQuery,
  patchSavedQueryStatus,
} from '../services/savedQueryApi';

interface Props {
  query: SavedQuery | null;
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function DeleteQueryConfirmDialog({
  query,
  open,
  onClose,
  onSuccess,
}: Props) {
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [usage, setUsage] = useState<SavedQueryUsageReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open && query) {
      setLoading(true);
      setError(null);
      getSavedQueryUsage(query.id)
        .then((report) => setUsage(report))
        .catch((err) => setError(err.message || 'Failed to check query usage'))
        .finally(() => setLoading(false));
    } else {
      setUsage(null);
      setError(null);
    }
  }, [open, query]);

  if (!query) return null;

  const handleDelete = async () => {
    setDeleting(true);
    setError(null);
    try {
      await deleteSavedQuery(query.id);
      onSuccess();
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to delete saved query');
    } finally {
      setDeleting(false);
    }
  };

  const handleDeprecate = async () => {
    setDeleting(true);
    setError(null);
    try {
      await patchSavedQueryStatus(query.id, 'deprecated');
      onSuccess();
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to deprecate saved query');
    } finally {
      setDeleting(false);
    }
  };

  const hasAdoptions = usage?.references.some((r) => r.type === 'core_adoption');
  const inUse = usage?.inUse ?? false;

  return (
    <Dialog open={open} onClose={deleting ? undefined : onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ fontWeight: 700 }}>
        {inUse ? 'Cannot Delete Query in Use' : `Delete "${query.name}"?`}
      </DialogTitle>

      <DialogContent dividers>
        {loading ? (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', py: 4 }}>
            <CircularProgress size={28} sx={{ mr: 2 }} />
            <Typography variant="body2" color="text.secondary">
              Scanning for page and query references...
            </Typography>
          </Box>
        ) : (
          <StackWrapper>
            {error && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {error}
              </Alert>
            )}

            {inUse ? (
              <>
                <Alert severity="warning" icon={<WarningAmberIcon />} sx={{ mb: 2 }}>
                  <AlertTitle sx={{ fontWeight: 700 }}>Query is Referenced Across the Workspace</AlertTitle>
                  This query cannot be deleted because it is currently used by{' '}
                  <strong>{usage?.references.length}</strong> active component(s). Deleting it would break live pages.
                </Alert>

                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                  Referenced In:
                </Typography>

                <List dense sx={{ bgcolor: 'background.paper', borderRadius: 1, border: 1, borderColor: 'divider' }}>
                  {usage?.references.map((ref, idx) => (
                    <ListItem key={idx}>
                      <ListItemIcon sx={{ minWidth: 32 }}>
                        {ref.type === 'page_published' && <LayersIcon color="success" fontSize="small" />}
                        {ref.type === 'page_draft' && <AutoFixHighIcon color="info" fontSize="small" />}
                        {ref.type === 'drill_through_target' && <NavigationIcon color="primary" fontSize="small" />}
                        {ref.type === 'core_adoption' && <BusinessIcon color="warning" fontSize="small" />}
                      </ListItemIcon>
                      <ListItemText
                        primary={
                          <Typography variant="body2" fontWeight={600}>
                            {ref.name || ref.ID || ref.sourceQueryID || 'Core Adoption'}
                            {ref.version ? ` (v${ref.version})` : ''}
                          </Typography>
                        }
                        secondary={
                          <Typography variant="caption" color="text.secondary">
                            {ref.type.replace('_', ' ').toUpperCase()} • {ref.location || (ref.count ? `${ref.count} adoption(s)` : ref.target)}
                          </Typography>
                        }
                      />
                    </ListItem>
                  ))}
                </List>

                {query.isCore && hasAdoptions && (
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
                    Tip: As a master tenant, you can <strong>Deprecate</strong> this core query so no new adoptions are created while preserving existing client instances.
                  </Typography>
                )}
              </>
            ) : (
              <>
                <Typography variant="body1" sx={{ mb: 1 }}>
                  Are you sure you want to delete <strong>{query.name}</strong>?
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  This query will be archived and hidden from lists and designer pickers. Existing bookmarks and run paths will degrade gracefully.
                </Typography>
              </>
            )}
          </StackWrapper>
        )}
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2 }}>
        <Button onClick={onClose} disabled={deleting} color="inherit">
          Cancel
        </Button>

        {inUse && query.isCore && hasAdoptions && (
          <Button
            onClick={handleDeprecate}
            disabled={deleting}
            variant="contained"
            color="warning"
          >
            {deleting ? 'Deprecating...' : 'Deprecate Query'}
          </Button>
        )}

        <Button
          onClick={handleDelete}
          disabled={inUse || loading || deleting}
          variant="contained"
          color="error"
        >
          {deleting ? 'Deleting...' : 'Delete Query'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function StackWrapper({ children }: { children: React.ReactNode }) {
  return <Box sx={{ display: 'flex', flexDirection: 'column' }}>{children}</Box>;
}
