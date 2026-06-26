import type { FC } from 'react';
import { Typography, TextField, Button, Divider, Box } from '@mui/material';
import { Plus } from 'lucide-react';

type Props = {
  expressionLibrary: string[];
  onExpressionChange: (index: number, value: string) => void;
  onAddExpression: () => void;
};

const ExpressionsEditor: FC<Props> = ({ expressionLibrary, onExpressionChange, onAddExpression }) => {
  return (
    <>
      <Box sx={{ mt: 1, mb: 1 }}>
        <Divider />
      </Box>
      <Typography variant="subtitle1">Expression Library</Typography>
      {expressionLibrary.map((expression, index) => (
        <TextField
          key={`expression_${index}`}
          fullWidth
          size="small"
          multiline
          minRows={2}
          label={`Expression ${index + 1}`}
          sx={{ mb: 1.5 }}
          value={expression}
          onChange={(e) => onExpressionChange(index, e.target.value)}
        />
      ))}
      <Button size="small" startIcon={<Plus size={14} />} onClick={onAddExpression}>Add Expression</Button>
    </>
  );
};

export default ExpressionsEditor;
