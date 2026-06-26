import React from 'react';
import { Autocomplete, TextField, Chip } from '@mui/material';

interface ArrayChipsProps {
  value: string[];
  onChange: (values: string[]) => void;
  label?: string;
  placeholder?: string;
  dataTestId?: string;
  // Optional map to display friendly labels for underlying stored ids
  displayMap?: Map<string, string> | null;
}

const ArrayChips: React.FC<ArrayChipsProps> = ({ value = [], onChange, label, placeholder, dataTestId, displayMap = null }) => {
  return (
    <Autocomplete
      multiple
      freeSolo
      options={[]}
      value={value}
      onChange={(e, newValue) => onChange(newValue as string[])}
        renderTags={(valueArray, getTagProps) => (
        valueArray.map((option, index) => (
          <Chip size="small" variant="outlined" label={displayMap?.get(String(option)) || option} {...getTagProps({ index })} key={index} />
        ))
      )}
      renderInput={(params) => (
        <TextField
          {...params}
          label={label}
          placeholder={placeholder}
          inputProps={{ ...params.inputProps, 'data-testid': dataTestId }}
        />
      )}
    />
  );
};

export default ArrayChips;
