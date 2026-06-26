import React, { useState } from 'react';
import {
  Box,
  Button,
  Card as _Card,
  CardContent as _CardContent,
  CardHeader as _CardHeader,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  TextField,
  Typography,
  Alert,
  Chip,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DownloadIcon from '@mui/icons-material/Download';

/**
 * Data generator for testing validation rules
 */
interface SampleDataGeneratorProps {
  entity: string;
  fields: Array<{
    name: string;
    dataType: 'string' | 'number' | 'date' | 'boolean' | 'email';
    format?: string;
  }>;
  onDataGenerated: (data: any[], format: 'json' | 'csv') => void;
}

/**
 * Sample Data Generator Component
 * 
 * Generates test data matching field definitions:
 * - Common patterns (Email validation, Phone, Date formats)
 * - Edge cases (null, empty, special characters)
 * - Industry patterns (HR, Finance, etc.)
 * - Custom value ranges
 */
const SampleDataGenerator: React.FC<SampleDataGeneratorProps> = ({
  entity,
  fields,
  onDataGenerated,
}) => {
  const [open, setOpen] = useState(false);
  const [recordCount, setRecordCount] = useState(10);
  const [includeEdgeCases, setIncludeEdgeCases] = useState(true);
  const [generatedData, setGeneratedData] = useState<any[] | null>(null);
  const [selectedFormat, setSelectedFormat] = useState<'json' | 'csv'>('json');

  // Generate sample values based on field type and format
  const generateFieldValue = (field: any, includeNull = false) => {
    if (includeNull && Math.random() < 0.1) return null;

    switch (field.dataType) {
      case 'string':
        if (field.format === 'email') {
          return `user${Math.floor(Math.random() * 10000)}@example.com`;
        }
        return `value_${Math.random().toString(36).substring(7)}`;

      case 'number':
        return Math.floor(Math.random() * 1000);

      case 'date':
        const date = new Date();
        date.setDate(date.getDate() - Math.floor(Math.random() * 365));
        return date.toISOString().split('T')[0];

      case 'boolean':
        return Math.random() > 0.5;

      case 'email':
        return `test${Math.floor(Math.random() * 10000)}@company.com`;

      default:
        return `sample_value`;
    }
  };

  // Generate edge case value
  const generateEdgeCaseValue = (field: any) => {
    const edgeCases: any = {
      string: ['', null, '  ', 'very_long_' + 'x'.repeat(100)],
      number: [0, -1, 999999999],
      date: ['', '2099-12-31', '1900-01-01'],
      boolean: [null, true, false],
      email: ['invalid-email', '@example.com', 'test@.com'],
    };

    const values = edgeCases[field.dataType] || [''];
    return values[Math.floor(Math.random() * values.length)];
  };

  // Generate sample data
  const handleGenerateData = () => {
    const data: any[] = [];

    for (let i = 0; i < recordCount; i++) {
      const record: any = {
        id: `${entity}_${i + 1}`,
      };

    fields.forEach((field, _idx) => {
        // Mix normal and edge cases
        if (includeEdgeCases && i < Math.ceil(recordCount * 0.2)) {
          record[field.name] = generateEdgeCaseValue(field);
        } else {
          record[field.name] = generateFieldValue(field, includeEdgeCases && Math.random() < 0.05);
        }
      });

      data.push(record);
    }

    setGeneratedData(data);
  };

  // Convert to CSV
  const toCSV = (data: any[]) => {
    if (data.length === 0) return '';

    const headers = Object.keys(data[0]);
    const rows = [headers.join(',')];

    data.forEach(record => {
      const values = headers.map(header => {
        const value = record[header];
        if (value === null || value === undefined) return '';
        if (typeof value === 'string' && (value.includes(',') || value.includes('"'))) {
          return `"${value.replace(/"/g, '""')}"`;
        }
        return value;
      });
      rows.push(values.join(','));
    });

    return rows.join('\n');
  };

  // Download data
  const handleDownload = () => {
    if (!generatedData) return;

    const data = selectedFormat === 'json'
      ? JSON.stringify(generatedData, null, 2)
      : toCSV(generatedData);

    const element = document.createElement('a');
    element.setAttribute(
      'href',
      `data:text/${selectedFormat === 'json' ? 'plain' : 'csv'};charset=utf-8,${encodeURIComponent(data)}`
    );
    element.setAttribute('download', `${entity}_sample_data.${selectedFormat === 'json' ? 'json' : 'csv'}`);
    element.style.display = 'none';
    document.body.appendChild(element);
    element.click();
    document.body.removeChild(element);
  };

  // Copy to clipboard
  const handleCopyToClipboard = () => {
    if (!generatedData) return;

    const data = selectedFormat === 'json'
      ? JSON.stringify(generatedData, null, 2)
      : toCSV(generatedData);

    navigator.clipboard.writeText(data);
  };

  // Confirm and pass back
  const handleConfirm = () => {
    if (generatedData) {
      onDataGenerated(generatedData, selectedFormat);
      setOpen(false);
      setGeneratedData(null);
    }
  };

  return (
    <Box>
      <Button
        variant="outlined"
        onClick={() => setOpen(true)}
        fullWidth
      >
        Generate Sample Data
      </Button>

      <Dialog open={open} onClose={() => setOpen(false)} maxWidth="lg" fullWidth>
        <DialogTitle>Sample Data Generator</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {!generatedData ? (
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <TextField
                  type="number"
                  label="Number of Records"
                  value={recordCount}
                  onChange={e => setRecordCount(parseInt(e.target.value))}
                  inputProps={{ min: 1, max: 1000 }}
                  fullWidth
                />
              </Grid>

              <Grid item xs={12}>
                <FormControl fullWidth>
                  <InputLabel>Options</InputLabel>
                  <Select
                    value={includeEdgeCases ? 'with-edges' : 'normal'}
                    label="Options"
                    onChange={e => setIncludeEdgeCases(e.target.value === 'with-edges')}
                  >
                    <MenuItem value="normal">Normal Data Only</MenuItem>
                    <MenuItem value="with-edges">Include Edge Cases (nulls, empty strings)</MenuItem>
                  </Select>
                </FormControl>
              </Grid>

              <Grid item xs={12}>
                <Alert severity="info">
                  Will generate {recordCount} sample records for {entity} with {fields.length} fields.
                  {includeEdgeCases && ' Includes edge cases like nulls and empty values.'}
                </Alert>
              </Grid>

              <Grid item xs={12}>
                <Button
                  variant="contained"
                  onClick={handleGenerateData}
                  startIcon={<RefreshIcon />}
                  fullWidth
                >
                  Generate Data
                </Button>
              </Grid>
            </Grid>
          ) : (
            <Box>
              <Box sx={{ mb: 2, display: 'flex', gap: 1, alignItems: 'center' }}>
                <Chip label={`${generatedData.length} records`} />
                <FormControl sx={{ minWidth: 120 }}>
                  <InputLabel>Format</InputLabel>
                  <Select
                    value={selectedFormat}
                    label="Format"
                    onChange={e => setSelectedFormat(e.target.value as 'json' | 'csv')}
                  >
                    <MenuItem value="json">JSON</MenuItem>
                    <MenuItem value="csv">CSV</MenuItem>
                  </Select>
                </FormControl>
              </Box>

              {/* Preview Table */}
              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                Preview (first 5 records)
              </Typography>
              <TableContainer sx={{ mb: 2, maxHeight: 300 }}>
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
                      {Object.keys(generatedData[0]).map(key => (
                        <TableCell key={key} sx={{ fontWeight: 'bold' }}>
                          {key}
                        </TableCell>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {generatedData.slice(0, 5).map((record, idx) => (
                      <TableRow key={idx}>
                        {Object.values(record).map((value, colIdx) => (
                          <TableCell key={colIdx}>
                            {value === null ? (
                              <Typography variant="caption" color="textSecondary">
                                NULL
                              </Typography>
                            ) : value === '' ? (
                              <Typography variant="caption" color="textSecondary">
                                (empty)
                              </Typography>
                            ) : (
                              String(value).substring(0, 30)
                            )}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>

              <Typography variant="caption" color="textSecondary">
                {selectedFormat === 'json'
                  ? `JSON: ${generatedData.length} records`
                  : `CSV: ${generatedData.length} rows + header`}
              </Typography>
            </Box>
          )}
        </DialogContent>

        <DialogActions>
          {generatedData && (
            <>
              <Button
                onClick={handleCopyToClipboard}
                startIcon={<ContentCopyIcon />}
              >
                Copy to Clipboard
              </Button>
              <Button
                onClick={handleDownload}
                startIcon={<DownloadIcon />}
              >
                Download
              </Button>
            </>
          )}
          <Box sx={{ flex: 1 }} />
          <Button onClick={() => {
            setOpen(false);
            setGeneratedData(null);
          }}>
            Cancel
          </Button>
          {generatedData && (
            <Button onClick={handleConfirm} variant="contained">
              Use This Data
            </Button>
          )}
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default SampleDataGenerator;
