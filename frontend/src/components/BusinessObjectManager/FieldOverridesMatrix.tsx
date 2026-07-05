import {
  Box,
  Chip,
  FormControl,
  IconButton,
  MenuItem,
  Select,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  AutoAwesome as AutoAwesomeIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';

import type {
  BindingRequirement,
  TermColumnMapping,
  WizardField,
  WizardSemanticTerm,
} from './bindingWizard.types';

interface FieldOverridesMatrixProps {
  fields: WizardField[];
  termMap: Record<string, WizardSemanticTerm>;
  onUpdateField: (termId: string, updates: Partial<WizardField>) => void;
  onMappingChange: (field: WizardField, mapping: TermColumnMapping | 'unresolved') => void;
  onRemoveField: (termId: string) => void;
}

const ROLE_OPTIONS = ['KEY', 'DIMENSION', 'MEASURE', 'ATTRIBUTE', 'IDENTIFIER'];
const BINDING_REQUIREMENT_OPTIONS: BindingRequirement[] = [
  'REQUIRED',
  'OPTIONAL',
  'BACKEND_SPECIFIC',
  'CALCULATED',
  'INTERNAL',
];

function getSourceChip(source: WizardField['eligibilitySource']) {
  switch (source) {
    case 'INHERITED':
      return (
        <Chip
          label="Inherited"
          size="small"
          color="default"
          variant="outlined"
          icon={<AutoAwesomeIcon fontSize="small" />}
        />
      );
    case 'OVERRIDE':
      return <Chip label="Override" size="small" color="warning" icon={<EditIcon fontSize="small" />} />;
    case 'DIRECT':
      return <Chip label="Tenant Custom" size="small" color="primary" />;
    case 'MANUAL':
      return <Chip label="Manual" size="small" color="info" />;
    case 'CALCULATED':
      return <Chip label="Calculated" size="small" color="success" />;
    case 'RELATED':
      return <Chip label="Related" size="small" color="secondary" />;
    default:
      return <Chip label={source} size="small" />;
  }
}

function getStatusColor(status: WizardField['bindingStatus']) {
  switch (status) {
    case 'RESOLVED':
      return 'success' as const;
    case 'PARTIAL':
      return 'warning' as const;
    case 'UNRESOLVED':
      return 'error' as const;
    default:
      return 'default' as const;
  }
}

export default function FieldOverridesMatrix({
  fields,
  termMap,
  onUpdateField,
  onMappingChange,
  onRemoveField,
}: FieldOverridesMatrixProps) {
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Semantic Field</TableCell>
            <TableCell>Origin Scope</TableCell>
            <TableCell>Role</TableCell>
            <TableCell>Requirement</TableCell>
            <TableCell>Column Mapping</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Local Override</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {fields.map((field) => {
            const term = termMap[field.semanticTermId];
            const mappings = term?.mappings || [];
            const mappingValue = field.selectedMapping?.columnNodeId || 'unresolved';
            const isInherited = field.eligibilitySource === 'INHERITED';
            const isOverride = field.eligibilitySource === 'OVERRIDE';
            const canOverride = isInherited || isOverride;
            const canRemove = field.eligibilitySource !== 'INHERITED';

            return (
              <TableRow
                key={field.semanticTermId}
                sx={{
                  bgcolor: isOverride ? 'warning.50' : 'inherit',
                }}
              >
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 500 }}>
                    {field.displayName}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {field.semanticTermName}
                  </Typography>
                </TableCell>
                <TableCell>{getSourceChip(field.eligibilitySource)}</TableCell>
                <TableCell>
                  <FormControl size="small" fullWidth>
                    <Select
                      value={field.role}
                      disabled={isInherited && !isOverride}
                      onChange={(e) => onUpdateField(field.semanticTermId, { role: e.target.value })}
                    >
                      {ROLE_OPTIONS.map((r) => (
                        <MenuItem key={r} value={r}>
                          {r}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </TableCell>
                <TableCell>
                  <FormControl size="small" fullWidth>
                    <Select
                      value={field.bindingRequirement}
                      disabled={isInherited && !isOverride}
                      onChange={(e) =>
                        onUpdateField(field.semanticTermId, {
                          bindingRequirement: e.target.value as BindingRequirement,
                        })
                      }
                    >
                      {BINDING_REQUIREMENT_OPTIONS.map((r) => (
                        <MenuItem key={r} value={r}>
                          {r.replace(/_/g, ' ')}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </TableCell>
                <TableCell>
                  <FormControl size="small" fullWidth>
                    <Select
                      value={mappingValue}
                      disabled={isInherited && !isOverride}
                      onChange={(e) => {
                        const value = e.target.value;
                        if (value === 'unresolved') {
                          onMappingChange(field, 'unresolved');
                        } else {
                          const mapping = mappings.find((m) => (m.columnNodeId || m.columnName) === value);
                          if (mapping) onMappingChange(field, mapping);
                        }
                      }}
                    >
                      <MenuItem value="unresolved">
                        <em>Unresolved</em>
                      </MenuItem>
                      {mappings.map((m) => (
                        <MenuItem key={m.columnNodeId || m.columnName} value={m.columnNodeId || m.columnName}>
                          {m.tableName}.{m.columnName}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </TableCell>
                <TableCell>
                  <Chip label={field.bindingStatus} size="small" color={getStatusColor(field.bindingStatus)} />
                </TableCell>
                <TableCell>
                  {canOverride ? (
                    <Stack spacing={1}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Switch
                          size="small"
                          checked={isOverride || !!field.hasLocalOverride}
                          onChange={(e) => {
                            const enabled = e.target.checked;
                            onUpdateField(field.semanticTermId, {
                              hasLocalOverride: enabled,
                              eligibilitySource: enabled ? 'OVERRIDE' : 'INHERITED',
                            });
                          }}
                        />
                        <Typography variant="caption" color="text.secondary">
                          Enable customizations
                        </Typography>
                      </Box>
                      {(isOverride || field.hasLocalOverride) && (
                        <>
                          {field.role === 'MEASURE' && (
                            <TextField
                              size="small"
                              placeholder="Local formula expression..."
                              value={field.localExpressionOverride || ''}
                              onChange={(e) =>
                                onUpdateField(field.semanticTermId, {
                                  localExpressionOverride: e.target.value,
                                })
                              }
                              inputProps={{ style: { fontSize: 12, fontFamily: 'monospace' } }}
                              fullWidth
                            />
                          )}
                          <TextField
                            size="small"
                            placeholder="Audit justification..."
                            value={field.overrideReason || ''}
                            onChange={(e) =>
                              onUpdateField(field.semanticTermId, { overrideReason: e.target.value })
                            }
                            inputProps={{ style: { fontSize: 12 } }}
                            fullWidth
                          />
                        </>
                      )}
                    </Stack>
                  ) : (
                    <Typography variant="caption" color="text.secondary">
                      N/A
                    </Typography>
                  )}
                </TableCell>
                <TableCell align="right">
                  <Tooltip title={canRemove ? 'Remove field' : 'Cannot remove core inherited fields'}>
                    <span>
                      <IconButton
                        size="small"
                        color="error"
                        disabled={!canRemove}
                        onClick={() => onRemoveField(field.semanticTermId)}
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </span>
                  </Tooltip>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
