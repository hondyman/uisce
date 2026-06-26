import React, { useState, useEffect } from 'react';
import { Box, Typography, TextField, Button, Paper, Stack, InputAdornment, MenuItem, Alert, Chip, Divider, Tabs, Tab, Slider, FormControlLabel, Switch } from '@mui/material';
import useUisceStore from '../hooks/useUisceStore';
import SaveIcon from '@mui/icons-material/Save';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import CurrencyExchangeIcon from '@mui/icons-material/CurrencyExchange';
import EventIcon from '@mui/icons-material/Event';
import LinkIcon from '@mui/icons-material/Link';
import axios from '@/utils/axiosClient';
import MonacoCodeEditor from '../../../components/UnifiedSemanticBuilder/MonacoCodeEditor';

const ConfigPanel = () => {
  const { nodes, selectedNodeId, updateNodeConfig, selectedBO } = useUisceStore();
  const selectedNode = nodes.find((n) => n.id === selectedNodeId);
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [tabIndex, setTabIndex] = useState(0);
  const [lookupDatasets, setLookupDatasets] = useState<any[]>([]);
  const [validationRules, setValidationRules] = useState<any[]>([]);
  const [availableBOs, setAvailableBOs] = useState<any[]>([]);
  const [targetBOFields, setTargetBOFields] = useState<any[]>([]);

  useEffect(() => {
    if (selectedNode) {
      setFormData(selectedNode.data.config || {});
    }
  }, [selectedNodeId, selectedNode]);

  useEffect(() => {
      // Fetch available lookups if we are configuring a List_Lookup
      const type = selectedNode?.data?.filterType || selectedNode?.data?.label; // fallback
      if (type && (type === 'List_Lookup' || type.includes('List_Lookup'))) {
          axios.get('/api/lookups')
            .then(res => setLookupDatasets(res.data.items || []))
            .catch(err => console.error("Failed to fetch lookups", err));
      }

      // Fetch validation rules if Policy Check and we know the context
      if (type === 'Policy_Check' && selectedBO?.id) {
          axios.get(`/api/validation-rules?target_entity_id=${selectedBO.id}`)
            .then(res => setValidationRules(res.data.items || []))
            .catch(err => console.error("Failed to fetch validation rules", err));
      }

      // Fetch BOs for Record Create
      if (type === 'Record_Create') {
          axios.get('/api/business-objects')
            .then(res => {
                // API returns map { id: bo }, convert to array and sort
                const boArray = Object.values(res.data || {});
                setAvailableBOs(boArray);
            })
            .catch(err => console.error("Failed to fetch BOs", err));
      }
      
      // Fetch fields if target BO is selected
      if (type === 'Record_Create' && selectedNode?.data?.config?.targetBOId) {
           axios.get(`/api/business-objects/${selectedNode.data.config.targetBOId}`)
            .then(res => setTargetBOFields(res.data.fields || []))
            .catch(err => console.error("Failed to fetch target fields", err));
      }
  }, [selectedNodeId, selectedBO?.id, selectedNode?.data?.config?.targetBOId]);

  if (!selectedNode) {
    return (
      <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: 'text.secondary', textAlign: 'center' }}>
        <AutoAwesomeIcon sx={{ fontSize: 48, mb: 2, color: 'action.disabled' }} />
        <Typography variant="body2">Select a node from the canvas to configure it.</Typography>
      </Box>
    );
  }

  const handleSave = () => {
    if (selectedNodeId) {
      updateNodeConfig(selectedNodeId, formData);
    }
  };

  const handleChange = (field: string, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabIndex(newValue);
  };

  // Determine Filter Type
  const filterType = selectedNode.data.filterType || 
                     (selectedNode.data.label.includes('Limit') ? 'Limit' : 
                      selectedNode.data.label.includes('Sanctions') ? 'Sanctions' : 
                      selectedNode.data.label.includes('AI') ? 'AI_Anomaly' : 
                      selectedNode.data.label.includes('Approval') ? 'Approval_Gate' :
                      selectedNode.data.label.includes('API') ? 'External_API' :
                      selectedNode.data.label.includes('Transform') ? 'Transform' :
                      selectedNode.data.label.includes('Aggregation') ? 'Aggregation' :
                      selectedNode.data.label.includes('Record') ? 'Record_Create' :
                      selectedNode.data.label.includes('Sub-Pipeline') || selectedNode.data.label.includes('Child') ? 'subPipeline' :
                      selectedNode.data.label.includes('Parallel') ? 'parallel' :
                      selectedNode.data.label.includes('For Each') || selectedNode.data.label.includes('ForEach') ? 'forEach' :
                      selectedNode.data.label.includes('Wait For Event') ? 'waitForEvent' :
                      selectedNode.data.label.includes('Wait') || selectedNode.data.label.includes('Timer') ? 'wait' :
                      selectedNode.data.label.includes('Try') || selectedNode.data.label.includes('Catch') ? 'tryCatch' :
                      selectedNode.data.label.includes('Compensate') || selectedNode.data.label.includes('Saga') ? 'compensate' :
                      selectedNode.data.label.includes('Switch') ? 'switch' :
                      selectedNode.data.label.includes('Publish') || selectedNode.data.label.includes('Event') ? 'publishEvent' :
                      selectedNode.data.label.includes('Alert') || selectedNode.data.label.includes('Notify') ? 'alert' :
                      selectedNode.data.label.includes('Interpretation') ? 'Interpretation' :
                      selectedNode.data.label.includes('Classification') ? 'Classification' :
                      selectedNode.data.label.includes('Drafting') ? 'Drafting' :
                      selectedNode.data.label.includes('Recommendation') ? 'Recommendation' :
                      selectedNode.data.label.includes('Explanation') || selectedNode.data.label.includes('ExceptionExplanation') ? 'ExceptionExplanation' :
                      selectedNode.data.label.includes('Conditional') ? 'Conditional' : 'Unknown');

  // BO Fields for Dropdowns (merged with pipeline variables)
  const pipelineVariables = nodes
    .filter(n => n.id !== selectedNodeId && n.data.config?.outputVariable)
    .map(n => ({
        name: n.data.config.outputVariable,
        label: `[VAR] ${n.data.config.outputVariable} (${n.data.label})`,
        data_type: 'any'
    }));

  const boFields = [...(selectedBO?.fields || []), ...pipelineVariables];

  const getLimitLogic = () => {
    return `if trade.${formData.fieldName || 'amount'} > ${formData.limit || 0} {\n    return error("Limit Exceeded")\n}`;
  };

  return (
    <Box sx={{ p: 0, height: '100%', bgcolor: 'rgba(255,255,255,0.8)', backdropFilter: 'blur(10px)', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ p: 3, borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
         <Typography variant="overline" color="text.secondary" fontWeight="bold" sx={{ display: 'flex', justifyContent: 'space-between' }}>
            Configuration <span>{filterType}</span>
         </Typography>
         <Typography variant="h6" fontWeight="800" sx={{ color: '#1e293b' }}>
            {selectedNode.data.label}
         </Typography>
         <Typography variant="caption" color="text.secondary">
            ID: {selectedNode.id}
         </Typography>
         {selectedBO && (
             <Chip 
                label={`Context: ${selectedBO.display_name}`} 
                size="small" 
                color="primary" 
                variant="outlined" 
                sx={{ mt: 1, height: 20, fontSize: '0.65rem' }} 
             />
         )}
      </Box>

      {/* Tabs */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={tabIndex} onChange={handleTabChange} aria-label="config tabs" variant="fullWidth">
            <Tab label="Properties" sx={{ fontWeight: 600 }} />
            <Tab label="Code (CUE)" sx={{ fontWeight: 600 }} />
        </Tabs>
      </Box>

      {/* Properties Tab */}
      <div role="tabpanel" hidden={tabIndex !== 0} style={{ flexGrow: 1, overflowY: 'auto' }}>
        {tabIndex === 0 && (
            <Stack spacing={3} sx={{ p: 3 }}>
                
                {/* Common Field: Description */}
                <TextField
                    label="Description"
                    fullWidth
                    size="small"
                    multiline
                    rows={2}
                    value={formData.description || ''}
                    onChange={(e) => handleChange('description', e.target.value)}
                />
                
                <Divider />

                {/* --- TYPE SPECIFIC FIELDS --- */}

                {/* 1. Trigger Configuration */}
            {filterType === 'Trigger' && (
                <>
                    <Typography variant="caption" sx={{ mb: 1, display: 'block', color: 'text.secondary' }}>
                        How this pipeline is initiated.
                    </Typography>
                    
                    <TextField
                        select
                        label="Trigger Type"
                        fullWidth
                        size="small"
                        value={formData.triggerType || 'api'}
                        onChange={(e) => handleChange('triggerType', e.target.value)}
                        sx={{ mb: 2 }}
                    >
                        <MenuItem value="api">REST API Webhook</MenuItem>
                        <MenuItem value="schedule">Scheduled (Cron)</MenuItem>
                        <MenuItem value="event">Event Bus (Kafka/RabbitMQ)</MenuItem>
                    </TextField>

                    {(!formData.triggerType || formData.triggerType === 'api') && (
                        <Box sx={{ mt: 1 }}>
                            <Typography variant="caption" fontWeight="bold">Invocation Endpoint</Typography>
                            <Box sx={{ bgcolor: '#f1f5f9', p: 1, borderRadius: 1, my: 1, fontFamily: 'monospace', fontSize: '0.75rem', wordBreak: 'break-all' }}>
                                POST /api/pipelines/{selectedNodeId?.split('-')[1] || '123'}/run
                            </Box>
                            
                            <Typography variant="caption" fontWeight="bold">Sample Payload</Typography>
                            <Box sx={{ bgcolor: '#1e293b', color: '#a5f3fc', p: 1.5, borderRadius: 1, mt: 1, fontFamily: 'monospace', fontSize: '0.7rem', whiteSpace: 'pre-wrap', maxHeight: 200, overflow: 'auto' }}>
                                {JSON.stringify({
                                    trace_id: "uuid-v4",
                                    context: {
                                        tenant_id: "tenant-1",
                                        bo_type: selectedBO?.name || "Trade"
                                    },
                                    data: selectedBO?.fields?.reduce((acc: any, f: any) => ({...acc, [f.name]: f.type === 'number' ? 1000 : 'example_value'}), {}) || {
                                        amount: 50000,
                                        currency: "USD",
                                        counterparty: "CPTY_A"
                                    }
                                }, null, 2)}
                            </Box>
                        </Box>
                    )}

                    {formData.triggerType === 'schedule' && (
                        <TextField
                            label="Cron Expression"
                            placeholder="0 0 * * *"
                            fullWidth
                            size="small"
                            value={formData.cron || ''}
                            onChange={(e) => handleChange('cron', e.target.value)}
                            helperText="Runs daily at midnight UTC"
                        />
                    )}
                </>
            )}

                {/* 1. Limit Filter */}
                {filterType === 'Limit' && (
                    <>
                        <TextField 
                            select
                            label="Field to Check"
                            fullWidth
                            size="small"
                            value={formData.fieldName || ''}
                            onChange={(e) => handleChange('fieldName', e.target.value)}
                        >
                            {boFields.map((f: any) => (
                                <MenuItem key={f.name} value={f.name}>{f.label} ({f.data_type})</MenuItem>
                            ))}
                            {!selectedBO && <MenuItem value="amount">Amount (Default)</MenuItem>}
                        </TextField>
                        <TextField 
                            label="Threshold Amount"
                            type="number" 
                            fullWidth
                            size="small"
                            value={formData.limit || 0}
                            onChange={(e) => handleChange('limit', Number(e.target.value))}
                            InputProps={{
                                startAdornment: <InputAdornment position="start"><CurrencyExchangeIcon fontSize="small" /></InputAdornment>,
                            }}
                        />
                    </>
                )}

                {/* 2. Sanctions Filter */}
                {filterType === 'Sanctions' && (
                    <>
                        <TextField 
                            select
                            label="Sanctions List"
                            fullWidth
                            size="small"
                            value={formData.listType || 'OFAC_SDN'}
                            onChange={(e) => handleChange('listType', e.target.value)}
                        >
                            <MenuItem value="OFAC_SDN">OFAC SDN List (USA)</MenuItem>
                            <MenuItem value="EU_CONSOLIDATED">EU Consolidated List</MenuItem>
                            <MenuItem value="UN_SECURITY_COUNCIL">UN Security Council</MenuItem>
                        </TextField>
                        <Alert severity="info" sx={{ fontSize: '0.75rem' }}>
                            Checks all party fields (sender, receiver, intermediaries) against selected list.
                        </Alert>
                    </>
                )}

                {/* 3. List Lookup */}
                {filterType === 'List_Lookup' && (
                    <>
                        <TextField 
                            select
                            label="Field to Lookup"
                            fullWidth
                            size="small"
                            value={formData.fieldName || ''}
                            onChange={(e) => handleChange('fieldName', e.target.value)}
                        >
                            {boFields.map((f: any) => (
                                <MenuItem key={f.name} value={f.name}>{f.label}</MenuItem>
                            ))}
                            <MenuItem value="custom">Custom Field...</MenuItem>
                        </TextField>

                        <FormControlLabel 
                            control={
                                <Switch 
                                    checked={formData.useDataset || false}
                                    onChange={(e) => handleChange('useDataset', e.target.checked)}
                                />
                            } 
                            label="Use Reference Dataset" 
                        />

                        {formData.useDataset ? (
                             <TextField 
                                select
                                label="Select Reference Dataset"
                                fullWidth
                                size="small"
                                value={formData.datasetId || ''}
                                onChange={(e) => handleChange('datasetId', e.target.value)}
                            >
                                {lookupDatasets.map((l) => (
                                    <MenuItem key={l.id} value={l.id}>
                                        {l.name} {l.description && `(${l.description})`}
                                    </MenuItem>
                                ))}
                                {lookupDatasets.length === 0 && <MenuItem disabled>No datasets available</MenuItem>}
                            </TextField>
                        ) : (
                            <TextField 
                                label="Reference List Values (comma separated)"
                                fullWidth
                                multiline
                                rows={3}
                                size="small"
                                value={formData.referenceList || ''}
                                onChange={(e) => handleChange('referenceList', e.target.value)}
                                placeholder="e.g. US, UK, CA, DE, FR"
                            />
                        )}
                    </>
                )}

                {/* 4. Date Validator */}
                {filterType === 'Date_Validator' && (
                    <>
                        <TextField 
                            select
                            label="Date Field"
                            fullWidth
                            size="small"
                            value={formData.fieldName || ''}
                            onChange={(e) => handleChange('fieldName', e.target.value)}
                        >
                            {boFields.filter((f:any) => f.data_type === 'date').map((f: any) => (
                                <MenuItem key={f.name} value={f.name}>{f.label}</MenuItem>
                            ))}
                            <MenuItem value="trade_date">Trade Date</MenuItem>
                            <MenuItem value="settlement_date">Settlement Date</MenuItem>
                        </TextField>
                        <TextField 
                            select
                            label="Validation Rule"
                            fullWidth
                            size="small"
                            value={formData.rule || 'today'}
                            onChange={(e) => handleChange('rule', e.target.value)}
                        >
                            <MenuItem value="future">Must be in Future</MenuItem>
                            <MenuItem value="past">Must be in Past</MenuItem>
                            <MenuItem value="business_day">Must be Business Day</MenuItem>
                            <MenuItem value="after">Must be After...</MenuItem>
                        </TextField>
                        {(formData.rule === 'after' || formData.rule === 'before') && (
                            <TextField 
                                type="date"
                                label="Reference Date"
                                fullWidth
                                size="small"
                                InputLabelProps={{ shrink: true }}
                                value={formData.referenceDate || ''}
                                onChange={(e) => handleChange('referenceDate', e.target.value)}
                            />
                        )}
                    </>
                )}

                {/* 5. Cross Reference */}
                {filterType === 'Cross_Reference' && (
                    <>
                        <TextField 
                            select
                            label="Local Field (ID)"
                            fullWidth
                            size="small"
                            value={formData.fieldName || ''}
                            onChange={(e) => handleChange('fieldName', e.target.value)}
                        >
                            {boFields.map((f: any) => (
                                <MenuItem key={f.name} value={f.name}>{f.label}</MenuItem>
                            ))}
                        </TextField>
                        <TextField 
                            select
                            label="Target Table"
                            fullWidth
                            size="small"
                            value={formData.targetTable || ''}
                            onChange={(e) => handleChange('targetTable', e.target.value)}
                        >
                            <MenuItem value="customers">Customers</MenuItem>
                            <MenuItem value="accounts">Accounts</MenuItem>
                            <MenuItem value="securities">Securities</MenuItem>
                            <MenuItem value="currencies">Currencies</MenuItem>
                        </TextField>
                        <Alert severity="success" icon={<LinkIcon />} sx={{ fontSize: '0.75rem' }}>
                            Validates referential integrity against the central catalog.
                        </Alert>
                    </>
                )}
                
                {/* 6. Formula */}
                {filterType === 'Formula' && (
                    <TextField 
                        label="Expression (Starlark/CUE)"
                        fullWidth
                        multiline
                        rows={4}
                        size="small"
                        value={formData.expression || ''}
                        onChange={(e) => handleChange('expression', e.target.value)}
                        placeholder="e.g. trade.amount * 0.1 < client.credit_limit"
                        helperText="Use available fields from the selected Business Object."
                    />
                )}

                {/* 7. Conditional */}
                {filterType === 'Conditional' && (
                    <TextField 
                        label="Condition (Expression)"
                        fullWidth
                        multiline
                        rows={3}
                        size="small"
                        value={formData.condition || ''}
                        onChange={(e) => handleChange('condition', e.target.value)}
                        placeholder="e.g. trade.currency == 'USD'"
                        helperText="If true, follows the top path. If false, follows the bottom path."
                    />
                )}

                {/* 8. Approval Gate */}
                {filterType === 'Approval_Gate' && (
                    <>
                        <TextField 
                            select
                            label="Required Role"
                            fullWidth
                            size="small"
                            SelectProps={{ multiple: true }}
                            value={formData.approverRoles || []}
                            onChange={(e) => handleChange('approverRoles', e.target.value)}
                        >
                            <MenuItem value="compliance_officer">Compliance Officer</MenuItem>
                            <MenuItem value="risk_manager">Risk Manager</MenuItem>
                            <MenuItem value="supervisor">Supervisor</MenuItem>
                            <MenuItem value="admin">System Admin</MenuItem>
                        </TextField>
                        <TextField 
                            label="Timeout (Hours)"
                            type="number" 
                            fullWidth
                            size="small"
                            value={formData.timeout || 24}
                            onChange={(e) => handleChange('timeout', Number(e.target.value))}
                        />
                        <FormControlLabel 
                            control={
                                <Switch 
                                    checked={formData.autoEscalate || false}
                                    onChange={(e) => handleChange('autoEscalate', e.target.checked)}
                                />
                            } 
                            label="Escalate on Timeout" 
                        />
                    </>
                )}

                {/* 9. External API */}
                {filterType === 'External_API' && (
                    <>
                        <TextField 
                            label="Output Variable Name"
                            fullWidth
                            size="small"
                            value={formData.outputVariable || ''}
                            onChange={(e) => handleChange('outputVariable', e.target.value)}
                            placeholder="e.g. fraud_score_api"
                            helperText="Variable to store the API response"
                            sx={{ mb: 2, bgcolor: '#f0f9ff' }}
                        />
                        <TextField 
                            label="API URL"
                            fullWidth
                            size="small"
                            value={formData.url || ''}
                            onChange={(e) => handleChange('url', e.target.value)}
                            placeholder="https://api.example.com/v1/check"
                        />
                        <Stack direction="row" spacing={2}>
                            <TextField 
                                select
                                label="Method"
                                fullWidth
                                size="small"
                                value={formData.method || 'POST'}
                                onChange={(e) => handleChange('method', e.target.value)}
                            >
                                <MenuItem value="GET">GET</MenuItem>
                                <MenuItem value="POST">POST</MenuItem>
                                <MenuItem value="PUT">PUT</MenuItem>
                            </TextField>
                            <TextField 
                                label="Timeout (ms)"
                                type="number"
                                fullWidth
                                size="small"
                                value={formData.timeoutMs || 5000}
                                onChange={(e) => handleChange('timeoutMs', Number(e.target.value))}
                            />
                        </Stack>
                        <TextField 
                            label="Auth Token (Key)"
                            fullWidth
                            size="small"
                            value={formData.authKey || ''}
                            onChange={(e) => handleChange('authKey', e.target.value)}
                            type="password"
                        />
                    </>
                )}

                {/* 10. Transform */}
                {filterType === 'Transform' && (
                    <>
                         <TextField 
                            label="Output Variable Name"
                            fullWidth
                            size="small"
                            value={formData.outputVariable || ''}
                            onChange={(e) => handleChange('outputVariable', e.target.value)}
                            placeholder="e.g. transformed_data"
                            helperText="Variable to store the result"
                            sx={{ mb: 2, bgcolor: '#f0f9ff' }}
                        />
                         <TextField 
                            select
                            label="Transform Type"
                            fullWidth
                            size="small"
                            value={formData.transformType || 'script'}
                            onChange={(e) => handleChange('transformType', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value="script">Custom Script (Starlark)</MenuItem>
                            <MenuItem value="map">Field Mapping</MenuItem>
                        </TextField>

                        {(!formData.transformType || formData.transformType === 'script') && (
                            <Box sx={{ border: '1px solid #e2e8f0', borderRadius: 1, overflow: 'hidden' }}>
                                <Box sx={{ p: 1, bgcolor: '#f8fafc', borderBottom: '1px solid #e2e8f0', display:'flex', alignItems:'center' }}>
                                     <Typography variant="caption" fontWeight="bold">Starlark Script</Typography>
                                     <Box sx={{ flexGrow: 1 }} />
                                     <Typography variant="caption" sx={{ color: '#64748b' }}>input.field | variables.name</Typography>
                                </Box>
                                <Box sx={{ height: 200 }}>
                                    <MonacoCodeEditor
                                        language="python"
                                        value={formData.script || 'def transform(input, variables):\n    output = {}\n    # Your logic here\n    return output'}
                                        onChange={(val: string) => handleChange('script', val)}
                                        dynamicCompletions={[
                                            ...boFields.map(f => ({ label: `input.${f.name}`, insertText: `input.${f.name}`, kind: 9 })), 
                                            ...pipelineVariables.map(v => ({ label: `variables.${v.name}`, insertText: `variables.${v.name}`, kind: 6 }))
                                        ]}
                                    />
                                </Box>
                            </Box>
                        )}
                    </>
                )}

                 {/* 11. Aggregation */}
                 {filterType === 'Aggregation' && (
                    <>
                        <TextField 
                            label="Output Variable Name"
                            fullWidth
                            size="small"
                            value={formData.outputVariable || ''}
                            onChange={(e) => handleChange('outputVariable', e.target.value)}
                            placeholder="e.g. total_exposure"
                            helperText="Variable to store the aggregated value"
                            sx={{ mb: 2, bgcolor: '#f0f9ff' }}
                        />
                        <TextField 
                            select
                            label="Group By"
                            fullWidth
                            size="small"
                            value={formData.groupBy || ''}
                            onChange={(e) => handleChange('groupBy', e.target.value)}
                        >
                             {boFields.map((f: any) => (
                                <MenuItem key={f.name} value={f.name}>{f.label}</MenuItem>
                            ))}
                            <MenuItem value="counterparty">Counterparty</MenuItem>
                        </TextField>
                        <Stack direction="row" spacing={2}>
                            <TextField 
                                select
                                label="Operation"
                                fullWidth
                                size="small"
                                value={formData.operation || 'sum'}
                                onChange={(e) => handleChange('operation', e.target.value)}
                            >
                                <MenuItem value="sum">Sum</MenuItem>
                                <MenuItem value="count">Count</MenuItem>
                                <MenuItem value="avg">Average</MenuItem>
                                <MenuItem value="max">Max</MenuItem>
                            </TextField>
                            <TextField 
                                select
                                label="Field"
                                fullWidth
                                size="small"
                                value={formData.targetField || ''}
                                onChange={(e) => handleChange('targetField', e.target.value)}
                            >
                                 {boFields.filter((f: any) => f.data_type === 'number').map((f: any) => (
                                    <MenuItem key={f.name} value={f.name}>{f.label}</MenuItem>
                                ))}
                                <MenuItem value="amount">Amount</MenuItem>
                            </TextField>
                        </Stack>
                        <TextField 
                            label="Window (Seconds)"
                            type="number"
                            fullWidth
                            size="small"
                            value={formData.windowSeconds || 60}
                            onChange={(e) => handleChange('windowSeconds', Number(e.target.value))}
                            helperText="Rolling window size for aggregation"
                        />
                    </>
                )}

                {/* 12. AI Anomaly & Prediction */}
                {(filterType === 'AI_Anomaly' || filterType === 'AI_Prediction') && (
                    <>
                        <TextField 
                            label="Output Variable Name"
                            fullWidth
                            size="small"
                            value={formData.outputVariable || ''}
                            onChange={(e) => handleChange('outputVariable', e.target.value)}
                            placeholder="e.g. anomaly_score"
                            helperText="Variable to store the model output"
                            sx={{ mb: 2, bgcolor: '#f0f9ff' }}
                        />
                        <Typography variant="caption" gutterBottom>
                            Sensitivity Threshold: {formData.sensitivity || 50}%
                        </Typography>
                        <Slider 
                            value={formData.sensitivity || 50}
                            onChange={(e, val) => handleChange('sensitivity', val)}
                            valueLabelDisplay="auto"
                            min={1}
                            max={99}
                            sx={{ mb: 2 }}
                        />
                        <TextField 
                            select
                            label="Model Version"
                            fullWidth
                            size="small"
                            value={formData.modelVersion || 'v2'}
                            onChange={(e) => handleChange('modelVersion', e.target.value)}
                        >
                            <MenuItem value="v1">v1.0 (Standard)</MenuItem>
                            <MenuItem value="v2">v2.1 (Enhanced with Graph)</MenuItem>
                            <MenuItem value="v3_beta">v3.0 Beta (Deep Learning)</MenuItem>
                        </TextField>
                         <Alert severity="warning" sx={{ fontSize: '0.75rem', mt: 1 }}>
                            Requires active GPU inference server connection.
                        </Alert>
                    </>
                )}

                {/* 13. Policy Check */}
                {/* 13. Policy Check */}
                {filterType === 'Policy_Check' && (
                    <>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={formData.useLinkedPolicy || false}
                                    onChange={(e) => handleChange('useLinkedPolicy', e.target.checked)}
                                    size="small"
                                />
                            }
                            label="Use Linked Policy"
                            sx={{ mb: 2 }}
                        />

                        {formData.useLinkedPolicy ? (
                             <TextField
                                select
                                label="Select Policy"
                                fullWidth
                                size="small"
                                value={formData.policyId || ''}
                                onChange={(e) => handleChange('policyId', e.target.value)}
                                helperText={validationRules.length === 0 ? "No policies found for this object." : "Select a pre-defined validation rule."}
                            >
                                {validationRules.map((rule: any) => (
                                    <MenuItem key={rule.id} value={rule.id}>
                                        {rule.rule_name}
                                    </MenuItem>
                                ))}
                            </TextField>
                        ) : (
                            <TextField 
                                label="Policy Definition"
                                fullWidth
                                multiline
                                rows={3}
                                size="small"
                                value={formData.policy || ''}
                                onChange={(e) => handleChange('policy', e.target.value)}
                                placeholder="e.g. block_if(risk_score > 80)"
                            />
                        )}
                       
                        <Stack direction="row" spacing={2} sx={{ mt: 2 }}>
                             <TextField 
                                select
                                label="Severity"
                                fullWidth
                                size="small"
                                value={formData.severity || 'critical'}
                                onChange={(e) => handleChange('severity', e.target.value)}
                            >
                                <MenuItem value="critical">Critical (Block)</MenuItem>
                                <MenuItem value="warning">Warning (Override)</MenuItem>
                                <MenuItem value="info">Info (Log Only)</MenuItem>
                            </TextField>
                        </Stack>
                        
                        <TextField 
                            label="Custom Error Message"
                            fullWidth
                            size="small"
                            value={formData.errorMessage || ''}
                            onChange={(e) => handleChange('errorMessage', e.target.value)}
                            placeholder="e.g. Credit score too low for approval"
                            sx={{ mt: 2 }}
                        />
                    </>
                )}

                {/* 14. Durable Ledger */}
                {filterType === 'Durable_Ledger' && (
                    <TextField 
                        select
                        label="Ledger Type"
                        fullWidth
                        size="small"
                        value={formData.ledgerType || 'immutable'}
                        onChange={(e) => handleChange('ledgerType', e.target.value)}
                    >
                        <MenuItem value="immutable">Immutable (Write Once)</MenuItem>
                        <MenuItem value="audit">Audit Log</MenuItem>
                        <MenuItem value="ephemeral">Ephemeral</MenuItem>
                    </TextField>
                )}

                {/* 15. Record Create */}
                {filterType === 'Record_Create' && (
                    <>
                        <TextField 
                            select
                            label="Target Business Object"
                            fullWidth
                            size="small"
                            value={formData.targetBOId || ''}
                            onChange={(e) => handleChange('targetBOId', e.target.value)}
                            helperText="Select the object type to create"
                            sx={{ mb: 2 }}
                        >
                            {availableBOs.map((bo: any) => (
                                <MenuItem key={bo.id} value={bo.id}>{bo.display_name} ({bo.name})</MenuItem>
                            ))}
                        </TextField>

                        {formData.targetBOId && targetBOFields.length > 0 && (
                            <Box sx={{ mt: 2 }}>
                                <Typography variant="caption" fontWeight="bold" sx={{ mb: 1, display: 'block' }}>
                                    Field Mapping (Target &lt;- Source)
                                </Typography>
                                {targetBOFields.map((field: any) => (
                                    <Stack direction="row" spacing={1} key={field.name} sx={{ mb: 1.5, alignItems: 'center' }}>
                                         <Typography variant="body2" sx={{ width: '40%', fontSize: '0.75rem', overflow:'hidden', textOverflow:'ellipsis' }}>
                                            {field.label} <span style={{color:'#94a3b8'}}>({field.name})</span>
                                         </Typography>
                                         <TextField
                                            select
                                            size="small"
                                            fullWidth
                                            value={formData.mapping?.[field.name] || ''}
                                            onChange={(e) => handleChange('mapping', { ...(formData.mapping || {}), [field.name]: e.target.value })}
                                            sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.8 } }}
                                         >
                                            <MenuItem value=""><em>None</em></MenuItem>
                                            <Divider textAlign="left"><Typography variant="caption">Variables</Typography></Divider>
                                            {pipelineVariables.map(v => (
                                                <MenuItem key={v.name} value={`$${v.name}`}>{v.label}</MenuItem>
                                            ))}
                                            <Divider textAlign="left"><Typography variant="caption">Context: {selectedBO?.display_name}</Typography></Divider>
                                             {selectedBO?.fields?.map((f:any) => (
                                                <MenuItem key={f.name} value={`source.${f.name}`}>{f.label}</MenuItem>
                                            ))}
                                         </TextField>
                                    </Stack>
                                ))}
                            </Box>
                        )}
                    </>
                )}

                {/* 16. Sub-Pipeline - Composable Workflow Execution */}
                {(filterType === 'subPipeline' || filterType === 'Child_Pipeline') && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Execute another pipeline as a child and wait for its result. 
                            The child pipeline will be terminated if this parent is cancelled.
                        </Alert>
                        
                        <TextField 
                            label="Pipeline ID"
                            fullWidth
                            size="small"
                            value={formData.pipeline_id || ''}
                            onChange={(e) => handleChange('pipeline_id', e.target.value)}
                            placeholder="e.g. global_kyc_verification_v1"
                            helperText="The unique identifier of the sub-pipeline to execute"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Output Variable"
                            fullWidth
                            size="small"
                            value={formData.output_variable || ''}
                            onChange={(e) => handleChange('output_variable', e.target.value)}
                            placeholder="e.g. kyc_results"
                            helperText="Variable name to store the sub-pipeline's output in parent context"
                            sx={{ mb: 2 }}
                        />
                        
                        <Divider sx={{ my: 2 }} />
                        
                        <Typography variant="subtitle2" fontWeight="bold" sx={{ mb: 1 }}>
                            Input Mapping
                        </Typography>
                        <Typography variant="caption" color="text.secondary" sx={{ mb: 2, display: 'block' }}>
                            Map parent context values to sub-pipeline input parameters
                        </Typography>
                        
                        {/* Dynamic input mapping */}
                        {(formData.input_mapping_keys || ['']).map((key: string, idx: number) => (
                            <Stack direction="row" spacing={1} key={idx} sx={{ mb: 1 }}>
                                <TextField
                                    size="small"
                                    placeholder="Sub-Pipeline Input Key"
                                    value={key}
                                    onChange={(e) => {
                                        const newKeys = [...(formData.input_mapping_keys || [''])];
                                        newKeys[idx] = e.target.value;
                                        handleChange('input_mapping_keys', newKeys);
                                    }}
                                    sx={{ width: '40%' }}
                                />
                                <TextField
                                    select
                                    size="small"
                                    fullWidth
                                    value={(formData.input_mapping_values || [])[idx] || ''}
                                    onChange={(e) => {
                                        const newVals = [...(formData.input_mapping_values || [''])];
                                        newVals[idx] = e.target.value;
                                        handleChange('input_mapping_values', newVals);
                                    }}
                                >
                                    <MenuItem value=""><em>Select source</em></MenuItem>
                                    <Divider textAlign="left"><Typography variant="caption">Input Fields</Typography></Divider>
                                    {selectedBO?.fields?.map((f: any) => (
                                        <MenuItem key={f.name} value={`$.input.${f.name}`}>{f.label}</MenuItem>
                                    ))}
                                    <Divider textAlign="left"><Typography variant="caption">Node Outputs</Typography></Divider>
                                    {pipelineVariables.map(v => (
                                        <MenuItem key={v.name} value={`$.nodes.${v.name}.output`}>{v.label}</MenuItem>
                                    ))}
                                    <Divider textAlign="left"><Typography variant="caption">Special</Typography></Divider>
                                    <MenuItem value="$.now">Current Timestamp</MenuItem>
                                </TextField>
                            </Stack>
                        ))}
                        <Button
                            size="small"
                            onClick={() => {
                                handleChange('input_mapping_keys', [...(formData.input_mapping_keys || ['']), '']);
                                handleChange('input_mapping_values', [...(formData.input_mapping_values || ['']), '']);
                            }}
                            sx={{ mt: 1 }}
                        >
                            + Add Mapping
                        </Button>
                    </>
                )}

                {/* 17. Parallel - Execute branches concurrently */}
                {filterType === 'parallel' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Execute multiple branches concurrently and wait for all to complete.
                        </Alert>
                        
                        <TextField 
                            label="Output Variable"
                            fullWidth
                            size="small"
                            value={formData.output_variable || ''}
                            onChange={(e) => handleChange('output_variable', e.target.value)}
                            placeholder="e.g. parallel_results"
                            helperText="Variable to store combined branch results"
                            sx={{ mb: 2 }}
                        />
                        
                        <Divider sx={{ my: 2 }} />
                        
                        <Typography variant="subtitle2" fontWeight="bold" sx={{ mb: 1 }}>
                            Branches
                        </Typography>
                        <Typography variant="caption" color="text.secondary" sx={{ mb: 2, display: 'block' }}>
                            Define branches to execute concurrently
                        </Typography>
                        
                        {(formData.branch_names || ['']).map((name: string, idx: number) => (
                            <Stack direction="row" spacing={1} key={idx} sx={{ mb: 1 }}>
                                <TextField
                                    size="small"
                                    placeholder="Branch Name"
                                    value={name}
                                    onChange={(e) => {
                                        const newNames = [...(formData.branch_names || [''])];
                                        newNames[idx] = e.target.value;
                                        handleChange('branch_names', newNames);
                                    }}
                                    sx={{ width: '40%' }}
                                />
                                <TextField
                                    size="small"
                                    placeholder="Node IDs (comma-separated)"
                                    value={(formData.branch_node_ids || [])[idx] || ''}
                                    onChange={(e) => {
                                        const newIds = [...(formData.branch_node_ids || [''])];
                                        newIds[idx] = e.target.value;
                                        handleChange('branch_node_ids', newIds);
                                    }}
                                    fullWidth
                                />
                            </Stack>
                        ))}
                        <Button
                            size="small"
                            onClick={() => {
                                handleChange('branch_names', [...(formData.branch_names || ['']), '']);
                                handleChange('branch_node_ids', [...(formData.branch_node_ids || ['']), '']);
                            }}
                            sx={{ mt: 1 }}
                        >
                            + Add Branch
                        </Button>
                    </>
                )}

                {/* 18. ForEach - Iterate over collection */}
                {filterType === 'forEach' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Iterate over a collection and execute body nodes for each item.
                        </Alert>
                        
                        <TextField 
                            select
                            label="Collection"
                            fullWidth
                            size="small"
                            value={formData.collection || ''}
                            onChange={(e) => handleChange('collection', e.target.value)}
                            helperText="The array to iterate over"
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value=""><em>Select collection</em></MenuItem>
                            {pipelineVariables.map(v => (
                                <MenuItem key={v.name} value={`$.${v.name}`}>{v.label}</MenuItem>
                            ))}
                            <MenuItem value="$.items">$.items</MenuItem>
                            <MenuItem value="$.records">$.records</MenuItem>
                        </TextField>
                        
                        <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
                            <TextField 
                                label="Item Variable"
                                size="small"
                                value={formData.item_variable || 'item'}
                                onChange={(e) => handleChange('item_variable', e.target.value)}
                                helperText="Current item"
                                fullWidth
                            />
                            <TextField 
                                label="Index Variable"
                                size="small"
                                value={formData.index_variable || 'index'}
                                onChange={(e) => handleChange('index_variable', e.target.value)}
                                helperText="Current index"
                                fullWidth
                            />
                        </Stack>
                        
                        <TextField 
                            select
                            label="Execution Mode"
                            fullWidth
                            size="small"
                            value={formData.mode || 'sequential'}
                            onChange={(e) => handleChange('mode', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value="sequential">Sequential (one at a time)</MenuItem>
                            <MenuItem value="parallel">Parallel (concurrent)</MenuItem>
                        </TextField>
                        
                        {formData.mode === 'parallel' && (
                            <TextField 
                                label="Max Concurrency"
                                type="number"
                                fullWidth
                                size="small"
                                value={formData.max_concurrency || 10}
                                onChange={(e) => handleChange('max_concurrency', parseInt(e.target.value))}
                                helperText="Maximum simultaneous iterations"
                                sx={{ mb: 2 }}
                            />
                        )}
                        
                        <TextField 
                            label="Output Variable"
                            fullWidth
                            size="small"
                            value={formData.output_variable || ''}
                            onChange={(e) => handleChange('output_variable', e.target.value)}
                            placeholder="e.g. processed_items"
                            helperText="Variable to store collected results"
                        />
                    </>
                )}

                {/* 19. Wait - Timer/Delay */}
                {filterType === 'wait' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Pause workflow execution for a specified duration.
                        </Alert>
                        
                        <TextField 
                            select
                            label="Wait Type"
                            fullWidth
                            size="small"
                            value={formData.wait_type || 'duration'}
                            onChange={(e) => handleChange('wait_type', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value="duration">Duration (relative)</MenuItem>
                            <MenuItem value="until">Until Time (absolute)</MenuItem>
                        </TextField>
                        
                        {formData.wait_type !== 'until' && (
                            <TextField 
                                label="Duration"
                                fullWidth
                                size="small"
                                value={formData.duration || ''}
                                onChange={(e) => handleChange('duration', e.target.value)}
                                placeholder="e.g. 5m, 1h, 24h"
                                helperText="Go duration format: 5m (5 minutes), 1h (1 hour), 24h (1 day)"
                            />
                        )}
                        
                        {formData.wait_type === 'until' && (
                            <TextField 
                                label="Until Time"
                                type="datetime-local"
                                fullWidth
                                size="small"
                                value={formData.until_time || ''}
                                onChange={(e) => handleChange('until_time', e.target.value)}
                                helperText="Wait until this specific date/time"
                                InputLabelProps={{ shrink: true }}
                            />
                        )}
                    </>
                )}

                {/* 20. Wait For Event - Signal Listener */}
                {filterType === 'waitForEvent' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Pause execution until a specific signal/event is received.
                        </Alert>
                        
                        <TextField 
                            label="Signal Name"
                            fullWidth
                            size="small"
                            value={formData.signal_name || ''}
                            onChange={(e) => handleChange('signal_name', e.target.value)}
                            placeholder="e.g. approval_received, payment_confirmed"
                            helperText="The name of the Temporal signal to wait for"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Timeout"
                            fullWidth
                            size="small"
                            value={formData.timeout || '24h'}
                            onChange={(e) => handleChange('timeout', e.target.value)}
                            placeholder="e.g. 1h, 24h, 7d"
                            helperText="Maximum time to wait for the signal"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            select
                            label="On Timeout"
                            fullWidth
                            size="small"
                            value={formData.timeout_action || 'continue'}
                            onChange={(e) => handleChange('timeout_action', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value="continue">Continue (proceed without signal)</MenuItem>
                            <MenuItem value="fail">Fail (stop workflow)</MenuItem>
                        </TextField>
                        
                        <TextField 
                            label="Output Variable"
                            fullWidth
                            size="small"
                            value={formData.output_variable || ''}
                            onChange={(e) => handleChange('output_variable', e.target.value)}
                            placeholder="e.g. signal_data"
                            helperText="Variable to store the signal payload"
                        />
                    </>
                )}

                {/* 21. Try/Catch - Error Handling */}
                {filterType === 'tryCatch' && (
                    <>
                        <Alert severity="warning" sx={{ mb: 2 }}>
                            Wrap nodes in try/catch/finally blocks for error handling.
                        </Alert>
                        
                        <TextField 
                            label="Try Node IDs"
                            fullWidth
                            size="small"
                            value={formData.try_node_ids || ''}
                            onChange={(e) => handleChange('try_node_ids', e.target.value)}
                            placeholder="node_1, node_2, node_3"
                            helperText="Comma-separated list of nodes to execute in try block"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Catch Node IDs"
                            fullWidth
                            size="small"
                            value={formData.catch_node_ids || ''}
                            onChange={(e) => handleChange('catch_node_ids', e.target.value)}
                            placeholder="error_handler_1, error_handler_2"
                            helperText="Nodes to execute if an error occurs"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Finally Node IDs (optional)"
                            fullWidth
                            size="small"
                            value={formData.finally_node_ids || ''}
                            onChange={(e) => handleChange('finally_node_ids', e.target.value)}
                            placeholder="cleanup_node"
                            helperText="Nodes to always execute regardless of success/failure"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Error Variable"
                            fullWidth
                            size="small"
                            value={formData.error_variable || 'last_error'}
                            onChange={(e) => handleChange('error_variable', e.target.value)}
                            helperText="Variable to store error information"
                        />
                    </>
                )}

                {/* 22. Compensate - Saga Pattern */}
                {filterType === 'compensate' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Register compensation/rollback actions for saga pattern.
                            Compensations execute in reverse order on failure.
                        </Alert>
                        
                        <TextField 
                            label="Compensation Name"
                            fullWidth
                            size="small"
                            value={formData.compensation_name || ''}
                            onChange={(e) => handleChange('compensation_name', e.target.value)}
                            placeholder="e.g. refund_payment, restore_inventory"
                            helperText="Identifier for this compensation action"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Compensation Node IDs"
                            fullWidth
                            size="small"
                            value={formData.compensation_node_ids || ''}
                            onChange={(e) => handleChange('compensation_node_ids', e.target.value)}
                            placeholder="rollback_1, rollback_2"
                            helperText="Comma-separated list of nodes to execute for rollback"
                            sx={{ mb: 2 }}
                        />
                        
                        <Typography variant="caption" color="text.secondary">
                            Compensations are stored and executed in LIFO order (last registered, first executed) when the workflow fails.
                        </Typography>
                    </>
                )}

                {/* 23. Switch - Multi-way Branch */}
                {filterType === 'switch' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Route to different paths based on a value (like a switch statement).
                        </Alert>
                        
                        <TextField 
                            select
                            label="Expression"
                            fullWidth
                            size="small"
                            value={formData.expression || ''}
                            onChange={(e) => handleChange('expression', e.target.value)}
                            helperText="The value to evaluate for switching"
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value=""><em>Select field</em></MenuItem>
                            {selectedBO?.fields?.map((f: any) => (
                                <MenuItem key={f.name} value={`$.input.${f.name}`}>{f.label}</MenuItem>
                            ))}
                            {pipelineVariables.map(v => (
                                <MenuItem key={v.name} value={`$.${v.name}`}>{v.label}</MenuItem>
                            ))}
                        </TextField>
                        
                        <Divider sx={{ my: 2 }} />
                        
                        <Typography variant="subtitle2" fontWeight="bold" sx={{ mb: 1 }}>
                            Cases
                        </Typography>
                        
                        {(formData.case_values || ['']).map((value: string, idx: number) => (
                            <Stack direction="row" spacing={1} key={idx} sx={{ mb: 1 }}>
                                <TextField
                                    size="small"
                                    placeholder="Value"
                                    value={value}
                                    onChange={(e) => {
                                        const newVals = [...(formData.case_values || [''])];
                                        newVals[idx] = e.target.value;
                                        handleChange('case_values', newVals);
                                    }}
                                    sx={{ width: '40%' }}
                                />
                                <TextField
                                    size="small"
                                    placeholder="Target Node ID"
                                    value={(formData.case_targets || [])[idx] || ''}
                                    onChange={(e) => {
                                        const newTargets = [...(formData.case_targets || [''])];
                                        newTargets[idx] = e.target.value;
                                        handleChange('case_targets', newTargets);
                                    }}
                                    fullWidth
                                />
                            </Stack>
                        ))}
                        <Button
                            size="small"
                            onClick={() => {
                                handleChange('case_values', [...(formData.case_values || ['']), '']);
                                handleChange('case_targets', [...(formData.case_targets || ['']), '']);
                            }}
                            sx={{ mt: 1, mb: 2 }}
                        >
                            + Add Case
                        </Button>
                        
                        <TextField 
                            label="Default Node ID"
                            fullWidth
                            size="small"
                            value={formData.default_id || ''}
                            onChange={(e) => handleChange('default_id', e.target.value)}
                            helperText="Node to go to if no case matches"
                        />
                    </>
                )}

                {/* 24. Publish Event - Multi-Broker Message Bus */}
                {filterType === 'publishEvent' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Publish events to RabbitMQ, Kafka, AWS SQS/SNS, Azure Service Bus, or GCP Pub/Sub.
                        </Alert>
                        
                        <TextField 
                            label="Event Name"
                            fullWidth
                            size="small"
                            value={formData.event_name || ''}
                            onChange={(e) => handleChange('event_name', e.target.value)}
                            placeholder="e.g. order.created, payment.processed"
                            helperText="The event type name"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            select
                            label="Message Broker"
                            fullWidth
                            size="small"
                            value={formData.broker_type || 'rabbitmq'}
                            onChange={(e) => handleChange('broker_type', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value="rabbitmq">RabbitMQ</MenuItem>
                            <MenuItem value="kafka">Apache Kafka</MenuItem>
                            <MenuItem value="aws_sqs">AWS SQS</MenuItem>
                            <MenuItem value="aws_sns">AWS SNS</MenuItem>
                            <MenuItem value="azure_servicebus">Azure Service Bus</MenuItem>
                            <MenuItem value="azure_eventhub">Azure Event Hub</MenuItem>
                            <MenuItem value="gcp_pubsub">Google Cloud Pub/Sub</MenuItem>
                        </TextField>
                        
                        {/* RabbitMQ Fields */}
                        {(formData.broker_type === 'rabbitmq' || !formData.broker_type) && (
                            <>
                                <TextField 
                                    label="Exchange"
                                    fullWidth
                                    size="small"
                                    value={formData.exchange || 'titan.events'}
                                    onChange={(e) => handleChange('exchange', e.target.value)}
                                    helperText="RabbitMQ exchange"
                                    sx={{ mb: 2 }}
                                />
                                <TextField 
                                    label="Routing Key"
                                    fullWidth
                                    size="small"
                                    value={formData.routing_key || ''}
                                    onChange={(e) => handleChange('routing_key', e.target.value)}
                                    placeholder="e.g. orders.us-east.created"
                                    helperText="Message routing key"
                                    sx={{ mb: 2 }}
                                />
                            </>
                        )}
                        
                        {/* Kafka Fields */}
                        {formData.broker_type === 'kafka' && (
                            <>
                                <TextField 
                                    label="Topic"
                                    fullWidth
                                    size="small"
                                    value={formData.topic || ''}
                                    onChange={(e) => handleChange('topic', e.target.value)}
                                    placeholder="e.g. orders-topic"
                                    helperText="Kafka topic"
                                    sx={{ mb: 2 }}
                                />
                                <TextField 
                                    label="Message Key (optional)"
                                    fullWidth
                                    size="small"
                                    value={formData.key || ''}
                                    onChange={(e) => handleChange('key', e.target.value)}
                                    placeholder="$.order_id or literal key"
                                    helperText="Key for partition assignment"
                                    sx={{ mb: 2 }}
                                />
                            </>
                        )}
                        
                        {/* AWS SQS Fields */}
                        {formData.broker_type === 'aws_sqs' && (
                            <>
                                <TextField 
                                    label="Queue URL"
                                    fullWidth
                                    size="small"
                                    value={formData.queue_url || ''}
                                    onChange={(e) => handleChange('queue_url', e.target.value)}
                                    placeholder="https://sqs.us-east-1.amazonaws.com/123456/my-queue"
                                    helperText="SQS Queue URL"
                                    sx={{ mb: 2 }}
                                />
                                <TextField 
                                    label="AWS Region"
                                    fullWidth
                                    size="small"
                                    value={formData.region || 'us-east-1'}
                                    onChange={(e) => handleChange('region', e.target.value)}
                                    sx={{ mb: 2 }}
                                />
                            </>
                        )}
                        
                        {/* AWS SNS Fields */}
                        {formData.broker_type === 'aws_sns' && (
                            <>
                                <TextField 
                                    label="Topic ARN"
                                    fullWidth
                                    size="small"
                                    value={formData.topic_arn || ''}
                                    onChange={(e) => handleChange('topic_arn', e.target.value)}
                                    placeholder="arn:aws:sns:us-east-1:123456:my-topic"
                                    helperText="SNS Topic ARN"
                                    sx={{ mb: 2 }}
                                />
                                <TextField 
                                    label="AWS Region"
                                    fullWidth
                                    size="small"
                                    value={formData.region || 'us-east-1'}
                                    onChange={(e) => handleChange('region', e.target.value)}
                                    sx={{ mb: 2 }}
                                />
                            </>
                        )}
                        
                        {/* Azure Service Bus Fields */}
                        {formData.broker_type === 'azure_servicebus' && (
                            <>
                                <TextField 
                                    label="Namespace"
                                    fullWidth
                                    size="small"
                                    value={formData.namespace || ''}
                                    onChange={(e) => handleChange('namespace', e.target.value)}
                                    placeholder="mycompany.servicebus.windows.net"
                                    helperText="Service Bus namespace"
                                    sx={{ mb: 2 }}
                                />
                                <TextField 
                                    label="Queue/Topic Name"
                                    fullWidth
                                    size="small"
                                    value={formData.queue_name || ''}
                                    onChange={(e) => handleChange('queue_name', e.target.value)}
                                    placeholder="orders-queue"
                                    sx={{ mb: 2 }}
                                />
                            </>
                        )}
                        
                        {/* Azure Event Hub Fields */}
                        {formData.broker_type === 'azure_eventhub' && (
                            <>
                                <TextField 
                                    label="Namespace"
                                    fullWidth
                                    size="small"
                                    value={formData.namespace || ''}
                                    onChange={(e) => handleChange('namespace', e.target.value)}
                                    placeholder="mycompany.servicebus.windows.net"
                                    sx={{ mb: 2 }}
                                />
                                <TextField 
                                    label="Event Hub Name"
                                    fullWidth
                                    size="small"
                                    value={formData.eventhub_name || ''}
                                    onChange={(e) => handleChange('eventhub_name', e.target.value)}
                                    placeholder="orders-hub"
                                    sx={{ mb: 2 }}
                                />
                            </>
                        )}
                        
                        {/* GCP Pub/Sub Fields */}
                        {formData.broker_type === 'gcp_pubsub' && (
                            <>
                                <TextField 
                                    label="Project ID"
                                    fullWidth
                                    size="small"
                                    value={formData.project_id || ''}
                                    onChange={(e) => handleChange('project_id', e.target.value)}
                                    placeholder="my-gcp-project"
                                    sx={{ mb: 2 }}
                                />
                                <TextField 
                                    label="Topic Name"
                                    fullWidth
                                    size="small"
                                    value={formData.topic_name || ''}
                                    onChange={(e) => handleChange('topic_name', e.target.value)}
                                    placeholder="orders-topic"
                                    sx={{ mb: 2 }}
                                />
                            </>
                        )}
                        
                        <Divider sx={{ my: 2 }} />
                        <Typography variant="subtitle2" fontWeight="bold" sx={{ mb: 1 }}>
                            Payload Mapping
                        </Typography>
                        
                        {(formData.payload_keys || ['']).map((key: string, idx: number) => (
                            <Stack direction="row" spacing={1} key={idx} sx={{ mb: 1 }}>
                                <TextField
                                    size="small"
                                    placeholder="Payload Key"
                                    value={key}
                                    onChange={(e) => {
                                        const newKeys = [...(formData.payload_keys || [''])];
                                        newKeys[idx] = e.target.value;
                                        handleChange('payload_keys', newKeys);
                                    }}
                                    sx={{ width: '40%' }}
                                />
                                <TextField
                                    select
                                    size="small"
                                    fullWidth
                                    value={(formData.payload_values || [])[idx] || ''}
                                    onChange={(e) => {
                                        const newVals = [...(formData.payload_values || [''])];
                                        newVals[idx] = e.target.value;
                                        handleChange('payload_values', newVals);
                                    }}
                                >
                                    <MenuItem value=""><em>Select source</em></MenuItem>
                                    {selectedBO?.fields?.map((f: any) => (
                                        <MenuItem key={f.name} value={`$.input.${f.name}`}>{f.label}</MenuItem>
                                    ))}
                                    {pipelineVariables.map(v => (
                                        <MenuItem key={v.name} value={`$.${v.name}`}>{v.label}</MenuItem>
                                    ))}
                                </TextField>
                            </Stack>
                        ))}
                        <Button
                            size="small"
                            onClick={() => {
                                handleChange('payload_keys', [...(formData.payload_keys || ['']), '']);
                                handleChange('payload_values', [...(formData.payload_values || ['']), '']);
                            }}
                        >
                            + Add Field
                        </Button>
                    </>
                )}

                {/* 25. Alert - Notification */}
                {filterType === 'alert' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Send a notification via email, Slack, or webhook.
                        </Alert>
                        
                        <TextField 
                            select
                            label="Channel"
                            fullWidth
                            size="small"
                            value={formData.channel || 'email'}
                            onChange={(e) => handleChange('channel', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value="email">Email</MenuItem>
                            <MenuItem value="slack">Slack</MenuItem>
                            <MenuItem value="webhook">Webhook</MenuItem>
                            <MenuItem value="sms">SMS</MenuItem>
                        </TextField>
                        
                        <TextField 
                            select
                            label="Severity"
                            fullWidth
                            size="small"
                            value={formData.severity || 'info'}
                            onChange={(e) => handleChange('severity', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value="info">Info</MenuItem>
                            <MenuItem value="warning">Warning</MenuItem>
                            <MenuItem value="error">Error</MenuItem>
                            <MenuItem value="critical">Critical</MenuItem>
                        </TextField>
                        
                        <TextField 
                            label="Subject"
                            fullWidth
                            size="small"
                            value={formData.subject || ''}
                            onChange={(e) => handleChange('subject', e.target.value)}
                            placeholder="Alert subject line"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Message"
                            fullWidth
                            size="small"
                            multiline
                            rows={3}
                            value={formData.message || ''}
                            onChange={(e) => handleChange('message', e.target.value)}
                            placeholder="Use {{field_name}} for variable interpolation"
                            helperText="Message body - supports template variables"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Recipients"
                            fullWidth
                            size="small"
                            value={formData.recipients || ''}
                            onChange={(e) => handleChange('recipients', e.target.value)}
                            placeholder="email1@example.com, email2@example.com"
                            helperText="Comma-separated list of recipients"
                        />
                    </>
                )}

                {/* ==================== LLM-ENHANCED STEPS ==================== */}

                {/* 26. Interpretation - Parse unstructured to structured */}
                {filterType === 'Interpretation' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Use LLM to parse unstructured input into structured data.
                        </Alert>
                        
                        <TextField 
                            select
                            label="LLM Profile"
                            fullWidth
                            size="small"
                            value={formData.profile_id || 'interpretation_default'}
                            onChange={(e) => handleChange('profile_id', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value="interpretation_default">Default Interpretation</MenuItem>
                            <MenuItem value="custom">Custom Profile</MenuItem>
                        </TextField>
                        
                        <TextField 
                            label="Fields to Extract"
                            fullWidth
                            size="small"
                            multiline
                            rows={3}
                            value={formData.fields_to_extract || ''}
                            onChange={(e) => handleChange('fields_to_extract', e.target.value)}
                            placeholder="name, email, phone, address"
                            helperText="Comma-separated list of fields to extract"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            select
                            label="Input Source"
                            fullWidth
                            size="small"
                            value={formData.input_source || ''}
                            onChange={(e) => handleChange('input_source', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value=""><em>Select input source</em></MenuItem>
                            {selectedBO?.fields?.map((f: any) => (
                                <MenuItem key={f.name} value={`$.input.${f.name}`}>{f.label}</MenuItem>
                            ))}
                            {pipelineVariables.map(v => (
                                <MenuItem key={v.name} value={`$.${v.name}`}>{v.label}</MenuItem>
                            ))}
                        </TextField>
                        
                        <TextField 
                            label="Output Variable"
                            fullWidth
                            size="small"
                            value={formData.output_variable || 'extracted_data'}
                            onChange={(e) => handleChange('output_variable', e.target.value)}
                            helperText="Variable name to store extracted result"
                        />
                    </>
                )}

                {/* 27. Classification - Categorize input */}
                {filterType === 'Classification' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Use LLM to classify input into categories.
                        </Alert>
                        
                        <TextField 
                            label="Categories"
                            fullWidth
                            size="small"
                            multiline
                            rows={3}
                            value={formData.categories || ''}
                            onChange={(e) => handleChange('categories', e.target.value)}
                            placeholder="high_risk, medium_risk, low_risk"
                            helperText="Comma-separated list of valid categories"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            select
                            label="Input to Classify"
                            fullWidth
                            size="small"
                            value={formData.input_source || ''}
                            onChange={(e) => handleChange('input_source', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value=""><em>Select input source</em></MenuItem>
                            {selectedBO?.fields?.map((f: any) => (
                                <MenuItem key={f.name} value={`$.input.${f.name}`}>{f.label}</MenuItem>
                            ))}
                            {pipelineVariables.map(v => (
                                <MenuItem key={v.name} value={`$.${v.name}`}>{v.label}</MenuItem>
                            ))}
                        </TextField>
                        
                        <TextField 
                            label="Output Variable"
                            fullWidth
                            size="small"
                            value={formData.output_variable || 'classification'}
                            onChange={(e) => handleChange('output_variable', e.target.value)}
                            helperText="Variable name to store classification result"
                        />
                    </>
                )}

                {/* 28. Drafting - Generate text for review */}
                {filterType === 'Drafting' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Use LLM to draft professional text for human review.
                        </Alert>
                        
                        <TextField 
                            label="Audience"
                            fullWidth
                            size="small"
                            value={formData.audience || ''}
                            onChange={(e) => handleChange('audience', e.target.value)}
                            placeholder="e.g., Client, Advisor, Compliance Team"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Purpose"
                            fullWidth
                            size="small"
                            value={formData.purpose || ''}
                            onChange={(e) => handleChange('purpose', e.target.value)}
                            placeholder="e.g., Explain portfolio changes, Request documents"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Context / Instructions"
                            fullWidth
                            size="small"
                            multiline
                            rows={4}
                            value={formData.context || ''}
                            onChange={(e) => handleChange('context', e.target.value)}
                            placeholder="Additional context or specific instructions for the draft"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Output Variable"
                            fullWidth
                            size="small"
                            value={formData.output_variable || 'draft_text'}
                            onChange={(e) => handleChange('output_variable', e.target.value)}
                            helperText="Variable name to store the generated draft"
                        />
                    </>
                )}

                {/* 29. Recommendation - Generate constrained recommendations */}
                {filterType === 'Recommendation' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Use LLM to generate recommendations constrained by policy.
                        </Alert>
                        
                        <TextField 
                            label="Context"
                            fullWidth
                            size="small"
                            multiline
                            rows={3}
                            value={formData.context || ''}
                            onChange={(e) => handleChange('context', e.target.value)}
                            placeholder="Describe the situation requiring a recommendation"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Constraints"
                            fullWidth
                            size="small"
                            multiline
                            rows={2}
                            value={formData.constraints || ''}
                            onChange={(e) => handleChange('constraints', e.target.value)}
                            placeholder="e.g., Must not exceed $100K, Must be diversified"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Policies to Comply With"
                            fullWidth
                            size="small"
                            value={formData.policies || ''}
                            onChange={(e) => handleChange('policies', e.target.value)}
                            placeholder="e.g., Suitability, Risk Limits, Regulatory"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Output Variable"
                            fullWidth
                            size="small"
                            value={formData.output_variable || 'recommendation'}
                            onChange={(e) => handleChange('output_variable', e.target.value)}
                            helperText="Variable name to store the recommendation"
                        />
                    </>
                )}

                {/* 30. ExceptionExplanation - Explain decisions/errors */}
                {filterType === 'ExceptionExplanation' && (
                    <>
                        <Alert severity="info" sx={{ mb: 2 }}>
                            Use LLM to explain exceptions, rejections, or decisions in plain language.
                        </Alert>
                        
                        <TextField 
                            select
                            label="Decision/Exception Source"
                            fullWidth
                            size="small"
                            value={formData.decision_source || ''}
                            onChange={(e) => handleChange('decision_source', e.target.value)}
                            sx={{ mb: 2 }}
                        >
                            <MenuItem value=""><em>Select source</em></MenuItem>
                            <MenuItem value="$.validation_result">Last Validation Result</MenuItem>
                            <MenuItem value="$._error">Last Error</MenuItem>
                            {pipelineVariables.map(v => (
                                <MenuItem key={v.name} value={`$.${v.name}`}>{v.label}</MenuItem>
                            ))}
                        </TextField>
                        
                        <TextField 
                            label="Additional Technical Details"
                            fullWidth
                            size="small"
                            multiline
                            rows={3}
                            value={formData.technical_details || ''}
                            onChange={(e) => handleChange('technical_details', e.target.value)}
                            placeholder="Any additional technical context to include"
                            sx={{ mb: 2 }}
                        />
                        
                        <TextField 
                            label="Output Variable"
                            fullWidth
                            size="small"
                            value={formData.output_variable || 'explanation'}
                            onChange={(e) => handleChange('output_variable', e.target.value)}
                            helperText="Variable name to store the explanation"
                        />
                    </>
                )}

                <Button 
                    variant="contained" 
                    fullWidth 
                    size="large"
                    startIcon={<SaveIcon />}
                    onClick={handleSave}
                    sx={{ mt: 2 }}
                >
                    Update Configuration
                </Button>
            </Stack>
        )}
      </div>

      {/* Code Tab */}
      <div role="tabpanel" hidden={tabIndex !== 1} style={{ flexGrow: 1, overflowY: 'auto', backgroundColor: '#1e293b' }}>
        {tabIndex === 1 && (
            <Box sx={{ p: 2 }}>
                <Typography variant="caption" sx={{ fontFamily: 'monospace', whiteSpace: 'pre-wrap', color: '#f8fafc' }}>
                    {filterType === 'Limit' ? getLimitLogic() : 
                     filterType === 'Formula' ? `// Custom Logic\n${formData.expression || ''}` : 
                     filterType === 'Conditional' ? `if (${formData.condition || 'true'}) {\n  return "path_A"\n} else {\n  return "path_B"\n}` :
                     filterType === 'Approval_Gate' ? `require_approval(roles=${JSON.stringify(formData.approverRoles || [])}, timeout="${formData.timeout || 24}h")` :
                     filterType === 'External_API' ? `http.${formData.method || 'POST'}("${formData.url || 'http://localhost'}")` :
                     filterType === 'Transform' ? `// Transform\n${formData.script || 'pass'}` :
                     filterType === 'AI_Anomaly' ? `detect_anomaly(model="${formData.modelVersion || 'v2'}", sensitivity=${(formData.sensitivity || 50) / 100})` :
                     `// Standard Policy Logic for ${filterType}`}
                </Typography>
            </Box>
        )}
      </div>
    </Box>
  );
};

export default ConfigPanel;
