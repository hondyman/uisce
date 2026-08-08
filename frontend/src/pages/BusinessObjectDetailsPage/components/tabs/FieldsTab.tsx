import {
  Box,
  Typography,
  Button,
  TextField,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Tooltip,
  InputAdornment,
  TableSortLabel,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Search as SearchIcon,
  MoreVert as MoreVertIcon,
  InfoOutlined as InfoOutlinedIcon,
  Numbers as NumberIcon,
  CalendarToday as DateIcon,
  Code as JsonIcon,
  ToggleOn as BooleanIcon,
  ShortText as TextIcon,
} from '@mui/icons-material';
import type { Field } from '../../../types/entity-schema';

type SortConfig = { key: string; direction: 'asc' | 'desc' };

interface FieldsTabProps {
  selectedNode: any;
  businessObject: any;
  searchFilter: string;
  showInheritedFields: boolean;
  sortedFilteredFields: Field[];
  sortConfig: SortConfig;
  onSearchChange: (value: string) => void;
  onToggleInherited: () => void;
  onAddField: () => void;
  onEditField: (field: Field) => void;
  onDeleteField: (field: Field) => void;
  onSort: (key: string) => void;
  getValidationIcon: (validation: any) => React.ReactNode;
}

export function FieldsTab({
  selectedNode,
  businessObject,
  searchFilter,
  showInheritedFields,
  sortedFilteredFields,
  sortConfig,
  onSearchChange,
  onToggleInherited,
  onAddField,
  onEditField,
  onDeleteField,
  onSort,
  getValidationIcon,
}: FieldsTabProps) {
  const getDataTypeConfig = (type: string) => {
    const t = type.toLowerCase();
    if (t.includes('int') || t.includes('number') || t.includes('decimal') || t.includes('float') || t.includes('double')) {
      return { icon: <NumberIcon sx={{ fontSize: 16 }} />, color: 'success', label: type };
    } else if (t.includes('date') || t.includes('time')) {
      return { icon: <DateIcon sx={{ fontSize: 16 }} />, color: 'secondary', label: type };
    } else if (t.includes('bool')) {
      return { icon: <BooleanIcon sx={{ fontSize: 16 }} />, color: 'warning', label: type };
    } else if (t.includes('json') || t.includes('obj') || t.includes('arr')) {
      return { icon: <JsonIcon sx={{ fontSize: 16 }} />, color: 'info', label: type };
    } else {
      return { icon: <TextIcon sx={{ fontSize: 16 }} />, color: 'primary', label: type };
    }
  };

  const isInheritedField = (field: Field) => {
    return selectedNode?.type === 'subtype' &&
      showInheritedFields &&
      (businessObject?.coreFields?.some((f: Field) => f.key === field.key) ||
        businessObject?.customFields?.some((f: Field) => f.key === field.key));
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1 }}>
      <Box sx={{ p: 3, borderBottom: '1px solid', borderBottomColor: 'divider' }}>
        <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }} spacing={2}>
          <Box>
            {selectedNode?.type === 'subtype' ? (
              <>
                <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
                  Fields for '{businessObject?.subtypes?.[selectedNode.subtypeKey!]?.displayName || selectedNode.subtypeKey}'
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Showing {showInheritedFields ? 'inherited + subtype-specific' : 'subtype-specific only'} fields.
                </Typography>
              </>
            ) : (
              <>
                <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
                  Fields for '{businessObject?.displayName}'
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Define data types, constraints, and display logic for this node.
                </Typography>
              </>
            )}
          </Box>
          <Stack direction="row" spacing={2} alignItems="center">
            {selectedNode?.type === 'subtype' && (
              <Button
                variant={showInheritedFields ? 'contained' : 'outlined'}
                color="primary"
                size="small"
                onClick={onToggleInherited}
              >
                {showInheritedFields ? 'Hide Inherited' : 'Show Inherited'}
              </Button>
            )}
            <Button
              variant="contained"
              color="primary"
              startIcon={<AddIcon />}
              size="small"
              onClick={onAddField}
            >
              Add Field
            </Button>
          </Stack>
        </Stack>
      </Box>

      <TextField
        fullWidth
        placeholder="Search fields..."
        variant="standard"
        value={searchFilter}
        onChange={(e) => onSearchChange(e.target.value)}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon fontSize="small" />
            </InputAdornment>
          ),
        }}
        sx={{
          px: 3,
          py: 1,
          mb: 2,
          '& .MuiInput-underline:before': {
            borderBottomColor: 'divider',
          },
        }}
      />

      <TableContainer sx={{ flex: 1 }}>
        <Table stickyHeader>
          <TableHead>
            <TableRow sx={{ bgcolor: 'action.hover' }}>
              <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                <TableSortLabel
                  active={sortConfig.key === 'technicalName'}
                  direction={sortConfig.key === 'technicalName' ? sortConfig.direction : 'asc'}
                  onClick={() => onSort('technicalName')}
                >
                  Technical Name
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                <TableSortLabel
                  active={sortConfig.key === 'businessName'}
                  direction={sortConfig.key === 'businessName' ? sortConfig.direction : 'asc'}
                  onClick={() => onSort('businessName')}
                >
                  Display Label
                </TableSortLabel>
              </TableCell>
              {selectedNode?.type === 'subtype' && (
                <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                  Type
                </TableCell>
              )}
              <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                <TableSortLabel
                  active={sortConfig.key === 'type'}
                  direction={sortConfig.key === 'type' ? sortConfig.direction : 'asc'}
                  onClick={() => onSort('type')}
                >
                  Data Type
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                Validation
              </TableCell>
              <TableCell align="right" sx={{ fontWeight: 700, textTransform: 'uppercase', fontSize: '0.75rem' }}>
                Actions
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {sortedFilteredFields.map((field) => {
              const isInherited = isInheritedField(field);
              const typeConfig = getDataTypeConfig(field.type);

              return (
                <TableRow
                  key={field.key}
                  hover
                  sx={{
                    '&:hover': { bgcolor: 'action.hover' },
                    opacity: isInherited ? 0.8 : 1,
                  }}
                >
                  <TableCell sx={{ fontWeight: 600, fontFamily: 'monospace', fontSize: '0.85rem' }}>
                    {field.technicalName || field.name}
                  </TableCell>
                  <TableCell>
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {field.businessName || field.name}
                      </Typography>
                      {field.description && (
                        <Tooltip title={field.description} arrow placement="right">
                          <InfoOutlinedIcon sx={{ fontSize: 16, color: 'text.secondary', cursor: 'help' }} />
                        </Tooltip>
                      )}
                    </Stack>
                  </TableCell>
                  {selectedNode?.type === 'subtype' && (
                    <TableCell>
                      <Chip
                        label={isInherited ? 'Inherited' : 'Assigned'}
                        size="small"
                        variant="filled"
                        color={isInherited ? 'default' : 'primary'}
                        sx={{ fontWeight: 600, fontSize: '0.7rem' }}
                      />
                    </TableCell>
                  )}
                  <TableCell>
                    <Chip
                      icon={typeConfig.icon}
                      label={typeConfig.label}
                      size="small"
                      color={typeConfig.color as any}
                      variant="outlined"
                      sx={{ fontWeight: 500, border: '1px solid' }}
                    />
                  </TableCell>
                  <TableCell>
                    <Stack direction="row" spacing={1} alignItems="center">
                      {getValidationIcon(field.validation)}
                      <Typography variant="body2" color="text.secondary">
                        {field.validationMessage || '-'}
                      </Typography>
                    </Stack>
                  </TableCell>
                  <TableCell align="right">
                    <Stack direction="row" spacing={0.5} justifyContent="flex-end">
                      <Tooltip title="Edit field">
                        <IconButton size="small" onClick={() => onEditField(field)} sx={{ '&:hover': { color: 'primary.main' } }}>
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <Tooltip title="Delete field">
                        <IconButton size="small" onClick={() => onDeleteField(field)} sx={{ '&:hover': { color: 'error.main' } }}>
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      <IconButton size="small" sx={{ '&:hover': { color: 'primary.main' } }}>
                        <MoreVertIcon fontSize="small" />
                      </IconButton>
                    </Stack>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}
