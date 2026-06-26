import React from 'react';
import {
  Drawer,
  Box,
  Typography,
  Divider,
  Chip,
  Button,
  Stack,
  IconButton,
} from '@mui/material';
import { Close as CloseIcon, Code as CodeIcon } from '@mui/icons-material';
import { Node } from 'reactflow';

interface NodeDetailDrawerProps {
  node: Node | null;
  open: boolean;
  onClose: () => void;
  boId: string;
}

export const NodeDetailDrawer: React.FC<NodeDetailDrawerProps> = ({
  node,
  open,
  onClose,
  boId,
}) => {
  if (!node) return null;

  const renderContent = () => {
    switch (node.type) {
      case 'bo':
      case 'related_bo':
        return (
          <>
            <Typography variant="h5" gutterBottom>
              {node.data.name || node.data.label}
            </Typography>
            <Chip
              label={node.type === 'related_bo' ? 'Related BO' : 'Business Object'}
              size="small"
              color="primary"
              sx={{ mb: 2 }}
            />

            {node.data.description && (
              <>
                <Typography variant="subtitle2" sx={{ mt: 2 }}>
                  Description
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {node.data.description}
                </Typography>
              </>
            )}

            {node.data.termCount !== undefined && (
              <>
                <Typography variant="subtitle2" sx={{ mt: 2 }}>
                  Statistics
                </Typography>
                <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                  <Chip label={`${node.data.termCount} Terms`} size="small" variant="outlined" />
                  {node.data.relatedBOCount > 0 && (
                    <Chip
                      label={`${node.data.relatedBOCount} Related BOs`}
                      size="small"
                      variant="outlined"
                    />
                  )}
                </Stack>
              </>
            )}

            {node.data.relationshipType && (
              <>
                <Typography variant="subtitle2" sx={{ mt: 2 }}>
                  Relationship
                </Typography>
                <Chip label={node.data.relationshipType} size="small" color="secondary" />
              </>
            )}
          </>
        );

      case 'term':
        return (
          <>
            <Typography variant="h6" gutterBottom>
              {node.data.termName || node.data.label}
            </Typography>
            <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
              <Chip label={node.data.termType || 'dimension'} size="small" color="primary" />
              {node.data.isKey && <Chip label="Primary Key" size="small" color="success" />}
              {node.data.isForeignKey && <Chip label="Foreign Key" size="small" color="warning" />}
            </Stack>

            <Divider sx={{ my: 2 }} />

            <Typography variant="subtitle2">Data Type</Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              {node.data.dataType || 'unknown'}
            </Typography>

            {node.data.aggregation && (
              <>
                <Typography variant="subtitle2">Aggregation</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  {node.data.aggregation}
                </Typography>
              </>
            )}

            {node.data.physicalMapping && (
              <>
                <Typography variant="subtitle2">Physical Mapping</Typography>
                <Box
                  sx={{
                    p: 1.5,
                    bgcolor: 'grey.100',
                    borderRadius: 1,
                    fontFamily: 'monospace',
                    fontSize: '0.875rem',
                    mb: 2,
                  }}
                >
                  {node.data.physicalMapping.schema && (
                    <div>Schema: {node.data.physicalMapping.schema}</div>
                  )}
                  <div>Table: {node.data.physicalMapping.table}</div>
                  <div>Column: {node.data.physicalMapping.column}</div>
                </Box>
              </>
            )}
          </>
        );

      case 'calculation':
        return (
          <>
            <Typography variant="h6" gutterBottom>
              {node.data.name || node.data.label}
            </Typography>
            <Chip label="Calculation" size="small" color="secondary" sx={{ mb: 2 }} />

            <Divider sx={{ my: 2 }} />

            <Typography variant="subtitle2">Formula</Typography>
            <Box
              sx={{
                p: 1.5,
                bgcolor: 'grey.100',
                borderRadius: 1,
                fontFamily: 'monospace',
                fontSize: '0.875rem',
                mb: 2,
                overflowX: 'auto',
              }}
            >
              {node.data.formula}
            </Box>

            {node.data.returnType && (
              <>
                <Typography variant="subtitle2">Return Type</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  {node.data.returnType}
                </Typography>
              </>
            )}

            <Button
              variant="outlined"
              startIcon={<CodeIcon />}
              fullWidth
              sx={{ mt: 2 }}
              onClick={() => {
                // TODO: Navigate to calculation detail or show SQL
                console.log('View SQL for calculation:', node.data.calcId);
              }}
            >
              View Resolved SQL
            </Button>
          </>
        );

      case 'table':
        return (
          <>
            <Typography variant="h6" gutterBottom>
              {node.data.tableName || node.data.label}
            </Typography>
            <Chip label="Physical Table" size="small" sx={{ mb: 2 }} />

            <Divider sx={{ my: 2 }} />

            {node.data.schema && (
              <>
                <Typography variant="subtitle2">Schema</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  {node.data.schema}
                </Typography>
              </>
            )}

            {node.data.rowCount && (
              <>
                <Typography variant="subtitle2">Row Count</Typography>
                <Typography variant="body2" color="text.secondary">
                  {node.data.rowCount.toLocaleString()}
                </Typography>
              </>
            )}
          </>
        );

      case 'column':
        return (
          <>
            <Typography variant="h6" gutterBottom>
              {node.data.columnName || node.data.label}
            </Typography>
            <Chip label="Physical Column" size="small" sx={{ mb: 2 }} />

            <Divider sx={{ my: 2 }} />

            <Typography variant="subtitle2">Table</Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              {node.data.schema && `${node.data.schema}.`}
              {node.data.tableName}
            </Typography>

            {node.data.dataType && (
              <>
                <Typography variant="subtitle2">Data Type</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  {node.data.dataType}
                </Typography>
              </>
            )}

            <Stack direction="row" spacing={1} sx={{ mt: 2 }}>
              {node.data.isPrimaryKey && <Chip label="Primary Key" size="small" color="success" />}
              {node.data.isForeignKey && <Chip label="Foreign Key" size="small" color="warning" />}
              {node.data.nullable && <Chip label="Nullable" size="small" variant="outlined" />}
            </Stack>
          </>
        );

      default:
        return (
          <Typography variant="body2" color="text.secondary">
            Select a node to view details
          </Typography>
        );
    }
  };

  return (
    <Drawer anchor="right" open={open} onClose={onClose}>
      <Box sx={{ width: 400, p: 3, position: 'relative' }}>
        <IconButton
          onClick={onClose}
          sx={{ position: 'absolute', right: 8, top: 8 }}
          size="small"
        >
          <CloseIcon />
        </IconButton>

        {renderContent()}
      </Box>
    </Drawer>
  );
};
