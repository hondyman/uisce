import React, { useEffect, useState } from 'react';
import {
  Dialog, DialogTitle, DialogContent, DialogActions, Button, Box, Typography,
  FormControl, InputLabel, Select, MenuItem, Checkbox, FormControlLabel, Stack, CircularProgress,
  Alert, Grid, Card, CardActionArea, CardContent, Chip,
} from '@mui/material';
import { apiClient } from '../../utils/apiClient';
import { useTenant } from '../../contexts/TenantContext';
import { LAYOUT_TEMPLATES, LayoutTemplate } from './layoutTemplates';
import { buildBODataSource } from './generatePageDraft';
import { PageStudioApi } from '../../api/pageStudio';
import { fetchBusinessObjectBindings, fetchBOTerms } from '../../features/query-builder/services/queryBuilderApi';
import type { SemanticTermView } from '../../features/query-builder/types/queryDef';
import type { CorePageDefinition, ComponentDefinition } from '../../types/pageStudio';

const slugify = (name: string) =>
  `${name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '') || 'page'}-${Math.random().toString(36).slice(2, 6)}`;

interface BOOption {
  id: string;
  key: string;
  name: string;
  display_name: string;
}

interface RelatedBusinessObject {
  targetObjectId: string;
  relatedObjectName: string;
  relationshipType: string;
  cardinality: string;
}

interface NewPageWizardProps {
  open: boolean;
  onClose: () => void;
  onCreate: (draft: Partial<CorePageDefinition>) => void;
}

/**
 * Recommends which layout templates fit a given object count best, ordered
 * most-to-least fitting. Not a hard restriction (the gallery still shows
 * every template) - just which ones get the "Recommended" badge and sort
 * first, since a page bound to more objects needs more sections to put
 * them in.
 */
const recommendedTemplateIds = (objectCount: number): string[] => {
  if (objectCount <= 1) return ['single-column', 'dashboard-grid'];
  if (objectCount === 2) return ['two-column', 'master-detail', 'two-column-side-panel'];
  return ['dashboard-grid', 'three-column'];
};

const Thumbnail: React.FC<{ rows: number[][] }> = ({ rows }) => (
  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, height: 48, width: '100%' }}>
    {rows.map((row, i) => (
      <Box key={i} sx={{ display: 'flex', gap: 0.5, flex: 1 }}>
        {row.map((weight, j) => (
          <Box key={j} sx={{ flex: weight, bgcolor: 'action.selected', borderRadius: 0.5 }} />
        ))}
      </Box>
    ))}
  </Box>
);

/**
 * Replaces the old flow of picking a blank layout template with no idea
 * what data the page is for. Step 1 asks which Business Object (and which
 * of its real related objects) the page is about; step 2 recommends
 * layout templates sized to that object count (one object needs one
 * section, three objects want a dashboard grid or three-column split) and
 * pre-wires a data source per selected object, so DataBindingsPanel and
 * every widget the author drags in afterward already have something to
 * bind to.
 */
