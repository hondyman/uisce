import React, { useState } from 'react';
import {
  Box,
  Typography,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  SelectChangeEvent,
  Chip,
  OutlinedInput,
  Checkbox,
  ListItemText,
  Button,
  Paper,
  Grid,
  TextField,
  InputAdornment
} from '@mui/material';
import { Search, FilterList, Clear } from '@mui/icons-material';

interface Fund {
  id: string;
  name: string;
  vintage: number;
  manager: string;
  strategy: string;
  geography: string;
  status: 'active' | 'liquidated' | 'realizing';
}

interface FundSelectorProps {
  availableFunds: Fund[];
  selectedFunds: string[];
  onSelectionChange: (fundIds: string[]) => void;
}

export const FundSelector: React.FC<FundSelectorProps> = ({
  availableFunds,
  selectedFunds,
  onSelectionChange
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStrategy, setFilterStrategy] = useState<string>('');
  const [filterGeography, setFilterGeography] = useState<string>('');
  const [filterVintage, setFilterVintage] = useState<string>('');

  // Get unique values for filters
  const strategies = [...new Set(availableFunds.map(fund => fund.strategy))];
  const geographies = [...new Set(availableFunds.map(fund => fund.geography))];
  const vintages = [...new Set(availableFunds.map(fund => fund.vintage))].sort((a, b) => b - a);

  // Filter funds based on search and filters
  const filteredFunds = availableFunds.filter(fund => {
    const matchesSearch = fund.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         fund.manager.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStrategy = !filterStrategy || fund.strategy === filterStrategy;
    const matchesGeography = !filterGeography || fund.geography === filterGeography;
    const matchesVintage = !filterVintage || fund.vintage.toString() === filterVintage;

    return matchesSearch && matchesStrategy && matchesGeography && matchesVintage;
  });

  const handleFundChange = (event: SelectChangeEvent<string[]>) => {
    const value = event.target.value as string[];
    onSelectionChange(value);
  };

  const handleSelectAll = () => {
    onSelectionChange(filteredFunds.map(fund => fund.id));
  };

  const handleClearAll = () => {
    onSelectionChange([]);
  };

  const handleFilterChange = (filterType: string, value: string) => {
    switch (filterType) {
      case 'strategy':
        setFilterStrategy(value);
        break;
      case 'geography':
        setFilterGeography(value);
        break;
      case 'vintage':
        setFilterVintage(value);
        break;
    }
  };

  const clearFilters = () => {
    setSearchTerm('');
    setFilterStrategy('');
    setFilterGeography('');
    setFilterVintage('');
  };

  const selectedFundObjects = availableFunds.filter(fund => selectedFunds.includes(fund.id));

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Fund Selection
      </Typography>

      <Grid container spacing={2}>
        {/* Search */}
        <Grid item xs={12} md={3}>
          <TextField
            fullWidth
            label="Search Funds"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <Search />
                </InputAdornment>
              ),
            }}
          />
        </Grid>

        {/* Filters */}
        <Grid item xs={12} md={2}>
          <FormControl fullWidth>
            <InputLabel>Strategy</InputLabel>
            <Select
              value={filterStrategy}
              label="Strategy"
              onChange={(e) => handleFilterChange('strategy', e.target.value)}
            >
              <MenuItem value="">
                <em>All Strategies</em>
              </MenuItem>
              {strategies.map(strategy => (
                <MenuItem key={strategy} value={strategy}>
                  {strategy}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </Grid>

        <Grid item xs={12} md={2}>
          <FormControl fullWidth>
            <InputLabel>Geography</InputLabel>
            <Select
              value={filterGeography}
              label="Geography"
              onChange={(e) => handleFilterChange('geography', e.target.value)}
            >
              <MenuItem value="">
                <em>All Regions</em>
              </MenuItem>
              {geographies.map(geography => (
                <MenuItem key={geography} value={geography}>
                  {geography}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </Grid>

        <Grid item xs={12} md={2}>
          <FormControl fullWidth>
            <InputLabel>Vintage</InputLabel>
            <Select
              value={filterVintage}
              label="Vintage"
              onChange={(e) => handleFilterChange('vintage', e.target.value)}
            >
              <MenuItem value="">
                <em>All Vintages</em>
              </MenuItem>
              {vintages.map(vintage => (
                <MenuItem key={vintage} value={vintage}>
                  {vintage}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </Grid>

        <Grid item xs={12} md={3}>
          <Box display="flex" gap={1}>
            <Button
              variant="outlined"
              startIcon={<FilterList />}
              onClick={clearFilters}
              size="small"
            >
              Clear Filters
            </Button>
            <Button
              variant="contained"
              onClick={handleSelectAll}
              size="small"
            >
              Select All
            </Button>
            <Button
              variant="outlined"
              onClick={handleClearAll}
              size="small"
            >
              Clear All
            </Button>
          </Box>
        </Grid>
      </Grid>

      {/* Fund Multi-Select */}
      <Box sx={{ mt: 2 }}>
        <FormControl fullWidth>
          <InputLabel>Selected Funds ({selectedFunds.length})</InputLabel>
          <Select
            multiple
            value={selectedFunds}
            onChange={handleFundChange}
            input={<OutlinedInput label={`Selected Funds (${selectedFunds.length})`} />}
            renderValue={(selected) => (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {selected.map((fundId) => {
                  const fund = availableFunds.find(f => f.id === fundId);
                  return fund ? (
                    <Chip
                      key={fundId}
                      label={`${fund.name} (${fund.vintage})`}
                      size="small"
                      onDelete={() => {
                        const newSelected = selectedFunds.filter(id => id !== fundId);
                        onSelectionChange(newSelected);
                      }}
                      deleteIcon={<Clear />}
                    />
                  ) : null;
                })}
              </Box>
            )}
            MenuProps={{
              PaperProps: {
                style: {
                  maxHeight: 300,
                },
              },
            }}
          >
            {filteredFunds.map((fund) => (
              <MenuItem key={fund.id} value={fund.id}>
                <Checkbox checked={selectedFunds.indexOf(fund.id) > -1} />
                <ListItemText
                  primary={`${fund.name} (${fund.vintage})`}
                  secondary={`${fund.manager} • ${fund.strategy} • ${fund.geography}`}
                />
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>

      {/* Selected Funds Summary */}
      {selectedFundObjects.length > 0 && (
        <Paper sx={{ mt: 2, p: 2, bgcolor: 'grey.50' }}>
          <Typography variant="subtitle2" gutterBottom>
            Selected Funds Summary
          </Typography>
          <Box display="flex" flexWrap="wrap" gap={1}>
            {selectedFundObjects.map(fund => (
              <Chip
                key={fund.id}
                label={`${fund.name} (${fund.vintage})`}
                size="small"
                color="primary"
                variant="outlined"
              />
            ))}
          </Box>
        </Paper>
      )}
    </Box>
  );
};
