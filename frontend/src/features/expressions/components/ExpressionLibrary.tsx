import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Tabs,
  Tab,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  PlayArrow as RunIcon,
  History as HistoryIcon,
} from '@mui/icons-material';
import ExpressionEditor from './ExpressionEditor';

// ============================================================================
// EXPRESSION LIBRARY
// Manages all Starlark expressions (validations, calculations, conditions)
// ============================================================================

interface Expression {
  id: string;
  name: string;
  rule_type: 'validation' | 'calculation' | 'condition';
  description?: string;
  script: string;
  is_active: boolean;
  version: number;
  created_at: string;
}

export const ExpressionLibrary: React.FC = () => {
  const [expressions, setExpressions] = useState<Expression[]>([]);
  const [filteredType, setFilteredType] = useState<string>('all');
  const [editorOpen, setEditorOpen] = useState(false);
  const [selectedExpression, setSelectedExpression] = useState<Expression | null>(null);

  useEffect(() => {
    loadExpressions();
  }, []);

  const loadExpressions = async () => {
    // TODO: Call API to get expressions
    // const response = await fetch('/api/expressions');
    // setExpressions(await response.json());
    
    // Mock data
    setExpressions([
      {
        id: '1',
        name: 'Transaction Amount Limit',
        rule_type: 'validation',
        description: 'Validates transaction amount does not exceed limit',
        script: 'result = (amount <= limit, "Amount exceeds limit") if amount > limit else (True, None)',
        is_active: true,
        version: 1,
        created_at: new Date().toISOString(),
      },
      {
        id: '2',
        name: 'Calculate Net Worth',
        rule_type: 'calculation',
        description: 'Calculates net worth from assets and liabilities',
        script: 'result = context.assets - context.liabilities',
        is_active: true,
        version: 1,
        created_at: new Date().toISOString(),
      },
      {
        id: '3',
        name: 'Approval Routing',
        rule_type: 'condition',
        description: 'Determines approval path based on amount',
        script: 'if amount > 1000000:\n    result = "committee_review"\nelif amount > 100000:\n    result = "manager_approval"\nelse:\n    result = "auto_approve"',
        is_active: true,
        version: 1,
        created_at: new Date().toISOString(),
      },
    ]);
  };

  const handleSaveExpression = async (expression: any) => {
    // TODO: Call API to save expression
    // const response = await fetch('/api/expressions', {
    //   method: 'POST',
    //   body: JSON.stringify(expression),
    // });
    
    setEditorOpen(false);
    loadExpressions();
  };

  const handleEditExpression = (expression: Expression) => {
    setSelectedExpression(expression);
    setEditorOpen(true);
  };

  const filteredExpressions = filteredType === 'all'
    ? expressions
    : expressions.filter((e) => e.rule_type === filteredType);

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'validation':
        return 'error';
      case 'calculation':
        return 'success';
      case 'condition':
        return 'warning';
      default:
        return 'default';
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4">Expression Library</Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => {
            setSelectedExpression(null);
            setEditorOpen(true);
          }}
        >
          New Expression
        </Button>
      </Box>

      <Card>
        <CardContent>
          <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
            <Tabs
              value={filteredType}
              onChange={(_, v) => setFilteredType(v)}
            >
              <Tab label="All" value="all" />
              <Tab label="Validations" value="validation" />
              <Tab label="Calculations" value="calculation" />
              <Tab label="Conditions" value="condition" />
            </Tabs>
          </Box>

          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Description</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Version</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredExpressions.map((expr) => (
                <TableRow key={expr.id} hover>
                  <TableCell>
                    <Typography variant="body2" fontWeight="medium">
                      {expr.name}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={expr.rule_type}
                      size="small"
                      color={getTypeColor(expr.rule_type) as any}
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary">
                      {expr.description}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={expr.is_active ? 'Active' : 'Inactive'}
                      size="small"
                      color={expr.is_active ? 'success' : 'default'}
                    />
                  </TableCell>
                  <TableCell>
                    <Chip label={`v${expr.version}`} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell align="right">
                    <IconButton
                      size="small"
                      onClick={() => handleEditExpression(expr)}
                    >
                      <EditIcon fontSize="small" />
                    </IconButton>
                    <IconButton size="small">
                      <RunIcon fontSize="small" />
                    </IconButton>
                    <IconButton size="small">
                      <HistoryIcon fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Expression Editor Dialog */}
      <Dialog
        open={editorOpen}
        onClose={() => setEditorOpen(false)}
        maxWidth="lg"
        fullWidth
      >
        <DialogTitle>
          {selectedExpression ? 'Edit Expression' : 'New Expression'}
        </DialogTitle>
        <DialogContent>
          <ExpressionEditor
            initialExpression={selectedExpression || undefined}
            onSave={handleSaveExpression}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditorOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ExpressionLibrary;