const NewPageWizard: React.FC<NewPageWizardProps> = ({ open, onClose, onCreate }) => {
  const { tenant } = useTenant();
  const [step, setStep] = useState<0 | 1 | 2>(0);
  const [bos, setBos] = useState<BOOption[]>([]);
  const [primaryId, setPrimaryId] = useState('');
  const [related, setRelated] = useState<RelatedBusinessObject[]>([]);
  const [selectedRelated, setSelectedRelated] = useState<Set<string>>(new Set());
  const [loadingRelated, setLoadingRelated] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // The page this wizard builds is the DETAIL page - the one worth
  // spending design time on (a single record's own page: a form, related
  // records, actions). A "browse everything" list page is comparatively
  // trivial - a filter + a table - so it's a checkbox that gets built and
  // saved for real automatically, already wired to open whichever record
  // is clicked into this detail page, rather than a second thing the user
  // has to design by hand.
  const [alsoCreateList, setAlsoCreateList] = useState(true);
  const [createdList, setCreatedList] = useState<{ id: string; name: string; slug: string } | null>(null);
  const [savedDetail, setSavedDetail] = useState<CorePageDefinition | null>(null);
  // Facets = which dimension fields become filters (one Slicer each) on the
  // auto-built list page - so "browse all Orders" can narrow by Status,
  // Region, etc. rather than only free-text search.
  const [primaryBindingId, setPrimaryBindingId] = useState('');
  const [facetOptions, setFacetOptions] = useState<SemanticTermView[]>([]);
  const [loadingFacets, setLoadingFacets] = useState(false);
  const [selectedFacets, setSelectedFacets] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!open) return;
    setStep(0);
    setPrimaryId('');
    setRelated([]);
    setSelectedRelated(new Set());
    setError(null);
    setAlsoCreateList(true);
    setCreatedList(null);
    setSavedDetail(null);
    setPrimaryBindingId('');
    setFacetOptions([]);
    setSelectedFacets(new Set());
    apiClient<unknown>('/business-objects', { headers: tenant?.id ? { 'X-Tenant-ID': tenant.id } : undefined })
      .then((data) => {
        const rawList = Array.isArray(data) ? data : data && typeof data === 'object' ? Object.values(data as Record<string, unknown>) : [];
        setBos(
          rawList
            .filter((item): item is Record<string, unknown> => !!item && typeof item === 'object')
            .map((item) => ({
              id: String(item.id ?? item.name),
              key: String(item.key ?? item.technicalName ?? item.technical_name ?? item.name ?? ''),
              name: String(item.name ?? ''),
              display_name: String(item.displayName ?? item.display_name ?? item.name ?? ''),
            }))
        );
      })
      .catch(() => setBos([]));
  }, [open, tenant?.id]);

  const primary = bos.find((b) => b.id === primaryId);

  const handlePickPrimary = (id: string) => {
    setPrimaryId(id);
    setSelectedRelated(new Set());
    setRelated([]);
    setPrimaryBindingId('');
    setFacetOptions([]);
    setSelectedFacets(new Set());
    if (!id) return;
    setLoadingRelated(true);
    apiClient<{ relatedObjects?: RelatedBusinessObject[] }>(`/business-objects/${id}/relationships`, {
      headers: tenant?.id ? { 'X-Tenant-ID': tenant.id } : undefined,
    })
      .then((data) => setRelated(data?.relatedObjects || []))
      .catch(() => setRelated([]))
      .finally(() => setLoadingRelated(false));

    setLoadingFacets(true);
    fetchBusinessObjectBindings(id)
      .then(async (bindings) => {
        const binding = bindings.find((b) => b.isDefault) || bindings[0];
        if (!binding) return;
        setPrimaryBindingId(binding.bindingId);
        const terms = await fetchBOTerms(id, binding.bindingId);
        const dims = terms.filter((t) => t.role === 'DIMENSION');
        setFacetOptions(dims);
        // A couple of default facets so the list page isn't filter-less out
        // of the box; the author can add/remove more before creating it.
        setSelectedFacets(new Set(dims.slice(0, 2).map((t) => t.termNodeId)));
      })
      .catch(() => setFacetOptions([]))
      .finally(() => setLoadingFacets(false));
  };

  const toggleRelated = (targetObjectId: string) => {
    setSelectedRelated((prev) => {
      const next = new Set(prev);
      if (next.has(targetObjectId)) next.delete(targetObjectId); else next.add(targetObjectId);
      return next;
    });
  };

  const toggleFacet = (termNodeId: string) => {
    setSelectedFacets((prev) => {
      const next = new Set(prev);
      if (next.has(termNodeId)) next.delete(termNodeId); else next.add(termNodeId);
      return next;
    });
  };

  const objectCount = 1 + selectedRelated.size;
  const recommended = recommendedTemplateIds(objectCount);
  const orderedTemplates = [...LAYOUT_TEMPLATES].sort((a, b) => {
    const ai = recommended.indexOf(a.id);
    const bi = recommended.indexOf(b.id);
    if (ai === -1 && bi === -1) return 0;
    if (ai === -1) return 1;
    if (bi === -1) return -1;
    return ai - bi;
  });

  const handlePickTemplate = async (template: LayoutTemplate) => {
    if (!primary) return;
    setCreating(true);
    setError(null);
    try {
      // Related BOs need their own name/display for the data source label -
      // the relationships response only has relatedObjectName (the driver
      // table name, e.g. "order_allocation"), close enough for both here.
      const relatedList = related.filter((r) => selectedRelated.has(r.targetObjectId));
      const relatedBoIds = relatedList.map((r) => r.targetObjectId);
      const primarySource = await buildBODataSource(primary.id, primary.key, primary.name, primary.display_name || primary.name, relatedBoIds);
      const relatedSources = await Promise.all(
        relatedList.map((r) => buildBODataSource(r.targetObjectId, r.relatedObjectName, r.relatedObjectName, r.relatedObjectName, []))
      );

      const { layout, sectionIds } = template.build();
      // No widgets are placed automatically here (unlike AI generation) -
      // the author drags those in themselves - but every section is ready
      // to receive one bound to whichever of these data sources fits.
      void sectionIds;
      const components: Record<string, ComponentDefinition> = {};
      const boLabel = primary.display_name || primary.name;

      const detailName = `${boLabel} Detail`;
      const detailDraft: Partial<CorePageDefinition> = {
        name: detailName,
        slug: slugify(detailName),
        layout,
        components,
        dataSources: [primarySource, ...relatedSources],
      };

      if (!alsoCreateList) {
        onCreate(detailDraft);
        return;
      }

      // The detail page needs a real id/slug before the list page's Table
      // can be wired to navigate to it, so it's saved now rather than left
      // as a draft the author saves later - the list page below is real
      // and working the moment this wizard closes.
      const detail = await PageStudioApi.savePage(detailDraft);

      const facets = facetOptions.filter((t) => selectedFacets.has(t.termNodeId));
      const facetIds = facets.map(() => `comp_${Math.random().toString(36).slice(2, 8)}`);
      const facetsRowId = 'row_facets';
      const tableId = `comp_${Math.random().toString(36).slice(2, 8)}`;
      const { layout: listLayout } = LAYOUT_TEMPLATES[0].build(); // single-column: facets above table
      const rootChildren = facetIds.length > 0 ? [facetsRowId, tableId] : [tableId];
      listLayout.nodes = {
        ...listLayout.nodes,
        root: { ...listLayout.nodes.root, children: rootChildren },
        ...(facetIds.length > 0
          ? { [facetsRowId]: { id: facetsRowId, type: 'Row', children: facetIds } }
          : {}),
      };
      const listComponents: Record<string, ComponentDefinition> = {};
      facets.forEach((facet, i) => {
        listComponents[facetIds[i]] = {
          id: facetIds[i],
          type: 'Slicer',
          label: facet.displayName,
          props: { dataSourceId: primarySource.id, dimensionTermIds: [facet.termNodeId] },
        };
      });
      listComponents[tableId] = {
        id: tableId,
        type: 'Table',
        label: `All ${boLabel}s`,
        props: { dataSourceId: primarySource.id, rowClickAction: 'navigate', rowClickTargetSlug: detail.slug },
      };
      const listName = `${boLabel} List`;
      const list = await PageStudioApi.savePage({
        name: listName,
        slug: slugify(listName),
        layout: listLayout,
        components: listComponents,
        dataSources: [primarySource],
        description: `List of all ${boLabel} records, with search${facets.length > 0 ? ' and facet filters (' + facets.map((f) => f.displayName).join(', ') + ')' : ''} - each row opens ${detail.name}.`,
      });

      setSavedDetail(detail);
      setCreatedList({ id: list.id, name: list.name, slug: list.slug });
      setStep(2);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to set up the page');
    } finally {
      setCreating(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{step === 0 ? 'What is this page about?' : step === 1 ? 'Choose a layout for the detail page' : 'List page created'}</DialogTitle>
      <DialogContent>
        {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>{error}</Alert>}
        {step === 0 ? (
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Typography variant="body2" color="text.secondary">
              Pick the main Business Object this page is for, and any related objects you want available too - the layout step next will size itself to how many you pick.
            </Typography>
            <FormControl fullWidth>
              <InputLabel id="wizard-primary-bo-label">Main Business Object</InputLabel>
              <Select
                labelId="wizard-primary-bo-label"
                label="Main Business Object"
                value={primaryId}
                onChange={(e) => handlePickPrimary(e.target.value as string)}
              >
                {bos.map((bo) => (
                  <MenuItem key={bo.id} value={bo.id}>{bo.display_name || bo.name}</MenuItem>
                ))}
              </Select>
            </FormControl>

            {primaryId && (
              <Box>
                <Typography variant="overline" color="text.secondary" fontWeight="bold">Related objects (optional)</Typography>
                {loadingRelated ? (
                  <Box sx={{ display: 'flex', justifyContent: 'center', p: 2 }}><CircularProgress size={20} /></Box>
                ) : related.length === 0 ? (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
                    No related objects found for this Business Object.
                  </Typography>
                ) : (
                  <Stack sx={{ mt: 0.5 }}>
                    {related.map((r) => (
                      <FormControlLabel
                        key={r.targetObjectId}
                        control={<Checkbox checked={selectedRelated.has(r.targetObjectId)} onChange={() => toggleRelated(r.targetObjectId)} />}
                        label={
                          <span>
                            {r.relatedObjectName}{' '}
                            <Typography component="span" variant="caption" color="text.secondary">
                              ({r.relationshipType}, {r.cardinality})
                            </Typography>
                          </span>
                        }
                      />
                    ))}
                  </Stack>
                )}
              </Box>
            )}
            <FormControlLabel
              control={<Checkbox checked={alsoCreateList} onChange={(e) => setAlsoCreateList(e.target.checked)} />}
              label={
                <span>
                  Also create a list/summary page
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                    A searchable, filterable table of every record, built automatically and already wired so
                    clicking a row opens the detail page you design next. Uncheck this if you only need the one page.
                  </Typography>
                </span>
              }
            />

            {alsoCreateList && primaryId && (
              <Box>
                <Typography variant="overline" color="text.secondary" fontWeight="bold">Facets (filters on the list page)</Typography>
                {loadingFacets ? (
                  <Box sx={{ display: 'flex', justifyContent: 'center', p: 2 }}><CircularProgress size={20} /></Box>
                ) : facetOptions.length === 0 ? (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
                    No filterable fields found for this Business Object.
                  </Typography>
                ) : (
                  <Stack direction="row" flexWrap="wrap" sx={{ mt: 0.5 }}>
                    {facetOptions.map((t) => (
                      <FormControlLabel
                        key={t.termNodeId}
                        sx={{ width: { xs: '100%', sm: '50%' }, mr: 0 }}
                        control={<Checkbox size="small" checked={selectedFacets.has(t.termNodeId)} onChange={() => toggleFacet(t.termNodeId)} />}
                        label={<Typography variant="body2">{t.displayName}</Typography>}
                      />
                    ))}
                  </Stack>
                )}
              </Box>
            )}
          </Stack>
        ) : step === 1 ? (
          <Box>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              Recommended for {objectCount} object{objectCount === 1 ? '' : 's'} ({[primary?.display_name, ...related.filter((r) => selectedRelated.has(r.targetObjectId)).map((r) => r.relatedObjectName)].filter(Boolean).join(', ')}):
            </Typography>
            <Grid container spacing={2}>
              {orderedTemplates.map((template) => {
                const isRecommended = recommended.includes(template.id);
                return (
                  <Grid key={template.id} size={{ xs: 12, sm: 6 }}>
                    <Card variant="outlined" sx={{ borderRadius: 2, height: '100%', borderColor: isRecommended ? 'primary.main' : undefined }}>
                      <CardActionArea sx={{ p: 2, height: '100%' }} onClick={() => handlePickTemplate(template)} disabled={creating}>
                        <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                          <Thumbnail rows={template.thumbnail} />
                          {isRecommended && <Chip size="small" color="primary" label="Recommended" sx={{ ml: 1 }} />}
                        </Stack>
                        <CardContent sx={{ p: 0, pt: 1.5, '&:last-child': { pb: 0 } }}>
                          <Typography variant="subtitle2" fontWeight={700}>{template.name}</Typography>
                          <Typography variant="caption" color="text.secondary">{template.description}</Typography>
                        </CardContent>
                      </CardActionArea>
                    </Card>
                  </Grid>
                );
              })}
            </Grid>
          </Box>
        ) : (
          <Stack spacing={2} sx={{ mt: 1 }}>
            <Alert severity="success">
              Created "{createdList?.name}" (/{createdList?.slug}) - a searchable, filterable table of every record,
              already wired so clicking a row opens the detail page you're about to design.
            </Alert>
            <Typography variant="body2" color="text.secondary">
              Continue to the detail page to lay out what one record's page should show - a form, related records,
              charts. Find the list page anytime from Page Designer.
            </Typography>
          </Stack>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={creating}>Cancel</Button>
        {step === 0 ? (
          <Button variant="contained" disabled={!primaryId} onClick={() => setStep(1)}>Next</Button>
        ) : step === 1 ? (
          <Button onClick={() => setStep(0)} disabled={creating}>Back</Button>
        ) : (
          <Button variant="contained" onClick={() => savedDetail && onCreate(savedDetail)}>
            Continue to Detail Page
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};

export default NewPageWizard;
