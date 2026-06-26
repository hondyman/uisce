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
import { useNodeTypes, useDeleteNodeType, useSearchNodeTypes } from '../../api/nodeTypes';
import { devError } from '../../utils/devLogger';
import { NodeTypeTable } from './NodeTypeTable';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import { NodeTypeFormModal } from './NodeTypeFormModal';
import type { NodeType } from '../../types/nodeTypes';

export const NodeTypeSetupPage: React.FC = () => {
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

  const { data: nodeTypes, isLoading, error } = useNodeTypes(tenantId);
  const deleteNodeType = useDeleteNodeType();
  const confirm = useConfirm();
  const notification = useNotification();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingNodeType, setEditingNodeType] = useState<NodeType | null>(null);
  const [searchQuery, setSearchQuery] = useState<string>('');

  // Use server-side search when there's a query, otherwise use the full list
  const { data: searchResults, isLoading: isSearching } = useSearchNodeTypes(tenantId, searchQuery);

  // Determine which data to display: search results or full list
  // Fallback to client-side filtering if server search returns nothing
  const displayedNodeTypes = React.useMemo(() => {
    const normalizedQuery = searchQuery.trim().toLowerCase();
    const baseList = searchResults ?? nodeTypes ?? [];

    if (!normalizedQuery) {
      return nodeTypes || [];
    }

    const filterByQuery = (items: NodeType[]) =>
      items.filter((nt) =>
        nt.catalog_type_name.toLowerCase().includes(normalizedQuery) ||
        (nt.description || '').toLowerCase().includes(normalizedQuery)
      );

    const filtered = filterByQuery(baseList);
    if (filtered.length > 0) {
      return filtered;
    }

    // Fallback: filter the full list if the base list doesn't contain matches
    return filterByQuery(nodeTypes || []);
  }, [searchQuery, searchResults, nodeTypes]);

  const handleCreate = () => {
    setEditingNodeType(null);
    setIsModalOpen(true);
  };

  const handleEdit = (nodeType: NodeType) => {
    setEditingNodeType(nodeType);
    setIsModalOpen(true);
  };

  const handleDelete = async (id: string) => {
    if (!(await confirm({ title: 'Delete node type', description: 'Are you sure you want to delete this node type? This action cannot be undone.' }))) return;

    try {
      await deleteNodeType.mutateAsync({ id, tenantId });
    } catch (error) {
      notification.error(`Failed to delete node type: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleModalClose = () => {
    setIsModalOpen(false);
    setEditingNodeType(null);
  };

  // Keep editingNodeType in parent up-to-date when modal operates in controlled mode
  const handleEditChange = (updated: Partial<NodeType>) => {
    setEditingNodeType((prev) => prev ? ({ ...prev, ...updated }) : prev);
  };

  // Listen for external requests to open an editor (from global search)
  React.useEffect(() => {
    const handler = (ev: Event) => {
      const custom = ev as CustomEvent;
      const detail = custom.detail as { kind: string; item: NodeType } | undefined;
      if (!detail) return;
      if (detail.kind === 'node') {
        setEditingNodeType(detail.item);
        setIsModalOpen(true);
        // scroll to top of page to ensure modal is visible if needed
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
            Please select a tenant using the tenant picker before managing node types.
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
            Node Type Management
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Define entity types and their hierarchies for your business glossary
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
          Create Node Type
        </Button>
      </Stack>

      {/* Search */}
      <Box mb={3}>
        <TextField
          fullWidth
          placeholder="Search node types by name or description..."
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
              Loading node types...
            </Typography>
          </Stack>
        </Box>
      )}

      {/* Error State */}
      {error && (
        <Alert severity="error" variant="filled" sx={{ mb: 3, borderRadius: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            Error Loading Node Types
          </Typography>
          <Typography variant="body2">
            {error instanceof Error ? error.message : 'An unknown error occurred'}
          </Typography>
        </Alert>
      )}

      {/* Table */}
      {!isLoading && !error && displayedNodeTypes && (
        <NodeTypeTable
          nodeTypes={displayedNodeTypes}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      )}

      {/* Modal */}
      <NodeTypeFormModal
        open={isModalOpen}
        onClose={handleModalClose}
        nodeType={editingNodeType}
        tenantId={tenantId}
        allNodeTypes={nodeTypes || []}
        onChange={handleEditChange}
      />
    </Container>
  );
};
