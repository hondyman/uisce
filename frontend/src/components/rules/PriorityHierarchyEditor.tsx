import { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Slider,
  Typography,
  Button,
  Stack,
  Chip,
  IconButton,
  Collapse,
  Alert,
  Paper,
  Tooltip,
} from '@mui/material';
import {
  DragIndicator as DragIcon,
  Delete as DeleteIcon,
  Visibility as PreviewIcon,
  ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

interface PriorityStep {
  id: string;
  priority: number;
  condition: {
    semanticTerm: string;
    operator: string;
    value: string;
  };
  action: {
    useField: string;
    confidence: number;
  };
  description: string;
}

interface SemanticTerm {
  id: string;
  name: string;
  dataType: string;
  governanceStatus: string;
}

interface PriorityHierarchyEditorProps {
  step: PriorityStep;
  isExpanded: boolean;
  onToggleExpand: () => void;
  onUpdate: (updates: Partial<PriorityStep>) => void;
  onDelete: () => void;
  terms?: SemanticTerm[];
  readOnly?: boolean;
  simulationResult?: any;
}

/**
 * PriorityHierarchyEditor Component (Material-UI)
 * Individual priority rule editor
 */
export const PriorityHierarchyEditor = ({
  step,
  isExpanded,
  onToggleExpand,
  onUpdate,
  onDelete,
  terms = [],
  readOnly = false,
  simulationResult,
}: PriorityHierarchyEditorProps) => {
  const [showConfidenceTooltip, setShowConfidenceTooltip] = useState(false);

  // Drag setup
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: step.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const selectedTerm = terms.find((t) => t.id === step.condition.semanticTerm);

  const operators =
    selectedTerm?.dataType === 'STRING'
      ? ['equals', 'contains', 'starts_with', 'in_list']
      : selectedTerm?.dataType === 'BOOLEAN'
        ? ['equals']
        : selectedTerm?.dataType === 'DATE'
          ? ['equals', 'after', 'before', 'between']
          : ['equals', 'greater_than', 'less_than', 'between'];

  const operatorLabels: Record<string, string> = {
    equals: 'equals',
    contains: 'contains',
    starts_with: 'starts with',
    in_list: 'in list',
    after: 'is after',
    before: 'is before',
    between: 'is between',
    greater_than: 'is greater than',
    less_than: 'is less than',
  };

  const getConfidenceColor = (value: number) => {
    if (value >= 90) return 'success';
    if (value >= 75) return 'primary';
    if (value >= 60) return 'warning';
    return 'error';
  };

  const getConfidenceLabel = (value: number) => {
    if (value >= 90) return 'Very High';
    if (value >= 75) return 'High';
    if (value >= 60) return 'Medium';
    return 'Low';
  };

  return (
    <Card
      ref={setNodeRef}
      style={style}
      sx={{
        border: isExpanded ? 2 : 1,
        borderColor: isExpanded ? 'primary.main' : 'divider',
        backgroundColor: isExpanded ? 'primary.lighter' : 'background.paper',
        boxShadow: isDragging ? 2 : 0,
        transition: 'all 200ms',
      }}
    >
      {/* Header */}
      <CardHeader
        onClick={onToggleExpand}
        sx={{
          cursor: 'pointer',
          p: 1.5,
          pb: isExpanded ? 1 : 1.5,
          '&:hover': { backgroundColor: 'action.hover' },
          display: 'flex',
          alignItems: 'center',
          gap: 1,
        }}
        avatar={
          !readOnly && (
            <Box
              {...attributes}
              {...listeners}
              sx={{
                cursor: 'grab',
                '&:active': { cursor: 'grabbing' },
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <DragIcon fontSize="small" />
            </Box>
          )
        }
        title={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
            <Chip
              label={step.priority}
              size="small"
              color="primary"
              variant="filled"
            />
            <Typography variant="body2" fontWeight="600">
              IF {selectedTerm?.name || 'Select a term'}
            </Typography>
            {selectedTerm && (
              <Typography variant="body2" color="textSecondary">
                {operatorLabels[step.condition.operator]} {step.condition.value}
              </Typography>
            )}
          </Box>
        }
        action={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {selectedTerm && (
              <Tooltip title={`${getConfidenceLabel(step.action.confidence)} confidence`}>
                <Chip
                  label={`${step.action.confidence}%`}
                  size="small"
                  color={getConfidenceColor(step.action.confidence)}
                  variant="outlined"
                />
              </Tooltip>
            )}
            {simulationResult && (
              <Chip
                label={simulationResult.won ? '✓ Winning' : `${simulationResult.matches} matches`}
                size="small"
                color={simulationResult.won ? 'success' : 'default'}
              />
            )}
            <IconButton size="small" onClick={onToggleExpand}>
              <ExpandMoreIcon
                sx={{
                  transform: isExpanded ? 'rotate(180deg)' : 'rotate(0deg)',
                  transition: 'transform 200ms',
                }}
              />
            </IconButton>
          </Box>
        }
      />

      {/* Expanded Content */}
      <Collapse in={isExpanded}>
        <CardContent sx={{ pt: 0, space: 2 }}>
          <Stack spacing={2}>
            {/* Semantic Term */}
            <FormControl fullWidth size="small">
              <InputLabel>Semantic Term</InputLabel>
              <Select
                value={step.condition.semanticTerm}
                label="Semantic Term"
                onChange={(e) =>
                  onUpdate({
                    condition: {
                      ...step.condition,
                      semanticTerm: e.target.value,
                      operator: 'equals',
                      value: '',
                    },
                  })
                }
                disabled={readOnly}
              >
                <MenuItem value="">
                  <em>Select a semantic term</em>
                </MenuItem>
                {terms.map((term) => (
                  <MenuItem key={term.id} value={term.id}>
                    {term.name} ({term.dataType})
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            {/* Operator */}
            {selectedTerm && (
              <FormControl fullWidth size="small">
                <InputLabel>Condition</InputLabel>
                <Select
                  value={step.condition.operator}
                  label="Condition"
                  onChange={(e) =>
                    onUpdate({
                      condition: {
                        ...step.condition,
                        operator: e.target.value,
                      },
                    })
                  }
                  disabled={readOnly}
                >
                  {operators.map((op) => (
                    <MenuItem key={op} value={op}>
                      {operatorLabels[op]}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            )}

            {/* Value Input */}
            {selectedTerm && (
              <FormControl fullWidth size="small">
                {selectedTerm.dataType === 'BOOLEAN' ? (
                  <>
                    <InputLabel>Value</InputLabel>
                    <Select
                      value={step.condition.value}
                      label="Value"
                      onChange={(e) =>
                        onUpdate({
                          condition: {
                            ...step.condition,
                            value: e.target.value,
                          },
                        })
                      }
                      disabled={readOnly}
                    >
                      <MenuItem value="">
                        <em>Select value</em>
                      </MenuItem>
                      <MenuItem value="true">True</MenuItem>
                      <MenuItem value="false">False</MenuItem>
                    </Select>
                  </>
                ) : (
                  <TextField
                    size="small"
                    type={selectedTerm.dataType === 'DATE' ? 'date' : 'text'}
                    label="Value"
                    value={step.condition.value}
                    onChange={(e) =>
                      onUpdate({
                        condition: {
                          ...step.condition,
                          value: e.target.value,
                        },
                      })
                    }
                    placeholder={`Enter ${selectedTerm.dataType.toLowerCase()} value`}
                    disabled={readOnly}
                    InputLabelProps={selectedTerm.dataType === 'DATE' ? { shrink: true } : undefined}
                  />
                )}
              </FormControl>
            )}

            {/* Confidence Slider */}
            {selectedTerm && (
              <Box>
                <Box
                  sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    mb: 1,
                  }}
                >
                  <Typography variant="subtitle2" fontWeight="600">
                    Trust/Confidence: <span style={{ color: '#1976d2' }}>{step.action.confidence}%</span>
                  </Typography>
                  <Chip
                    label={getConfidenceLabel(step.action.confidence)}
                    size="small"
                    color={getConfidenceColor(step.action.confidence)}
                  />
                </Box>
                <Slider
                  value={step.action.confidence}
                  onChange={(e, newValue) =>
                    onUpdate({
                      action: {
                        ...step.action,
                        confidence: newValue as number,
                      },
                    })
                  }
                  step={5}
                  min={0}
                  max={100}
                  marks
                  disabled={readOnly}
                  onMouseEnter={() => setShowConfidenceTooltip(true)}
                  onMouseLeave={() => setShowConfidenceTooltip(false)}
                  valueLabelDisplay="auto"
                />
                {showConfidenceTooltip && (
                  <Paper sx={{ mt: 1.5, p: 1.5, backgroundColor: 'action.hover' }}>
                    <Typography variant="caption" display="block" sx={{ fontWeight: 600, mb: 0.5 }}>
                      What this means:
                    </Typography>
                    <Typography variant="caption" display="block" color="textSecondary">
                      How much to trust this data source when it says the value is "{step.condition.value}"
                    </Typography>
                    <Typography variant="caption" display="block" color="textSecondary" sx={{ mt: 0.5 }}>
                      {step.action.confidence >= 90
                        ? 'Primary source - use unless another source has higher confidence'
                        : step.action.confidence >= 70
                          ? 'Well-trusted source'
                          : 'Fallback source - use if higher confidence sources conflict'}
                    </Typography>
                  </Paper>
                )}
              </Box>
            )}

            {/* Description */}
            {selectedTerm && (
              <TextField
                fullWidth
                multiline
                rows={2}
                size="small"
                label="Rule Description (optional)"
                value={step.description}
                onChange={(e) =>
                  onUpdate({
                    description: e.target.value,
                  })
                }
                placeholder="Why is this rule important? What business logic does it implement?"
                disabled={readOnly}
              />
            )}

            {/* Action Summary */}
            {selectedTerm && (
              <Alert severity="info">
                <Typography variant="body2">
                  <strong>THEN:</strong> USE {step.action.useField} with{' '}
                  <strong>{step.action.confidence}% confidence</strong>
                </Typography>
              </Alert>
            )}

            {/* Actions */}
            <Box sx={{ display: 'flex', gap: 1, pt: 1, borderTop: 1, borderColor: 'divider' }}>
              <Button
                size="small"
                startIcon={<PreviewIcon />}
                variant="outlined"
                onClick={() => console.log('Preview rule results')}
              >
                Preview
              </Button>
              {!readOnly && (
                <Button
                  size="small"
                  startIcon={<DeleteIcon />}
                  variant="outlined"
                  color="error"
                  onClick={onDelete}
                >
                  Delete
                </Button>
              )}
            </Box>
          </Stack>
        </CardContent>
      </Collapse>
    </Card>
  );
};

export default PriorityHierarchyEditor;
