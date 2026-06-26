import React, { useState, lazy as _lazy, Suspense as _Suspense } from 'react';
import { 
  Box, 
  Button, 
  Typography, 
  Alert, 
  CircularProgress, 
  Container,
  Paper,
  Stack,
  TextField,
  InputAdornment
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import SearchIcon from '@mui/icons-material/Search';
import { useEdgeTypes, useDeleteEdgeType, useSearchEdgeTypes } from '../../api/edgeTypes';
import { useNodeTypes } from '../../api/nodeTypes';
import { EdgeTypeTable } from './EdgeTypeTable';
import { EdgeTypeFormModal } from './EdgeTypeFormModal';
import type { EdgeType } from '../../types/edgeTypes';
import { devError } from '../../utils/devLogger';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';

export const EdgeTypeSetupPage: React.FC = () => {
  const [tenantId] = useState<string>(() => {
    try {
      const stored = localStorage.getItem('selected_tenant');
      if (stored) {
        const parsed = JSON.parse(stored);
        return parsed.id || '';
      }
    } catch (e) {
      devError('Failed to parse selected_tenant:', e);
    }
    return '';
  });

  const { data: edgeTypes, isLoading, error } = useEdgeTypes(tenantId);
  const { data: nodeTypes } = useNodeTypes(tenantId);
  const deleteEdgeType = useDeleteEdgeType();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingEdgeType, setEditingEdgeType] = useState<EdgeType | null>(null);
  const [searchQuery, setSearchQuery] = useState<string>('');

  // Use server-side search when there's a query, otherwise use the full list
  const { data: searchResults, isLoading: isSearching } = useSearchEdgeTypes(tenantId, searchQuery);
  const confirm = useConfirm();
  const notification = useNotification();

  // Determine which data to display: search results or full list
  // Fallback to client-side filtering if server search returns nothing
  const nodeTypeLookup = React.useMemo(() => {
    const lookup = new Map<string, string>();
    (nodeTypes ?? []).forEach((nodeType) => {
      lookup.set(nodeType.id, (nodeType.catalog_type_name || '').toLowerCase());
    });
    return lookup;
  }, [nodeTypes]);

  const displayedEdgeTypes = React.useMemo(() => {
    const normalizedQuery = searchQuery.trim().toLowerCase();
    const baseList = searchResults ?? edgeTypes ?? [];

    if (!normalizedQuery) {
      return edgeTypes || [];
    }

    const filterByQuery = (items: EdgeType[]) =>
      items.filter((et) => {
        const predicateMatch = (et.edge_type_name || '').toLowerCase().includes(normalizedQuery);
        const descriptionMatch = (et.description || '').toLowerCase().includes(normalizedQuery);

        const subjectName = (et.subject_node_type_id && nodeTypeLookup.get(et.subject_node_type_id)) || '';
        const objectName = (et.object_node_type_id && nodeTypeLookup.get(et.object_node_type_id)) || '';
        const subjectId = (et.subject_node_type_id || '').toLowerCase();
        const objectId = (et.object_node_type_id || '').toLowerCase();

        const directionVariants = [
          `${subjectName} ${objectName}`,
          `${subjectName}->${objectName}`,
          `${subjectName} -> ${objectName}`,
          `${subjectId} ${objectId}`,
          `${subjectId}->${objectId}`,
          `${subjectId} -> ${objectId}`,
        ].some((variant) => variant.includes(normalizedQuery));

        return predicateMatch || descriptionMatch || directionVariants;
      });

    const filtered = filterByQuery(baseList);
    if (filtered.length > 0) {
      return filtered;
    }

    return filterByQuery(edgeTypes || []);
  }, [searchQuery, searchResults, edgeTypes, nodeTypeLookup]);

  const handleCreate = () => {
    setEditingEdgeType(null);
    setIsModalOpen(true);
  };

  const handleEdit = (edgeType: EdgeType) => {
    setEditingEdgeType(edgeType);
    setIsModalOpen(true);
  };

  const handleDelete = async (id: string) => {
    if (!(await confirm({ title: 'Delete edge type', description: 'Are you sure you want to delete this edge type? This action cannot be undone.' }))) {
      return;
    }

    try {
      await deleteEdgeType.mutateAsync({ id, tenantId });
      notification.success('Edge type deleted successfully');
    } catch (error) {
      notification.error(`Failed to delete edge type: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleModalClose = () => {
    setIsModalOpen(false);
    setEditingEdgeType(null);
  };

  // Listen for external requests to open an editor (from global search)
  React.useEffect(() => {
    const handler = (ev: Event) => {
      const custom = ev as CustomEvent;
      const detail = custom.detail as { kind: string; item: EdgeType } | undefined;
      if (!detail) return;
      if (detail.kind === 'edge') {
        setEditingEdgeType(detail.item);
        setIsModalOpen(true);
        window.scrollTo({ top: 0, behavior: 'smooth' });
      }
    };

    window.addEventListener('catalog-open-edit', handler as EventListener);
    return () => window.removeEventListener('catalog-open-edit', handler as EventListener);
  }, []);

  if (!tenantId) {
    return (
      <Container maxWidth="lg" sx={{ py: 8 }}>
        <Paper elevation={0} sx={{ p: 6, textAlign: 'center', bgcolor: 'warning.lighter', border: 1, borderColor: 'warning.light' }}>
          <Typography variant="h5" gutterBottom color="warning.dark" sx={{ fontWeight: 600 }}>
            Tenant Required
          </Typography>
          <Typography variant="body1" color="warning.dark">
            Please select a tenant using the tenant picker before managing edge types.
          </Typography>
        </Paper>
      </Container>
    );
  }

  return (
    <Container maxWidth="xl" sx={{ py: 4 }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="flex-start" mb={4}>
        <Box>
          <Typography variant="h5" component="h2" gutterBottom sx={{ fontWeight: 600 }}>
            Edge Type Management
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Define relationships between node types to structure your business glossary
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleCreate}
          size="large"
          sx={{ 
            textTransform: 'none',
            fontWeight: 600,
            px: 3,
            boxShadow: 2,
            '&:hover': {
              boxShadow: 4,
            }
          }}
        >
          Create Edge Type
        </Button>
      </Stack>

      {/* Search */}
      <Box mb={3}>
        <TextField
          fullWidth
          placeholder="Search edge types by name, description, or direction..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
          sx={{
            '& .MuiOutlinedInput-root': {
              bgcolor: 'background.paper',
            }
          }}
        />
      </Box>

      {/* Loading State */}
      {(isLoading || isSearching) && (
        <Box display="flex" justifyContent="center" alignItems="center" py={12}>
          <Stack spacing={2} alignItems="center">
            <CircularProgress size={48} />
            <Typography variant="body2" color="text.secondary">
              Loading edge types...
            </Typography>
          </Stack>
        </Box>
      )}

      {/* Error State */}
      {error && (
        <Alert severity="error" variant="filled" sx={{ mb: 3, borderRadius: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            Error Loading Edge Types
          </Typography>
          <Typography variant="body2">
            {error instanceof Error ? error.message : 'An unknown error occurred'}
          </Typography>
        </Alert>
      )}

      {/* Table */}
      {!isLoading && !error && displayedEdgeTypes && (
        <EdgeTypeTable
          edgeTypes={displayedEdgeTypes}
          nodeTypes={nodeTypes || []}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      )}

      {/* Modal */}
      <EdgeTypeFormModal
        open={isModalOpen}
        onClose={handleModalClose}
        edgeType={editingEdgeType}
        tenantId={tenantId}
      />
    </Container>
  );
};
