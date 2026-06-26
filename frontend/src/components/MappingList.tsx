import { Box, Card, Typography, Stack, Pagination, Select, MenuItem } from '@mui/material';
import { Database } from 'lucide-react';
import { MappingRow } from './MappingRow';
import { getMappingUniqueId } from './semantic-mapper/utils/mappingId';

interface MappingListProps {
  loading: boolean;
  mappings: any[];
  paginatedMappings: any[];
  page: number;
  setPage: (page: number) => void;
  pageSize: number;
  setPageSize: (size: number) => void;
  // MappingRow props
  editingMapping: any | null;
  savedRows: Set<string>;
  compactRows: boolean;
  keyboardExpanded: boolean;
  setKeyboardExpanded: (expanded: boolean) => void;
  toggleMapping: (id: string) => void;
  openEditing: (mapping: any) => void;
  confirmEditing: (id: string, term?: string) => void;
  cancelEditing: (id: string) => void;
  searchSemanticTerms: (query: string) => Promise<any[]>;
  selectSemanticTerm: (term: any, mappingId: string) => void;
  handleCreateAndSelectTerm: (mappingId: string, termName: string) => Promise<any>;
  setOverride: (id: string, value: boolean) => void;
  setIgnored: (id: string, value: boolean) => void;
  openReplaceConfirm: (index: number) => void;
  openLineageModal: (mapping: any) => void;
}

export function MappingList(props: MappingListProps) {
  if (props.loading && props.mappings.length === 0) {
    return null; // Let parent handle initial loading state
  }

  if (props.mappings.length === 0) {
    return (
      <Card sx={{ p: 6, textAlign: 'center', borderRadius: 2 }} elevation={2}>
        <Database width={64} height={64} style={{ margin: '0 auto 16px', color: '#ccc' }} />
        <Typography variant="h6" sx={{ mb: 1 }}>No Mappings Found</Typography>
        <Typography variant="body2" color="text.secondary">
          No database columns found to map. Check your database connection or scope filters.
        </Typography>
      </Card>
    );
  }

  return (
    <Stack spacing={2}>
      {props.paginatedMappings.map((mapping, idx) => (
        <MappingRow key={getMappingUniqueId(mapping)} mapping={mapping} idx={idx} {...props} />
      ))}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mt: 1, flexWrap: 'wrap', gap: 1 }}>
        <Pagination count={Math.ceil(props.mappings.length / props.pageSize)} page={props.page + 1} onChange={(_e, value) => props.setPage(value - 1)} size="small" />
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography variant="caption" color="text.secondary">{props.mappings.length} items</Typography>
          <Select size="small" value={props.pageSize} onChange={(e) => { props.setPageSize(Number(e.target.value)); props.setPage(0); }}>
            <MenuItem value={10}>10</MenuItem> <MenuItem value={25}>25</MenuItem> <MenuItem value={50}>50</MenuItem>
          </Select>
        </Box>
      </Box>
    </Stack>
  );
}