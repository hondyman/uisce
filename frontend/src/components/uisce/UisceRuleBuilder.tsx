/**
 * UisceRuleBuilder - Main canvas for no-code rule creation
 * Business owners see logic and impact, never code.
 * 
 * Features:
 * - Metadata-driven field dropdowns
 * - Compound AND/OR condition logic
 * - Impact simulation against historical data
 * - CUE code generation (hidden from user)
 */
import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Divider,
  Alert,
  Chip,
  CircularProgress,
} from '@mui/material';
import {
  PlayArrow as SimulateIcon,
  Publish as PublishIcon,
  Code as CodeIcon,
  Visibility as PreviewIcon,
} from '@mui/icons-material';
import ConditionGroup, { ConditionGroupType } from './ConditionGroup';
import ImpactReport, { ImpactReportData } from './ImpactReport';
import { FieldDefinition } from './ConditionRow';

export interface UIRule {
  name: string;
  description: string;
  conditionGroups: ConditionGroupType[];
  logic: 'AND' | 'OR';
  action: 'REJECT' | 'FLAG_FOR_REVIEW' | 'APPROVE' | 'LOG';
  severity: 'error' | 'warning' | 'info';
}

interface UisceRuleBuilderProps {
  tenantId: string;
  datasourceId: string;
  targetEntity?: string;
  onPublish?: (rule: UIRule, cueCode: string) => void;
}

const ACTION_OPTIONS = [
  { value: 'REJECT', label: 'Reject Transaction', color: 'error' },
  { value: 'FLAG_FOR_REVIEW', label: 'Flag for Review', color: 'warning' },
  { value: 'APPROVE', label: 'Auto-Approve', color: 'success' },
  { value: 'LOG', label: 'Log Only', color: 'info' },
];

