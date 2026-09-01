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
  DriveFileMove as MoveIcon,
} from '@mui/icons-material';
import { Menu, MenuItem, ListItemIcon, ListItemText } from '@mui/material';
import { useState } from 'react';
import type { Field } from '../../../types/entity-schema';
import { SubtypeScopeIcon } from '../../../../components/common/CoreCustomIcons';

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
  onMoveField?: (field: Field, targetScope: string) => void;
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
  onMoveField,
  onSort,
  getValidationIcon,
}: FieldsTabProps) {
  const [moveMenuAnchor, setMoveMenuAnchor] = useState<null | HTMLElement>(null);
  const [fieldForMove, setFieldForMove] = useState<Field | null>(null);

  const handleOpenMoveMenu = (e: React.MouseEvent<HTMLElement>, field: Field) => {
    setMoveMenuAnchor(e.currentTarget);
    setFieldForMove(field);
  };

  const handleCloseMoveMenu = () => {
    setMoveMenuAnchor(null);
    setFieldForMove(null);
  };

  const handleSelectMoveTarget = (targetScope: string) => {
    if (fieldForMove && onMoveField) {
      onMoveField(fieldForMove, targetScope);
    }
    handleCloseMoveMenu();
  };
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
    if (selectedNode?.type !== 'subtype' || !selectedNode?.subtypeKey) return false;
    const subtypeFields = businessObject?.subtypes?.[selectedNode.subtypeKey]?.subtypeFields || [];
    const fieldIdent = (field.key || field.technicalName || field.id || field.name || '').toLowerCase();
    const isAssigned = subtypeFields.some((f: any) =>
      (f.key || f.technicalName || f.id || f.name || '').toLowerCase() === fieldIdent
    );
    return !isAssigned;
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1 }}>
      <Box sx={{ p: 3, borderBottom: '1px solid', borderBottomColor: 'divider' }}>
        <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }} spacing={2}>
          <Box>
            {selectedNode?.type === 'subtype' ? (
              <>
                <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
                  Subtype: {businessObject?.subtypes?.[selectedNode.subtypeKey!]?.displayName || selectedNode.subtypeKey}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {showInheritedFields
                    ? 'Showing full union (subtype-assigned fields + baseline inherited fields).'
                    : 'Showing subtype-assigned fields only.'}
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
                color={showInheritedFields ? 'info' : 'primary'}
                size="small"
                onClick={onToggleInherited}
              >
                {showInheritedFields ? 'Hide Baseline (Showing Union)' : 'Include Baseline Core Fields'}
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
            {sortedFilteredFields.map((field, idx) => {
              const isInherited = isInheritedField(field);
              const typeConfig = getDataTypeConfig(field.type);

              return (
                <TableRow
                  key={field.id || field.fieldId || field.field_id || field.key || `${field.technicalName || field.name || idx}-${idx}`}
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
                    <TableCell align="center">
                      <SubtypeScopeIcon isInherited={isInherited} fontSize={18} />
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
                      {businessObject?.subtypes && Object.keys(businessObject.subtypes).length > 0 && onMoveField && (
                        <Tooltip title="Move to Subtype / Root">
                          <IconButton
                            size="small"
                            onClick={(e) => handleOpenMoveMenu(e, field)}
                            sx={{ color: 'info.main', '&:hover': { bgcolor: 'info.light', color: 'info.dark' } }}
                          >
                            <MoveIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      )}
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

            {sortedFilteredFields.length === 0 && (
              <TableRow>
                <TableCell colSpan={selectedNode?.type === 'subtype' ? 6 : 5} align="center" sx={{ py: 6 }}>
                  <Typography variant="body1" color="text.secondary" sx={{ mb: 1, fontWeight: 500 }}>
                    {selectedNode?.type === 'subtype'
                      ? !showInheritedFields
                        ? `No fields specifically assigned to subtype '${businessObject?.subtypes?.[selectedNode.subtypeKey]?.displayName || selectedNode.subtypeKey}' yet.`
                        : 'No fields found.'
                      : 'No fields defined for this business object.'}
                  </Typography>
                  {selectedNode?.type === 'subtype' && !showInheritedFields && (
                    <Stack direction="row" spacing={2} justifyContent="center" sx={{ mt: 2 }}>
                      <Button variant="outlined" size="small" onClick={onToggleInherited}>
                        Include Baseline Core Fields
                      </Button>
                      <Button variant="contained" size="small" onClick={onAddField}>
                        Add Field to Subtype
                      </Button>
                    </Stack>
                  )}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Move Target Menu */}
      <Menu
        anchorEl={moveMenuAnchor}
        open={Boolean(moveMenuAnchor)}
        onClose={handleCloseMoveMenu}
        transformOrigin={{ horizontal: 'right', vertical: 'top' }}
        anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
      >
        <MenuItem disabled sx={{ fontSize: '0.75rem', fontWeight: 700, opacity: 1, textTransform: 'uppercase' }}>
          Move Field To
        </MenuItem>
        <MenuItem onClick={() => handleSelectMoveTarget('root')}>
          <ListItemIcon>
            <MoveIcon fontSize="small" color="primary" />
          </ListItemIcon>
          <ListItemText primary={`Root Object (${businessObject?.displayName || 'Main'})`} />
        </MenuItem>
        {businessObject?.subtypes && Object.entries(businessObject.subtypes).map(([key, st]: [string, any]) => (
          <MenuItem key={key} onClick={() => handleSelectMoveTarget(key)}>
            <ListItemIcon>
              <MoveIcon fontSize="small" color="info" />
            </ListItemIcon>
            <ListItemText primary={`Subtype: ${st.displayName || st.name || key}`} />
          </MenuItem>
        ))}
      </Menu>
    </Box>
  );
}
