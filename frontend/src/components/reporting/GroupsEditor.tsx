import type { FC } from 'react';
import { Box, Grid, Typography, Button, IconButton, TextField, FormControl, InputLabel, Select, MenuItem, Divider, FormControlLabel, Switch } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import ExpressionInputControl from './ExpressionInputControl';
import { aggregateHandlers } from './reportingUtils';

type Aggregate = any;
type Group = any;

type Props = {
  groupDefinitions: Group[];
  onAddGroup: () => void;
  onRemoveGroup: (groupId: string) => void;
  onGroupChange: (groupId: string, key: string, value: any) => void;
  onAddAggregate: (groupId: string) => void;
  onAggregateChange: (groupId: string, aggregateId: string, key: string, value: any) => void;
  onRemoveAggregate: (groupId: string, aggregateId: string) => void;
};

const GroupsEditor: FC<Props> = ({ groupDefinitions, onAddGroup, onRemoveGroup, onGroupChange, onAddAggregate, onAggregateChange, onRemoveAggregate }) => {
  return (
    <>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="subtitle1">Groups & Aggregations</Typography>
        <Button size="small" startIcon={<AddIcon sx={{ fontSize: 16 }} />} onClick={onAddGroup}>Add Group</Button>
      </Box>

      {groupDefinitions.map((group) => (
        <Box key={group.id} sx={{ mb: 2, p: 2, border: '1px solid #e5e7eb', borderRadius: 1, position: 'relative', backgroundColor: '#fafafa' }}>
          <Box sx={{ position: 'absolute', top: 8, right: 8 }}>
            <IconButton size="small" onClick={() => onRemoveGroup(group.id)}><DeleteIcon fontSize="small" /></IconButton>
          </Box>
          <Grid container spacing={2}>
            <Grid size={{ 'xs': 12, 'sm': 4 }}><TextField fullWidth size="small" label="Group Name" value={group.name} onChange={(e) => onGroupChange(group.id, 'name', e.target.value)} /></Grid>
            <Grid size={{ 'xs': 12, 'sm': 4 }}>
              <ExpressionInputControl
                label="Expression"
                property={group.expression || ''}
                defaultFormula="=Fields!Region"
                onChange={(prop) => onGroupChange(group.id, 'expression', prop.isExpression ? prop.formula : prop.value)}
                renderStaticControl={(val, setVal) => (
                  <TextField fullWidth size="small" label="Expression" helperText="Example: =Fields!Region" value={String(val ?? '')} onChange={(e) => setVal(e.target.value)} />
                )}
              />
            </Grid>
            <Grid   size={{ xs: 12, sm: 4 }}><FormControl fullWidth size="small"><InputLabel>Parent Group</InputLabel><Select label="Parent Group" value={group.parent ?? ''} onChange={(e) => onGroupChange(group.id, 'parent', e.target.value ? e.target.value : null)}><MenuItem value="">(None)</MenuItem>{groupDefinitions.filter((c) => c.id !== group.id).map((candidate) => (<MenuItem key={`${group.id}_parent_${candidate.id}`} value={candidate.id}>{candidate.name}</MenuItem>))}</Select></FormControl></Grid>
            <Grid size={{ 'xs': 12, 'sm': 6 }}><FormControlLabel control={<Switch size="small" checked={Boolean(group.pageBreakBefore)} onChange={(e) => onGroupChange(group.id, 'pageBreakBefore', e.target.checked)} />} label="Page break before" /></Grid>
            <Grid size={{ 'xs': 12, 'sm': 6 }}><FormControlLabel control={<Switch size="small" checked={Boolean(group.pageBreakAfter)} onChange={(e) => onGroupChange(group.id, 'pageBreakAfter', e.target.checked)} />} label="Page break after" /></Grid>
          </Grid>
          <Divider sx={{ my: 2 }}>Aggregates</Divider>
          {group.aggregates.map((aggregate: Aggregate) => (
            <Grid container spacing={1.5} key={aggregate.id} alignItems="center" sx={{ mb: 1 }}>
              <Grid size={{ 'xs': 12, 'sm': 3 }}><TextField fullWidth size="small" label="Field" value={aggregate.field} onChange={(e) => onAggregateChange(group.id, aggregate.id, 'field', e.target.value)} /></Grid>
              <Grid size={{ 'xs': 12, 'sm': 3 }}><FormControl fullWidth size="small"><InputLabel>Function</InputLabel><Select label="Function" value={aggregate.function} onChange={(e) => onAggregateChange(group.id, aggregate.id, 'function', e.target.value)}>{Object.keys(aggregateHandlers).map((fn) => (<MenuItem key={`${aggregate.id}_${fn}`} value={fn}>{fn}</MenuItem>))}</Select></FormControl></Grid>
              <Grid   size={{ xs: 12, sm: 3 }}><FormControl fullWidth size="small"><InputLabel>Scope</InputLabel><Select label="Scope" value={aggregate.scope} onChange={(e) => onAggregateChange(group.id, aggregate.id, 'scope', e.target.value)}><MenuItem value="Group">Group</MenuItem><MenuItem value="Report">Report</MenuItem></Select></FormControl></Grid>
              <Grid size={{ 'xs': 12, 'sm': 2 }}><TextField fullWidth size="small" label="Display Name" value={aggregate.displayName ?? ''} onChange={(e) => onAggregateChange(group.id, aggregate.id, 'displayName', e.target.value)} /></Grid>
              <Grid size={{ 'xs': 12, 'sm': 1 }}><IconButton size="small" onClick={() => onRemoveAggregate(group.id, aggregate.id)}><DeleteIcon fontSize="small" /></IconButton></Grid>
            </Grid>
          ))}
          <Button size="small" variant="text" startIcon={<AddIcon sx={{ fontSize: 14 }} />} onClick={() => onAddAggregate(group.id)}>Add Aggregate</Button>
        </Box>
      ))}

      {!groupDefinitions.length && (<Typography variant="body2" color="text.secondary">No groups defined yet.</Typography>)}
    </>
  );
};

export default GroupsEditor;
