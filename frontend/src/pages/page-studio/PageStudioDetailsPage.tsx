import React, { useEffect, useState } from 'react';
import { useNavigate, useParams, Link as RouterLink } from 'react-router-dom';
import { Box, CircularProgress, Alert, Breadcrumbs, Link, Typography, TextField, IconButton, Tooltip } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import { PageStudioApi } from '../../api/pageStudio';
import type { CorePageDefinition } from '../../types/pageStudio';
import PageEditor from './PageEditor';
import NewPageWizard from './NewPageWizard';
import { useTenant } from '../../contexts/TenantContext';

const BLANK_PAGE: CorePageDefinition = {
  id: '',
  name: 'New Page',
  slug: 'new-page',
  env: 'production',
  layout: { root: 'root', nodes: { root: { id: 'root', type: 'Row', children: [] } } },
  components: {},
  dataSources: [],
  dataBindings: { sources: {}, bindings: [] },
  visibility: { roles: ['advisor'] },
  createdAt: '',
  updatedAt: '',
  version: 1,
};

/**
 * The details/tabs half of the list->detail split (see
 * PageStudioListPage.tsx). Reached at /page-studio/:id - "new" is a
 * reserved id that skips the fetch and opens PageEditor on a blank draft
 * instead, so creating a page doesn't need its own separate route/component.
 */
const PageStudioDetailsPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { tenant } = useTenant();
  const [page, setPage] = useState<CorePageDefinition | null>(null);
  const [loading, setLoading] = useState(id !== 'new');
  const [error, setError] = useState<string | null>(null);
  const [editingName, setEditingName] = useState(false);
  const [nameDraft, setNameDraft] = useState('');
  const [renaming, setRenaming] = useState(false);
  // A brand-new page picks its primary/related Business Objects and a
  // layout recommended for that object count before the editor opens; an
  // existing page (or one already past this dialog) skips it.
  const [showWizard, setShowWizard] = useState(id === 'new');

  // page_definitions never persists tenant_id to the client (the backend
  // hides it, json:"-" on PageDefinition.TenantID - a page's rows/queries
  // still need to be scoped to whoever is VIEWING it right now, not to
  // whichever tenant happened to author it), so this always overlays the
  // current session's tenant onto the draft rather than trusting
  // page.tenantId, which is never populated by a real API response.
  useEffect(() => {
    if (!tenant?.id) return;
    if (id === 'new') { setPage({ ...BLANK_PAGE, tenantId: tenant.id }); setLoading(false); return; }
    if (!id) return;
    let cancelled = false;
    setLoading(true);
    PageStudioApi.getPage(id)
      .then((p) => { if (!cancelled) setPage({ ...p, tenantId: tenant.id }); })
      .catch((err) => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load page'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [id, tenant?.id]);

  if (loading) return <Box sx={{ display: 'flex', justifyContent: 'center', p: 6 }}><CircularProgress /></Box>;
  if (error) return <Alert severity="error" sx={{ m: 3 }}>{error}</Alert>;
  if (!page) return null;

  const startRename = () => { setNameDraft(page.name); setEditingName(true); };
  const commitRename = async () => {
    const trimmed = nameDraft.trim();
    if (!trimmed || trimmed === page.name || !page.id) { setEditingName(false); return; }
    setRenaming(true);
    try {
      const updated = await PageStudioApi.updatePage(page.id, { name: trimmed });
      setPage(updated);
      setEditingName(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to rename page');
    } finally {
      setRenaming(false);
    }
  };

  // PageEditor takes `page` only as its initial state (a plain useState,
  // not synced to prop changes) - mounting it before the wizard has
  // finished, then merging the wizard's dataSources into this component's
  // own `page` state, updated a prop PageEditor had already stopped
  // reading. Not rendering PageEditor until the wizard is done (or was
  // never needed) sidesteps that instead of trying to force a resync.
  if (showWizard) {
    return (
      <NewPageWizard
        open={showWizard}
        onClose={() => setShowWizard(false)}
        onCreate={(draft) => {
          // The wizard saves the detail page for real up front when it also
          // built a paired list page (so the list's Table has something to
          // navigate to) - in that case `draft` already has an id, so this
          // just re-routes to the normal existing-page URL instead of
          // treating it as an unsaved draft PageEditor would try to POST.
          if (draft.id) { navigate(`/page-studio/${draft.id}`, { replace: true }); return; }
          setPage((prev) => (prev ? { ...prev, ...draft } : prev));
          setShowWizard(false);
        }}
      />
    );
  }

  return (
    <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', px: 2, pt: 1.5, pb: 1, gap: 1 }}>
        <Breadcrumbs sx={{ flex: 1, minWidth: 0 }}>
          <Link component={RouterLink} to="/page-studio" underline="hover" color="inherit" sx={{ fontWeight: 600 }}>
            Page Designer
          </Link>
          {editingName ? (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <TextField
                autoFocus
                size="small"
                variant="standard"
                value={nameDraft}
                onChange={(e) => setNameDraft(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') commitRename(); if (e.key === 'Escape') setEditingName(false); }}
                disabled={renaming}
              />
              <IconButton size="small" onClick={commitRename} disabled={renaming}><CheckIcon fontSize="small" /></IconButton>
              <IconButton size="small" onClick={() => setEditingName(false)} disabled={renaming}><CloseIcon fontSize="small" /></IconButton>
            </Box>
          ) : (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              <Typography variant="body2" fontWeight={700} color="text.primary" noWrap>
                {page.name || 'Untitled page'}
              </Typography>
              {id !== 'new' && (
                <Tooltip title="Rename page">
                  <IconButton size="small" onClick={startRename}><EditIcon sx={{ fontSize: 15 }} /></IconButton>
                </Tooltip>
              )}
            </Box>
          )}
        </Breadcrumbs>
      </Box>
      <Box sx={{ flex: 1, overflow: 'hidden' }}>
        <PageEditor
          page={page}
          onSave={(updated) => {
            setPage(updated);
            if (id === 'new' && updated.id) navigate(`/page-studio/${updated.id}`, { replace: true });
          }}
        />
      </Box>
    </Box>
  );
};

export default PageStudioDetailsPage;
