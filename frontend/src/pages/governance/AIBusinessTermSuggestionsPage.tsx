import React, { useState } from 'react';
import { 
  Box, Typography, Paper, Grid, TextField, MenuItem, 
  Button, Drawer 
} from '@mui/material';
import SuggestionsTable from '../../components/governance/SuggestionsTable';
import BusinessTermReviewDrawer from '../../components/governance/BusinessTermReviewDrawer';

const AIBusinessTermSuggestionsPage = () => {
    // Mock Data Fetching
    const [selectedDraft, setSelectedDraft] = useState(null);
    const [filterStatus, setFilterStatus] = useState('DRAFT_AI');

    const mockDrafts = [
      {
        businessTermId: "draft-bt-client-address",
        name: "Client Address",
        piiFlag: true,
        sensitivity: "HIGH",
        residency: "UNKNOWN",
        hierarchy: { level1: "Wealth Management", level2: "Client", level3: "Client Address" },
        status: "DRAFT_AI"
      },
      // ... more mock data
    ];

    return (
        <Box p={3}>
            <Typography variant="h4" gutterBottom>AI Business Term Suggestions</Typography>
            
            {/* Filters Bar */}
            <Paper sx={{ p: 2, mb: 3 }}>
                <Grid container spacing={2} alignItems="center">
                    <Grid item xs={3}>
                        <TextField 
                            select fullWidth label="Status" 
                            value={filterStatus}
                            onChange={(e) => setFilterStatus(e.target.value)}
                        >
                            <MenuItem value="DRAFT_AI">Draft AI</MenuItem>
                            <MenuItem value="APPROVED">Approved</MenuItem>
                            <MenuItem value="REJECTED">Rejected</MenuItem>
                        </TextField>
                    </Grid>
                    <Grid item xs={3}>
                        <TextField fullWidth label="Search" placeholder="Name or Definition..." />
                    </Grid>
                    <Grid item xs={6} display="flex" justifyContent="flex-end">
                        <Button variant="contained">Generate New</Button>
                    </Grid>
                </Grid>
            </Paper>

            {/* Suggestions Table */}
            <SuggestionsTable 
                drafts={mockDrafts} 
                onSelect={(draft) => setSelectedDraft(draft)} 
            />

            {/* Review Drawer */}
            <Drawer 
                anchor="right" 
                open={!!selectedDraft} 
                onClose={() => setSelectedDraft(null)}
                PaperProps={{ sx: { width: '40%' } }}
            >
                {selectedDraft && (
                    <BusinessTermReviewDrawer 
                        draft={selectedDraft} 
                        onClose={() => setSelectedDraft(null)} 
                    />
                )}
            </Drawer>
        </Box>
    );
};

export default AIBusinessTermSuggestionsPage;
