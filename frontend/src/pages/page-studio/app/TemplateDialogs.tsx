import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Box, Button, Chip, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, List, ListItemButton, ListItemText, MenuItem, Stack, TextField, Typography } from '@mui/material';
import { PageStudioApi, type InstantiatedTemplate, type PageTemplateSummary } from '../../../api/pageStudio';
import type { CorePageDefinition } from '../../../types/pageStudio';

/**
 * The templates gallery: start a page from a template (the page and the
 * fragments it needs, copied once), and save a page as a template. Both are
 * thin over the templates API; the page a template makes opens as an unsaved
 * draft, so nothing is created until the author saves it.
 */

const slugOf = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
const message = (e: unknown, fallback: string) => (e instanceof Error && e.message ? e.message : fallback);

export function TemplateGalleryDialog({ open, onClose, onChosen }: {
  open: boolean;
  onClose: () => void;
  onChosen: (draft: Partial<CorePageDefinition>) => void;
}) {
  const [templates, setTemplates] = useState<PageTemplateSummary[] | null>(null);
  const [category, setCategory] = useState('');
  const [picked, setPicked] = useState<PageTemplateSummary | null>(null);
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setTemplates(null); setPicked(null); setError(null); setName('');
    PageStudioApi.listTemplates().then(setTemplates).catch((e) => { setTemplates([]); setError(message(e, 'Could not load templates.')); });
  }, [open]);

  const categories = useMemo(() => Array.from(new Set((templates ?? []).map((t) => t.category))).sort(), [templates]);
  const shown = (templates ?? []).filter((t) => !category || t.category === category);

  const start = async () => {
    if (!picked) return;
    setBusy(true); setError(null);
    try {
      const made: InstantiatedTemplate = await PageStudioApi.instantiateTemplate(picked.slug, picked.version, { name: name.trim(), slug: slugOf(name) });
      onChosen(made.page);
    } catch (e) {
      setError(message(e, 'Could not start a page from this template.'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Start from a template</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {templates === null && <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}><CircularProgress size={24} /></Box>}
          {templates !== null && templates.length === 0 && !error && (
            <Typography variant="body2" color="text.secondary">No templates yet. Open a page's menu and choose "Save as template…" to make one.</Typography>
          )}
          {categories.length > 1 && (
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
              <Chip size="small" label="All" color={category ? 'default' : 'primary'} onClick={() => setCategory('')} />
              {categories.map((c) => <Chip key={c} size="small" label={c} color={category === c ? 'primary' : 'default'} onClick={() => setCategory(c)} />)}
            </Stack>
          )}
          {shown.length > 0 && (
            <List dense disablePadding sx={{ maxHeight: 320, overflow: 'auto', border: 1, borderColor: 'divider', borderRadius: 1 }}>
              {shown.map((t) => (
                <ListItemButton key={t.id} selected={picked?.id === t.id} onClick={() => { setPicked(t); setName((n) => n || t.name); }}>
                  <ListItemText
                    primary={<Stack direction="row" spacing={1} alignItems="center"><span>{t.name}</span>{t.isCore && <Chip size="small" label="Core" variant="outlined" />}</Stack>}
                    secondary={t.description || t.category}
                  />
                </ListItemButton>
              ))}
            </List>
          )}
          {picked && <TextField size="small" label="New page name" value={name} onChange={(e) => setName(e.target.value)} helperText={name.trim() ? `/${slugOf(name)}` : undefined} />}
          {error && <Alert severity="warning">{error}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" disabled={!picked || !name.trim() || !slugOf(name) || busy} onClick={start}>Start</Button>
      </DialogActions>
    </Dialog>
  );
}

const CATEGORIES = ['general', 'dashboard', 'list', 'detail', 'form', 'operations'];

export function SaveAsTemplateDialog({ page, onClose, onSaved }: {
  page: CorePageDefinition | null;
  onClose: () => void;
  onSaved: (t: PageTemplateSummary) => void;
}) {
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [category, setCategory] = useState('general');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!page) return;
    setName(page.name); setSlug(slugOf(page.name)); setDescription(page.description ?? ''); setCategory('general'); setError(null);
  }, [page]);

  const save = async () => {
    if (!page) return;
    setBusy(true); setError(null);
    try {
      onSaved(await PageStudioApi.publishTemplate({ slug, name: name.trim(), description: description.trim(), category, pageId: page.id }));
    } catch (e) {
      setError(message(e, 'Could not save the template.'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={!!page} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Save as template</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          <Typography variant="body2" color="text.secondary">
            Saves this page, and the fragments it uses, as it is now. Changing the page later does not change the template; saving again makes a new version.
          </Typography>
          <TextField size="small" label="Template name" value={name} onChange={(e) => { setName(e.target.value); setSlug(slugOf(e.target.value)); }} />
          <TextField size="small" label="Template id" value={slug} onChange={(e) => setSlug(e.target.value)} helperText="Lowercase letters, digits and hyphens." />
          <TextField size="small" label="Description" value={description} onChange={(e) => setDescription(e.target.value)} multiline minRows={2} />
          <TextField select size="small" label="Category" value={category} onChange={(e) => setCategory(e.target.value)}>
            {CATEGORIES.map((c) => <MenuItem key={c} value={c}>{c}</MenuItem>)}
          </TextField>
          {error && <Alert severity="warning">{error}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" disabled={!name.trim() || !/^[a-z][a-z0-9-]{1,62}$/.test(slug) || busy} onClick={save}>Save template</Button>
      </DialogActions>
    </Dialog>
  );
}
