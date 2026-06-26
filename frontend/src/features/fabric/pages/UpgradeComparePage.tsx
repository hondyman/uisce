import React, { useState, useEffect } from 'react';
import { devLog } from '../../../utils/devLogger';
import { useLocation } from 'react-router-dom';
import {
  Box, Typography, Paper, Button, CircularProgress, Alert, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, Checkbox, TextField, Chip, Collapse, IconButton, Accordion, AccordionSummary, AccordionDetails
} from '@mui/material';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { createPatch } from 'diff';
import { parseDiff, Diff, Hunk } from 'react-diff-view';
import 'react-diff-view/style/index.css';

import { FieldChange, ModelDiff, CompareResponse, RuleGroup } from './types';
import getErrorMessage from '../../../utils/errors';
import { useNotification } from '../../../hooks/useNotification';

// A simple CommentBox component
const CommentBox: React.FC<{ onSave: (comment: string) => void }> = ({ onSave }) => {
  const [comment, setComment] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = async () => {
    if (!comment.trim()) return;
    setIsSaving(true);
    await onSave(comment);
    setIsSaving(false);
    setComment(''); // Clear after saving
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, mt: 1 }}>
      <TextField
        fullWidth
        multiline
        rows={2}
        size="small"
        label="Add a comment"
        value={comment}
        onChange={(e) => setComment(e.target.value)}
        disabled={isSaving}
      />
      <Button size="small" variant="contained" onClick={handleSave} disabled={isSaving || !comment.trim()}>
        {isSaving ? 'Saving...' : 'Save Comment'}
      </Button>
    </Box>
  );
};

const DiffDetailRow: React.FC<{
  modelDiff: ModelDiff;
  isSelected: boolean;
  onSelectionChange: (modelName: string) => void;
  onSaveComment: (model: string, comment: string) => void;
}> = ({ modelDiff, isSelected, onSelectionChange, onSaveComment }) => {
  const [open, setOpen] = useState(false);

  return (
    <React.Fragment>
      <TableRow sx={{ '& > *': { borderBottom: 'unset' } }}>
        <TableCell padding="checkbox">
          <Checkbox checked={isSelected} onChange={() => onSelectionChange(modelDiff.model)} />
        </TableCell>
        <TableCell component="th" scope="row">{modelDiff.model}</TableCell>
        <TableCell>
          <Chip label={modelDiff.change_type} color={modelDiff.change_type === 'added' ? 'success' : modelDiff.change_type === 'removed' ? 'error' : 'warning'} size="small" />
        </TableCell>
        <TableCell>
          <IconButton aria-label="expand row" size="small" onClick={() => setOpen(!open)}>
            {open ? <KeyboardArrowUpIcon /> : <KeyboardArrowDownIcon />}
          </IconButton>
          {modelDiff.field_changes.length} field(s) changed
        </TableCell>
        <TableCell>
          <CommentBox onSave={(comment) => onSaveComment(modelDiff.model, comment)} />
        </TableCell>
      </TableRow>
  <TableRow>
  <TableCell className="no-vertical-padding-cell" colSpan={6}>
          <Collapse in={open} timeout="auto" unmountOnExit>
            <Box sx={{ margin: 1, p: 2, backgroundColor: 'action.hover', borderRadius: 1 }}>
              <Typography variant="h6" gutterBottom component="div">
                Field-Level Changes for {modelDiff.model}
              </Typography>
              <Table size="small" aria-label="field changes">
                <TableHead>
                  <TableRow>
                    <TableCell>Path</TableCell>
                    <TableCell sx={{ width: '60%' }}>Change</TableCell>
                    <TableCell>Rule</TableCell>
                    <TableCell>Provenance</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {modelDiff.field_changes.map((change: FieldChange, index: number) => {
                    const oldStr = JSON.stringify(change.before, null, 2) ?? 'null';
                    const newStr = JSON.stringify(change.after, null, 2) ?? 'null';
                    const patch = createPatch(change.path, oldStr, newStr, '', '', { context: 1 });
                    const [file] = parseDiff(patch);

                    return (
                      <TableRow key={index}>
                      <TableCell><code>{change.path}</code></TableCell>
                        <TableCell sx={{ fontSize: '0.75rem', '& pre': { m: 0 } }}>
                          <Diff viewType="unified" diffType={file.type} hunks={file.hunks || []}>
                            {(hunks) => hunks.map((hunk) => <Hunk key={hunk.content} hunk={hunk} />)}
                          </Diff>
                        </TableCell>
                      <TableCell>{change.rule_id && <Chip label={change.rule_id} size="small" variant="outlined" />}</TableCell>
                      <TableCell><code>{change.provenance}</code></TableCell>
                    </TableRow>
                  );
                  })}
                </TableBody>
              </Table>
            </Box>
          </Collapse>
        </TableCell>
      </TableRow>
    </React.Fragment>
  );
};

