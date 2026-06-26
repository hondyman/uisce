import type { FC } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  Chip,
  Stack,
  Divider as _Divider,
  TextField,
  MenuItem,
  Checkbox,
  FormControlLabel,
  Paper as _Paper,
} from '@mui/material';
import { PropertyDef as _PropertyDef } from './PropertySchemaEditor';

interface DisplayProperty {
  name: string;
  label?: string;
  data_type: string;
  input_type: string;
  required?: boolean;
  options?: string[];
  original_data_type?: string;
  original_input_type?: string;
  nullable?: boolean;
  default_value?: any;
  format?: string;
  validation?: any;
  syntax_language?: 'sql' | 'yaml' | 'json' | null;
}

const PropertyPreview: FC<{ property: DisplayProperty }> = ({ property }) => {
  const renderFormControl = () => {
    const label = property.label || property.name;
    const isRequired = property.required || (property.validation?.required && !property.nullable);

    switch (property.input_type) {
      case 'text':
      case 'textarea':
        return (
          <TextField
            label={label}
            placeholder={`Enter ${property.data_type} value`}
            multiline={property.input_type === 'textarea'}
            rows={property.input_type === 'textarea' ? 3 : 1}
            fullWidth
            size="small"
            required={isRequired}
            disabled
          />
        );

      case 'number':
        return (
          <TextField
            label={label}
            type="number"
            placeholder="Enter number"
            fullWidth
            size="small"
            required={isRequired}
            disabled
          />
        );

      case 'select':
      case 'dropdown':
        return (
          <TextField
            select
            label={label}
            fullWidth
            size="small"
            required={isRequired}
            disabled
            value=""
            SelectProps={{
              displayEmpty: true,
            }}
          >
            <MenuItem value="" disabled>
              Select an option...
            </MenuItem>
            {property.options?.map((option, idx) => (
              <MenuItem key={idx} value={option}>
                {option}
              </MenuItem>
            ))}
          </TextField>
        );

      case 'checkbox':
        return (
          <FormControlLabel
            control={<Checkbox disabled checked={false} />}
            label={label}
          />
        );

      case 'date':
      case 'date-picker':
        return (
          <TextField
            label={label}
            type="date"
            fullWidth
            size="small"
            required={isRequired}
            disabled
            InputLabelProps={{ shrink: true }}
          />
        );

      case 'json-editor':
        return (
          <TextField
            label={label}
            placeholder='{"key": "value"}'
            multiline
            rows={4}
            fullWidth
            size="small"
            required={isRequired}
            disabled
          />
        );

      case 'code-editor':
        return (
          <TextField
            label={label}
            placeholder={`Enter ${property.syntax_language || 'code'} content`}
            multiline
            rows={4}
            fullWidth
            size="small"
            required={isRequired}
            disabled
          />
        );

      default:
        return (
          <TextField
            label={label}
            placeholder={`Enter ${property.data_type} value`}
            fullWidth
            size="small"
            required={isRequired}
            disabled
          />
        );
    }
  };

  return (
    <Box sx={{ mb: 2 }}>
      {renderFormControl()}
    </Box>
  );
};

interface PropertiesModalProps {
  open: boolean;
  onClose: () => void;
  properties: DisplayProperty[];
  title: string;
}

export const PropertiesModal: FC<PropertiesModalProps> = ({
  open,
  onClose,
  properties,
  title,
}) => {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: { borderRadius: 2 },
      }}
    >
      <DialogTitle sx={{ pb: 1 }}>
        <Typography variant="h6" component="div" sx={{ fontWeight: 600 }}>
          {title}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {properties.length} {properties.length === 1 ? 'property' : 'properties'}
        </Typography>
      </DialogTitle>

      <DialogContent dividers>
        {properties.length === 0 ? (
          <Box sx={{ textAlign: 'center', py: 4 }}>
            <Typography variant="body2" color="text.secondary">
              No properties defined
            </Typography>
          </Box>
        ) : (
          <Stack spacing={2}>
            {properties.map((property, index) => (
              <Box key={index} sx={{ p: 2, border: '1px solid', borderColor: 'divider', borderRadius: 1 }}>
                <Stack spacing={2}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                      {property.label || property.name}
                    </Typography>
                    {(property.required || (property.validation?.required && !property.nullable)) && (
                      <Chip
                        label="Required"
                        size="small"
                        color="error"
                        variant="outlined"
                        sx={{ fontSize: '0.7rem', height: 20 }}
                      />
                    )}
                    {property.nullable === false && (
                      <Chip
                        label="Not Null"
                        size="small"
                        color="warning"
                        variant="outlined"
                        sx={{ fontSize: '0.7rem', height: 20 }}
                      />
                    )}
                  </Box>

                  {/* Form Preview */}
                  <Box>
                    <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 500, mb: 1, display: 'block' }}>
                      Form Preview:
                    </Typography>
                    <PropertyPreview property={property} />
                  </Box>

                  {/* Property Metadata */}
                  <Box>
                    <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 500, mb: 1, display: 'block' }}>
                      Property Details:
                    </Typography>
                    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
                      <Chip
                        label={`Type: ${property.data_type}`}
                        size="small"
                        color="primary"
                        variant="outlined"
                        sx={{ fontSize: '0.7rem' }}
                      />
                      <Chip
                        label={`Input: ${property.input_type}`}
                        size="small"
                        color="secondary"
                        variant="outlined"
                        sx={{ fontSize: '0.7rem' }}
                      />
                      {property.original_data_type && property.original_data_type !== property.data_type && (
                        <Chip
                          label={`Original: ${property.original_data_type}`}
                          size="small"
                          color="info"
                          variant="outlined"
                          sx={{ fontSize: '0.7rem' }}
                        />
                      )}
                    </Box>

                    {property.options && property.options.length > 0 && (
                      <Box sx={{ mt: 1 }}>
                        <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 500 }}>
                          Options:
                        </Typography>
                        <Box sx={{ mt: 0.5 }}>
                          <Stack direction="row" spacing={0.5} flexWrap="wrap">
                            {property.options.map((option, optIndex) => (
                              <Chip
                                key={optIndex}
                                label={option}
                                size="small"
                                variant="outlined"
                                sx={{ fontSize: '0.7rem', height: 24 }}
                              />
                            ))}
                          </Stack>
                        </Box>
                      </Box>
                    )}

                    <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace', mt: 1, display: 'block' }}>
                      Machine name: {property.name}
                    </Typography>
                  </Box>
                </Stack>
              </Box>
            ))}
          </Stack>
        )}
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2 }}>
        <Button onClick={onClose} variant="outlined">
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default PropertiesModal;