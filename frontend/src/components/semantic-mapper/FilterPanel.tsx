import type { ChangeEvent } from 'react';
import {
  Card, CardContent, Typography, TextField, Divider, Stack, Button,
  Chip, Box, IconButton, Tooltip
} from '@mui/material';
import { Filter, X, Check, Database, Link, Unlink } from 'lucide-react';
import CatalogNodeTypeahead from '../CatalogNodeTypeahead';
import { useScope } from '../../contexts/ScopeContext';

type CatalogNode = {
  id: string
  node_name: string
  qualified_path: string
  catalog_type: string
  parent_id?: string
  properties?: any
  created_at: string
};

interface FilterPanelProps {
  globalSearchTerm: string;
  setGlobalSearchTerm: (term: string) => void;
  // sort handled in header now
  mappedFilter: 'all' | 'mapped' | 'unmapped';
  setMappedFilter: (v: 'all' | 'mapped' | 'unmapped') => void;
  mappingCounts: { all: number; mapped: number; unmapped: number };
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
  mappedFilter,
  setMappedFilter,
  mappingCounts,
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
    <Card sx={{ borderRadius: 2 }} elevation={2}>
      <CardContent>
        <Typography variant="h6" sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
          <Filter width={20} height={20} />
          Filters
        </Typography>

        {/* Search input at the top */}
        <Box sx={{ mb: 2 }}>
                <TextField
                  variant="outlined"
                  size="small"
                  placeholder="Type to search..."
                  value={globalSearchTerm}
                  onChange={(e: ChangeEvent<HTMLInputElement>) => setGlobalSearchTerm(e.target.value)}
                  fullWidth
                />
        </Box>

        {/* Filter icons with counts */}
        <Box sx={{ display: 'flex', gap: 1, mb: 2, justifyContent: 'center' }}>
          <Tooltip title={`All mappings (${mappingCounts.all})`}>
            <IconButton
              size="small"
              onClick={() => setMappedFilter('all')}
              sx={{
                bgcolor: mappedFilter === 'all' ? 'primary.main' : 'grey.100',
                color: mappedFilter === 'all' ? 'white' : 'text.secondary',
                '&:hover': { bgcolor: mappedFilter === 'all' ? 'primary.dark' : 'grey.200' }
              }}
            >
              <Database width={16} height={16} />
              <Typography variant="caption" sx={{ ml: 0.5, fontSize: '10px' }}>
                {mappingCounts.all}
              </Typography>
            </IconButton>
          </Tooltip>

          <Tooltip title={`Mapped (${mappingCounts.mapped})`}>
            <IconButton
              size="small"
              onClick={() => setMappedFilter('mapped')}
              sx={{
                bgcolor: mappedFilter === 'mapped' ? 'success.main' : 'grey.100',
                color: mappedFilter === 'mapped' ? 'white' : 'text.secondary',
                '&:hover': { bgcolor: mappedFilter === 'mapped' ? 'success.dark' : 'grey.200' }
              }}
            >
              <Link width={16} height={16} />
              <Typography variant="caption" sx={{ ml: 0.5, fontSize: '10px' }}>
                {mappingCounts.mapped}
              </Typography>
            </IconButton>
          </Tooltip>

          <Tooltip title={`Unmapped (${mappingCounts.unmapped})`}>
            <IconButton
              size="small"
              onClick={() => setMappedFilter('unmapped')}
              sx={{
                bgcolor: mappedFilter === 'unmapped' ? 'warning.main' : 'grey.100',
                color: mappedFilter === 'unmapped' ? 'white' : 'text.secondary',
                '&:hover': { bgcolor: mappedFilter === 'unmapped' ? 'warning.dark' : 'grey.200' }
              }}
            >
              <Unlink width={16} height={16} />
              <Typography variant="caption" sx={{ ml: 0.5, fontSize: '10px' }}>
                {mappingCounts.unmapped}
              </Typography>
            </IconButton>
          </Tooltip>
        </Box>

        <Divider sx={{ mb: 2 }} />

        <Stack spacing={2.5}>
          <Box>
            <Typography variant="caption" color="text.secondary" sx={{ mb: 0.5, display: 'block', fontWeight: 600 }}>Schema</Typography>
            <CatalogNodeTypeahead
              nodeType="schema"
              value={schemaIds}
              multiple
              onChange={(value: string | string[] | null) => {
                const v = (value as string[] | string | null) || [];
                setSchemaIds(Array.isArray(v) ? v : [v].filter(Boolean));
                setTableIds([]); setTableNames([]); setColumnIds([]); setColumnNames([]);
              }}
              onSelect={(nodes: CatalogNode | CatalogNode[] | null) => {
                if (!nodes) return setSchemaNames([]);
                if (Array.isArray(nodes)) setSchemaNames(nodes.map((n: CatalogNode) => n.node_name));
                else setSchemaNames([nodes.node_name]);
              }}
              label=""
              placeholder="Select schemas..."
            />
            {schemaNames.length > 0 && (
              <Box sx={{ mt: 1, display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {schemaNames.map((schema: string) => (
                  <Chip key={schema} label={schema} size="small" onDelete={() => {
                    // Use current values from context rather than functional updater to match typed signatures
                    setSchemaNames(schemaNames.filter((s: string) => s !== schema));
                    setSchemaIds(schemaIds.filter((_: string, idx: number) => schemaNames[idx] !== schema));
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
              onChange={(value: string | string[] | null) => {
                const v = (value as string[] | string | null) || [];
                setTableIds(Array.isArray(v) ? v : [v].filter(Boolean));
                setColumnIds([]); setColumnNames([]);
              }}
              onSelect={(nodes: CatalogNode | CatalogNode[] | null) => {
                if (!nodes) return setTableNames([]);
                if (Array.isArray(nodes)) setTableNames(nodes.map((n: CatalogNode) => n.node_name));
                else setTableNames([nodes.node_name]);
              }}
              label=""
              placeholder={schemaIds.length > 0 ? "Select tables..." : "Select schema first"}
            />
                  {tableNames.length > 0 && (
                <Box sx={{ mt: 1, display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {tableNames.map((table: string) => (
                    <Chip key={table} label={table} size="small" onDelete={() => {
                      setTableNames(tableNames.filter((t: string) => t !== table));
                      setTableIds(tableIds.filter((_: string, idx: number) => tableNames[idx] !== table));
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
              onChange={(value: string | string[] | null) => {
                const v = (value as string[] | string | null) || [];
                setColumnIds(Array.isArray(v) ? v : [v].filter(Boolean));
              }}
              onSelect={(nodes: CatalogNode | CatalogNode[] | null) => {
                if (!nodes) return setColumnNames([]);
                if (Array.isArray(nodes)) setColumnNames(nodes.map((n: CatalogNode) => n.node_name));
                else setColumnNames([nodes.node_name]);
              }}
              label=""
              placeholder={tableIds.length > 0 ? "Select columns..." : "Select table first"}
            />
                  {columnNames.length > 0 && (
                <Box sx={{ mt: 1, display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {columnNames.map((col: string) => (
                    <Chip key={col} label={col} size="small" onDelete={() => {
                      setColumnNames(columnNames.filter((c: string) => c !== col));
                      setColumnIds(columnIds.filter((_: string, idx: number) => columnNames[idx] !== col));
                    }} />
                  ))}
                </Box>
              )}
          </Box>

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
