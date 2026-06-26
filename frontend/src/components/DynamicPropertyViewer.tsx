import React from 'react';
import { Box, Typography } from '@mui/material';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import CancelOutlinedIcon from '@mui/icons-material/CancelOutlined';

export interface PropertyMetadata {
  name: string;
  label: string;
  order: number;
  data_type: string;
  input_type: string;
}

export interface DynamicPropertyViewerProps {
  properties: PropertyMetadata[];
  values: Record<string, any>;
}

const DynamicPropertyViewer: React.FC<DynamicPropertyViewerProps> = ({
  properties,
  values,
}) => {
  if (!properties || properties.length === 0 || !values) {
    return null;
  }

  // Sort properties by order
  const sortedProperties = [...properties].sort((a, b) => (a.order ?? 999) - (b.order ?? 999));

  // Filter to only show properties that have values (or show "False" for booleans explicitly if desired, but typically we show what is set)
  // For viewing, usually we want to see everything that is set.
  const propsWithValues = sortedProperties.filter(p => values[p.name] !== undefined && values[p.name] !== null && values[p.name] !== '');

  if (propsWithValues.length === 0) {
      return <Typography variant="body2" color="text.secondary">No property values set.</Typography>;
  }

  return (
    <div className="metadata-grid">
      {propsWithValues.map((prop) => {
        const val = values[prop.name];
        let displayVal: React.ReactNode = String(val);

        if (prop.data_type === 'boolean' || prop.input_type === 'checkbox') {
             displayVal = val ? (
                 <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                     <CheckCircleOutlineIcon fontSize="small" color="success" /> <Typography variant="body2">Yes</Typography>
                 </Box>
             ) : (
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                    <CancelOutlinedIcon fontSize="small" color="disabled" /> <Typography variant="body2">No</Typography>
                </Box>
             );
        } else if (prop.name === 'sql' || prop.input_type === 'code') {
            displayVal = (
                <pre style={{ margin: 0, padding: '4px', backgroundColor: '#f5f5f5', borderRadius: '4px', fontSize: '0.8rem', overflowX: 'auto' }}>
                    {String(val)}
                </pre>
            );
        } else if (typeof val === 'object') {
            displayVal = <pre>{JSON.stringify(val, null, 2)}</pre>;
        }

        return (
          <div className="metadata-item" key={prop.name}>
            <span className="metadata-label">{prop.label || prop.name}:</span>
            <span className="metadata-value">{displayVal}</span>
          </div>
        );
      })}
    </div>
  );
};

export default DynamicPropertyViewer;
