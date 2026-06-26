import React from 'react';
import {
  Box,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Select,
  MenuItem,
  Paper,
  FormControl,
  Stack,
  Alert,
  Tooltip,
  IconButton,
  Button
} from '@mui/material';
import { 
  Link as LinkIcon, 
  LinkOff as LinkOffIcon, 
  Delete as DeleteIcon,
  Add as AddIcon,
  Info as InfoIcon
} from '@mui/icons-material';
import { AvailableSemanticTerm } from '../../hooks/useBORelationships';
import { DataTypeChip } from './DataTypeChip';
import { TermInfoTooltip } from './TermInfoTooltip';

interface SemanticMappingTabProps {
  fields: any[];
  availableTerms: AvailableSemanticTerm[];
  onUpdateField: (index: number, updates: any) => void;
  onAddField: () => void;
  onRemoveField: (index: number) => void;
  readOnly?: boolean;
}

const ROLES = [
  'DIMENSION',
  'MEASURE',
  'VALIDITY_START',
  'VALIDITY_END',
  'EVENT_DATE',
  'PARTITION_KEY'
];

export const SemanticMappingTab: React.FC<SemanticMappingTabProps> = ({
  fields,
  availableTerms,
  onUpdateField,
  onAddField,
  onRemoveField,
  readOnly = false,
}) => {
  return (
    <Box>
      <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 2 }}>
        <Box>
          <Typography variant="subtitle2" sx={{ fontWeight: 600, color: 'primary.main' }}>
            🔗 Fields & Mappings
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Manage your business object fields and their semantic mappings.
          </Typography>
        </Box>
        <Button
          variant="contained"
          size="small"
          startIcon={<AddIcon />}
          onClick={onAddField}
          disabled={readOnly}
        >
          Add Fields
        </Button>
      </Stack>

      {fields.length === 0 ? (
        <Alert severity="info" sx={{ mt: 2 }} action={
          <Button color="inherit" size="small" onClick={onAddField}>
            Add Fields
          </Button>
        }>
          No fields defined yet. Click "Add Fields" to select from the semantic layer.
        </Alert>
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead sx={{ bgcolor: 'action.hover' }}>
              <TableRow>
                <TableCell sx={{ fontWeight: 600 }}>Field Name</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Semantic Term</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Role</TableCell>
                <TableCell sx={{ fontWeight: 600 }} align="center">Status</TableCell>
                <TableCell sx={{ fontWeight: 600 }} align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {fields.map((field, idx) => (
                <TableRow key={field.key || idx} hover>
                  <TableCell>
                    <Stack direction="row" alignItems="center" spacing={1}>
                      <Box>
                         <Typography variant="body2" sx={{ fontWeight: 500 }}>
                          {field.name || field.display_name || field.key}
                        </Typography>
                        <Stack direction="row" spacing={0.5} alignItems="center">
                           <Typography variant="caption" color="text.secondary" fontFamily="monospace">
                            {field.key}
                          </Typography>
                          <DataTypeChip type={field.type} sx={{ height: 18, fontSize: '0.65rem' }} />
                        </Stack>
                      </Box>
                      {/* Show info tooltip for the FIELD itself if it has metadata, or fall back to mapped term */}
                      <TermInfoTooltip term={{
                        node_name: field.name || field.key,
                        description: field.description,
                        qualified_path: field.qualified_path, // If passed
                        properties: { sql: field.source_column || field.sql } 
                      }} />
                    </Stack>
                  </TableCell>
                  <TableCell>
                    <FormControl fullWidth size="small" variant="standard">
                      <Select
                        value={field.semanticTermId || ''}
                        onChange={(e) => onUpdateField(idx, { 
                          semanticTermId: e.target.value,
                          semanticTermName: availableTerms.find(t => t.id === e.target.value)?.node_name,
                          // Also update type if term changes? Maybe optional.
                        })}
                        displayEmpty
                        disableUnderline
                        readOnly={readOnly}
                        renderValue={(selected) => {
                           if (!selected) return <em>Unmapped</em>;
                           const term = availableTerms.find(t => t.id === selected);
                           return term ? term.node_name : selected;
                        }}
                      >
                        <MenuItem value="">
                          <em>None (Unmapped)</em>
                        </MenuItem>
                        {availableTerms.map((term) => (
                          <MenuItem key={term.id} value={term.id}>
                            <Stack direction="row" spacing={1} alignItems="center" sx={{ width: '100%' }}>
                               <Box sx={{ flex: 1 }}>
                                 <Typography variant="body2">{term.display_name || term.node_name}</Typography>
                                 <Typography variant="caption" color="text.secondary" display="block">{term.qualified_path}</Typography>
                               </Box>
                               <DataTypeChip type={term.dataType} />
                            </Stack>
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  </TableCell>
                  <TableCell>
                     <FormControl fullWidth size="small" variant="standard">
                      <Select
                        value={field.role || 'DIMENSION'}
                        onChange={(e) => onUpdateField(idx, { role: e.target.value })}
                        disableUnderline
                        readOnly={readOnly}
                      >
                        {ROLES.map((role) => (
                          <MenuItem key={role} value={role}>
                            {role}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  </TableCell>
                  <TableCell align="center">
                     {field.semanticTermId ? (
                       <Tooltip title="Mapped to Semantic Layer">
                         <LinkIcon color="success" fontSize="small" />
                       </Tooltip>
                     ) : (
                       <Tooltip title="Not Mapped">
                         <LinkOffIcon color="disabled" fontSize="small" />
                       </Tooltip>
                     )}
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title="Remove Field">
                      <IconButton size="small" onClick={() => onRemoveField(idx)} disabled={readOnly} color="error">
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {availableTerms.length === 0 && fields.length > 0 && (
        <Alert severity="warning" sx={{ mt: 3 }} icon={<InfoIcon />}>
          No suggested semantic terms found for this driver table. Ensure the table has been scanned and enriched in the Semantic Explorer.
        </Alert>
      )}
    </Box>
  );
};
