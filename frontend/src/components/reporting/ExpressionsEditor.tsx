import type { FC } from 'react';
import { Typography, Button, Divider, Box, IconButton } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import ExpressionEditorField from '../ExpressionBuilder/ExpressionEditorField';

type Props = {
  expressionLibrary: string[];
  onExpressionChange: (index: number, value: string) => void;
  onAddExpression: () => void;
  onRemoveExpression: (index: number) => void;
  boName?: string;
};

const ExpressionsEditor: FC<Props> = ({ expressionLibrary, onExpressionChange, onAddExpression, onRemoveExpression, boName }) => {
  return (
    <>
      <Box sx={{ mt: 1, mb: 1 }}>
        <Divider />
      </Box>
      <Typography variant="subtitle1">Expression Library</Typography>
      {expressionLibrary.map((expression, index) => (
        <Box key={`expression_${index}`} sx={{ display: 'flex', alignItems: 'flex-start', gap: 1, mb: 1.5 }}>
          <Box sx={{ flex: 1 }}>
            <ExpressionEditorField
              label={`Expression ${index + 1}`}
              value={expression}
              onChange={(v) => onExpressionChange(index, v)}
              boName={boName}
              minHeight={80}
            />
          </Box>
          <IconButton size="small" onClick={() => onRemoveExpression(index)} sx={{ mt: 0.5 }}>
            <DeleteIcon fontSize="small" />
          </IconButton>
        </Box>
      ))}
      <Button size="small" startIcon={<AddIcon sx={{ fontSize: 14 }} />} onClick={onAddExpression}>Add Expression</Button>
    </>
  );
};

export default ExpressionsEditor;
