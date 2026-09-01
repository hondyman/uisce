import React from 'react';
import {
  Box,
  Typography,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Grid,
  Switch,
  FormControlLabel,
  Divider,
  Chip,
  IconButton,
  TextField,
  Button,
  Tooltip,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import AddIcon from '@mui/icons-material/Add';
import CodeIcon from '@mui/icons-material/Code';
import AdvancedConditionBuilder, {
  ConditionGroup,
  FieldDefinition,
} from '../ExpressionBuilder/AdvancedConditionBuilder';
import { REPORT_SECTIONS } from './reportingUtils';

interface TokenEntry {
  id: string;
  text: string;
  mode: 'static' | 'expression';
  expression?: string;
}

const getSectionLabel = (section: string) => {
  switch (section) {
    case REPORT_SECTIONS.REPORT_HEADER: return 'Report Header';
    case REPORT_SECTIONS.PAGE_HEADER: return 'Page Header';
    case REPORT_SECTIONS.GROUP_HEADER: return 'Group Header';
    case REPORT_SECTIONS.BODY: return 'Body (Detail)';
    case REPORT_SECTIONS.GROUP_FOOTER: return 'Group Footer';
    case REPORT_SECTIONS.PAGE_FOOTER: return 'Page Footer';
    case REPORT_SECTIONS.REPORT_FOOTER: return 'Report Footer';
    default: return section;
  }
};

function uid() {
  return `${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
}

function emptyConditionGroup(): ConditionGroup {
  return { id: uid(), type: 'group', operator: 'AND', conditions: [] };
}

interface SectionPropertyPanelProps {
  selectedSection: string;
  sectionConfig: Record<string, any>;
  onSectionConfigChange: (section: string, update: Partial<any>) => void;
  availableFieldDefs?: (FieldDefinition & { _scope?: 'root' | 'subtype'; _subtypeKey?: string })[];
  layoutSettings?: any;
  onLayoutSettingsChange?: (key: string, value: any) => void;
}

const SectionPropertyPanel: React.FC<SectionPropertyPanelProps> = ({
  selectedSection,
  sectionConfig,
  onSectionConfigChange,
  availableFieldDefs = [],
  layoutSettings = {},
  onLayoutSettingsChange,
}) => {
  const sectionLabel = getSectionLabel(selectedSection);
  const config = sectionConfig[selectedSection] || {};
  const isHeader = selectedSection === REPORT_SECTIONS.PAGE_HEADER || selectedSection === REPORT_SECTIONS.REPORT_HEADER;
  const isFooter = selectedSection === REPORT_SECTIONS.PAGE_FOOTER || selectedSection === REPORT_SECTIONS.REPORT_FOOTER;

  const tokens: TokenEntry[] = React.useMemo(() => {
    const raw = layoutSettings.headerTokens || [];
    if (raw.length === 0) return [];
    return raw.map((t: string | TokenEntry) => {
      if (typeof t === 'object' && t !== null && 'mode' in t) return t as TokenEntry;
      return { id: uid(), text: String(t), mode: 'static' } as TokenEntry;
    });
  }, [layoutSettings.headerTokens]);

  const footerTokens: TokenEntry[] = React.useMemo(() => {
    const raw = layoutSettings.footerTokens || [];
    if (raw.length === 0) return [];
    return raw.map((t: string | TokenEntry) => {
      if (typeof t === 'object' && t !== null && 'mode' in t) return t as TokenEntry;
      return { id: uid(), text: String(t), mode: 'static' } as TokenEntry;
    });
  }, [layoutSettings.footerTokens]);

  const handleTokenChange = (key: 'headerTokens' | 'footerTokens', id: string, field: keyof TokenEntry, value: any) => {
    const list = key === 'headerTokens' ? tokens : footerTokens;
    const updated = list.map((t: TokenEntry) => t.id === id ? { ...t, [field]: value } : t);
    onLayoutSettingsChange?.(key, updated);
  };

  const handleAddToken = (key: 'headerTokens' | 'footerTokens') => {
    const list = key === 'headerTokens' ? tokens : footerTokens;
    const updated = [...list, { id: uid(), text: 'New Token', mode: 'static' } as TokenEntry];
    onLayoutSettingsChange?.(key, updated);
  };

  const handleRemoveToken = (key: 'headerTokens' | 'footerTokens', id: string) => {
    const list = key === 'headerTokens' ? tokens : footerTokens;
    const updated = list.filter((t: TokenEntry) => t.id !== id);
    onLayoutSettingsChange?.(key, updated);
  };

  const visibilityValue = config.visibilityCondition ?? emptyConditionGroup();
  const hasVisibilityCondition = config.visibilityCondition && Object.keys(config.visibilityCondition).length > 0;

  return (
    <Box sx={{ p: 2, maxHeight: 'calc(100vh - 120px)', overflowY: 'auto' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
        <Typography variant="subtitle1" fontWeight="700">
          {sectionLabel}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Section Properties
        </Typography>
      </Box>

      {/* ─── 1. General & Visibility ─── */}
      <Accordion defaultExpanded disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight="600">General & Visibility</Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0 }}>
          <Grid container spacing={1.5}>
            <Grid size={12}>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                {sectionLabel} — {isHeader ? 'Header' : isFooter ? 'Footer' : 'Body'} section of the report.
              </Typography>
            </Grid>

            {/* Hide toggle */}
            <Grid size={12}>
              <FormControlLabel
                control={
                  <Switch
                    size="small"
                    checked={config.visible !== false}
                    onChange={(e) => onSectionConfigChange(selectedSection, { visible: e.target.checked ? undefined : false })}
                  />
                }
                label={config.visible === false ? 'Hidden' : 'Visible'}
              />
            </Grid>

            {/* Visibility expression — always-on AdvancedConditionBuilder */}
            <Grid size={12}>
              <Divider sx={{ my: 0.5 }} />
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="caption" color="text.secondary" fontWeight="600">
                  Conditional Visibility Expression
                </Typography>
                {hasVisibilityCondition ? (
                  <Chip
                    size="small"
                    label="Rule active"
                    icon={<VisibilityOffIcon sx={{ fontSize: 11 }} />}
                    color="warning"
                    sx={{ height: 18, fontSize: '0.6rem' }}
                  />
                ) : (
                  <Typography variant="caption" color="text.disabled" sx={{ fontSize: '0.65rem' }}>
                    No rule — always visible
                  </Typography>
                )}
              </Box>
              <AdvancedConditionBuilder
                value={visibilityValue as ConditionGroup}
                onChange={(tree) => onSectionConfigChange(selectedSection, { visibilityCondition: tree })}
                availableFields={availableFieldDefs}
                entityName={sectionLabel}
              />
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>

      {/* ─── 2. Page & Layout ─── */}
      <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight="600">Page & Layout</Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0 }}>
          <Grid container spacing={1.5}>
            {(isHeader || isFooter) && (
              <>
                <Grid size={12}>
                  <FormControlLabel
                    control={
                      <Switch
                        size="small"
                        checked={layoutSettings.pageBreakBeforeGroup ?? false}
                        onChange={(e) => onLayoutSettingsChange?.('pageBreakBeforeGroup', e.target.checked)}
                      />
                    }
                    label="Page break before section"
                  />
                </Grid>
                <Grid size={12}>
                  <FormControlLabel
                    control={
                      <Switch
                        size="small"
                        checked={layoutSettings.pageBreakAfterGroup ?? true}
                        onChange={(e) => onLayoutSettingsChange?.('pageBreakAfterGroup', e.target.checked)}
                      />
                    }
                    label="Page break after section"
                  />
                </Grid>
              </>
            )}
            {selectedSection === REPORT_SECTIONS.BODY && (
              <>
                <Grid size={6}>
                  <TextField
                    fullWidth
                    size="small"
                    type="number"
                    label="Columns"
                    value={layoutSettings.columns ?? 1}
                    onChange={(e) => onLayoutSettingsChange?.('columns', Math.max(1, Number(e.target.value) || 1))}
                  />
                </Grid>
                <Grid size={6}>
                  <TextField
                    fullWidth
                    size="small"
                    type="number"
                    label="Column Spacing (px)"
                    value={layoutSettings.columnSpacing ?? 24}
                    onChange={(e) => onLayoutSettingsChange?.('columnSpacing', Math.max(0, Number(e.target.value) || 0))}
                  />
                </Grid>
              </>
            )}
          </Grid>
        </AccordionDetails>
      </Accordion>

      {/* ─── 3. Header / Footer Tokens ─── */}
      {(isHeader || isFooter) && (
        <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight="600">{isHeader ? 'Header' : 'Footer'} Tokens</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0 }}>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              {isHeader && tokens.map((token: TokenEntry) => (
                <TokenEditorRow
                  key={token.id}
                  token={token}
                  onChange={(field, value) => handleTokenChange('headerTokens', token.id, field, value)}
                  onRemove={() => handleRemoveToken('headerTokens', token.id)}
                />
              ))}
              {isFooter && footerTokens.map((token: TokenEntry) => (
                <TokenEditorRow
                  key={token.id}
                  token={token}
                  onChange={(field, value) => handleTokenChange('footerTokens', token.id, field, value)}
                  onRemove={() => handleRemoveToken('footerTokens', token.id)}
                />
              ))}
              <Button
                size="small"
                startIcon={<AddIcon sx={{ fontSize: 14 }} />}
                onClick={() => handleAddToken(isHeader ? 'headerTokens' : 'footerTokens')}
                sx={{ textTransform: 'none', fontSize: '0.72rem' }}
              >
                Add {isHeader ? 'Header' : 'Footer'} Token
              </Button>
            </Box>
          </AccordionDetails>
        </Accordion>
      )}
    </Box>
  );
};

interface TokenEditorRowProps {
  token: TokenEntry;
  onChange: (field: keyof TokenEntry, value: any) => void;
  onRemove: () => void;
}

const TokenEditorRow: React.FC<TokenEditorRowProps> = ({ token, onChange, onRemove }) => {
  const [showBuilder, setShowBuilder] = React.useState(false);

  return (
    <Box sx={{ p: 1, border: '1px solid', borderColor: 'divider', borderRadius: 1, bgcolor: 'background.default' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
        <FormControlLabel
          control={
            <Switch
              size="small"
              checked={token.mode === 'expression'}
              onChange={(e) => {
                onChange('mode', e.target.checked ? 'expression' : 'static');
                if (e.target.checked) setShowBuilder(true);
              }}
            />
          }
          label=""
          sx={{ mr: 0, my: -0.5 }}
        />
        <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.65rem', mr: 0.5 }}>
          {token.mode === 'expression' ? 'Expression' : 'Static'}
        </Typography>
        <Box sx={{ flex: 1 }} />
        <Tooltip title="Remove token">
          <IconButton size="small" onClick={onRemove} sx={{ p: 0.25 }}>
            <DeleteOutlineIcon sx={{ fontSize: 14 }} />
          </IconButton>
        </Tooltip>
      </Box>

      {token.mode === 'static' ? (
        <TextField
          fullWidth
          size="small"
          placeholder="e.g. Page {PageNumber} of {TotalPages}"
          value={token.text}
          onChange={(e) => onChange('text', e.target.value)}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' } }}
        />
      ) : (
        <Box>
          {showBuilder ? (
            <AdvancedConditionBuilder
              value={token.expression ? { id: uid(), type: 'group', operator: 'AND', conditions: [] } : emptyConditionGroup()}
              onChange={(tree) => {
                onChange('expression', `ConditionGroup:${tree.id}`);
              }}
              compact
            />
          ) : (
            <Box sx={{ display: 'flex', gap: 0.5 }}>
              <TextField
                fullWidth
                size="small"
                placeholder="Enter expression"
                value={token.expression || ''}
                onChange={(e) => onChange('expression', e.target.value)}
                InputProps={{
                  sx: { fontSize: '0.72rem', fontFamily: 'monospace' },
                }}
              />
              <Button
                size="small"
                variant="outlined"
                startIcon={<CodeIcon sx={{ fontSize: 12 }} />}
                onClick={() => setShowBuilder(true)}
                sx={{ textTransform: 'none', fontSize: '0.65rem', whiteSpace: 'nowrap' }}
              >
                Builder
              </Button>
            </Box>
          )}
        </Box>
      )}
    </Box>
  );
};

export default SectionPropertyPanel;
