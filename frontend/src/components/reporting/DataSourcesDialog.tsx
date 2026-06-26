import type { FC } from 'react';
import { Dialog, DialogTitle, DialogContent, DialogActions, Button, Box, TableContainer, Paper, Table, TableHead, TableRow, TableCell, TableBody, IconButton } from '@mui/material';
import { Plus, Settings, Trash2 } from 'lucide-react';
import { dataSources } from './reportingUtils';

type DataSource = {
  id: string;
  name: string;
  type: string;
  connectionString?: string;
  url?: string;
};

type Props = {
  open: boolean;
  onClose: () => void;
};

const DataSourcesDialog: FC<Props> = ({ open, onClose }) => {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Data Sources</DialogTitle>
      <DialogContent>
        <Box sx={{ mb: 2 }}>
          <Button variant="contained" startIcon={<Plus />} sx={{ mb: 2 }}>Add Data Source</Button>
        </Box>
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Connection</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {dataSources.map((ds: DataSource) => (
                <TableRow key={ds.id}>
                  <TableCell>{ds.name}</TableCell>
                  <TableCell>{ds.type}</TableCell>
                  <TableCell sx={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis' }}>{ds.connectionString || ds.url}</TableCell>
                  <TableCell>
                    <IconButton size="small"><Settings size={16} /></IconButton>
                    <IconButton size="small"><Trash2 size={16} /></IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};

export default DataSourcesDialog;
