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
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import PropertiesModal from '../../components/properties/PropertiesModal';
import type { EdgeType, EdgeProperty } from '../../types/edgeTypes';
import type { NodeType } from '../../types/nodeTypes';

interface EdgeTypeTableProps {
  edgeTypes: EdgeType[];
  nodeTypes: NodeType[];
  onEdit: (edgeType: EdgeType) => void;
  onDelete: (id: string) => void;
}

// Convert EdgeProperty to DisplayProperty for PropertiesModal
const convertEdgePropertiesToDisplayProperties = (properties: EdgeProperty[]) => {
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

export const EdgeTypeTable: React.FC<EdgeTypeTableProps> = ({
  edgeTypes,
  nodeTypes,
  onEdit,
  onDelete,
}) => {
  const [propertiesModalOpen, setPropertiesModalOpen] = useState(false);
  const [selectedEdgeType, setSelectedEdgeType] = useState<EdgeType | null>(null);

  const getNodeTypeName = (id: string | null | undefined): string => {
    if (!id) return '—';
    const nodeType = nodeTypes.find((nt) => nt.id === id);
    return nodeType?.catalog_type_name || id;
  };

  if (edgeTypes.length === 0) {
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
          <ArrowForwardIcon sx={{ fontSize: 32, color: 'primary.main' }} />
        </Box>
        <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
          No Edge Types Yet
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Create your first edge type to define relationships between node types in your glossary.
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
              Relationship
            </TableCell>
            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Description
            </TableCell>
            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Node Types
            </TableCell>
            <TableCell align="center" sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Properties
            </TableCell>
            <TableCell sx={{ fontWeight: 700, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5 }}>
              Direction
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
          {edgeTypes.map((edgeType) => {
            const propertiesCount = edgeType.properties?.length || 0;

            return (
              <TableRow
                key={edgeType.id}
                sx={{
                  '&:hover': {
                    bgcolor: 'grey.100',
                  },
                  transition: 'background-color 0.2s',
                }}
              >
                {/* Name Column */}
                <TableCell>
                  <Box>
                    <Typography variant="body2" sx={{ fontWeight: 600, mb: 0.5 }}>
                      {edgeType.edge_type_name}
                    </Typography>
                    <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                      {edgeType.id.substring(0, 8)}...
                    </Typography>
                  </Box>
                </TableCell>

                {/* Description Column */}
                <TableCell>
                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{
                      maxWidth: 300,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {edgeType.description || '—'}
                  </Typography>
                </TableCell>

                {/* Node Types Column */}
                <TableCell>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Chip
                      label={edgeType.subject_node_type_name || getNodeTypeName(edgeType.subject_node_type_id)}
                      size="small"
                      color="primary"
                      variant="outlined"
                      sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                    />
                    <ArrowForwardIcon sx={{ fontSize: 16, color: 'text.secondary' }} />
                    <Chip
                      label={edgeType.object_node_type_name || getNodeTypeName(edgeType.object_node_type_id)}
                      size="small"
                      color="secondary"
                      variant="outlined"
                      sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                    />
                  </Stack>
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
                        setSelectedEdgeType(edgeType);
                        setPropertiesModalOpen(true);
                      }
                    }}
                  />
                </TableCell>

                {/* Direction Column */}
                <TableCell>
                  <Chip
                    label={edgeType.is_directed ? 'Directed' : 'Undirected'}
                    size="small"
                    variant="outlined"
                    sx={{ fontWeight: 600, fontSize: '0.75rem' }}
                  />
                </TableCell>

              {/* Status Column */}
              <TableCell align="center">
                <Stack direction="row" spacing={0.5} alignItems="center" justifyContent="center">
                  <Chip
                    icon={edgeType.is_active ? <CheckCircleIcon /> : <CancelIcon />}
                    label={edgeType.is_active ? "Active" : "Inactive"}
                    size="small"
                    color={edgeType.is_active ? "success" : "default"}
                    sx={{ fontWeight: 600 }}
                  />
                  {edgeType.type && (
                    <Chip
                      label={edgeType.type === 'core' ? 'Core' : 'Custom'}
                      size="small"
                      color={edgeType.type === 'core' ? 'primary' : 'warning'}
                      variant={edgeType.type === 'core' ? 'filled' : 'outlined'}
                      sx={{ fontWeight: 600 }}
                    />
                  )}
                </Stack>
              </TableCell>

              {/* Actions Column */}
              <TableCell align="right">
                <Stack direction="row" spacing={1} justifyContent="flex-end">
                  <Tooltip title="Edit edge type">
                    <IconButton
                      size="small"
                      onClick={() => onEdit(edgeType)}
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
                  <Tooltip title="Delete edge type">
                    <IconButton
                      size="small"
                      onClick={() => onDelete(edgeType.id)}
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
        setSelectedEdgeType(null);
      }}
      properties={selectedEdgeType?.properties ? convertEdgePropertiesToDisplayProperties(selectedEdgeType.properties) : []}
      title={`Properties for ${selectedEdgeType?.edge_type_name || 'Edge Type'}`}
    />
    </Box>
  );
};
