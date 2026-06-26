// frontend/src/components/validation/HierarchyValidationBuilder.tsx

import React, { useState, useEffect, useCallback } from 'react';

// MUI Imports
import {
  Select,
  TextField,
  Button,
  Card,
  CardContent,
  CardHeader,
  Stack,
  CircularProgress,
  List,
  ListItem,
  ListItemText,
  ListItemAvatar,
  Alert,
  AlertTitle,
  Tooltip,
  MenuItem,
  FormControl,
  InputLabel,
  Grid,
  Typography,
  Chip,
  Snackbar,
  Paper,
  SelectChangeEvent,
} from '@mui/material';
import { SimpleTreeView as TreeView, TreeItem } from '@mui/x-tree-view';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';

import {
  GitBranch, Layers, Filter as _Filter, TrendingUp as _TrendingUp, HelpCircle as _HelpCircle, Play, Code, BarChart2, BrainCircuit, Link, MessageSquareText,
} from 'lucide-react';
import { devError, devDebug } from '../../utils/devLogger';

// ============================================================================
// TYPES
// ============================================================================

interface HierarchyField {
  key: string; // Unique key for Antd Tree, e.g., "order.total"
  title: string; // Display name, e.g., "Total"
  fullPath: string; // The full dot-notation path, e.g., "order.total"
  type: 'string' | 'number' | 'date' | 'boolean' | 'object' | 'array';
  children?: HierarchyField[];
  profile?: { // Data profiling information
    min?: any;
    max?: any;
    avg?: number;
    null_percentage?: number;
    distinct_values?: number;
  };
}

interface HierarchyRule {
  id?: string;
  name: string;
  description?: string;
  ruleType: 'parent_only' | 'sub_only' | 'sub_parent' | 'aggregate';
  parentFullPath?: string;
  subFullPath?: string;
  operator?: string;
  value?: any;
  aggregationType?: 'sum' | 'count' | 'avg' | 'min' | 'max';
  severity?: 'error' | 'warning' | 'info';
  valueFromParentField?: boolean;
}

