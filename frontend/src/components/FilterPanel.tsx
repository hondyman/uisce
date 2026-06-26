import {
  Card, CardContent, Typography, TextField, Divider, Stack, Button, Select, MenuItem,
  FormControlLabel, Switch, Chip, Box
} from '@mui/material';
import { Filter, X, Check } from 'lucide-react';
import CatalogNodeTypeahead from '../CatalogNodeTypeahead';
import { useScope } from '../../contexts/ScopeContext';

interface FilterPanelProps {
  globalSearchTerm: string;
  setGlobalSearchTerm: (term: string) => void;
  sortBy: 'confidence' | 'name' | 'none';
  setSortBy: (sort: 'confidence' | 'name' | 'none') => void;
  compactRows: boolean;
  setCompactRows: (compact: boolean) => void;
  clearAll: () => void;
  selectAll: () => void;
  allVisibleSelected: boolean;
  openConfirm: () => void;
  selectedMappingsCount: number;
  loading: boolean;
}

export function FilterPanel({
  globalSearchTerm,
  setGlobalSearchTerm,
  sortBy,
  setSortBy,
  compactRows,
  setCompactRows,
  clearAll,
  selectAll,
  allVisibleSelected,
  openConfirm,
  selectedMappingsCount,
  loading,
}: FilterPanelProps) {
  const {
    schemaIds, setSchemaIds, schemaNames, setSchemaNames,
    tableIds, setTableIds, tableNames, setTableNames,
    columnIds, setColumnIds, columnNames, setColumnNames,
  } = useScope();

  return (
    <Card sx={{ position: { md: 'sticky' }, top: { md: 16 }, borderRadius: 2 }} elevation={2}>
      <CardContent>
        <Typography variant="h6" sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
          <Filter width={20} height={20} />
          Filters
        </Typography>
        <TextField
          fullWidth
          variant="outlined"
          size="small"
          placeholder="Type to search..."
          value={globalSearchTerm}
          onChange={(e) => setGlobalSearchTerm(e.target.value)}
          sx={{ mb: 2 }}
        />
        <Divider sx={{ mb: 2 }} />

        <Stack spacing={2.5}>
          <Box>
            <Typography variant="caption" color="text.secondary" sx={{ mb: 0.5, display: 'block', fontWeight: 600 }}>Schema</Typography>
            <CatalogNodeTypeahead
              nodeType="schema"
              value={schemaIds}
              multiple
              onChange={(value: string[] | null) => {
                setSchemaIds((value as string[]) || []);
                setTableIds([]); setTableNames([]); setColumnIds([]); setColumnNames([]);
              }}
              onSelect={(nodes: any[]) => setSchemaNames((nodes as any[]).map(n => n.node_name))}
              label=""
              placeholder="Select schemas..."
            />
            {schemaNames.length > 0 && (
              <Box sx={{ mt: 1, display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {schemaNames.map((schema: string) => (
                  <Chip key={schema} label={schema} size="small" onDelete={() => {
                    setSchemaNames((prev: string[]) => prev.filter((s: string) => s !== schema));
                    setSchemaIds((prev: string[]) => prev.filter((_, idx: number) => schemaNames[idx] !== schema));
                    setTableIds([]); setTableNames([]); setColumnIds([]); setColumnNames([]);
                  }} />
                ))}
              </Box>
            )}
          </Box>

          <Box>
            <Typography variant="caption" color="text.secondary" sx={{ mb: 0.5, display: 'block', fontWeight: 600 }}>
              Table {schemaIds.length === 0 && <Typography component="span" variant="caption" color="text.disabled">(select schema first)</Typography>}
            </Typography>
            <CatalogNodeTypeahead
              nodeType="table"
              parentId={schemaIds.length === 1 ? schemaIds[0] : undefined}
              value={tableIds}
              multiple
              onChange={(value: string[] | null) => { setTableIds((value as string[]) || []); setColumnIds([]); setColumnNames([]); }}
              onSelect={(nodes: any[]) => setTableNames((nodes as any[]).map(n => n.node_name))}
              label=""
              placeholder={schemaIds.length > 0 ? "Select tables..." : "Select schema first"}
            />
            {tableNames.length > 0 && (
              <Box sx={{ mt: 1, display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {tableNames.map((table: string) => (
                  <Chip key={table} label={table} size="small" onDelete={() => {
                    setTableNames((prev: string[]) => prev.filter((t: string) => t !== table));
                    setTableIds((prev: string[]) => prev.filter((_, idx: number) => tableNames[idx] !== table));
                    setColumnIds([]); setColumnNames([]);
                  }} />
                ))}
              </Box>
            )}
          </Box>

          <Box>
            <Typography variant="caption" color="text.secondary" sx={{ mb: 0.5, display: 'block', fontWeight: 600 }}>
              Columns {tableIds.length === 0 && <Typography component="span" variant="caption" color="text.disabled">(select table first)</Typography>}
            </Typography>
            <CatalogNodeTypeahead
              nodeType="column"
              parentId={tableIds.length === 1 ? tableIds[0] : undefined}
              value={columnIds}
              multiple
              onChange={(value: string[] | null) => setColumnIds((value as string[]) || [])}
              onSelect={(nodes: any[]) => setColumnNames((nodes as any[]).map(n => n.node_name))}
              label=""
              placeholder={tableIds.length > 0 ? "Select columns..." : "Select table first"}
            />
            {columnNames.length > 0 && (
              <Box sx={{ mt: 1, display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {columnNames.map((col: string) => (
                  <Chip key={col} label={col} size="small" onDelete={() => {
                    setColumnNames((prev: string[]) => prev.filter((c: string) => c !== col));
                    setColumnIds((prev: string[]) => prev.filter((_, idx: number) => columnNames[idx] !== col));
                  }} />
                ))}
              </Box>
            )}
          </Box>

          <Divider />

          <Box>
            <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block', fontWeight: 600 }}>Sort By</Typography>
            <Select value={sortBy} onChange={(e) => setSortBy(e.target.value as 'confidence' | 'name' | 'none')} size="small" fullWidth>
              <MenuItem value="confidence">🎯 Confidence (High → Low)</MenuItem>
              <MenuItem value="name">🔤 Column Name (A → Z)</MenuItem>
              <MenuItem value="none">📋 Original Order</MenuItem>
            </Select>
          </Box>

          <Divider />

          <FormControlLabel
            control={<Switch checked={compactRows} onChange={(e) => setCompactRows(e.target.checked)} size="small" />}
            label={<Typography variant="caption" sx={{ fontWeight: 700 }}>Compact rows</Typography>}
          />

          <Divider />

          <Stack spacing={1}>
            <Button onClick={clearAll} variant="outlined" color="secondary" size="small" startIcon={<X width={16} height={16} />} fullWidth>
              Clear All Filters
            </Button>
            <Button onClick={selectAll} variant="outlined" size="small" startIcon={allVisibleSelected ? <X width={16} height={16} /> : <Check width={16} height={16} />} fullWidth>
              {allVisibleSelected ? 'Deselect All Visible' : 'Select All Visible'}
            </Button>
            <Button onClick={openConfirm} disabled={selectedMappingsCount === 0 || loading} variant="contained" color="success" size="small" startIcon={<Check width={16} height={16} />} fullWidth>
              Create {selectedMappingsCount} Edge{selectedMappingsCount !== 1 ? 's' : ''}
            </Button>
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
}