import React from 'react';
import {
  Box,
  Typography,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Grid,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Divider,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import FormatToolbar from './FormatToolbar';
import ExpressionInputControl from './ExpressionInputControl';
import TableColumnEditor from './TableColumnEditor';
import TotalsEditor from './tableStyling/TotalsEditor';
import BandingEditor from './tableStyling/BandingEditor';
import FreezePaneEditor from './tableStyling/FreezePaneEditor';
import ConditionalRuleEditor from './tableStyling/ConditionalRuleEditor';
import NamedStyleManager from './tableStyling/NamedStyleManager';
import SparklinePicker from './tableStyling/SparklinePicker';
import PaginationEditor from './tableStyling/PaginationEditor';
import SectionPropertyPanel from './SectionPropertyPanel';
import { FormPropertiesPanel } from './FormPropertiesPanel';
import { ColumnConfig, TotalsConfig, BandingConfig, FreezePaneConfig, PaginationConfig, ConditionalRule, NamedStyle, createDefaultTotalsConfig, createDefaultBandingConfig, createDefaultFreezePaneConfig, createDefaultPaginationConfig } from './tableColumnModel';
import { ELEMENT_TYPES, datasets, sanitizeInput } from './reportingUtils';
import AdvancedConditionBuilder, { FieldDefinition, ConditionGroup } from '../ExpressionBuilder/AdvancedConditionBuilder';

const FORMAT_TYPES = [
  { value: 'Auto', label: 'Auto / Default' },
  { value: 'Currency', label: 'Currency ($1,234)' },
  { value: 'Percent', label: 'Percentage (12.3%)' },
  { value: 'Decimal', label: 'Decimal (1,234.56)' },
  { value: 'Integer', label: 'Integer (1,234)' },
  { value: 'Date', label: 'Date (MM/DD/YYYY)' },
  { value: 'Text', label: 'Plain Text' },
];

const BORDER_STYLES = ['solid', 'dashed', 'dotted', 'double', 'none'];

interface PropertiesPanelProps {
  selectedElement: any | null;
  onElementUpdate: (id: string, updates: Partial<any>) => void;
  activeDatasets?: any[];
  availableFieldDefs?: (FieldDefinition & { _scope?: 'root' | 'subtype'; _subtypeKey?: string })[];
  groupDefinitions?: any[];
  selectedBO?: any;
  businessObjects?: any[];
  selectedSection?: string | null;
  sectionConfig?: Record<string, any>;
  onSectionConfigChange?: (section: string, update: Partial<any>) => void;
  layoutSettings?: any;
  onLayoutSettingsChange?: (key: string, value: any) => void;
  formRegistry?: Record<string, any>;
}

const PropertiesPanel: React.FC<PropertiesPanelProps> = ({
  selectedElement,
  onElementUpdate,
  activeDatasets,
  availableFieldDefs = [],
  formRegistry = {},
  selectedBO,
  selectedSection,
  sectionConfig = {},
  onSectionConfigChange,
  layoutSettings = {},
  onLayoutSettingsChange,
}) => {
  if (selectedSection) {
    return (
      <SectionPropertyPanel
        selectedSection={selectedSection}
        sectionConfig={sectionConfig}
        onSectionConfigChange={onSectionConfigChange!}
        availableFieldDefs={availableFieldDefs}
        layoutSettings={layoutSettings}
        onLayoutSettingsChange={onLayoutSettingsChange}
      />
    );
  }

  if (!selectedElement) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="body2" color="text.secondary">
          Select an element or section on the canvas to configure its properties, styling, and expressions.
        </Typography>
      </Box>
    );
  }

  // formReference elements use the dedicated FormPropertiesPanel
  if (selectedElement.type === ELEMENT_TYPES.FORM_REFERENCE) {
    return (
      <FormPropertiesPanel
        element={selectedElement}
        formRegistry={formRegistry}
        availableFields={availableFieldDefs}
        onUpdate={(patch) => onElementUpdate(selectedElement.id, patch)}
        onNavigateToFormTab={() => {}}
      />
    );
  }

  const properties = selectedElement.properties || {};

  const updateProperty = (property: string, value: any) => {
    const sanitizedValue = typeof value === 'string' ? sanitizeInput(value) : value;
    onElementUpdate(selectedElement.id, {
      properties: { ...properties, [property]: sanitizedValue },
    });
  };

  const availableDatasets = activeDatasets && activeDatasets.length > 0 ? activeDatasets : datasets;

  return (
    <Box sx={{ p: 2, maxHeight: 'calc(100vh - 120px)', overflowY: 'auto' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
        <Typography variant="subtitle1" fontWeight="700">
          {String(selectedElement.type).toUpperCase()} Properties
        </Typography>
        <Typography variant="caption" color="text.secondary">
          ID: {selectedElement.id}
        </Typography>
      </Box>

      {/* ─── 1. General & Content ─── */}
      <Accordion defaultExpanded disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight="600">Content & Data</Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0 }}>
          <Grid container spacing={1.5}>
            <Grid size={12}>
              <TextField
                fullWidth
                size="small"
                label="Element Name / Title"
                value={properties.name || ''}
                onChange={(e) => updateProperty('name', e.target.value)}
              />
            </Grid>

            {selectedElement.type === ELEMENT_TYPES.TEXTBOX && (
              <>
                <Grid size={12}>
                  <TextField
                    fullWidth
                    size="small"
                    multiline
                    minRows={2}
                    label="Display Text / Expression"
                    helperText="Use [field_name] or standard text"
                    value={properties.text !== undefined ? properties.text : ''}
                    onChange={(e) => updateProperty('text', e.target.value)}
                  />
                </Grid>
                <Grid size={12}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Binding Field Name"
                    placeholder="e.g. market_value, name, status"
                    value={properties.fieldName || ''}
                    onChange={(e) => updateProperty('fieldName', e.target.value)}
                  />
                </Grid>
              </>
            )}

            {(selectedElement.type === ELEMENT_TYPES.TABLE || selectedElement.type === ELEMENT_TYPES.MATRIX || selectedElement.type === ELEMENT_TYPES.LIST) && (
              <>
                <Grid size={12}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Data Source</InputLabel>
                    <Select
                      value={properties.dataSource || availableDatasets[0]?.id || ''}
                      label="Data Source"
                      onChange={(e) => updateProperty('dataSource', e.target.value)}
                    >
                      {availableDatasets.map((ds: any) => (
                        <MenuItem key={ds.id} value={ds.id}>
                          {ds.name}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>

                {availableFieldDefs && availableFieldDefs.length > 0 && (
                  <Grid size={12}>
                    <Typography variant="caption" fontWeight="600" sx={{ display: 'block', mb: 0.5 }}>
                      Column Manager
                    </Typography>
                    <TableColumnEditor
                      columns={(properties.columns as ColumnConfig[]) || []}
                      onChange={(cols) => updateProperty('columns', cols)}
                      availableFields={availableFieldDefs.map((f: any) => ({
                        name: f.name,
                        type: f.type || 'string',
                        label: f.label || f.name,
                      }))}
                    />
                  </Grid>
                )}
              </>
            )}
          </Grid>
        </AccordionDetails>
      </Accordion>

      {/* ─── 2. PowerBI/Looker Typography & Text Styling ─── */}
      <Accordion defaultExpanded disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight="600">Typography & Alignment</Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0 }}>
          <Box sx={{ mb: 1.5 }}>
            <FormatToolbar properties={properties} onUpdate={updateProperty} />
          </Box>
          <Grid container spacing={1.5}>
            <Grid size={6}>
              <FormControl fullWidth size="small">
                <InputLabel>Font Weight</InputLabel>
                <Select
                  value={properties.fontWeight || (properties.bold ? 700 : 400)}
                  label="Font Weight"
                  onChange={(e) => updateProperty('fontWeight', Number(e.target.value))}
                >
                  <MenuItem value={300}>Light (300)</MenuItem>
                  <MenuItem value={400}>Regular (400)</MenuItem>
                  <MenuItem value={500}>Medium (500)</MenuItem>
                  <MenuItem value={600}>Semi-Bold (600)</MenuItem>
                  <MenuItem value={700}>Bold (700)</MenuItem>
                  <MenuItem value={800}>Extra Bold (800)</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid size={6}>
              <FormControl fullWidth size="small">
                <InputLabel>Text Transform</InputLabel>
                <Select
                  value={properties.textTransform || 'none'}
                  label="Text Transform"
                  onChange={(e) => updateProperty('textTransform', e.target.value)}
                >
                  <MenuItem value="none">None</MenuItem>
                  <MenuItem value="uppercase">UPPERCASE</MenuItem>
                  <MenuItem value="lowercase">lowercase</MenuItem>
                  <MenuItem value="capitalize">Capitalize</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>

      {/* ─── 3. Looker/PowerBI Style & Color Palette ─── */}
      <Accordion defaultExpanded disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight="600">Colors, Borders & Dynamic fx</Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0 }}>
          <Grid container spacing={1.5}>
            <Grid size={12}>
              <ExpressionInputControl
                label="Text Color"
                property={properties.textColor || '#111827'}
                defaultFormula="=IIF(Fields!amount.Value > 1000, '#10B981', '#EF4444')"
                onChange={(prop) => updateProperty('textColor', prop.isExpression ? prop : prop.value)}
                renderStaticControl={(val, setVal) => (
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <input
                      type="color"
                      value={typeof val === 'string' && val.startsWith('#') ? val : '#111827'}
                      onChange={(e) => setVal(e.target.value)}
                      style={{ width: 36, height: 32, padding: 0, border: 'none', borderRadius: 4, cursor: 'pointer' }}
                    />
                    <TextField
                      size="small"
                      value={val || '#111827'}
                      onChange={(e) => setVal(e.target.value)}
                      sx={{ '& input': { py: 0.5, fontSize: '0.75rem' } }}
                    />
                  </Box>
                )}
              />
            </Grid>

            <Grid size={12}>
              <ExpressionInputControl
                label="Background Fill Color"
                property={properties.backgroundColor || 'transparent'}
                defaultFormula="=IIF(Fields!status.Value == 'Active', 'rgba(16,185,129,0.1)', 'transparent')"
                onChange={(prop) => updateProperty('backgroundColor', prop.isExpression ? prop : prop.value)}
                renderStaticControl={(val, setVal) => (
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <input
                      type="color"
                      value={typeof val === 'string' && val.startsWith('#') ? val : '#ffffff'}
                      onChange={(e) => setVal(e.target.value)}
                      style={{ width: 36, height: 32, padding: 0, border: 'none', borderRadius: 4, cursor: 'pointer' }}
                    />
                    <TextField
                      size="small"
                      value={val || 'transparent'}
                      onChange={(e) => setVal(e.target.value)}
                      sx={{ '& input': { py: 0.5, fontSize: '0.75rem' } }}
                    />
                  </Box>
                )}
              />
            </Grid>

            <Grid size={4}>
              <TextField
                fullWidth
                size="small"
                type="number"
                label="Border Width"
                value={properties.borderWidth ?? 0}
                onChange={(e) => updateProperty('borderWidth', Number(e.target.value))}
              />
            </Grid>
            <Grid size={4}>
              <FormControl fullWidth size="small">
                <InputLabel>Border Style</InputLabel>
                <Select
                  value={properties.borderStyle || 'solid'}
                  label="Border Style"
                  onChange={(e) => updateProperty('borderStyle', e.target.value)}
                >
                  {BORDER_STYLES.map((st) => (
                    <MenuItem key={st} value={st}>{st}</MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={4}>
              <TextField
                fullWidth
                size="small"
                type="number"
                label="Border Radius"
                value={properties.borderRadius ?? 4}
                onChange={(e) => updateProperty('borderRadius', Number(e.target.value))}
              />
            </Grid>

            <Grid size={6}>
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                  Border Color
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <input
                    type="color"
                    value={properties.borderColor || '#cccccc'}
                    onChange={(e) => updateProperty('borderColor', e.target.value)}
                    style={{ width: 36, height: 32, padding: 0, border: 'none', borderRadius: 4, cursor: 'pointer' }}
                  />
                  <TextField
                    size="small"
                    value={properties.borderColor || 'transparent'}
                    onChange={(e) => updateProperty('borderColor', e.target.value)}
                    sx={{ '& input': { py: 0.5, fontSize: '0.75rem' } }}
                  />
                </Box>
              </Box>
            </Grid>

            <Grid size={6}>
              <TextField
                fullWidth
                size="small"
                type="number"
                label="Padding (px)"
                value={properties.padding ?? 4}
                onChange={(e) => updateProperty('padding', Number(e.target.value))}
              />
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>

      {/* ─── 4. Number & Value Formatting ─── */}
      <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight="600">Value Formatting</Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0 }}>
          <Grid container spacing={1.5}>
            <Grid size={12}>
              <FormControl fullWidth size="small">
                <InputLabel>Format Type</InputLabel>
                <Select
                  value={properties.formatType || 'Auto'}
                  label="Format Type"
                  onChange={(e) => updateProperty('formatType', e.target.value)}
                >
                  {FORMAT_TYPES.map((f) => (
                    <MenuItem key={f.value} value={f.value}>
                      {f.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={6}>
              <TextField
                fullWidth
                size="small"
                label="Prefix"
                placeholder="e.g. $"
                value={properties.formatPrefix || ''}
                onChange={(e) => updateProperty('formatPrefix', e.target.value)}
              />
            </Grid>
            <Grid size={6}>
              <TextField
                fullWidth
                size="small"
                label="Suffix"
                placeholder="e.g. USD, %"
                value={properties.formatSuffix || ''}
                onChange={(e) => updateProperty('formatSuffix', e.target.value)}
              />
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>

      {/* ─── 5. Dynamic Expressions & Calculations ─── */}
      <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight="600">Dynamic Expressions & Calcs</Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0 }}>
          <Grid container spacing={1.5}>
            <Grid size={12}>
              <ExpressionInputControl
                label="Value Expression"
                property={properties.valueExpression || ''}
                defaultFormula="=Fields!Amount.Value"
                onChange={(prop) => updateProperty('valueExpression', prop.isExpression ? prop.formula : prop.value)}
                renderStaticControl={(val, setVal) => (
                  <TextField
                    fullWidth
                    size="small"
                    multiline
                    minRows={2}
                    label="Value Expression"
                    helperText="e.g. [Account.balance] * 1.05 or =Fields!Amount.Value"
                    value={String(val ?? '')}
                    onChange={(e) => setVal(e.target.value)}
                  />
                )}
              />
            </Grid>
            <Grid size={12}>
              <ExpressionInputControl
                label="Conditional Style Expression"
                property={properties.conditionalColorExpression || ''}
                defaultFormula="=IIF(Fields!Growth.Value < 0, '#EF4444', '#22C55E')"
                onChange={(prop) => updateProperty('conditionalColorExpression', prop.isExpression ? prop.formula : prop.value)}
                renderStaticControl={(val, setVal) => (
                  <TextField
                    fullWidth
                    size="small"
                    multiline
                    minRows={2}
                    label="Conditional Style Expression"
                    helperText="e.g. =IIF(Fields!Growth.Value < 0, '#EF4444', '#22C55E') — Use vibrant colors that work in both light/dark modes"
                    value={String(val ?? '')}
                    onChange={(e) => setVal(e.target.value)}
                  />
                )}
              />
            </Grid>

            {/* Element Conditional Visibility — direct AdvancedConditionBuilder */}
            <Grid size={12}>
              <Divider sx={{ my: 1 }} />
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                <Typography variant="caption" color="text.secondary" fontWeight="600">
                  Conditional Visibility
                </Typography>
                {properties.visibilityCondition ? (
                  <Chip size="small" label="Rule active" color="warning" sx={{ height: 18, fontSize: '0.6rem' }} />
                ) : (
                  <Typography variant="caption" color="text.disabled" sx={{ fontSize: '0.65rem' }}>
                    No rule — always visible
                  </Typography>
                )}
              </Box>
              <AdvancedConditionBuilder
                value={properties.visibilityCondition || { id: `vis_el_${Date.now()}`, type: 'group', operator: 'AND', conditions: [] }}
                onChange={(tree: ConditionGroup) => updateProperty('visibilityCondition', tree)}
                availableFields={availableFieldDefs}
                entityName={selectedBO?.name || 'Entity'}
              />
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>

      {/* ─── 6. Position & Geometry ─── */}
      <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2" fontWeight="600">Size & Layout</Typography>
        </AccordionSummary>
        <AccordionDetails sx={{ pt: 0 }}>
          <Grid container spacing={1.5}>
            <Grid size={6}>
              <TextField
                fullWidth
                size="small"
                type="number"
                label="Width (px)"
                value={selectedElement.size.width}
                onChange={(e) =>
                  onElementUpdate(selectedElement.id, {
                    size: { ...selectedElement.size, width: Number(e.target.value) },
                  })
                }
              />
            </Grid>
            <Grid size={6}>
              <TextField
                fullWidth
                size="small"
                type="number"
                label="Height (px)"
                value={selectedElement.size.height}
                onChange={(e) =>
                  onElementUpdate(selectedElement.id, {
                    size: { ...selectedElement.size, height: Number(e.target.value) },
                  })
                }
              />
            </Grid>
            <Grid size={6}>
              <TextField
                fullWidth
                size="small"
                type="number"
                label="X Position"
                value={selectedElement.position.x}
                onChange={(e) =>
                  onElementUpdate(selectedElement.id, {
                    position: { ...selectedElement.position, x: Number(e.target.value) },
                  })
                }
              />
            </Grid>
            <Grid size={6}>
              <TextField
                fullWidth
                size="small"
                type="number"
                label="Y Position"
                value={selectedElement.position.y}
                onChange={(e) =>
                  onElementUpdate(selectedElement.id, {
                    position: { ...selectedElement.position, y: Number(e.target.value) },
                  })
                }
              />
            </Grid>
          </Grid>
        </AccordionDetails>
      </Accordion>

      {/* ─── TABLE/MATRIX: Totals ─── */}
      {(selectedElement.type === ELEMENT_TYPES.TABLE || selectedElement.type === ELEMENT_TYPES.MATRIX) && (
        <Accordion defaultExpanded disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight="600">Totals &amp; Aggregation</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0 }}>
            <TotalsEditor
              totals={(properties.totals as TotalsConfig) || createDefaultTotalsConfig()}
              onChange={(totals) => updateProperty('totals', totals)}
              columns={(properties.columns as ColumnConfig[]) || []}
              onColumnsChange={(cols) => updateProperty('columns', cols)}
            />
          </AccordionDetails>
        </Accordion>
      )}

      {/* ─── TABLE/MATRIX: Page Layout ─── */}
      {(selectedElement.type === ELEMENT_TYPES.TABLE || selectedElement.type === ELEMENT_TYPES.MATRIX) && (
        <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight="600">Page Layout</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0 }}>
            <PaginationEditor
              pagination={(properties.pagination as PaginationConfig) || createDefaultPaginationConfig()}
              onChange={(pagination) => updateProperty('pagination', pagination)}
            />
          </AccordionDetails>
        </Accordion>
      )}

      {/* ─── TABLE/MATRIX: Banding & Gridlines ─── */}
      {(selectedElement.type === ELEMENT_TYPES.TABLE || selectedElement.type === ELEMENT_TYPES.MATRIX) && (
        <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight="600">Banding &amp; Gridlines</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0 }}>
            <BandingEditor
              banding={(properties.banding as BandingConfig) || createDefaultBandingConfig()}
              onChange={(banding) => updateProperty('banding', banding)}
            />
          </AccordionDetails>
        </Accordion>
      )}

      {/* ─── TABLE/MATRIX: Freeze Panes ─── */}
      {(selectedElement.type === ELEMENT_TYPES.TABLE || selectedElement.type === ELEMENT_TYPES.MATRIX) && (
        <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight="600">Freeze Panes</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0 }}>
            <FreezePaneEditor
              freezePane={(properties.freezePane as FreezePaneConfig) || createDefaultFreezePaneConfig()}
              onChange={(fp) => updateProperty('freezePane', fp)}
            />
          </AccordionDetails>
        </Accordion>
      )}

      {/* ─── TABLE/MATRIX: Sparklines ─── */}
      {(selectedElement.type === ELEMENT_TYPES.TABLE || selectedElement.type === ELEMENT_TYPES.MATRIX) && (
        <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight="600">Sparklines</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0 }}>
            <SparklinePicker
              sparkline={undefined}
              onChange={(sp) => {
                if (!sp) return;
                const cols = (properties.columns as ColumnConfig[]) || [];
                if (cols.length > 0) {
                  updateProperty('columns', cols.map((c, i) => i === cols.length - 1 ? { ...c, sparkline: sp } : c));
                }
              }}
            />
          </AccordionDetails>
        </Accordion>
      )}

      {/* ─── TABLE/MATRIX: Conditional Formatting ─── */}
      {(selectedElement.type === ELEMENT_TYPES.TABLE || selectedElement.type === ELEMENT_TYPES.MATRIX) && (
        <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight="600">Conditional Formatting</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0 }}>
            <ConditionalRuleEditor
              rules={(properties.conditionalRules as ConditionalRule[]) || []}
              onChange={(rules) => updateProperty('conditionalRules', rules)}
              columnIds={((properties.columns as ColumnConfig[]) || []).map(c => c.id)}
            />
          </AccordionDetails>
        </Accordion>
      )}

      {/* ─── TABLE/MATRIX: Named Styles ─── */}
      {(selectedElement.type === ELEMENT_TYPES.TABLE || selectedElement.type === ELEMENT_TYPES.MATRIX) && (
        <Accordion disableGutters sx={{ mb: 1, '&:before': { display: 'none' } }}>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" fontWeight="600">Named Styles</Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ pt: 0 }}>
            <NamedStyleManager
              styles={(properties.namedStyles as NamedStyle[]) || []}
              onChange={(styles) => updateProperty('namedStyles', styles)}
            />
          </AccordionDetails>
        </Accordion>
      )}
    </Box>
  );
};

export default PropertiesPanel;
