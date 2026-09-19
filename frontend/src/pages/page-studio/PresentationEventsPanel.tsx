import React, { useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Chip,
  IconButton,
  MenuItem,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import type {
  CorePageDefinition,
  PresentationAction,
  PresentationEventKind,
  PresentationRule,
  BusinessObjectDataSourceConfig,
} from '../../types/pageStudio';
import ExpressionEditorField from '../../components/ExpressionBuilder/ExpressionEditorField';
import {
  PRESENTATION_EVENT_LABELS,
  collectPresentationTargets,
  newPresentationRule,
} from './presentationEvents';

const EVENTS: PresentationEventKind[] = ['pageActivate', 'fieldChange', 'fieldEdit', 'selectionChange'];
const ACTION_PROPS = [
  { key: 'hidden', label: 'Hidden' },
  { key: 'readOnly', label: 'Read only (display)' },
  { key: 'collapsed', label: 'Collapsed (panel)' },
] as const;

interface PresentationEventsPanelProps {
  draft: CorePageDefinition;
  setDraft: (updater: (prev: CorePageDefinition) => CorePageDefinition) => void;
}

const PresentationEventsPanel: React.FC<PresentationEventsPanelProps> = ({ draft, setDraft }) => {
  const rules = draft.presentationEvents || [];
  const boSource = (draft.dataSources || []).find((d) => d.type === 'business_object');
  const boKey = (boSource?.config as unknown as BusinessObjectDataSourceConfig | undefined)?.boKey;
  const targets = collectPresentationTargets(draft);
  const [expanded, setExpanded] = useState<PresentationEventKind | false>('fieldChange');

  const setRules = (next: PresentationRule[]) => {
    setDraft((prev) => ({ ...prev, presentationEvents: next }));
  };

  const updateRule = (id: string, patch: Partial<PresentationRule>) => {
    setRules(rules.map((r) => (r.id === id ? { ...r, ...patch } : r)));
  };

  const removeRule = (id: string) => setRules(rules.filter((r) => r.id !== id));

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="overline" color="text.secondary">Events</Typography>
      <Alert severity="info" sx={{ mt: 1, mb: 2, py: 0.5, '& .MuiAlert-message': { fontSize: 12 } }}>
        These rules <strong>format</strong> the page (hide, color, collapse, relabel).
        They do not save or validate data — Business Object events own CRUD.
        Include / Place / Bind stay on Data Binding and Design.
      </Alert>

      {EVENTS.map((event) => {
        const group = rules.filter((r) => r.event === event);
        return (
          <Accordion
            key={event}
            expanded={expanded === event}
            onChange={(_, on) => setExpanded(on ? event : false)}
            disableGutters
            elevation={0}
            sx={{ '&:before': { display: 'none' }, bgcolor: 'transparent', borderBottom: '1px solid', borderColor: 'divider' }}
          >
            <AccordionSummary expandIcon={<ExpandMoreIcon />} sx={{ px: 0 }}>
              <Typography variant="body2" fontWeight={700}>
                {PRESENTATION_EVENT_LABELS[event]}
              </Typography>
              <Chip size="small" label={group.length} sx={{ ml: 1 }} />
            </AccordionSummary>
            <AccordionDetails sx={{ px: 0, pt: 0 }}>
              {group.length === 0 && (
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                  No {PRESENTATION_EVENT_LABELS[event].toLowerCase()} rules.
                </Typography>
              )}
              {group.map((rule) => (
                <RuleCard
                  key={rule.id}
                  rule={rule}
                  boKey={boKey}
                  targets={targets}
                  onChange={(patch) => updateRule(rule.id, patch)}
                  onRemove={() => removeRule(rule.id)}
                />
              ))}
              <Button
                size="small"
                startIcon={<AddIcon />}
                onClick={() => setRules([...rules, newPresentationRule(event, { label: `${PRESENTATION_EVENT_LABELS[event]} rule` })])}
              >
                Add {PRESENTATION_EVENT_LABELS[event].toLowerCase()}
              </Button>
            </AccordionDetails>
          </Accordion>
        );
      })}
    </Box>
  );
};

