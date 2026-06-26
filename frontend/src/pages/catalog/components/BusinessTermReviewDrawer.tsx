import React from 'react';
import {
    Drawer,
    Typography,
    Box,
    Button,
    Divider,
    Stack,
    Chip,
    TextField,
    Alert,
    Paper,
} from '@mui/material';
import { AIBusinessTermDraft } from '../../../api/catalogApi';

interface BusinessTermReviewDrawerProps {
    open: boolean;
    suggestion: AIBusinessTermDraft | null;
    onClose: () => void;
    onApprove: (id: string) => void;
    onReject: (id: string, reason: string) => void;
}

export const BusinessTermReviewDrawer: React.FC<BusinessTermReviewDrawerProps> = ({
    open,
    suggestion,
    onClose,
    onApprove,
    onReject,
}) => {
    const [rejectReason, setRejectReason] = React.useState('');
    const [showRejectInput, setShowRejectInput] = React.useState(false);

    React.useEffect(() => {
        if (!open) {
            setShowRejectInput(false);
            setRejectReason('');
        }
    }, [open]);

    if (!suggestion) return null;

    const handleRejectClick = () => {
        if (showRejectInput) {
            if (rejectReason.trim()) {
                onReject(suggestion.id, rejectReason);
            }
        } else {
            setShowRejectInput(true);
        }
    };

    return (
        <Drawer
            anchor="right"
            open={open}
            onClose={onClose}
            PaperProps={{ sx: { width: 500, padding: 3 } }}
        >
            <Box mb={2}>
                <Typography variant="h5" gutterBottom>
                    Review Suggestion
                </Typography>
                <Chip
                    label={suggestion.status}
                    color={
                        suggestion.status === 'APPROVED'
                            ? 'success'
                            : suggestion.status === 'REJECTED'
                            ? 'error'
                            : 'warning'
                    }
                    size="small"
                />
            </Box>

            <Divider sx={{ mb: 3 }} />

            <Stack spacing={3}>
                <Box>
                    <Typography variant="subtitle2" color="text.secondary">
                        Proposed Term
                    </Typography>
                    <Typography variant="h6">{suggestion.name}</Typography>
                </Box>

                <Box>
                    <Typography variant="subtitle2" color="text.secondary">
                        Definition
                    </Typography>
                    <Typography variant="body1">{suggestion.definition}</Typography>
                </Box>

                <Box>
                    <Typography variant="subtitle2" color="text.secondary">
                        Hierarchy
                    </Typography>
                    <Paper variant="outlined" sx={{ p: 1.5, mt: 1, bgcolor: '#f5f5f5' }}>
                        <Stack spacing={0.5}>
                            <Typography variant="body2" color="text.secondary">L1: {suggestion.hierarchy.level1}</Typography>
                            <Typography variant="body2" color="text.secondary">L2: {suggestion.hierarchy.level2}</Typography>
                            <Typography variant="body2" fontWeight="medium">L3: {suggestion.hierarchy.level3}</Typography>
                        </Stack>
                    </Paper>
                </Box>

                <Box>
                    <Typography variant="subtitle2" color="text.secondary">
                        Compliance & Governance
                    </Typography>
                    <Stack direction="row" spacing={1} mt={1}>
                        <Chip 
                            label={suggestion.piiFlag ? "PII: YES" : "PII: NO"} 
                            color={suggestion.piiFlag ? "error" : "default"} 
                            size="small" 
                        />
                        <Chip 
                            label={`Sensitivity: ${suggestion.sensitivity}`} 
                            color={suggestion.sensitivity === 'HIGH' ? "warning" : "default"} 
                            size="small" 
                        />
                        <Chip label={`Residency: ${suggestion.residency}`} size="small" />
                    </Stack>
                </Box>

                 <Box>
                    <Typography variant="subtitle2" color="text.secondary">
                        Tags
                    </Typography>
                    <Stack direction="row" spacing={1} flexWrap="wrap" mt={1}>
                        {suggestion.tags.map((tag) => (
                            <Chip key={tag} label={`#${tag}`} size="small" sx={{ bgcolor: '#e3f2fd' }} />
                        ))}
                    </Stack>
                </Box>

                <Box>
                    <Typography variant="subtitle2" color="text.secondary">
                        Source Semantic Terms
                    </Typography>
                    <Stack direction="row" spacing={1} flexWrap="wrap" mt={1}>
                        {suggestion.sourceSemanticTerms.map((term) => (
                            <Chip key={term} label={term} size="small" variant="outlined" />
                        ))}
                         {suggestion.sourceSemanticTerms.length === 0 && <Typography variant="caption" color="text.secondary">None</Typography>}
                    </Stack>
                </Box>

                <Box>
                    <Typography variant="subtitle2" color="text.secondary">
                        Source Columns
                    </Typography>
                    <Stack direction="row" spacing={1} flexWrap="wrap" mt={1}>
                        {suggestion.sourceColumns.map((col) => (
                            <Chip key={col} label={col} size="small" variant="outlined" />
                        ))}
                    </Stack>
                </Box>
            </Stack>

            <Box mt="auto" pt={3}>
                {showRejectInput && (
                    <TextField
                        fullWidth
                        label="Rejection Reason"
                        value={rejectReason}
                        onChange={(e) => setRejectReason(e.target.value)}
                        multiline
                        rows={2}
                        sx={{ mb: 2 }}
                        autoFocus
                    />
                )}
                {suggestion.status === 'DRAFT_AI' && (
                    <Stack direction="row" spacing={2} justifyContent="flex-end">
                        <Button
                            variant="outlined"
                            color="error"
                            onClick={handleRejectClick}
                            disabled={showRejectInput && !rejectReason.trim()}
                        >
                            {showRejectInput ? 'Confirm Reject' : 'Reject'}
                        </Button>
                        <Button
                            variant="contained"
                            color="success"
                            onClick={() => onApprove(suggestion.id)}
                            disabled={showRejectInput}
                        >
                            Approve
                        </Button>
                    </Stack>
                )}
            </Box>
        </Drawer>
    );
};
