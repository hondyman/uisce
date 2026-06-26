import type React from 'react';
import {
    Box,
    Button,
    Alert,
    Typography,
    CircularProgress,
} from '@mui/material';

interface BundleActionsProps {
    onCancel: () => void;
    onSave: () => void;
    loading: boolean;
    publishErrors: string[];
}

export const BundleActions: React.FC<BundleActionsProps> = ({
    onCancel,
    onSave,
    loading,
    publishErrors,
}) => {
    return (
        <Box sx={{ mt: 4, display: 'flex', justifyContent: 'flex-end', gap: 2 }}>
            {publishErrors.length > 0 && (
                <Alert severity="error" sx={{ flex: 1 }}>
                    <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
                        Publish Validation Issues:
                    </Typography>
                                            <ul>
                        {publishErrors.map((error, idx) => (
                            <li key={idx}>{error}</li>
                        ))}
                    </ul>
                </Alert>
            )}
            <Button onClick={onCancel} sx={{ mr: 2 }}>
                Cancel
            </Button>
            <Button variant="contained" onClick={onSave} disabled={loading}>
                {loading ? <CircularProgress size={24} /> : 'Save Bundle'}
            </Button>
        </Box>
    );
};