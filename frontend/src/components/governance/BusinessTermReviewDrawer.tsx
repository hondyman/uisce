import React from 'react';
import { 
  Box, Typography, Button, Divider, Chip, Grid 
} from '@mui/material';

const BusinessTermReviewDrawer = ({ draft, onClose }) => {
    // Mock Detail Data (would fetch in real app)
    const detail = {
        ...draft,
        definition: "The mailing address associated with a client, used for communication, identity verification, and regulatory reporting.",
        sourceSemanticTerms: ["st-client_address_line1", "st-client_city"],
        sourceColumns: ["client.address_line1", "client.city"],
        tags: ["contact", "address", "client", "regulatory", "pii"]
    };

    return (
        <Box p={3} role="presentation">
            <Typography variant="h5" gutterBottom>Review Suggestion</Typography>
            
            <Box mb={3}>
                <Typography variant="h6">{detail.name}</Typography>
                <Typography variant="body1" color="textSecondary">{detail.definition}</Typography>
            </Box>

            <Divider sx={{ mb: 2 }} />

            <Typography variant="subtitle1" gutterBottom>Compliance</Typography>
            <Grid container spacing={2} mb={2}>
                <Grid item xs={4}>
                    <Typography variant="caption">PII</Typography>
                    <Box><Chip label={detail.piiFlag ? "Yes" : "No"} color="error" size="small" /></Box>
                </Grid>
                <Grid item xs={4}>
                    <Typography variant="caption">Sensitivity</Typography>
                    <Box><Chip label={detail.sensitivity} color="warning" size="small" /></Box>
                </Grid>
                <Grid item xs={4}>
                    <Typography variant="caption">Residency</Typography>
                    <Box><Chip label={detail.residency} variant="outlined" size="small" /></Box>
                </Grid>
            </Grid>

            <Divider sx={{ mb: 2 }} />

            <Typography variant="subtitle1" gutterBottom>Hierarchy</Typography>
            <Box mb={2}>
                <Typography variant="body2">• L1: {detail.hierarchy.level1}</Typography>
                <Typography variant="body2">• L2: {detail.hierarchy.level2}</Typography>
                <Typography variant="body2">• L3: {detail.hierarchy.level3}</Typography>
            </Box>

            <Divider sx={{ mb: 2 }} />

            <Typography variant="subtitle1" gutterBottom>Source</Typography>
            <Box mb={2}>
                <Typography variant="caption" display="block">Semantic Terms:</Typography>
                {detail.sourceSemanticTerms.map(t => <Typography key={t} variant="body2" sx={{ ml: 1 }}>- {t}</Typography>)}
                <Typography variant="caption" display="block" sx={{ mt: 1 }}>Columns:</Typography>
                {detail.sourceColumns.map(c => <Typography key={c} variant="body2" sx={{ ml: 1 }}>- {c}</Typography>)}
            </Box>

            <Box mt={4} display="flex" gap={2}>
                <Button variant="contained" color="primary" fullWidth>Accept & Create</Button>
                <Button variant="outlined" color="primary" fullWidth>Edit</Button>
            </Box>
            <Box mt={2}>
                <Button variant="text" color="error" fullWidth>Reject</Button>
            </Box>
        </Box>
    );
};

export default BusinessTermReviewDrawer;
