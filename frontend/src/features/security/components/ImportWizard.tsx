import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  Alert,
  LinearProgress,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Chip,
} from '@mui/material';
import {
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { accessRulesApi, AccessRuleInput } from '../../../api/accessRules';

interface ImportResult {
  success: number;
  failed: number;
  errors: Array<{ line: number; error: string; rule?: any }>;
}

interface ImportWizardProps {
  open: boolean;
  onClose: () => void;
  onComplete: () => void;
}

export const ImportWizard: React.FC<ImportWizardProps> = ({ open, onClose, onComplete }) => {
  const [file, setFile] = useState<File | null>(null);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = event.target.files?.[0];
    if (selectedFile) {
      setFile(selectedFile);
      setResult(null);
    }
  };

  const handleImport = async () => {
    if (!file) return;

    setImporting(true);
    try {
      const text = await file.text();
      const rules: AccessRuleInput[] = JSON.parse(text);

      let success = 0;
      let failed = 0;
      const errors: ImportResult['errors'] = [];

      for (let i = 0; i < rules.length; i++) {
        try {
          await accessRulesApi.create(rules[i]);
          success++;
        } catch (error: any) {
          failed++;
          errors.push({
            line: i + 1,
            error: error?.message || 'Unknown error',
            rule: rules[i],
          });
        }
      }

      setResult({ success, failed, errors });
      if (failed === 0) {
        setTimeout(() => {
          onComplete();
          handleClose();
        }, 2000);
      }
    } catch (error: any) {
      setResult({
        success: 0,
        failed: 1,
        errors: [{ line: 0, error: error?.message || 'Invalid JSON file' }],
      });
    } finally {
      setImporting(false);
    }
  };

  const handleClose = () => {
    setFile(null);
    setResult(null);
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>Import Access Rules</DialogTitle>
      <DialogContent>
        <Box sx={{ py: 2 }}>
          {!result && (
            <>
              <Alert severity="info" sx={{ mb: 3 }}>
                Upload a JSON file containing access rules. The file should be an array of rule objects.
              </Alert>

              <input
                type="file"
                accept=".json"
                onChange={handleFileChange}
                style={{ display: 'none' }}
                id="import-file-input"
              />
              <label htmlFor="import-file-input">
                <Button variant="outlined" component="span" fullWidth>
                  {file ? file.name : 'Choose File'}
                </Button>
              </label>

              {importing && (
                <Box sx={{ mt: 3 }}>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                    Importing rules...
                  </Typography>
                  <LinearProgress />
                </Box>
              )}
            </>
          )}

          {result && (
            <Box>
              {result.failed === 0 ? (
                <Alert severity="success" icon={<SuccessIcon />} sx={{ mb: 2 }}>
                  Successfully imported {result.success} rule(s)
                </Alert>
              ) : (
                <>
                  <Alert severity="warning" icon={<WarningIcon />} sx={{ mb: 2 }}>
                    Imported {result.success} rule(s), {result.failed} failed
                  </Alert>

                  <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                    Errors:
                  </Typography>
                  <List dense>
                    {result.errors.map((err, index) => (
                      <ListItem key={index}>
                        <ListItemIcon>
                          <ErrorIcon color="error" />
                        </ListItemIcon>
                        <ListItemText
                          primary={`Line ${err.line}: ${err.error}`}
                          secondary={err.rule ? JSON.stringify(err.rule).substring(0, 100) : undefined}
                        />
                      </ListItem>
                    ))}
                  </List>
                </>
              )}
            </Box>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>
          {result ? 'Close' : 'Cancel'}
        </Button>
        {!result && (
          <Button
            variant="contained"
            onClick={handleImport}
            disabled={!file || importing}
          >
            Import
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};

export default ImportWizard;
