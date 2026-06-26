import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { 
  Box, Typography, Paper, Chip, Table, TableBody, TableCell, 
  TableHead, TableRow, Button, Grid, IconButton, Dialog, DialogTitle,
  DialogContent, DialogActions, TextField, List, ListItem, ListItemText,
  ListItemSecondaryAction
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import { useBusinessTerm, useAddMappings } from '../../api/complianceApi'; // Assume hook exists or will be created

const BusinessTermDetail = () => {
    const { id } = useParams();
    const { data: term, isLoading } = useBusinessTerm(id);
    const [openMappingModal, setOpenMappingModal] = useState(false);

    if (isLoading) return <Typography>Loading...</Typography>;
    if (!term) return <Typography>Term not found</Typography>;

    return (
        <Box p={3}>
            {/* 1. Header Section */}
            <Paper sx={{ p: 3, mb: 3 }}>
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <Typography variant="h4">{term.name}</Typography>
                        <Typography variant="subtitle1" color="textSecondary">ID: {term.id}</Typography>
                    </Grid>
                    <Grid item xs={12}>
                        <Typography variant="body1">{term.description}</Typography>
                    </Grid>
                </Grid>
            </Paper>

            {/* 2. Compliance Metadata */}
            <Paper sx={{ p: 3, mb: 3 }}>
                <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="h6">Compliance Metadata</Typography>
                    <Button startIcon={<EditIcon />}>Edit Metadata</Button>
                </Box>
                <Grid container spacing={3}>
                    <Grid item xs={4}>
                        <Typography variant="subtitle2">PII Flag</Typography>
                        <Chip label={term.piiFlag ? "Yes" : "No"} color={term.piiFlag ? "error" : "default"} />
                    </Grid>
                    <Grid item xs={4}>
                        <Typography variant="subtitle2">Residency</Typography>
                        <Chip label={term.residency || "Global"} color="primary" variant="outlined" />
                    </Grid>
                    <Grid item xs={4}>
                        <Typography variant="subtitle2">Sensitivity</Typography>
                        <Chip 
                            label={term.sensitivity} 
                            color={term.sensitivity === 'HIGH' ? "error" : term.sensitivity === 'MEDIUM' ? "warning" : "success"} 
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <Typography variant="caption" display="block">
                            Last Updated By: {term.updatedBy} at {new Date(term.updatedAt).toLocaleString()}
                        </Typography>
                    </Grid>
                </Grid>
            </Paper>

            {/* 3. Linked Semantic Terms */}
            <Paper sx={{ p: 3 }}>
                <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                    <Typography variant="h6">Linked Semantic Terms</Typography>
                    <Button 
                        startIcon={<AddIcon />} 
                        variant="contained" 
                        color="primary"
                        onClick={() => setOpenMappingModal(true)}
                    >
                        Add Mapping
                    </Button>
                </Box>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Semantic Term Name</TableCell>
                            <TableCell>Type</TableCell>
                            <TableCell>Inherited PII</TableCell>
                            <TableCell>Residency</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {term.semanticTerms && term.semanticTerms.map((st) => (
                            <TableRow key={st.id}>
                                <TableCell>{st.name}</TableCell>
                                <TableCell>Field</TableCell> {/* Mock type for now */}
                                <TableCell>{term.piiFlag ? "Yes" : "No"}</TableCell>
                                <TableCell>{term.residency || "Global"}</TableCell>
                                <TableCell align="right">
                                    <IconButton size="small" color="secondary"><DeleteIcon /></IconButton>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </Paper>

            <AddMappingModal 
                open={openMappingModal} 
                onClose={() => setOpenMappingModal(false)} 
                businessTermId={id}
            />
        </Box>
    );
};

// 1.2 Add Semantic Term Mapping Modal
const AddMappingModal = ({ open, onClose, businessTermId }) => {
    const [searchTerm, setSearchTerm] = useState('');
    // Mock search results
    const results = [
        { id: 'st-client_address_line1', name: 'client_address_line1', type: 'Field' },
        { id: 'st-client_city', name: 'client_city', type: 'Field' },
        { id: 'st-other', name: 'other_term', type: 'Field' },
    ].filter(r => r.name.includes(searchTerm));

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>Add Semantic Term Mapping</DialogTitle>
            <DialogContent>
                <TextField 
                    fullWidth 
                    label="Search Semantic Terms" 
                    margin="normal" 
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                />
                <List>
                    {results.map((res) => (
                        <ListItem key={res.id} divider>
                            <ListItemText primary={res.name} secondary={res.type} />
                            <ListItemSecondaryAction>
                                <Button size="small" variant="outlined">Add Mapping</Button>
                            </ListItemSecondaryAction>
                        </ListItem>
                    ))}
                </List>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
            </DialogActions>
        </Dialog>
    );
};

export default BusinessTermDetail;