const UpgradeComparePage: React.FC = () => {
  const location = useLocation();
  const [diffData, setDiffData] = useState<CompareResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedModels, setSelectedModels] = useState<Set<string>>(new Set());
  const notification = useNotification();

  useEffect(() => {
    const fetchDiff = async () => {
      setLoading(true);
      setError(null);
      try {
        // Using POST as per blueprint, even though we get params from URL
        const params = new URLSearchParams(location.search);
        const response = await fetch('/api/fabric/models/compare', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            tenant_instance_id: params.get('tenant_instance_id'),
            model_name: params.get('model_name'), // Assuming this might be used for scope
          }),
        });
        if (!response.ok) throw new Error('Failed to fetch comparison data.');
        const data: CompareResponse = await response.json();
        setDiffData(data);
        // Initially select all models for convenience
        setSelectedModels(new Set(data.diff_details.map(d => d.model)));
      } catch (e: unknown) {
        setError(getErrorMessage(e, 'Failed to fetch comparison data.'));
      } finally {
        setLoading(false);
      }
    };
    fetchDiff();
  }, [location.search]);

  const handleSelectionChange = (modelName: string) => {
    const newSelection = new Set(selectedModels);
    if (newSelection.has(modelName)) {
      newSelection.delete(modelName);
    } else {
      newSelection.add(modelName);
    }
    setSelectedModels(newSelection);
  };

  const handleSaveComment = async (model: string, comment: string) => {
  devLog(`Saving comment for ${model}: ${comment}`);
    // API call to POST /api/fabric/models/diff/comment
    await fetch('/api/fabric/models/diff/comment', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        diff_id: diffData?.diff_id,
        model,
        comment,
      }),
    });
    // In a real app, you might want to show a confirmation
  };

  const handleApply = async () => {
    if (!diffData) return;
    await fetch('/api/fabric/models/upgrade/apply', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        diff_id: diffData.diff_id,
        selected_models: Array.from(selectedModels),
      }),
    });
    notification.success(`Applied changes for: ${Array.from(selectedModels).join(', ')}`);
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Model Upgrade Comparison
      </Typography>
      {loading && <CircularProgress />}
      {error && <Alert severity="error">{error}</Alert>}
      {diffData && (
        <>
          <Paper sx={{ p: 2, mb: 2, display: 'flex', gap: 2 }}>
            <Chip label={`Added: ${diffData.diff_summary.added}`} color="success" />
            <Chip label={`Modified: ${diffData.diff_summary.modified}`} color="warning" />
            <Chip label={`Removed: ${diffData.diff_summary.removed}`} color="error" />
          </Paper>

          <Paper sx={{ p: 2, mb: 2 }}>
            <Typography variant="h6" gutterBottom>Rule Groups</Typography>
            {Object.values(diffData.groups || {}).map((group: RuleGroup) => (
              <Accordion key={group.rule_id}>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Typography sx={{ flexShrink: 0, mr: 2 }}>
                    <Chip label={group.rule_id} variant="outlined" />
                  </Typography>
                  <Typography sx={{ color: 'text.secondary' }}>{group.changes.length} changes driven by this rule</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <Table size="small">
                    <TableHead>
                      <TableRow><TableCell>Path</TableCell><TableCell>Before</TableCell><TableCell>After</TableCell></TableRow>
                    </TableHead>
                    <TableBody>
                      {group.changes.map((change: FieldChange, i: number) => (
                        <TableRow key={i}>
                          <TableCell><code>{change.path}</code></TableCell><TableCell><code>{JSON.stringify(change.before)}</code></TableCell><TableCell><code>{JSON.stringify(change.after)}</code></TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </AccordionDetails>
              </Accordion>
            ))}
          </Paper>
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell padding="checkbox">
                    <Checkbox
                      indeterminate={selectedModels.size > 0 && selectedModels.size < diffData.diff_details.length}
                      checked={diffData.diff_details.length > 0 && selectedModels.size === diffData.diff_details.length}
                      onChange={() => {
                        const allSelected = selectedModels.size === diffData.diff_details.length;
                        setSelectedModels(allSelected ? new Set() : new Set(diffData.diff_details.map(d => d.model)));
                      }}
                    />
                  </TableCell>
                  <TableCell>Model</TableCell>
                  <TableCell>Change Type</TableCell>
                  <TableCell>Field Changes</TableCell>
                  <TableCell>Reviewer Comments</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {diffData.diff_details.map((modelDiff: ModelDiff) => (
                  <DiffDetailRow
                    key={modelDiff.model}
                    modelDiff={modelDiff}
                    isSelected={selectedModels.has(modelDiff.model)}
                    onSelectionChange={handleSelectionChange}
                    onSaveComment={handleSaveComment}
                  />
                ))}
              </TableBody>
            </Table>
          </TableContainer>
          <Box sx={{ mt: 2, display: 'flex', justifyContent: 'flex-end' }}>
            <Button variant="contained" onClick={handleApply} disabled={selectedModels.size === 0}>
              Approve & Apply ({selectedModels.size})
            </Button>
          </Box>
        </>
      )}
    </Box>
  );
};

export default UpgradeComparePage;