export const UisceRuleBuilder: React.FC<UisceRuleBuilderProps> = ({
  tenantId,
  datasourceId,
  targetEntity,
  onPublish,
}) => {
  // Field definitions from metadata
  const [fields, setFields] = useState<FieldDefinition[]>([]);
  const [loadingFields, setLoadingFields] = useState(false);

  // Rule state
  const [rule, setRule] = useState<UIRule>({
    name: '',
    description: '',
    conditionGroups: [{
      id: `group-${Date.now()}`,
      logic: 'AND',
      conditions: [],
    }],
    logic: 'AND',
    action: 'REJECT',
    severity: 'error',
  });

  // Simulation state
  const [simulating, setSimulating] = useState(false);
  const [impactReport, setImpactReport] = useState<ImpactReportData | null>(null);
  const [simulationError, setSimulationError] = useState<string | null>(null);

  // Publishing state
  const [publishing, setPublishing] = useState(false);
  const [generatedCue, setGeneratedCue] = useState<string | null>(null);
  const [showCuePreview, setShowCuePreview] = useState(false);

  // Load field definitions from metadata
  useEffect(() => {
    const loadFields = async () => {
      setLoadingFields(true);
      try {
        // Try to load from business object or semantic terms
        const response = await fetch(
          `/api/business-objects/${targetEntity}/fields?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`
        );
        
        if (response.ok) {
          const data = await response.json();
          const mappedFields: FieldDefinition[] = (data.fields || []).map((f: any) => ({
            name: f.name || f.key || f.field_name,
            type: mapFieldType(f.type || f.field_type),
            label: f.label || f.display_name || f.name,
            options: f.options || f.enum_values,
          }));
          setFields(mappedFields);
        } else {
          // Fallback to default fields for demo
          setFields(getDefaultFields());
        }
      } catch (error) {
        console.error('Failed to load field definitions:', error);
        setFields(getDefaultFields());
      } finally {
        setLoadingFields(false);
      }
    };

    if (tenantId && datasourceId) {
      loadFields();
    } else {
      setFields(getDefaultFields());
    }
  }, [tenantId, datasourceId, targetEntity]);

  const mapFieldType = (type: string): FieldDefinition['type'] => {
    const typeMap: Record<string, FieldDefinition['type']> = {
      'number': 'number',
      'integer': 'number',
      'decimal': 'number',
      'float': 'number',
      'string': 'string',
      'text': 'string',
      'boolean': 'boolean',
      'bool': 'boolean',
      'date': 'date',
      'datetime': 'date',
      'enum': 'enum',
      'select': 'enum',
    };
    return typeMap[type?.toLowerCase()] || 'string';
  };

  const getDefaultFields = (): FieldDefinition[] => [
    { name: 'amount', type: 'number', label: 'Transaction Amount' },
    { name: 'currency', type: 'enum', label: 'Currency', options: ['USD', 'EUR', 'GBP', 'JPY'] },
    { name: 'counterparty_rating', type: 'enum', label: 'Counterparty Rating', options: ['AAA', 'AA', 'A', 'BBB', 'BB', 'B', 'CCC'] },
    { name: 'trade_type', type: 'enum', label: 'Trade Type', options: ['BUY', 'SELL', 'TRANSFER'] },
    { name: 'trade_date', type: 'date', label: 'Trade Date' },
    { name: 'is_internal', type: 'boolean', label: 'Internal Trade' },
    { name: 'client_name', type: 'string', label: 'Client Name' },
  ];

  const handleGroupChange = (index: number, group: ConditionGroupType) => {
    const newGroups = [...rule.conditionGroups];
    newGroups[index] = group;
    setRule({ ...rule, conditionGroups: newGroups });
  };

  const handleSimulate = useCallback(async () => {
    setSimulating(true);
    setSimulationError(null);
    setImpactReport(null);

    try {
      const payload = {
        rule: {
          name: rule.name,
          description: rule.description,
          conditions: rule.conditionGroups.flatMap(g => g.conditions),
          logic: rule.logic,
          action: rule.action,
        },
        time_range: '24h',
      };

      const response = await fetch(
        `/api/rules/simulate?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        }
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Simulation failed');
      }

      const result = await response.json();
      setImpactReport(result);
    } catch (error) {
      const err = error instanceof Error ? error.message : 'Unknown error';
      setSimulationError(err);
    } finally {
      setSimulating(false);
    }
  }, [rule, tenantId, datasourceId]);

  const handleGenerateCue = useCallback(async () => {
    try {
      const payload = {
        name: rule.name,
        description: rule.description,
        conditions: rule.conditionGroups.flatMap(g => g.conditions),
        logic: rule.logic,
        action: rule.action,
      };

      const response = await fetch(
        `/api/rules/generate-cue?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        }
      );

      if (response.ok) {
        const data = await response.json();
        setGeneratedCue(data.cue_code);
        setShowCuePreview(true);
      }
    } catch (error) {
      console.error('CUE generation failed:', error);
    }
  }, [rule, tenantId, datasourceId]);

  const handlePublish = useCallback(async () => {
    if (!rule.name.trim()) {
      alert('Please enter a rule name');
      return;
    }

    if (rule.conditionGroups.every(g => g.conditions.length === 0)) {
      alert('Please add at least one condition');
      return;
    }

    setPublishing(true);
    try {
      // First generate CUE
      await handleGenerateCue();
      
      // Then call publish
      if (onPublish && generatedCue) {
        onPublish(rule, generatedCue);
      }
    } finally {
      setPublishing(false);
    }
  }, [rule, generatedCue, handleGenerateCue, onPublish]);

  const hasConditions = rule.conditionGroups.some(g => g.conditions.length > 0);

  return (
    <Box sx={{ maxWidth: 900, mx: 'auto' }}>
      <Card elevation={2}>
        <CardContent>
          {/* Header */}
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
            <Box>
              <Typography variant="h5" fontWeight="bold">
                Uisce Compliance Builder
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Build validation rules visually • See impact before you publish
              </Typography>
            </Box>
            <Chip 
              label={targetEntity || 'All Entities'} 
              color="primary" 
              variant="outlined" 
            />
          </Box>

          <Divider sx={{ mb: 3 }} />

          {/* Rule Info */}
          <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
            <TextField
              label="Rule Name"
              value={rule.name}
              onChange={(e) => setRule({ ...rule, name: e.target.value })}
              fullWidth
              placeholder="e.g., High Value Trade Check"
            />
            <FormControl sx={{ minWidth: 180 }}>
              <InputLabel>Action</InputLabel>
              <Select
                value={rule.action}
                label="Action"
                onChange={(e) => setRule({ ...rule, action: e.target.value as UIRule['action'] })}
              >
                {ACTION_OPTIONS.map((opt) => (
                  <MenuItem key={opt.value} value={opt.value}>
                    <Chip size="small" label={opt.label} color={opt.color as any} />
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Box>

          <TextField
            label="Description"
            value={rule.description}
            onChange={(e) => setRule({ ...rule, description: e.target.value })}
            fullWidth
            multiline
            rows={2}
            placeholder="Describe what this rule validates and why..."
            sx={{ mb: 3 }}
          />

          <Divider sx={{ mb: 3 }} />

          {/* Conditions */}
          <Typography variant="subtitle1" fontWeight="bold" gutterBottom>
            Conditions
          </Typography>

          {loadingFields ? (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, py: 4, justifyContent: 'center' }}>
              <CircularProgress size={24} />
              <Typography>Loading field definitions...</Typography>
            </Box>
          ) : (
            <Box sx={{ mb: 3 }}>
              {rule.conditionGroups.map((group, index) => (
                <ConditionGroup
                  key={group.id}
                  group={group}
                  fields={fields}
                  onChange={(g) => handleGroupChange(index, g)}
                />
              ))}
            </Box>
          )}

          <Divider sx={{ mb: 3 }} />

          {/* Actions */}
          <Box sx={{ display: 'flex', gap: 2, justifyContent: 'flex-end' }}>
            <Button
              variant="outlined"
              startIcon={<CodeIcon />}
              onClick={handleGenerateCue}
              disabled={!hasConditions}
            >
              Preview CUE
            </Button>
            <Button
              variant="contained"
              color="secondary"
              startIcon={simulating ? <CircularProgress size={20} color="inherit" /> : <SimulateIcon />}
              onClick={handleSimulate}
              disabled={!hasConditions || simulating}
            >
              {simulating ? 'Simulating...' : 'Run Simulation'}
            </Button>
            <Button
              variant="contained"
              color="primary"
              startIcon={publishing ? <CircularProgress size={20} color="inherit" /> : <PublishIcon />}
              onClick={handlePublish}
              disabled={!hasConditions || publishing || !rule.name.trim()}
            >
              {publishing ? 'Publishing...' : 'Publish Rule'}
            </Button>
          </Box>
        </CardContent>
      </Card>

      {/* Impact Report */}
      {(impactReport || simulationError || simulating) && (
        <Box sx={{ mt: 3 }}>
          <ImpactReport
            data={impactReport}
            loading={simulating}
            error={simulationError}
          />
        </Box>
      )}

      {/* CUE Preview Dialog */}
      {showCuePreview && generatedCue && (
        <Card variant="outlined" sx={{ mt: 3 }}>
          <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
              <Typography variant="subtitle1" fontWeight="bold">
                Generated CUE Code (Internal)
              </Typography>
              <Button size="small" onClick={() => setShowCuePreview(false)}>
                Close
              </Button>
            </Box>
            <Alert severity="info" sx={{ mb: 2 }}>
              This is the validation code generated from your visual rule. Most users never need to see this.
            </Alert>
            <Box
              component="pre"
              sx={{
                p: 2,
                bgcolor: 'grey.900',
                color: 'grey.100',
                borderRadius: 1,
                overflow: 'auto',
                fontSize: '0.85rem',
                fontFamily: 'monospace',
              }}
            >
              {generatedCue}
            </Box>
          </CardContent>
        </Card>
      )}
    </Box>
  );
};

export default UisceRuleBuilder;
