import React, { useEffect, useState } from 'react';
import { Box, Typography, Alert } from '@mui/material';
import { ComponentDefinition } from '../../types/pageStudio';
import { evaluateExpressionTextWasm } from '../../rules/wasmRuntime';

interface TextWidgetProps {
  component: ComponentDefinition;
  /** BO key for the record context an expression evaluates against, when the page has one bound. */
  boName?: string;
}

/**
 * Renders a Text component's content: either the literal `props.text` an
 * author typed, or - when `props.contentMode === 'expression'` - the
 * result of evaluating `props.textExpression` (authored via the same
 * centralized ExpressionEditorField/ASL engine as the Rule Builder)
 * against a sample context. There is no live record context on the
 * canvas yet (Page Studio has no "current row" concept outside a bound
 * Table/Form), so expression preview here always runs against
 * SAMPLE_CONTEXT - the same placeholder-data convention the Rule
 * Builder's own preview panel uses - and is clearly labeled as such
 * rather than presented as live data.
 */
const SAMPLE_CONTEXT = { total: 150, status: 'pending', is_gift: false, TargetQuantity: 100 };

const TextWidget: React.FC<TextWidgetProps> = ({ component }) => {
  const mode = (component.props?.contentMode as string) || 'literal';
  const literalText = (component.props?.text as string) || '';
  const expression = (component.props?.textExpression as string) || '';

  const [evaluated, setEvaluated] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (mode !== 'expression' || !expression.trim()) {
      setEvaluated(null);
      setError(null);
      return;
    }
    let cancelled = false;
    evaluateExpressionTextWasm(expression, SAMPLE_CONTEXT)
      .then(({ result }) => { if (!cancelled) { setEvaluated(String(result)); setError(null); } })
      .catch((err) => { if (!cancelled) { setError(err instanceof Error ? err.message : String(err)); setEvaluated(null); } });
    return () => { cancelled = true; };
  }, [mode, expression]);

  const variant = (component.props?.variant as string) || 'body1';

  if (mode === 'expression') {
    if (error) return <Alert severity="warning" sx={{ fontSize: 12 }}>{error}</Alert>;
    return (
      <Box>
        <Typography variant={variant as any}>{evaluated ?? '…'}</Typography>
        <Typography variant="caption" color="text.secondary">Preview uses sample data - live values apply on the published page.</Typography>
      </Box>
    );
  }

  return <Typography variant={variant as any}>{literalText || <em>Empty text - set content in the Style panel</em>}</Typography>;
};

export default TextWidget;
