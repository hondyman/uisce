import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Typography,
  Box,
  Divider,
  Chip,
} from '@mui/material';
import {
  Storage as StorageIcon,
  Inventory as ProductIcon,
  Cable as ConnectionIcon,
  DeleteOutline as DeleteIcon,
} from '@mui/icons-material';

interface ResourceStats {
  productId: string;
  productName: string;
  connections: {
    id: string;
    name: string;
    type: string;
  }[];
}

interface InstanceResourcesDialogProps {
  open: boolean;
  onClose: () => void;
  instanceName: string;
  resources: ResourceStats[];
  onDeleteConnection: (connectionId: string) => Promise<void>;
}

export const InstanceResourcesDialog: React.FC<InstanceResourcesDialogProps> = ({
  open,
  onClose,
  instanceName,
  resources,
  onDeleteConnection,
}) => {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        Resources Linked to {instanceName}
      </DialogTitle>
      <DialogContent>
        {resources.length === 0 ? (
          <Box sx={{ py: 4, textAlign: 'center', color: 'text.secondary' }}>
            <Typography>No resources linked to this instance.</Typography>
          </Box>
        ) : (
          <List>
            {resources.map((resource, index) => (
              <React.Fragment key={resource.productId}>
                <ListItem alignItems="flex-start" sx={{ flexDirection: 'column', alignItems: 'stretch', py: 2 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                    <ListItemIcon sx={{ minWidth: 40 }}>
                      <ProductIcon color="primary" />
                    </ListItemIcon>
                    <ListItemText
                      primary={
                        <Typography variant="subtitle1" fontWeight="bold">
                          {resource.productName}
                        </Typography>
                      }
                      secondary={`${resource.connections.length} Connection${resource.connections.length !== 1 ? 's' : ''}`}
                    />
                  </Box>
                  
                  {resource.connections.length > 0 && (
                    <Box sx={{ ml: 7 }}>
                      <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
                        Connections:
                      </Typography>
                      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                        {resource.connections.map((conn) => (
                          <Chip
                            key={conn.id}
                            icon={<ConnectionIcon fontSize="small" />}
                            label={`${conn.name} (${conn.type})`}
                            size="small"
                            variant="outlined"
                            onDelete={() => onDeleteConnection(conn.id)}
                            deleteIcon={<DeleteIcon fontSize="small" color="error" />}
                          />
                        ))}
                      </Box>
                    </Box>
                  )}
                </ListItem>
                {index < resources.length - 1 && <Divider component="li" />}
              </React.Fragment>
            ))}
          </List>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};

export default InstanceResourcesDialog;
