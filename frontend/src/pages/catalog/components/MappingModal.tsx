import React, { useState } from 'react';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Button,
    TextField,
    List,
    ListItem,
    ListItemText,
    Checkbox,
    ListItemIcon,
    Typography,
    Box,
    CircularProgress,
} from '@mui/material';
// We'll need a way to search semantic terms. 
// Assuming there might be an API for this, or we mock it for now.
// For now, let's assume we can pass a search function or just use a mock list.

// Mocking for MVP until a proper search API is available/confirmed
const MOCK_SEMANTIC_TERMS = [
    { id: 'sem_1', name: 'customer_id' },
    { id: 'sem_2', name: 'total_revenue' },
    { id: 'sem_3', name: 'email_address' },
    { id: 'sem_4', name: 'shipping_address' },
    { id: 'sem_5', name: 'order_count' },
];

interface MappingModalProps {
    open: boolean;
    onClose: () => void;
    onAdd: (selectedIds: string[]) => Promise<void>;
}

export const MappingModal: React.FC<MappingModalProps> = ({ open, onClose, onAdd }) => {
    const [search, setSearch] = useState('');
    const [selected, setSelected] = useState<string[]>([]);
    const [submitting, setSubmitting] = useState(false);

    // In a real implementation, we'd debounce a search API call here
    // For now, client-side filter mock
    const filteredTerms = MOCK_SEMANTIC_TERMS.filter(t => 
        t.name.toLowerCase().includes(search.toLowerCase())
    );

    const handleToggle = (id: string) => {
        const currentIndex = selected.indexOf(id);
        const newChecked = [...selected];

        if (currentIndex === -1) {
            newChecked.push(id);
        } else {
            newChecked.splice(currentIndex, 1);
        }

        setSelected(newChecked);
    };

    const handleSubmit = async () => {
        if (selected.length === 0) return;
        setSubmitting(true);
        try {
            await onAdd(selected);
            onClose();
            setSelected([]);
            setSearch('');
        } catch (error) {
            console.error(error);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>Add Semantic Mappings</DialogTitle>
            <DialogContent dividers>
                <Box mb={2}>
                    <TextField
                        autoFocus
                        margin="dense"
                        label="Search Semantic Terms"
                        type="text"
                        fullWidth
                        variant="outlined"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        placeholder="e.g. customer_id..."
                    />
                </Box>
                
                <Typography variant="caption" color="text.secondary" display="block" mb={1}>
                    Select terms to link:
                </Typography>
                
                <List sx={{ height: 300, overflow: 'auto', border: '1px solid #eee', borderRadius: 1 }}>
                    {filteredTerms.map((term) => (
                        <ListItem key={term.id} button onClick={() => handleToggle(term.id)}>
                            <ListItemIcon>
                                <Checkbox
                                    edge="start"
                                    checked={selected.indexOf(term.id) !== -1}
                                    tabIndex={-1}
                                    disableRipple
                                />
                            </ListItemIcon>
                            <ListItemText primary={term.name} />
                        </ListItem>
                    ))}
                    {filteredTerms.length === 0 && (
                        <ListItem>
                            <ListItemText primary="No matching terms found" sx={{ color: 'gray', textAlign: 'center' }} />
                        </ListItem>
                    )}
                </List>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose} disabled={submitting}>Cancel</Button>
                <Button 
                    onClick={handleSubmit} 
                    variant="contained" 
                    disabled={selected.length === 0 || submitting}
                >
                    {submitting ? 'Adding...' : 'Add Selected'}
                </Button>
            </DialogActions>
        </Dialog>
    );
};
