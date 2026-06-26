import type { Dispatch, SetStateAction } from 'react';
import { Card, CardHeader, CardContent, Typography, Button, TextField, Chip, Grid, Box, InputAdornment } from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import TableChartIcon from '@mui/icons-material/TableChart';
import StorageIcon from '@mui/icons-material/Storage';
import { SemanticModelConfig, DATABASE_SCHEMA } from './types'
import { addDimensionFromColumn, addMeasureFromColumn, isNumericType } from './utils';

interface ExplorerTabProps {
  config: SemanticModelConfig;
  setConfig: Dispatch<SetStateAction<SemanticModelConfig>>;
  searchTerm: string;
  setSearchTerm: Dispatch<SetStateAction<string>>;
  toast: (options: { title: string; description: string; variant?: string }) => void;
}

export default function ExplorerTab({ config, setConfig, searchTerm, setSearchTerm, toast }: ExplorerTabProps) {
  const filteredTables = DATABASE_SCHEMA.filter(table =>
    table.table_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    table.columns.some(col => col.column_name.toLowerCase().includes(searchTerm.toLowerCase()))
  );
  
  return (
    <Box sx={{ pt: 2 }}>
      <TextField
        fullWidth
        placeholder="Search tables and columns..."
        value={searchTerm}
        onChange={(e) => setSearchTerm(e.target.value)}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon />
            </InputAdornment>
          ),
        }}
        sx={{ mb: 2 }}
      />

      <Grid container spacing={2}>
        <Grid item xs={12} md={8}>
          {filteredTables.map((table) => (
            <Card key={table.table_name} sx={{ mb: 2 }}>
              <CardHeader title={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <TableChartIcon />
                  <Typography variant="h6">{table.table_name}</Typography>
                </Box>
              } />
              <CardContent>
                {table.columns
                  .filter(column => 
                    !searchTerm || 
                    column.column_name.toLowerCase().includes(searchTerm.toLowerCase())
                  )
                  .map((column) => (
                    <Box key={column.column_name} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', p: 1, border: '1px solid lightgray', borderRadius: 1, mb: 1 }}>
                      <div>
                        <Typography variant="subtitle1">{column.column_name} <Chip label={column.data_type} size="small" /></Typography>
                      </div>
                      <div>
                        <Button size="small" onClick={() => addDimensionFromColumn(table.table_name, column, true, setConfig, toast)} sx={{ mr: 1 }}>Add Core Dimension</Button>
                        <Button size="small" variant="outlined" onClick={() => addDimensionFromColumn(table.table_name, column, false, setConfig, toast)} sx={{ mr: 1 }}>Add Custom Dimension</Button>
                        {isNumericType(column.data_type) && (
                          <>
                            <Button size="small" onClick={() => addMeasureFromColumn(table.table_name, column, true, setConfig, toast)} sx={{ mr: 1 }}>Add Core Measure</Button>
                            <Button size="small" variant="outlined" onClick={() => addMeasureFromColumn(table.table_name, column, false, setConfig, toast)}>Add Custom Measure</Button>
                          </>
                        )}
                      </div>
                    </Box>
                  ))}
              </CardContent>
            </Card>
          ))}
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardHeader title={
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <StorageIcon />
                <Typography variant="h6">Quick Stats</Typography>
              </Box>
            } />
            <CardContent sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography>Core Dimensions:</Typography>
                <Chip label={config.core.dimensions.length} color="primary" />
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography>Custom Dimensions:</Typography>
                <Chip label={config.custom.dimensions.length} />
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography>Core Measures:</Typography>
                <Chip label={config.core.measures.length} color="primary" />
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography>Custom Measures:</Typography>
                <Chip label={config.custom.measures.length} />
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography>Available Tables:</Typography>
                <Chip label={DATABASE_SCHEMA.length} color="secondary" />
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}