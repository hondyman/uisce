import React, { useState } from 'react';
import { 
  Box, 
  Button, 
  Typography, 
  Stepper, 
  Step, 
  StepLabel, 
  Paper, 
  CircularProgress,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Checkbox,
  TextField,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert
} from '@mui/material';
import AutoFixHighIcon from '@mui/icons-material/AutoFixHigh';
import SearchIcon from '@mui/icons-material/Search';
import SaveIcon from '@mui/icons-material/Save';
import { useAbbreviationWizard, useAbbreviations, SuggestionResult } from '../utils/abbreviationApi';

// Step definitions
const STEPS = ['Scan Database', 'Review Candidates', 'Get Suggestions', 'Final Review'];

interface AbbreviationWizardProps {
  onCompletion?: () => void;
}

export const AbbreviationWizard: React.FC<AbbreviationWizardProps> = ({ onCompletion }) => {
  const [activeStep, setActiveStep] = useState(0);
  const [scannedCandidates, setScannedCandidates] = useState<string[]>([]);
  const [selectedCandidates, setSelectedCandidates] = useState<string[]>([]);
  const [suggestions, setSuggestions] = useState<SuggestionResult>({});
  const [finalReviewData, setFinalReviewData] = useState<{abbr: string, full: string}[]>([]);
  
  const { scan, suggest, loading: wizardLoading, error: wizardError } = useAbbreviationWizard();
  const { addAbbreviation } = useAbbreviations();
  const [bulkSaveLoading, setBulkSaveLoading] = useState(false);

  // --- Step Handlers ---

  const handleScan = async () => {
    const result = await scan();
    if (result) {
      setScannedCandidates(result.candidates);
      setSelectedCandidates(result.candidates); // Select all by default
      setActiveStep(1);
    }
  };

  const handleSuggest = async () => {
    const result = await suggest(selectedCandidates);
    if (result) {
      setSuggestions(result);
      // Pre-populate final review data
      const reviewData = selectedCandidates.map(abbr => ({
        abbr,
        full: result[abbr] || ''
      }));
      setFinalReviewData(reviewData);
      setActiveStep(3);
    }
  };

  const handleSave = async () => {
    setBulkSaveLoading(true);
    let successCount = 0;
    
    // Filter out items with empty full words
    const itemsToSave = finalReviewData.filter(item => item.full.trim() !== '');
    
    for (const item of itemsToSave) {
      const success = await addAbbreviation(item.abbr, item.full, 'Added via Wizard');
      if (success) successCount++;
    }
    
    setBulkSaveLoading(false);
    if (onCompletion) onCompletion();
    // Maybe show toast here? "Saved X abbreviations"
  };

  // --- Step Content Renderers ---

  const renderScanStep = () => (
    <Box sx={{ textAlign: 'center', py: 4 }}>
      <SearchIcon sx={{ 
        fontSize: 60, 
        color: wizardLoading ? 'primary.main' : 'text.secondary', 
        mb: 2,
        animation: wizardLoading ? 'spin 2s linear infinite' : 'none',
        '@keyframes spin': {
          '0%': { transform: 'rotate(0deg)' },
          '100%': { transform: 'rotate(360deg)' },
        },
      }} />
      <Typography variant="h6" gutterBottom>
        Scan Database for Abbreviations
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        This will scan all database columns to identify frequent unknown tokens that might be abbreviations.
      </Typography>
      <Button 
        variant="contained" 
        onClick={handleScan} 
        disabled={wizardLoading}
        startIcon={wizardLoading ? <CircularProgress size={20} /> : <SearchIcon />}
      >
        {wizardLoading ? 'Scanning...' : 'Start Scan'}
      </Button>
      {wizardError && <Alert severity="error" sx={{ mt: 2 }}>{wizardError}</Alert>}
    </Box>
  );

  const renderReviewCandidatesStep = () => (
    <Box>
      <Typography variant="subtitle1" gutterBottom>
        Found {scannedCandidates.length} potential abbreviations. Select the ones you want to process.
      </Typography>
      <Paper variant="outlined" sx={{ maxHeight: 400, overflow: 'auto', mb: 2 }}>
        <List dense>
          {scannedCandidates.map((abbr) => {
             const labelId = `checkbox-list-label-${abbr}`;
             return (
               <ListItem
                 key={abbr}
                 button
                 onClick={() => {
                   const currentIndex = selectedCandidates.indexOf(abbr);
                   const newChecked = [...selectedCandidates];
                   if (currentIndex === -1) {
                     newChecked.push(abbr);
                   } else {
                     newChecked.splice(currentIndex, 1);
                   }
                   setSelectedCandidates(newChecked);
                 }}
               >
                 <ListItemIcon>
                   <Checkbox
                     edge="start"
                     checked={selectedCandidates.indexOf(abbr) !== -1}
                     tabIndex={-1}
                     disableRipple
                     inputProps={{ 'aria-labelledby': labelId }}
                   />
                 </ListItemIcon>
                 <ListItemText id={labelId} primary={abbr} />
               </ListItem>
             );
          })}
        </List>
      </Paper>
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1 }}>
        <Button onClick={() => setActiveStep(0)}>Back</Button>
        <Button 
          variant="contained" 
          onClick={() => setActiveStep(2)}
          disabled={selectedCandidates.length === 0}
        >
          Next
        </Button>
      </Box>
    </Box>
  );

  const renderGetSuggestionsStep = () => (
    <Box sx={{ textAlign: 'center', py: 4 }}>
      <AutoFixHighIcon sx={{ 
        fontSize: 60, 
        color: 'primary.main', 
        mb: 2,
        animation: wizardLoading ? 'glow 1.5s ease-in-out infinite' : 'none',
        '@keyframes glow': {
          '0%, 100%': { filter: 'brightness(1)' },
          '50%': { filter: 'brightness(1.5)' },
        },
      }} />
      <Typography variant="h6" gutterBottom>
        Generate AI Suggestions
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
        Using AI to suggest full word expansions for {selectedCandidates.length} selected abbreviations...
      </Typography>
      <Button 
        variant="contained" 
        onClick={handleSuggest} 
        disabled={wizardLoading}
        startIcon={wizardLoading ? <CircularProgress size={20} /> : <AutoFixHighIcon />}
      >
        {wizardLoading ? 'Generating...' : 'Generate Suggestions'}
      </Button>
      {wizardError && <Alert severity="error" sx={{ mt: 2 }}>{wizardError}</Alert>}
      <Box sx={{ mt: 2 }}>
        <Button onClick={() => setActiveStep(1)} disabled={wizardLoading}>Back</Button>
      </Box>
    </Box>
  );

  const renderFinalReviewStep = () => (
    <Box>
      <Typography variant="subtitle1" gutterBottom>
        Review and edit the suggested expansions before saving.
      </Typography>
      <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 400, mb: 2 }}>
        <Table stickyHeader size="small">
          <TableHead>
            <TableRow>
              <TableCell>Abbreviation</TableCell>
              <TableCell>Full Word (Editable)</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {finalReviewData.map((row, index) => (
              <TableRow key={row.abbr}>
                <TableCell>{row.abbr}</TableCell>
                <TableCell>
                  <TextField 
                    fullWidth 
                    variant="standard" 
                    value={row.full} 
                    onChange={(e) => {
                      const newData = [...finalReviewData];
                      newData[index].full = e.target.value.toUpperCase();
                      setFinalReviewData(newData);
                    }}
                    placeholder="Enter full word..."
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1 }}>
        <Button onClick={() => setActiveStep(2)} disabled={bulkSaveLoading}>Back</Button>
        <Button 
          variant="contained" 
          onClick={handleSave}
          disabled={bulkSaveLoading}
          startIcon={bulkSaveLoading ? <CircularProgress size={20} /> : <SaveIcon />}
        >
          {bulkSaveLoading ? 'Saving...' : 'Save All'}
        </Button>
      </Box>
    </Box>
  );

  return (
    <Box sx={{ width: '100%', p: 2 }}>
      <Stepper activeStep={activeStep} alternativeLabel sx={{ mb: 4 }}>
        {STEPS.map((label, index) => (
          <Step key={label} completed={index < activeStep}>
            <StepLabel 
              sx={{
                '& .MuiStepLabel-label': {
                  color: index === activeStep ? 'primary.main' : index < activeStep ? 'success.main' : 'text.secondary',
                  fontWeight: index === activeStep ? 'bold' : 'normal',
                },
                '& .MuiStepIcon-root': {
                  color: index < activeStep ? 'success.main' : undefined,
                  '&.Mui-active': {
                    color: 'primary.main',
                    animation: 'pulse 1.5s ease-in-out infinite',
                  },
                },
                '@keyframes pulse': {
                  '0%, 100%': { transform: 'scale(1)' },
                  '50%': { transform: 'scale(1.1)' },
                },
              }}
            >
              {label}
            </StepLabel>
          </Step>
        ))}
      </Stepper>

      {activeStep === 0 && renderScanStep()}
      {activeStep === 1 && renderReviewCandidatesStep()}
      {activeStep === 2 && renderGetSuggestionsStep()}
      {activeStep === 3 && renderFinalReviewStep()}
    </Box>
  );
};
