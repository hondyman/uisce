import React, { useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  Chip,
  Typography,
  Box,
  Tooltip,
  Stack,
  alpha,
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import PropertiesModal from '../../components/properties/PropertiesModal';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import CategoryIcon from '@mui/icons-material/Category';
import type { NodeType, NodeProperty } from '../../types/nodeTypes';

interface NodeTypeTableProps {
  nodeTypes: NodeType[];
  onEdit: (nodeType: NodeType) => void;
  onDelete: (id: string) => void;
}

// Convert NodeProperty to DisplayProperty for PropertiesModal
const convertNodePropertiesToDisplayProperties = (properties: NodeProperty[]) => {
  return properties.map(prop => ({
    name: prop.name,
    label: prop.label,
    data_type: prop.data_type === 'integer' || prop.data_type === 'float' ? 'number' :
               prop.data_type === 'text' || prop.data_type === 'json' ? 'string' :
               prop.data_type,
    input_type: prop.input_type === 'date-picker' ? 'date' :
                prop.input_type === 'textarea' || prop.input_type === 'number' || prop.input_type === 'json-editor' ? 'text' :
                prop.input_type,
    required: !prop.nullable,
    options: prop.options || [],
    original_data_type: prop.data_type,
    original_input_type: prop.input_type,
    nullable: prop.nullable,
    default_value: prop.default_value,
    format: prop.format,
    validation: prop.validation,
    syntax_language: prop.syntax_language || null,
  }));
};

export const NodeTypeTable: React.FC<NodeTypeTableProps> = ({
  nodeTypes,
  onEdit,
  onDelete,
}) => {
  const [propertiesModalOpen, setPropertiesModalOpen] = useState(false);
  const [selectedNodeType, setSelectedNodeType] = useState<NodeType | null>(null);

  const getParentNodeType = (parentId: string | null | undefined): NodeType | undefined => {
    if (!parentId) return undefined;
    return nodeTypes.find((nt) => nt.id === parentId);
  };

  if (nodeTypes.length === 0) {
    return (
      <Paper
        elevation={0}
        sx={{
          p: 8,
          textAlign: 'center',
          bgcolor: 'grey.50',
          border: 1,
          borderColor: 'divider',
          borderRadius: 2,
        }}
      >
        <Box
          sx={{
            width: 64,
            height: 64,
            borderRadius: '50%',
            bgcolor: 'primary.lighter',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            mx: 'auto',
            mb: 2,
          }}
        >
          <AccountTreeIcon sx={{ fontSize: 32, color: 'primary.main' }} />
        </Box>
        <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
          No Node Types Yet
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Create your first node type to define entities in your business glossary.
        </Typography>
      </Paper>
    );
  }

  return (
    <Box>
      <TableContainer
      component={Paper}
      elevation={2}
      sx={{
        borderRadius: 2,
        overflow: 'hidden',
      }}
    >
      <Table sx={{ minWidth: 650 }}>
        <TableHead>
          <TableRow sx={{ bgcolor: 'grey.50' }}>
            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Node Type
            </TableCell>
            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Description
            </TableCell>
            <TableCell align="center" sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Properties
            </TableCell>
            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Parent Type
            </TableCell>
            <TableCell align="center" sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Status
            </TableCell>
            <TableCell align="right" sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Actions
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {nodeTypes.map((nodeType) => {
            const propertiesCount = nodeType.properties?.length || 0;
            const parentType = getParentNodeType(nodeType.parent_type_id);

            return (
              <TableRow
                key={nodeType.id}
                sx={{
                  '&:hover': {
                    bgcolor: alpha('#1976d2', 0.04),
                  },
                  transition: 'background-color 0.2s',
                }}
              >
                {/* Node Type Name Column */}
                <TableCell>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Box
                      sx={{
                        width: 32,
                        height: 32,
                        borderRadius: 1,
                        bgcolor: 'primary.lighter',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      <CategoryIcon sx={{ fontSize: 18, color: 'primary.main' }} />
                    </Box>
                    <Box>
                      <Typography variant="body2" sx={{ fontWeight: 600, mb: 0.5 }}>
                        {nodeType.catalog_type_name}
                      </Typography>
                      <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                        {nodeType.id.substring(0, 8)}...
                      </Typography>
                    </Box>
                  </Stack>
                </TableCell>

                {/* Description Column */}
                <TableCell>
                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{
                      maxWidth: 350,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {nodeType.description || '—'}
                  </Typography>
                </TableCell>

                {/* Properties Count Column */}
                <TableCell align="center">
                  <Chip
                    label={`${propertiesCount} ${propertiesCount === 1 ? 'property' : 'properties'}`}
                    size="small"
                    color="info"
                    variant="outlined"
                    sx={{
                      fontWeight: 600,
                      fontSize: '0.75rem',
                      cursor: propertiesCount > 0 ? 'pointer' : 'default',
                      '&:hover': propertiesCount > 0 ? {
                        bgcolor: 'info.lighter',
                        borderColor: 'info.main',
                      } : {},
                    }}
                    onClick={() => {
                      if (propertiesCount > 0) {
                        setSelectedNodeType(nodeType);
                        setPropertiesModalOpen(true);
                      }
                    }}
                  />
                </TableCell>

                {/* Parent Type Column */}
                <TableCell>
                  {parentType ? (
                    <Chip
                      icon={<AccountTreeIcon />}
                      label={parentType.catalog_type_name}
                      size="small"
                      variant="outlined"
                      sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                    />
                  ) : (
                    <Typography variant="body2" color="text.secondary">
                      —
                    </Typography>
                  )}
                </TableCell>

                {/* Status Column */}
                <TableCell align="center">
                  {nodeType.is_active ? (
                    <Chip
                      icon={<CheckCircleIcon />}
                      label="Active"
                      size="small"
                      color="success"
                      sx={{ fontWeight: 600 }}
                    />
                  ) : (
                    <Chip
                      icon={<CancelIcon />}
                      label="Inactive"
                      size="small"
                      color="default"
                      sx={{ fontWeight: 600 }}
                    />
                  )}
                </TableCell>

                {/* Actions Column */}
                <TableCell align="right">
                  <Stack direction="row" spacing={1} justifyContent="flex-end">
                    <Tooltip title="Edit node type">
                      <IconButton
                        size="small"
                        onClick={() => onEdit(nodeType)}
                        sx={{
                          color: 'primary.main',
                          '&:hover': {
                            bgcolor: 'primary.lighter',
                          },
                        }}
                      >
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Delete node type">
                      <IconButton
                        size="small"
                        onClick={() => onDelete(nodeType.id)}
                        sx={{
                          color: 'error.main',
                          '&:hover': {
                            bgcolor: 'error.lighter',
                          },
                        }}
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </Stack>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </TableContainer>

    <PropertiesModal
      open={propertiesModalOpen}
      onClose={() => {
        setPropertiesModalOpen(false);
        setSelectedNodeType(null);
      }}
      properties={selectedNodeType?.properties ? convertNodePropertiesToDisplayProperties(selectedNodeType.properties) : []}
      title={`Properties for ${selectedNodeType?.catalog_type_name || 'Node Type'}`}
    />
    </Box>
  );
};
