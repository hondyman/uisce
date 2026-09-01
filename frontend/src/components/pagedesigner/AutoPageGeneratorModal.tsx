import React, { useState, useEffect } from 'react';
import {
  Dialog, DialogTitle, DialogContent, DialogActions,
  Button, TextField, FormControl, InputLabel, Select,
  MenuItem, Checkbox, FormControlLabel, FormGroup, Typography,
  Divider, Stack, Box, Chip
} from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import { DiscoveredSubtype, DiscoveredRelationship, PageGroupSpec } from './AutoPageTypes';

interface AutoPageGeneratorModalProps {
  open: boolean;
  boKey: string;
  onClose: () => void;
  onGenerated: (pageGroup: PageGroupSpec) => void;
}

export const AutoPageGeneratorModal: React.FC<AutoPageGeneratorModalProps> = ({
  open,
  boKey,
  onClose,
  onGenerated,
}) => {
  const [title, setTitle] = useState(`${boKey.toUpperCase()} Management Hub`);
  const [topology, setTopology] = useState<'TABBED_BY_SUBTYPE' | 'SINGLE_SCROLL_PANE'>('TABBED_BY_SUBTYPE');
  const [allowCreate, setAllowCreate] = useState(true);
  const [allowUpdate, setAllowUpdate] = useState(true);
  const [allowDelete, setAllowDelete] = useState(false);

  const [discoveredSubtypes, setDiscoveredSubtypes] = useState<DiscoveredSubtype[]>([]);
  const [selectedSubtypes, setSelectedSubtypes] = useState<string[]>([]);
  const [discoveredRelationships, setDiscoveredRelationships] = useState<DiscoveredRelationship[]>([]);
  const [selectedRelationships, setSelectedRelationships] = useState<string[]>([]);
  const [generating, setGenerating] = useState(false);

  useEffect(() => {
    if (!boKey || !open) return;
    setTitle(`${boKey.toUpperCase()} Management Hub`);
    // Inspect BO graph topology (Subtypes + 1:1/1:N Relationships)
    fetch(`/api/v1/bo/${boKey}/topology-summary`)
      .then(async (res) => {
        if (!res.ok) throw new Error('Topology endpoint unavailable');
        return res.json();
      })
      .then((data) => {
        const subtypes = data.subtypes || [];
        setDiscoveredSubtypes(subtypes);
        setSelectedSubtypes(subtypes.map((s: DiscoveredSubtype) => s.subtypeCode));

        const rels = data.relationships || [];
        setDiscoveredRelationships(rels);
        setSelectedRelationships(rels.map((r: DiscoveredRelationship) => r.relKey));
      })
      .catch(() => {
        // Fallback default subtypes & relations
        const defaultSubtypes: DiscoveredSubtype[] = [
          { subtypeCode: 'institutional', displayName: 'Institutional Client', isSatelliteTable: false, assignedFieldsCount: 14 },
          { subtypeCode: 'retail_wealth', displayName: 'Retail Wealth', isSatelliteTable: false, assignedFieldsCount: 12 },
          { subtypeCode: 'sma', displayName: 'SMA Managed Account', isSatelliteTable: false, assignedFieldsCount: 10 },
        ];
        const defaultRels: DiscoveredRelationship[] = [
          { relKey: 'mandate_info', relName: 'Account Mandate Info', targetBoKey: 'mandate', targetBoName: 'Mandate', cardinality: '1:1', isSubtypeScoped: false },
          { relKey: 'positions', relName: 'Account Positions', targetBoKey: 'position', targetBoName: 'Position', cardinality: '1:N', isSubtypeScoped: false },
          { relKey: 'trade_orders', relName: 'Trade Orders', targetBoKey: 'trade_order', targetBoName: 'Trade Order', cardinality: '1:N', isSubtypeScoped: false },
        ];
        setDiscoveredSubtypes(defaultSubtypes);
        setSelectedSubtypes(defaultSubtypes.map((s) => s.subtypeCode));
        setDiscoveredRelationships(defaultRels);
        setSelectedRelationships(defaultRels.map((r) => r.relKey));
      });
  }, [boKey, open]);

  const handleGenerate = async () => {
    setGenerating(true);
    try {
      const res = await fetch('/api/v1/page-designer/auto-compile', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          rootBoKey: boKey,
          pageGroupTitle: title,
          layoutTopology: topology,
          includeSubtypes: selectedSubtypes,
          includeRelationships: selectedRelationships,
          crudEntitlements: { allowCreate, allowUpdate, allowDelete },
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data: PageGroupSpec = await res.json();
      onGenerated(data);
      onClose();
    } catch (err) {
      console.warn('Backend auto-compile fallback to client-side compiler:', err);
      // Client-side fallback compilation
      const fallbackSpec: PageGroupSpec = {
        pageGroupId: `pg_${boKey}_${Date.now()}`,
        rootBoKey: boKey,
        title: title,
        tabs: [
          {
            tabId: 'core',
            tabTitle: 'Overview (Core)',
            sections: [
              {
                id: 'sec_master_core',
                title: 'Master Entity Information',
                flow: 'COLUMN',
                widgets: [
                  {
                    id: 'w_form_core',
                    type: 'BO_FORM',
                    title: `${boKey.toUpperCase()} Overview`,
                    boKey: boKey,
                    gridSpan: { xs: 12, md: 12, lg: 12 },
                    subscribedParams: ['selected_id'],
                    entitlements: { allowCreate, allowUpdate, allowDelete },
                  },
                ],
              },
              ...selectedRelationships.map((relKey) => ({
                id: `sec_rel_${relKey}`,
                title: `Related: ${relKey.replace(/_/g, ' ')}`,
                flow: 'COLUMN' as const,
                widgets: [
                  {
                    id: `w_grid_${relKey}`,
                    type: 'BO_GRID' as const,
                    title: `Associated ${relKey.replace(/_/g, ' ').toUpperCase()}`,
                    boKey: relKey,
                    gridSpan: { xs: 12, md: 12, lg: 12 },
                    subscribedParams: ['selected_id'],
                    events: {
                      onRowSelect: [{ targetChannel: 'selected_child_id', sourcePropertyKey: 'id', actionType: 'SET_PARAMETER' as const }],
                      onRowDoubleClick: [{ targetChannel: 'active_modal_record_id', sourcePropertyKey: 'id', actionType: 'LAUNCH_MODAL_FORM' as const }],
                    },
                  },
                ],
              })),
            ],
          },
          ...(topology === 'TABBED_BY_SUBTYPE'
            ? selectedSubtypes.map((stCode) => ({
                tabId: stCode,
                tabTitle: `${stCode.replace(/_/g, ' ')} Subtype`,
                subtypeCode: stCode,
                sections: [
                  {
                    id: `sec_master_${stCode}`,
                    title: `${stCode} Subtype Information`,
                    flow: 'COLUMN' as const,
                    widgets: [
                      {
                        id: `w_form_${stCode}`,
                        type: 'BO_FORM' as const,
                        title: `${stCode.toUpperCase()} Subtype Details`,
                        boKey: boKey,
                        gridSpan: { xs: 12, md: 12, lg: 12 },
                        subscribedParams: ['selected_id'],
                        entitlements: { allowCreate, allowUpdate, allowDelete },
                      },
                    ],
                  },
                ],
              }))
            : []),
        ],
      };
      onGenerated(fallbackSpec);
      onClose();
    } finally {
      setGenerating(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth PaperProps={{ sx: { bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B' } }}>
      <DialogTitle sx={{ borderBottom: '1px solid #1E293B', display: 'flex', alignItems: 'center', gap: 1 }}>
        <AutoAwesomeIcon sx={{ color: '#00D4FF' }} />
        <Typography variant="subtitle1" fontWeight={700}>Self-Assembling Page Generator: {boKey}</Typography>
      </DialogTitle>

      <DialogContent sx={{ py: 2 }}>
        <Stack spacing={2.5}>
          <TextField
            fullWidth
            size="small"
            label="Page Group Title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            sx={{ mt: 1, input: { color: '#F8FAFC' }, label: { color: '#94A3B8' } }}
          />

          <FormControl fullWidth size="small">
            <InputLabel sx={{ color: '#94A3B8' }}>Layout Topology</InputLabel>
            <Select
              value={topology}
              label="Layout Topology"
              onChange={(e) => setTopology(e.target.value as any)}
              sx={{ color: '#F8FAFC' }}
            >
              <MenuItem value="TABBED_BY_SUBTYPE">Multi-Tab PageGroup (Tab per Subtype)</MenuItem>
              <MenuItem value="SINGLE_SCROLL_PANE">Single Scroll Pane (Unified Canvas)</MenuItem>
            </Select>
          </FormControl>

          <Divider sx={{ borderColor: '#1E293B' }} />

          {/* Subtypes Selection */}
          <Box>
            <Typography variant="caption" sx={{ fontWeight: 700, color: '#00D4FF', textTransform: 'uppercase' }}>
              Discovered Subtypes ({discoveredSubtypes.length})
            </Typography>
            <FormGroup row sx={{ mt: 1 }}>
              {discoveredSubtypes.map((st) => (
                <FormControlLabel
                  key={st.subtypeCode}
                  control={
                    <Checkbox
                      checked={selectedSubtypes.includes(st.subtypeCode)}
                      onChange={(e) => {
                        setSelectedSubtypes((prev) =>
                          e.target.checked ? [...prev, st.subtypeCode] : prev.filter((k) => k !== st.subtypeCode)
                        );
                      }}
                      sx={{ color: '#00D4FF', '&.Mui-checked': { color: '#00D4FF' } }}
                    />
                  }
                  label={<Typography variant="body2" sx={{ fontSize: 12 }}>{st.displayName}</Typography>}
                />
              ))}
            </FormGroup>
          </Box>

          <Divider sx={{ borderColor: '#1E293B' }} />

          {/* CRUD Entitlements */}
          <Box>
            <Typography variant="caption" sx={{ fontWeight: 700, color: '#10B981', textTransform: 'uppercase' }}>
              CRUD Action Entitlements
            </Typography>
            <FormGroup row sx={{ mt: 1 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={allowCreate}
                    onChange={(e) => setAllowCreate(e.target.checked)}
                    sx={{ color: '#10B981', '&.Mui-checked': { color: '#10B981' } }}
                  />
                }
                label={<Typography variant="body2" sx={{ fontSize: 12 }}>Allow Create (INSERT)</Typography>}
              />
              <FormControlLabel
                control={
                  <Checkbox
                    checked={allowUpdate}
                    onChange={(e) => setAllowUpdate(e.target.checked)}
                    sx={{ color: '#10B981', '&.Mui-checked': { color: '#10B981' } }}
                  />
                }
                label={<Typography variant="body2" sx={{ fontSize: 12 }}>Allow Update (MUTATION)</Typography>}
              />
              <FormControlLabel
                control={
                  <Checkbox
                    checked={allowDelete}
                    onChange={(e) => setAllowDelete(e.target.checked)}
                    sx={{ color: '#EF4444', '&.Mui-checked': { color: '#EF4444' } }}
                  />
                }
                label={<Typography variant="body2" sx={{ fontSize: 12 }}>Allow Delete (Soft-Delete)</Typography>}
              />
            </FormGroup>
          </Box>

          <Divider sx={{ borderColor: '#1E293B' }} />

          {/* Relationships & Cardinality */}
          <Box>
            <Typography variant="caption" sx={{ fontWeight: 700, color: '#38BDF8', textTransform: 'uppercase' }}>
              Related Entities & Cardinality ({discoveredRelationships.length})
            </Typography>
            <Stack spacing={1} sx={{ mt: 1 }}>
              {discoveredRelationships.map((rel) => (
                <Box
                  key={rel.relKey}
                  sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', p: 1, bgcolor: '#0B1E36', borderRadius: 1 }}
                >
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={selectedRelationships.includes(rel.relKey)}
                        onChange={(e) => {
                          setSelectedRelationships((prev) =>
                            e.target.checked ? [...prev, rel.relKey] : prev.filter((k) => k !== rel.relKey)
                          );
                        }}
                        sx={{ color: '#38BDF8', '&.Mui-checked': { color: '#38BDF8' } }}
                      />
                    }
                    label={<Typography variant="body2" sx={{ fontSize: 12 }}>{rel.relName}</Typography>}
                  />
                  <Chip
                    size="small"
                    label={rel.cardinality}
                    color={rel.cardinality === '1:1' ? 'info' : 'warning'}
                    sx={{ fontSize: 10, height: 20 }}
                  />
                </Box>
              ))}
            </Stack>
          </Box>
        </Stack>
      </DialogContent>

      <DialogActions sx={{ borderTop: '1px solid #1E293B', px: 3, py: 1.5 }}>
        <Button onClick={onClose} sx={{ color: '#94A3B8', textTransform: 'none' }}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleGenerate}
          disabled={generating}
          startIcon={<AutoAwesomeIcon />}
          sx={{ bgcolor: '#0284C7', textTransform: 'none' }}
        >
          {generating ? 'Compiling Blueprint...' : 'Generate PageGroup'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
