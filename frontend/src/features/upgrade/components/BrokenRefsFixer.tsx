import { useEffect, useState } from 'react';
import { Alert, Box, Button, Chip, CircularProgress, List, ListItem, ListItemText, Stack, TextField, Typography } from '@mui/material';
import { applyFixes, listBrokenRefs } from '../api';
import getErrorMessage from '../../../utils/errors';

type BrokenRef = { path: string; reason: string; suggestions: string[] };

export default function BrokenRefsFixer({ version }: { version: string }) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [items, setItems] = useState<BrokenRef[]>([]);
  const [patches, setPatches] = useState<Record<string, string>>({});
  const [applied, setApplied] = useState(false);

  useEffect(() => {
    if (!version) return;
    setApplied(false);
    setLoading(true); setError(null);
    (async () => {
      try {
        const data = await listBrokenRefs(version);
        setItems(data);
        setPatches({});
      } catch (e: unknown) {
        setError(getErrorMessage(e, 'Failed to load broken references'));
      } finally { setLoading(false); }
    })();
  }, [version]);

  const setPatch = (path: string, value: string) => {
    setPatches((p) => ({ ...p, [path]: value }));
  };

  const onApply = async () => {
    try {
      await applyFixes(version, patches);
      setApplied(true);
    } catch (e: unknown) {
      setError(getErrorMessage(e, 'Failed to apply fixes'));
    }
  };

  if (!version) return <Typography variant="body2">Select a version</Typography>;
  if (loading) return <CircularProgress size={18} />;
  if (error) return <Alert severity="error">{error}</Alert>;

  return (
    <Box>
      {items.length === 0 ? (
        <Typography variant="body2" color="text.secondary">No broken references</Typography>
      ) : (
        <List dense>
          {items.map((it) => (
            <ListItem key={it.path} alignItems="flex-start" disableGutters sx={{ mb: 1 }}>
              <ListItemText
                primary={<>
                  <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>{it.path}</Typography>
                  <Typography variant="caption" color="text.secondary">{it.reason}</Typography>
                </>}
                secondary={
                  <Stack spacing={1} sx={{ mt: 1 }}>
                    <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap' }}>
                      {it.suggestions?.map((s) => (
                        <Chip key={s} size="small" label={s} onClick={() => setPatch(it.path, s)} />
                      ))}
                    </Stack>
                    <TextField
                      size="small"
                      placeholder="Replacement path"
                      value={patches[it.path] || ''}
                      onChange={(e) => setPatch(it.path, e.target.value)}
                    />
                  </Stack>
                }
              />
            </ListItem>
          ))}
        </List>
      )}

      <Stack direction="row" spacing={1}>
        <Button variant="outlined" size="small" disabled={Object.keys(patches).length === 0} onClick={onApply}>Apply fixes</Button>
        {applied && <Chip color="success" size="small" label="Applied" />}
      </Stack>
    </Box>
  );
}
