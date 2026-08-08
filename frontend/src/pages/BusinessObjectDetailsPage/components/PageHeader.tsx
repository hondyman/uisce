import { Stack, Typography, Chip, IconButton, Tooltip } from '@mui/material';
import { Delete as DeleteIcon, Edit as EditIcon, Add as AddIcon, Functions as FunctionsIcon, ImportExport as ImportExportIcon } from '@mui/icons-material';

interface PageHeaderProps {
  isNewObject: boolean;
  businessObject?: {
    displayName?: string;
    description?: string;
    status?: string;
    isActive?: boolean;
  };
  onDeleteObject: () => void;
  onEditObject: () => void;
  onAddSubtype: () => void;
  onAddCalculatedField: () => void;
  onExportImport: () => void;
}

export function PageHeader({
  isNewObject,
  businessObject,
  onDeleteObject,
  onEditObject,
  onAddSubtype,
  onAddCalculatedField,
  onExportImport,
}: PageHeaderProps) {
  return (
    <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }} spacing={2} sx={{ mb: 4 }}>
      <Stack direction="column" spacing={1}>
        <Stack direction="row" spacing={2} alignItems="center">
          <Typography variant="h4" sx={{ fontWeight: 900 }}>
            {isNewObject ? 'Create New Business Object' : businessObject?.displayName}
          </Typography>
          {!isNewObject && (
            <Chip
              label={businessObject?.status === 'active' ? 'Active' : 'Draft'}
              color={businessObject?.status === 'active' ? 'success' : 'warning'}
              variant="filled"
              size="small"
            />
          )}
        </Stack>
        <Typography variant="body2" color="text.secondary">
          {isNewObject 
            ? 'Define a new business object and configure its fields and hierarchy.' 
            : businessObject?.description || 'Core data model for business operations.'}
        </Typography>
      </Stack>

      {isNewObject ? (
        <Chip label="New" color="success" size="small" />
      ) : (
        <Stack direction="row" alignItems="center" spacing={1}>
          {businessObject?.isActive ? (
            <Chip label="Active" color="success" size="small" />
          ) : (
            <Chip label="Draft" color="default" size="small" />
          )}
          <Tooltip title="Delete Business Object">
            <IconButton 
              size="small" 
              onClick={onDeleteObject}
              sx={{ color: 'text.secondary', '&:hover': { color: 'error.main' } }}
            >
              <DeleteIcon />
            </IconButton>
          </Tooltip>

          <Tooltip title="Edit Object">
            <IconButton
              size="medium"
              onClick={onEditObject}
              sx={{ color: 'primary.main', ml: 1 }}
              disabled={!businessObject}
            >
              <EditIcon sx={{ fontSize: 32 }} />
            </IconButton>
          </Tooltip>

          <Tooltip title="Add Subtype">
            <IconButton
              size="medium"
              onClick={onAddSubtype}
              sx={{ color: 'primary.main' }}
              disabled={!businessObject}
            >
              <AddIcon sx={{ fontSize: 32 }} />
            </IconButton>
          </Tooltip>

          <Tooltip title="Add Calculated Field">
             <IconButton
                size="medium"
                onClick={onAddCalculatedField}
                sx={{ color: 'secondary.main' }}
                disabled={!businessObject}
             >
                <FunctionsIcon sx={{ fontSize: 32 }} />
             </IconButton>
          </Tooltip>
          
          <Tooltip title="Export/Import">
             <IconButton
                size="medium"
                onClick={onExportImport}
                sx={{ color: 'info.main' }}
                disabled={!businessObject}
             >
                <ImportExportIcon sx={{ fontSize: 32 }} />
             </IconButton>
          </Tooltip>
        </Stack>
      )}
    </Stack>
  );
}
