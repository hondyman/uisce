import React, { useState } from 'react';
import {
  Typography,
  TextField,
  Button,
  Divider,
  Box,
  Paper,
  IconButton,
  Tooltip,
  Chip,
  useTheme,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import CodeIcon from '@mui/icons-material/Code';
import { Code2, Sparkles, Plus } from 'lucide-react';
import ExpressionInputControl from './ExpressionInputControl';
import UnifiedExpressionBuilderModal from './UnifiedExpressionBuilderModal';

type Props = {
  expressionLibrary: string[];
  onExpressionChange: (index: number, value: string) => void;
  onAddExpression: () => void;
  onRemoveExpression?: (index: number) => void;
};

export const ExpressionsEditor: React.FC<Props> = ({
  expressionLibrary,
  onExpressionChange,
  onAddExpression,
  onRemoveExpression,
}) => {
  const theme = useTheme();
  const [activeModalIndex, setActiveModalIndex] = useState<number | null>(null);

  // Resolve MUI palette colors to hex strings for expression placeholders
  const positiveColor = theme.palette.success.main;
  const negativeColor = theme.palette.error.main;
  const defaultFormula = `=IIF(Fields!Status.Value == "Active", "${positiveColor}", "${negativeColor}")`;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box>
          <Typography variant="subtitle2" fontWeight="700" sx={{ display: 'flex', alignItems: 'center', gap: 1, color: 'text.primary' }}>
            <Code2 size={18} /> Expression Library &amp; Dynamic Formulas
          </Typography>
          <Typography variant="caption" color="text.secondary">
            Define reusable SSRS formulas, conditional formatting expressions, and multi-tenant variables.
          </Typography>
        </Box>
        <Button
          size="small"
          variant="outlined"
          startIcon={<Plus size={14} />}
          onClick={onAddExpression}
          sx={{ textTransform: 'none', fontSize: '0.75rem', fontWeight: 700, borderRadius: 1.5 }}
        >
          Add Expression
        </Button>
      </Box>

      {expressionLibrary.length === 0 ? (
        <Paper sx={{ p: 2.5, textAlign: 'center', bgcolor: 'action.hover', border: `1px dashed ${theme.palette.divider}`, borderRadius: 2 }}>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
            No expressions in library. Click &quot;Add Expression&quot; to build dynamic report formulas.
          </Typography>
          <Button
            size="small"
            variant="contained"
            startIcon={<Plus size={14} />}
            onClick={onAddExpression}
            sx={{ bgcolor: 'primary.main', color: 'primary.contrastText', textTransform: 'none', fontSize: '0.75rem', fontWeight: 700, '&:hover': { bgcolor: 'primary.dark' } }}
          >
            Create First Expression
          </Button>
        </Paper>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          {expressionLibrary.map((expression, index) => (
            <Paper
              key={`expression_${index}`}
              sx={{
                p: 2,
                bgcolor: 'action.hover',
                border: `1px solid ${theme.palette.divider}`,
                borderRadius: 2,
                display: 'flex',
                alignItems: 'flex-start',
                gap: 1.5,
              }}
            >
              <Box sx={{ flexGrow: 1 }}>
                <ExpressionInputControl
                  label={`Expression ${index + 1}`}
                  property={{ isExpression: true, value: expression, formula: expression || defaultFormula }}
                  defaultFormula={defaultFormula}
                  onChange={(prop) =>
                    onExpressionChange(index, prop.isExpression ? prop.formula || '' : String(prop.value || ''))
                  }
                  renderStaticControl={(val, setVal) => (
                    <TextField
                      fullWidth
                      size="small"
                      label={`Expression ${index + 1}`}
                      value={String(val ?? '')}
                      onChange={(e) => setVal(e.target.value)}
                    />
                  )}
                />
              </Box>

              <Tooltip title="Open Unified Expression Builder Modal">
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => setActiveModalIndex(index)}
                  startIcon={<Code2 size={13} />}
                  sx={{ textTransform: 'none', fontSize: '0.72rem', fontWeight: 700, mt: 3, borderRadius: 1.5 }}
                >
                  Builder (fx)
                </Button>
              </Tooltip>

              {onRemoveExpression && (
                <Tooltip title="Remove Expression">
                  <IconButton
                    size="small"
                    onClick={() => onRemoveExpression(index)}
                    sx={{ color: 'text.secondary', mt: 3, '&:hover': { color: 'error.main' } }}
                  >
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              )}
            </Paper>
          ))}
        </Box>
      )}

      {/* Direct Modal Launcher if triggered */}
      {activeModalIndex !== null && (
        <UnifiedExpressionBuilderModal
          open={activeModalIndex !== null}
          onClose={() => setActiveModalIndex(null)}
          title={`Expression ${activeModalIndex + 1} Builder`}
          label={`Expression ${activeModalIndex + 1}`}
          initialFormula={expressionLibrary[activeModalIndex] || defaultFormula}
          onApply={(newFormula) => {
            onExpressionChange(activeModalIndex, newFormula);
            setActiveModalIndex(null);
          }}
        />
      )}
    </Box>
  );
};

export default ExpressionsEditor;
