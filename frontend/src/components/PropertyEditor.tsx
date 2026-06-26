import React, { useState } from 'react';
import {
  Box,
  Typography,
  IconButton,
  Button,
  TextField,
  MenuItem,
  Stack,
  Collapse,
  Card,
  Divider,
  alpha,
  useTheme,
  Tooltip
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowRightIcon from '@mui/icons-material/KeyboardArrowRight';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';

export interface PropertyDefinition {
  name: string;
  label: string;
  data_type: 'string' | 'integer' | 'boolean' | 'date' | 'float' | 'json' | 'text' | 'object';
  nullable: boolean;
  description?: string;
  required?: boolean;
  properties?: PropertyDefinition[]; // For hierarchical objects
}

interface PropertyEditorProps {
  properties: PropertyDefinition[];
  onChange: (properties: PropertyDefinition[]) => void;
}

const PROPERTY_TYPES = [
  { value: 'string', label: 'String' },
  { value: 'integer', label: 'Integer' },
  { value: 'float', label: 'Float' },
  { value: 'boolean', label: 'Boolean' },
  { value: 'date', label: 'Date' },
  { value: 'text', label: 'Long Text' },
  { value: 'object', label: 'Nested Object' },
  { value: 'json', label: 'JSON' },
];

export const PropertyEditor: React.FC<PropertyEditorProps> = ({ properties, onChange }) => {
  const theme = useTheme();

  const handleUpdate = (path: number[], updatedProp: PropertyDefinition | null) => {
    const newProps = [...properties];
    let current: any = newProps;

    for (let i = 0; i < path.length - 1; i++) {
      current = current[path[i]].properties;
    }

    const index = path[path.length - 1];
    if (updatedProp === null) {
      current.splice(index, 1);
    } else {
      current[index] = updatedProp;
    }

    onChange(newProps);
  };

  const handleAdd = (path: number[] = []) => {
    const newProps = [...properties];
    const newProp: PropertyDefinition = {
      name: `prop_${Date.now()}`,
      label: '',
      data_type: 'string',
      nullable: true,
      required: false
    };

    if (path.length === 0) {
      newProps.push(newProp);
    } else {
      let current: any = newProps;
      for (let i = 0; i < path.length; i++) {
        if (!current[path[i]].properties) {
          current[path[i]].properties = [];
        }
        current = current[path[i]].properties;
      }
      current.push(newProp);
    }

    onChange(newProps);
  };

  return (
    <Box sx={{ width: '100%' }}>
      <Stack spacing={2}>
        {properties.map((prop, idx) => (
          <PropertyItem
            key={idx}
            property={prop}
            path={[idx]}
            onUpdate={handleUpdate}
            onAddChild={handleAdd}
          />
        ))}
        <Button
          startIcon={<AddIcon />}
          onClick={() => handleAdd()}
          variant="outlined"
          sx={{ 
            mt: 1, 
            borderStyle: 'dashed',
            color: theme.palette.text.secondary,
            borderColor: theme.palette.divider,
            '&:hover': {
              borderColor: theme.palette.primary.main,
              background: alpha(theme.palette.primary.main, 0.05)
            }
          }}
          fullWidth
        >
          Add Property
        </Button>
      </Stack>
    </Box>
  );
};

interface PropertyItemProps {
  property: PropertyDefinition;
  path: number[];
  onUpdate: (path: number[], updatedProp: PropertyDefinition | null) => void;
  onAddChild: (path: number[]) => void;
}

const PropertyItem: React.FC<PropertyItemProps> = ({ property, path, onUpdate, onAddChild }) => {
  const theme = useTheme();
  const [expanded, setExpanded] = useState(true);

  const handleChange = (field: keyof PropertyDefinition, value: any) => {
    onUpdate(path, { ...property, [field]: value });
  };

  const isContainer = property.data_type === 'object';

  return (
    <Card 
      variant="outlined" 
      sx={{ 
        p: 2, 
        border: `1px solid ${theme.palette.divider}`,
        transition: 'all 0.2s',
        '&:hover': {
          borderColor: theme.palette.primary.light,
          boxShadow: `0 2px 8px ${alpha(theme.palette.primary.main, 0.1)}`
        }
      }}
    >
      <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', minWidth: 24 }}>
          {isContainer ? (
            <IconButton size="small" onClick={() => setExpanded(!expanded)}>
              {expanded ? <KeyboardArrowDownIcon /> : <KeyboardArrowRightIcon />}
            </IconButton>
          ) : (
            <Box sx={{ width: 28 }} />
          )}
        </Box>

        <TextField
          size="small"
          label="Field Name"
          value={property.name}
          onChange={(e) => handleChange('name', e.target.value)}
          sx={{ flex: 1 }}
        />

        <TextField
          size="small"
          label="Label"
          value={property.label || ''}
          onChange={(e) => handleChange('label', e.target.value)}
          sx={{ flex: 1 }}
        />

        <TextField
          select
          size="small"
          label="Type"
          value={property.data_type}
          onChange={(e) => handleChange('data_type', e.target.value)}
          sx={{ minWidth: 150 }}
        >
          {PROPERTY_TYPES.map((option) => (
            <MenuItem key={option.value} value={option.value}>
              {option.label}
            </MenuItem>
          ))}
        </TextField>

        <Stack direction="row" spacing={0.5}>
          <Tooltip title="Delete Property">
            <IconButton 
              size="small" 
              color="error" 
              onClick={() => onUpdate(path, null)}
            >
              <DeleteOutlineIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Stack>
      </Box>

      {/* Nested Properties */}
      {isContainer && (
        <Collapse in={expanded} timeout="auto" unmountOnExit>
          <Box sx={{ ml: 6, mt: 2, pl: 2, borderLeft: `2px solid ${theme.palette.divider}` }}>
            <Stack spacing={2}>
              {property.properties?.map((child, cIdx) => (
                <PropertyItem
                  key={cIdx}
                  property={child}
                  path={[...path, cIdx]}
                  onUpdate={(childPath, updated) => {
                    const localIdx = childPath[childPath.length - 1];
                    const newProps = [...(property.properties || [])];
                    if (updated === null) {
                      newProps.splice(localIdx, 1);
                    } else {
                      newProps[localIdx] = updated;
                    }
                    handleChange('properties', newProps);
                  }}
                  onAddChild={onAddChild}
                />
              ))}
              <Button
                size="small"
                startIcon={<AddIcon />}
                onClick={() => {
                  const newProps = [...(property.properties || [])];
                  newProps.push({
                    name: `field_${Date.now()}`,
                    label: '',
                    data_type: 'string',
                    nullable: true,
                    required: false
                  });
                  handleChange('properties', newProps);
                }}
                sx={{ alignSelf: 'flex-start', color: theme.palette.text.secondary }}
              >
                Add Nested Field
              </Button>
            </Stack>
          </Box>
        </Collapse>
      )}
    </Card>
  );
};
