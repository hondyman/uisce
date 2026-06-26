import { useState, Fragment } from 'react';
import {
  Box,
  Typography,
  Chip,
  Dialog,
  DialogContent,
  DialogActions,
  Button,
  List,
  ListItem,
  ListItemText,
  Divider,
  Alert,
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import type { UpgradeArtifacts } from '../../../types/upgrade-generated';

interface SchemaVersionDisplayProps {
  artifact: UpgradeArtifacts;
  backendVersion?: string;
}

export const SchemaVersionDisplay: React.FC<SchemaVersionDisplayProps> = ({
  artifact,
  backendVersion,
}) => {
  const [changelogOpen, setChangelogOpen] = useState(false);

  const hasVersionMismatch = backendVersion && backendVersion !== artifact.schema_version;

  return (
    <Box sx={{ mb: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
        <Typography variant="body2" color="text.secondary">
          Schema Version:
        </Typography>
        <Chip
          label={artifact.schema_version}
          size="small"
          color={hasVersionMismatch ? 'warning' : 'primary'}
          variant="outlined"
        />
        {hasVersionMismatch && (
          <Alert severity="warning" sx={{ py: 0, px: 1 }}>
            Backend version: {backendVersion}
          </Alert>
        )}
      </Box>

      {artifact.changelog && artifact.changelog.length > 0 && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography variant="body2" color="text.secondary">
            Latest Change:
          </Typography>
          <Typography variant="body2" sx={{ fontStyle: 'italic' }}>
            {artifact.changelog[0].description}
          </Typography>
          <Button
            size="small"
            variant="text"
            onClick={() => setChangelogOpen(true)}
          >
            View History
          </Button>
        </Box>
      )}

      <Dialog
        open={changelogOpen}
        onClose={() => setChangelogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <ModalHeader title="Schema Changelog" onClose={() => setChangelogOpen(false)} />
        <DialogContent>
          <List>
            {artifact.changelog?.map((entry, index) => (
              <Fragment key={entry.version}>
                <ListItem>
                  <ListItemText
                    primary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Chip label={entry.version} size="small" />
                        <Typography variant="caption" color="text.secondary">
                          {new Date(entry.date).toLocaleDateString()}
                        </Typography>
                      </Box>
                    }
                    secondary={entry.description}
                  />
                </ListItem>
                {index < (artifact.changelog?.length || 0) - 1 && <Divider />}
              </Fragment>
            ))}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setChangelogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