const RuleCard: React.FC<{
  rule: PresentationRule;
  boKey?: string;
  targets: ReturnType<typeof collectPresentationTargets>;
  onChange: (patch: Partial<PresentationRule>) => void;
  onRemove: () => void;
}> = ({ rule, boKey, targets, onChange, onRemove }) => {
  const setAction = (index: number, patch: Partial<PresentationAction>) => {
    const actions = rule.actions.map((a, i) => (i === index ? { ...a, ...patch } : a));
    onChange({ actions });
  };
  const addAction = () => {
    const first = targets[0];
    onChange({
      actions: [
        ...rule.actions,
        { target: first ? { kind: first.kind, id: first.id, fieldName: first.fieldName } : { kind: 'widget', id: '' } },
      ],
    });
  };
  return (
    <Box sx={{ mb: 2, p: 1.25, border: '1px solid', borderColor: 'divider', borderRadius: 1 }}>
      <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
        <TextField
          size="small"
          fullWidth
          label="Name"
          value={rule.label || ''}
          onChange={(e) => onChange({ label: e.target.value })}
        />
        <IconButton size="small" onClick={onRemove}><DeleteIcon fontSize="small" /></IconButton>
      </Stack>
      <ExpressionEditorField
        label="If (empty = always on this event)"
        value={rule.when}
        onChange={(v) => onChange({ when: v })}
        boName={boKey}
        minHeight={80}
      />
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1.5, mb: 0.5 }}>
        Then set properties
      </Typography>
      {rule.actions.map((action, i) => (
        <Box key={i} sx={{ mb: 1 }}>
          <TextField
            select
            size="small"
            fullWidth
            label="Target"
            value={`${action.target.kind}:${action.target.id}:${action.target.fieldName || ''}`}
            onChange={(e) => {
              const [kind, id, fieldName] = e.target.value.split(':');
              setAction(i, { target: { kind: kind as PresentationAction['target']['kind'], id, fieldName: fieldName || undefined } });
            }}
            sx={{ mb: 0.75 }}
          >
            {targets.map((t) => (
              <MenuItem key={`${t.kind}:${t.id}:${t.fieldName || ''}`} value={`${t.kind}:${t.id}:${t.fieldName || ''}`}>
                {t.label}
              </MenuItem>
            ))}
          </TextField>
          <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', rowGap: 0.5, mb: 0.5 }}>
            {ACTION_PROPS.map((p) => (
              <Chip
                key={p.key}
                size="small"
                label={p.label}
                color={action[p.key] ? 'primary' : 'default'}
                variant={action[p.key] ? 'filled' : 'outlined'}
                onClick={() => setAction(i, { [p.key]: !action[p.key] })}
              />
            ))}
          </Stack>
          <Stack direction="row" spacing={1}>
            <TextField
              size="small"
              type="color"
              label="Color"
              InputLabelProps={{ shrink: true }}
              value={action.style?.color || '#000000'}
              onChange={(e) => setAction(i, { style: { ...action.style, color: e.target.value } })}
              sx={{ width: 88 }}
            />
            <TextField
              size="small"
              type="color"
              label="Fill"
              InputLabelProps={{ shrink: true }}
              value={action.style?.backgroundColor || '#ffffff'}
              onChange={(e) => setAction(i, { style: { ...action.style, backgroundColor: e.target.value } })}
              sx={{ width: 88 }}
            />
            <TextField
              size="small"
              label="Label"
              value={action.label || ''}
              onChange={(e) => setAction(i, { label: e.target.value || undefined })}
              sx={{ flex: 1 }}
            />
            <IconButton size="small" onClick={() => onChange({ actions: rule.actions.filter((_, j) => j !== i) })}>
              <DeleteIcon fontSize="small" />
            </IconButton>
          </Stack>
        </Box>
      ))}
      <Button size="small" onClick={addAction}>Add action</Button>
    </Box>
  );
};

export default PresentationEventsPanel;
export { RuleCard };