interface AISuggestedRelationship {
  targetEntityId: string;
  targetEntityName: string;
  reason: string;
  sharedTerms: string[];
  confidenceScore: number;
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const HierarchyValidationBuilder: React.FC<{
  entity: string;
  onRuleSaved?: (rule: HierarchyRule) => void;
}> = ({ entity, onRuleSaved }) => {

  // Form state management
  const [formState, setFormState] = useState<Partial<HierarchyRule>>({
    ruleType: 'sub_parent',
    severity: 'error',
  });
  const [ruleType, setRuleType] = useState<string>('sub_parent');
  const [selectedPaths, setSelectedPaths] = useState<{
    parent?: string;
    sub?: string;
  }>({});

  // Snackbar state for notifications
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' | 'info' | 'warning' }>({
    open: false,
    message: '',
    severity: 'info',
  });

  const handleFormChange = (event: React.ChangeEvent<HTMLInputElement | { name?: string; value: unknown }> | SelectChangeEvent<any>) => {
    setFormState(prev => ({ ...prev, [event.target.name!]: event.target.value }));
  };

  const [dynamicEntityHierarchy, setDynamicEntityHierarchy] = useState<HierarchyField[]>([]);
  const [loadingSchema, setLoadingSchema] = useState<boolean>(true);
  const [testData, setTestData] = useState<string>('');
  const [testResults, setTestResults] = useState<any>(null);
  const [testingLoading, setTestingLoading] = useState<boolean>(false);
  const [aiSuggestions, setAiSuggestions] = useState<AISuggestedRelationship[]>([]);
  const [_loadingAiSuggestions, _setLoadingAiSuggestions] = useState<boolean>(false);
  const [naturalLanguageRule, setNaturalLanguageRule] = useState<string>('');
  const [generatingRule, setGeneratingRule] = useState<boolean>(false);

  // Fetch dynamic schema
  useEffect(() => {
    const fetchSchema = async () => {
      setLoadingSchema(true);
      try {
        // Assuming an API endpoint /api/schema/:entity exists
        const response = await fetch(`/api/schema/${entity}?include_profiling=true`);
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
  const _schemaData = await response.json();
        // Transform schemaData if necessary to match HierarchyField structure
        // For now, let's use a mock structure if the API isn't ready
        const mockSchema: HierarchyField[] = [
          {
            key: 'order',
            title: 'Order',
            fullPath: 'order',
            type: 'object',
            children: [
              { key: 'order.id', title: 'ID', fullPath: 'order.id', type: 'string' },
              {
                key: 'order.total', title: 'Total', fullPath: 'order.total', type: 'number',
                profile: { min: 10, max: 15000, avg: 2540.50, null_percentage: 0 }
              },
              {
                key: 'order.customer_id', title: 'Customer ID', fullPath: 'order.customer_id', type: 'string'
              },
              {
                key: 'order.line_items',
                title: 'Line Items',
                fullPath: 'order.line_items',
                type: 'array',
                children: [
                  { key: 'order.line_items.id', title: 'ID', fullPath: 'order.line_items.id', type: 'string' },
                  {
                    key: 'order.line_items.qty', title: 'Quantity', fullPath: 'order.line_items.qty', type: 'number',
                    profile: { min: 1, max: 200, avg: 3.1, null_percentage: 0 }
                  },
                  {
                    key: 'order.line_items.price', title: 'Price', fullPath: 'order.line_items.price', type: 'number',
                    profile: { min: 5.99, max: 999.99, avg: 89.50, null_percentage: 0 }
                  },
                  {
                    key: 'order.line_items.product',
                    title: 'Product',
                    fullPath: 'order.line_items.product',
                    type: 'object',
                    children: [
                      { key: 'order.line_items.product.id', title: 'ID', fullPath: 'order.line_items.product.id', type: 'string' },
                      { key: 'order.line_items.product.category', title: 'Category', fullPath: 'order.line_items.product.category', type: 'string' },
                    ],
                  },
                ],
              },
            ],
          },
        ];
        setDynamicEntityHierarchy(mockSchema); // Use mock for now, replace with schemaData
      } catch (e) {
          devError('Failed to load entity schema:', e);
          setSnackbar({ open: true, message: `Failed to load entity schema: ${e instanceof Error ? e.message : String(e)}`, severity: 'error' }); // Corrected error message
      } finally {
        setLoadingSchema(false);
      }
    };

    fetchSchema();
  }, [entity]);

  // Fetch AI suggestions when a relevant field is selected
  useEffect(() => {
    const fetchAiSuggestions = async (entityId: string) => {
      if (!entityId) return;
  _setLoadingAiSuggestions(true);
      try {
        const response = await fetch(`/api/ai/discover-relationships/${entityId}`);
        if (response.ok) {
          const suggestions = await response.json();
          setAiSuggestions(suggestions);
        }
      } catch (e: any) { // Explicitly type 'e' as 'any' for broader error handling
        devError("Failed to fetch AI suggestions", e);
      } finally {
        _setLoadingAiSuggestions(false);
      }
    };

    // Trigger when a field that looks like an ID is selected
    const triggerField = selectedPaths.parent || selectedPaths.sub;
    if (triggerField && triggerField.toLowerCase().includes('id')) {
      fetchAiSuggestions(triggerField);
    }
  }, [selectedPaths]);

  const handleSelectPath = (nodeId: string, context: 'parent' | 'sub') => {
    // In MUI TreeView, nodeId is the key/fullPath
    if (context === 'parent') {
      setSelectedPaths(prev => ({ ...prev, parent: nodeId }));
      setFormState(prev => ({ ...prev, parentFullPath: nodeId }));
    } else {
      setSelectedPaths(prev => ({ ...prev, sub: nodeId }));
      setFormState(prev => ({ ...prev, subFullPath: nodeId }));
    }
  };


  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => { // Explicitly type event
    event.preventDefault();

    // Validate required fields before submitting
    if (!formState.name || !formState.ruleType) {
      setSnackbar({ open: true, message: 'Rule Name and Type are required.', severity: 'warning' });
      return;
    }

    // Helper to get sub-entity path and field name
    const getSubPathAndField = (path?: string) => {
        if (!path) return { subEntityPath: undefined, subFieldName: undefined };
        const segments = path.split('.');
        return {
            subEntityPath: segments.slice(0, segments.length - 1).join('.'),
            subFieldName: segments[segments.length - 1],
        };
    };

  const { subEntityPath: _subEntityPath, subFieldName: _subFieldName } = getSubPathAndField(formState.subFullPath); // Use formState.subFullPath
    
    const rule: HierarchyRule = {
      ...(formState as HierarchyRule), // Cast to HierarchyRule, assuming validation passed
      parentFullPath: selectedPaths.parent, // Use state for paths
      subFullPath: selectedPaths.sub, // Use state for paths
    };

    if (onRuleSaved) {
      onRuleSaved(rule as HierarchyRule); // Cast to HierarchyRule
    }

  devDebug('Rule to save:', rule);
    setFormState({ name: '', description: '', ruleType: 'sub_parent', severity: 'error', parentFullPath: '', subFullPath: '', operator: '', value: undefined, aggregationType: undefined });
    setSelectedPaths({});
    setTestResults(null); // Clear test results on new rule save
  };
  const generateRulePreviewText = useCallback(() => {
  const { name, ruleType, operator, value, aggregationType, valueFromParentField: _valueFromParentField } = formState;
    const parentPath = selectedPaths.parent; // Use selectedPaths for consistency
    const subPath = selectedPaths.sub;

    let preview = `Rule: "${name || 'Unnamed Rule'}"\n`;

    const getSubPathAndField = (path?: string) => {
        if (!path) return { subEntityPath: undefined, subFieldName: undefined };
        const segments = path.split('.');
        return {
            subEntityPath: segments.slice(0, segments.length - 1).join('.'),
            subFieldName: segments[segments.length - 1],
        };
    };
    const { subEntityPath, subFieldName } = getSubPathAndField(subPath);

    switch (ruleType) {
      case 'parent_only':
        if (parentPath && operator && value !== undefined) {
          preview += `The parent field '${parentPath}' must be ${operatorToText(operator)} ${value}.`;
        }
        break;
      case 'sub_only':
        if (subPath && operator && value !== undefined) {
          preview += `Each '${subFieldName}' in '${subEntityPath}' must be ${operatorToText(operator)} ${value}.`;
        }
        break;
      case 'sub_parent': // Corrected from 'parent_sub'
        if (subPath && parentPath && operator && selectedPaths.parent) { // Use selectedPaths.parent for comparison
          preview += `Each '${subFieldName}' in '${subEntityPath}' must be ${operatorToText(operator)} the parent field '${parentPath}'.`;
        }
        break;
      case 'aggregate':
        if (parentPath && subPath && aggregationType && operator) {
          preview += `The parent field '${parentPath}' must be ${operatorToText(operator)} the ${aggregationType} of '${subFieldName}' in '${subEntityPath}'.`;
        }
        break;
      default:
        preview += 'Select a rule type and fields to see a preview.';
    }
    return preview.trim(); // Trim to remove leading/trailing newlines if no rule details
  }, [formState, ruleType, selectedPaths]);

  const operatorToText = (op: string) => {
    switch (op) {
      case 'equals': return 'equal to';
      case 'not_equals': return 'not equal to';
      case 'greater_than': return 'greater than';
      case 'less_than': return 'less than';
      case 'greater_equal': return 'greater than or equal to';
      case 'less_equal': return 'less than or equal to';
      case 'equals_aggregate': return 'equal to';
      default: return op;
    }
  };

  const handleTestRule = async () => {
    setTestingLoading(true);
    setTestResults(null);
    try {
      const ruleDefinition = formState; // Use formState directly
      const payload = {
        rule: {
          ...ruleDefinition,
          parentFullPath: selectedPaths.parent,
          subFullPath: selectedPaths.sub,
        },
        data: JSON.parse(testData),
      };
      const response = await fetch('/api/rules/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (!response.ok) {
        const errorBody = await response.json();
        throw new Error(errorBody.message || `HTTP error! status: ${response.status}`);
      }
      const result = await response.json();
      setTestResults(result);
    } catch (e: any) {
      setTestResults({ valid: false, errors: [{ message: `Test failed: ${e.message || e}` }] }); // Ensure errors is an array for consistency
      setSnackbar({ open: true, message: `Failed to test rule: ${e.message || e}`, severity: 'error' });
    } finally {
      setTestingLoading(false);
    }
  };

  const renderConditionFields = () => {
    switch (ruleType) {
      case 'parent_only': // parent_only and sub_only use a literal value
      case 'sub_only':
        return (
          <Stack direction="row" spacing={2} alignItems="flex-start"> {/* Align items to start for better vertical alignment */}
            <FormControl sx={{ minWidth: 150 }}>
              <InputLabel id="operator-label">Operator</InputLabel>
              <Select name="operator" value={formState.operator || ''} onChange={handleFormChange} label="Operator">
                <MenuItem value="equals">=</MenuItem>
                <MenuItem value="not_equals">≠</MenuItem>
                <MenuItem value="greater_than">&gt;</MenuItem>
                <MenuItem value="less_than">&lt;</MenuItem>
                <MenuItem value="greater_equal">≥</MenuItem>
                <MenuItem value="less_equal">≤</MenuItem>
              </Select>
            </FormControl>
            <TextField
              name="value"
              label="Value"
              type="number"
              value={formState.value || ''}
              onChange={handleFormChange}
            />
          </Stack>
        );
      case 'sub_parent': // sub_parent compares a sub-entity field against a parent field
        return (
          <Stack direction="row" spacing={2} alignItems="flex-start">
            <FormControl sx={{ minWidth: 150 }}>
              <InputLabel id="operator-label-sub-parent">Operator</InputLabel>
              <Select name="operator" value={formState.operator || ''} onChange={handleFormChange} label="Operator">
                <MenuItem value="equals">=</MenuItem>
                <MenuItem value="not_equals">≠</MenuItem>
                <MenuItem value="greater_than">&gt;</MenuItem>
                <MenuItem value="less_than">&lt;</MenuItem>
                <MenuItem value="greater_equal">≥</MenuItem>
                <MenuItem value="less_equal">≤</MenuItem>
              </Select>
            </FormControl>
            <TextField
              name="valueFromParentField" // Add name for formState
              label="Compare to Parent Field" // Corrected label
              placeholder="Parent Field (auto-filled)"
              disabled
              value={selectedPaths.parent || ''}
              fullWidth
            />
          </Stack>
        );
      case 'aggregate':
        return (
          <Stack direction="row" spacing={2} alignItems="flex-start"> // Use Stack for layout
            <FormControl sx={{ minWidth: 120 }}>
              <InputLabel id="aggregation-label">Aggregation</InputLabel>
              <Select name="aggregationType" value={formState.aggregationType || 'sum'} onChange={handleFormChange} label="Aggregation">
                <MenuItem value="sum">Sum</MenuItem>
                <MenuItem value="count">Count</MenuItem>
                <MenuItem value="avg">Average</MenuItem>
                <MenuItem value="min">Min</MenuItem>
                <MenuItem value="max">Max</MenuItem>
              </Select>
            </FormControl>
            <FormControl sx={{ minWidth: 120 }}>
              <InputLabel id="operator-label-agg">Operator</InputLabel>
              <Select name="operator" value={formState.operator || 'equals_aggregate'} onChange={handleFormChange} label="Operator">
                <MenuItem value="equals_aggregate">=</MenuItem>
                <MenuItem value="greater_than">&gt;</MenuItem>
                <MenuItem value="less_than">&lt;</MenuItem>
              </Select>
            </FormControl>
            <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>vs. Parent Field</Typography> {/* Added margin-top for alignment */} // Corrected text
          </Stack>
        );
      default:
        return null;
    }
  };

  const handleGenerateRule = async () => {
    if (!naturalLanguageRule.trim()) {
      setSnackbar({ open: true, message: 'Please enter a natural language rule.', severity: 'warning' });
      return;
    }
    setGeneratingRule(true);
    try {
      const response = await fetch('/api/ai/generate-rule', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          naturalLanguageRule: naturalLanguageRule,
          entityContext: entity,
        }),
      });
      if (!response.ok) {
        const errorBody = await response.json();
        throw new Error(errorBody.message || `HTTP error! status: ${response.status}`);
      }
      const { rule } = await response.json();
      setFormState(rule); // Populate form with AI-generated rule
      setRuleType(rule.ruleType); // Update ruleType state for conditional rendering
      setSelectedPaths({ parent: rule.parentFullPath, sub: rule.subFullPath }); // Update selected paths // Corrected to use rule.parentFullPath and rule.subFullPath
      setSnackbar({ open: true, message: 'Rule generated successfully from natural language!', severity: 'success' });
    } catch (e: any) {
      setSnackbar({ open: true, message: `Failed to generate rule: ${e.message || e}`, severity: 'error' });
    } finally {
      setGeneratingRule(false);
    }
  };

  const renderNodeTitle = (node: HierarchyField) => (
    <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ width: '100%' }}>
      <span>{node.title}</span>
      {node.profile && (
        <Tooltip
          title={
            <div className="text-xs">
              <Typography variant="body2"><strong>Profiling Stats</strong></Typography>
              {node.profile.min !== undefined && <Typography variant="caption">Min: {node.profile.min}</Typography>}
              {node.profile.max !== undefined && <Typography variant="caption">Max: {node.profile.max}</Typography>}
              {node.profile.avg !== undefined && <Typography variant="caption">Avg: {node.profile.avg.toFixed(2)}</Typography>}
              {node.profile.null_percentage !== undefined && <Typography variant="caption">Nulls: {node.profile.null_percentage}%</Typography>}
              {node.profile.distinct_values !== undefined && <Typography variant="caption">Distinct: {node.profile.distinct_values}</Typography>}
            </div>
          }
        >
          <BarChart2 size={14} style={{ color: '#1976d2', cursor: 'help', marginLeft: '8px' }} />
        </Tooltip>
      )}
    </Stack>
  );

  const renderTree = (nodes: HierarchyField): React.ReactNode => (
    <TreeItem key={nodes.key} itemId={nodes.key} label={renderNodeTitle(nodes)}>
      {Array.isArray(nodes.children) ? nodes.children.map((node) => renderTree(node)) : null}
    </TreeItem>
  );

  return (
    <Stack spacing={3}>
      <Paper elevation={0} sx={{ p: 2, background: 'linear-gradient(to right, #f3e5f5, #e3f2fd)' }}>
        <Stack direction="row" spacing={2} alignItems="center" mb={1}>
          <GitBranch style={{ color: '#7e57c2' }} size={24} />
          <Typography variant="h5" component="h2">
            Hierarchical Validation Rule Builder
          </Typography>
        </Stack>
        <Typography variant="body2" color="text.secondary">
          Create rules that validate parent entities against their sub-entities (e.g., Order vs Line Items).
        </Typography>
      </Paper>

      <Alert
        severity="info"
        icon={<Layers size={20} />}
      >
        <AlertTitle>Workday-Style Hierarchy Support</AlertTitle>
        You can now validate parent records against their sub-entities. Example: Order total must match the sum of its line item prices.
      </Alert>

      <form onSubmit={handleSubmit}> {/* Use form element for proper submission */} // Corrected form tag
        <Stack spacing={3}>
        <Card>
          <CardHeader title="1. Rule Details" />
          <CardContent>
            <Stack spacing={2}>
              <TextField
                name="name"
                label="Rule Name" // Corrected label
                value={formState.name || ''} // Controlled component
                onChange={handleFormChange}
                placeholder="e.g., Line Item Quantity Check"
                required
                fullWidth
              />
              <TextField
                name="description"
                label="Description"
                value={formState.description || ''} // Controlled component // Corrected label
                onChange={handleFormChange}
                placeholder="Describe what this rule validates"
                multiline
                rows={2}
                fullWidth
              />
            </Stack>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="2. Rule Type" />
          <CardContent>
            <FormControl fullWidth> // Corrected FormControl
              <InputLabel id="rule-type-label">Select Hierarchy Type</InputLabel>
              <Select // Controlled component
                name="ruleType"
                value={ruleType}
                label="Select Hierarchy Type"
                onChange={(e) => {
                  setRuleType(e.target.value);
                  handleFormChange(e);
                }}
              >
                <MenuItem value="parent_only">Parent Only</MenuItem>
                <MenuItem value="sub_only">Sub-Entity Only</MenuItem>
                <MenuItem value="sub_parent">Sub-Entity vs. Parent</MenuItem>
                <MenuItem value="aggregate">Aggregate Sub-Entities vs. Parent</MenuItem>
              </Select>
            </FormControl>
            <Stack direction="row" spacing={2} alignItems="flex-start" sx={{ mt: 3 }}> // Corrected Stack
              <TextField
                name="naturalLanguageRule"
                label="Describe Rule in Plain English"
                value={naturalLanguageRule}
                onChange={(e) => setNaturalLanguageRule(e.target.value)}
                placeholder="e.g., 'Ensure each line item quantity is greater than zero'"
                multiline
                rows={2}
                fullWidth // Corrected fullWidth
                InputProps={{
                  startAdornment: <MessageSquareText size={20} style={{ marginRight: '8px' }} />,
                }}
              />
              <Button
                variant="contained"
                onClick={handleGenerateRule}
                disabled={generatingRule || !naturalLanguageRule.trim()}
                startIcon={generatingRule ? <CircularProgress size={20} /> : <BrainCircuit />}
                sx={{ minWidth: '150px', height: '56px' }} // Match TextField height // Corrected sx
              >
                Generate Rule
              </Button>
            </Stack>
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="3. Path Selection" />
          <CardContent>
          <Grid container spacing={3}>
            { (ruleType === 'parent_only' || ruleType === 'sub_parent' || ruleType === 'aggregate') && (
              <Grid item xs={12} lg={6}>
                <Typography variant="subtitle1" gutterBottom>Parent Field</Typography>
                {loadingSchema ? <CircularProgress /> : ( // Use Paper for consistent styling
                  <Paper variant="outlined" sx={{ p: 1, height: 250, overflow: 'auto' }}>
                    <TreeView
                      onSelectedItemsChange={(_event, itemIds) => itemIds && handleSelectPath(itemIds[0], 'parent')}
                      slots={{
                        collapseIcon: ExpandMoreIcon,
                        expandIcon: ChevronRightIcon,
                      }}
                    >
                      {dynamicEntityHierarchy.map(renderTree)}
                    </TreeView>
                  </Paper>
                )}
                {selectedPaths.parent && <Alert severity="success" sx={{ mt: 1 }}>Parent: <Chip label={selectedPaths.parent} size="small" /></Alert>} // Corrected Chip
              </Grid>
            )}

            { (ruleType === 'sub_only' || ruleType === 'sub_parent' || ruleType === 'aggregate') && ( // Corrected ruleType
              <Grid item xs={12} lg={6}>
                <Typography variant="subtitle1" gutterBottom>Sub-Entity Field</Typography>
                 {loadingSchema ? <CircularProgress /> : ( // Use Paper for consistent styling
                   <Paper variant="outlined" sx={{ p: 1, height: 250, overflow: 'auto' }}>
                    <TreeView
                      onSelectedItemsChange={(_event, itemIds) => itemIds && handleSelectPath(itemIds[0], 'sub')}
                      slots={{
                        collapseIcon: ExpandMoreIcon,
                        expandIcon: ChevronRightIcon,
                      }}
                    >
                      {dynamicEntityHierarchy.map(renderTree)}
                    </TreeView>
                  </Paper>
                 )}
                {selectedPaths.sub && <Alert severity="info" sx={{ mt: 1 }}>Sub: <Chip label={selectedPaths.sub} size="small" /></Alert>} // Corrected Chip
              </Grid>
            )}
          </Grid>
          </CardContent>
        </Card>

        {aiSuggestions.length > 0 && (
          <Card>
            <CardHeader avatar={<BrainCircuit />} title="AI Suggestions" />
            <CardContent>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>Based on your catalog, you might want to create rules related to these entities:</Typography>
              <List dense >
              {aiSuggestions.map(item => (
                <ListItem
                  key={item.targetEntityId}
                  secondaryAction={<Button size="small">Create Rule</Button>}
                >
                  <ListItemAvatar>
                    <Link />
                  </ListItemAvatar>
                  <ListItemText // Use ListItemText for proper typography // Corrected ListItemText
                    primary={item.targetEntityName}
                    secondary={`Shares concepts: ${item.sharedTerms.join(', ')}`}
                  />
                </ListItem>
              ))}
              </List>
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader title="4. Condition Logic" />
          <CardContent>
            {renderConditionFields()}
          </CardContent>
        </Card>

        <Button type="submit" variant="contained" size="large" fullWidth>
          Create Hierarchical Rule {/* Submit button for the form */}
        </Button>

        {/* Rule Preview */}
        <Card>
          <CardHeader avatar={<Code />} title="Rule Preview" />
          <CardContent>
          <Typography component="pre" sx={{ p: 2, bgcolor: 'grey.100', borderRadius: 1, whiteSpace: 'pre-wrap' }}>
            {generateRulePreviewText() || 'Your rule preview will appear here as you build it.'} {/* Placeholder text */} // Corrected placeholder text
          </Typography>
          </CardContent>
        </Card>

        {/* Integrated Test Harness */}
        <Card>
          <CardHeader avatar={<Play />} title="Test Rule with Sample Data" />
          <CardContent>
          <Stack spacing={2}> {/* Use Stack for layout */} // Corrected Stack
            <TextField
              label="Sample JSON Data"
              multiline
              rows={8}
              value={testData}
              onChange={(e) => setTestData(e.target.value)}
              placeholder={`Enter JSON data for the '${entity}' entity, e.g.:\n{\n  "id": "ORD001",\n  "total": 100,\n  "line_items": [\n    { "qty": 1, "price": 100 }\n  ]\n}`}
              fullWidth
            />
          <Button // Test button // Corrected button
            variant="outlined"
            onClick={handleTestRule}
            startIcon={testingLoading ? <CircularProgress size={20} /> : <Play />}
            disabled={testingLoading}
            fullWidth
          >
            Run Test
          </Button>
          {testResults && (
            <Alert severity={testResults.valid ? 'success' : 'error'} sx={{ mt: 2 }}> {/* Alert for test results */} // Corrected Alert
              <AlertTitle>{testResults.valid ? 'Rule Passed!' : 'Rule Failed!'}</AlertTitle>
              <Typography component="pre" sx={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(testResults, null, 2)}</Typography>
            </Alert>
          )}
          </Stack>
          </CardContent>
        </Card>
        </Stack>
      </form>
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar(prev => ({ ...prev, open: false }))}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={() => setSnackbar(prev => ({ ...prev, open: false }))} severity={snackbar.severity} sx={{ width: '100%' }}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Stack>
  );
};

export default HierarchyValidationBuilder;