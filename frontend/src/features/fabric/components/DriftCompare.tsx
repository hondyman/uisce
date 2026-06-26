import type { FC } from 'react';
import { Typography, Paper, Grid, Box } from '@mui/material';
import { groupBySeverity, DiffResult, DriftLogEntry, ChangedEntry } from '../pages/diff';
import { createPatch } from 'diff';
import { parseDiff, Diff, Hunk } from 'react-diff-view';
import 'react-diff-view/style/index.css';
import SeverityBadge from './SeverityBadge';

interface DriftCompareProps {
  diff: DiffResult;
}

const getSeverityIcon = (severity: string) => {
  switch (severity) {
    case 'breaking': return '🚨';
    case 'medium': return '⚠️';
    case 'low': return '✅';
    default: return '➡️';
  }
};

const SeveritySection: FC<{ severity: string; entries: DriftLogEntry[] }> = ({ severity, entries }) => {
  if (entries.length === 0) return null;

  return (
    <Box component="section" mb={2}>
      <Typography variant="h6" sx={{ display: 'flex', alignItems: 'center', gap: 1, textTransform: 'capitalize', mb: 1, fontSize: '1rem' }}>
        {getSeverityIcon(severity)} {severity}
      </Typography>
      <Box component="ul" sx={{ pl: 2, m: 0, listStyle: 'none' }}>
        {entries.map(e => (
          <Paper component="li" key={e.qualified_path} variant="outlined" sx={{ p: 1.5, mb: 1 }}>
            <Typography variant="body2" component="strong" sx={{ fontFamily: 'monospace' }}>{e.qualified_path}</Typography>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
              {e.explanation}
            </Typography>
          </Paper>
        ))}
      </Box>
    </Box>
  );
};

const ChangedEntryDiff: FC<ChangedEntry> = ({ before, after, severityChanged, explanationChanged }) => {
  return (
    <Paper variant="outlined" sx={{ mb: 2, p: 2, '& .diff-line-old': { backgroundColor: 'rgba(255, 0, 0, 0.1)' }, '& .diff-line-new': { backgroundColor: 'rgba(0, 255, 0, 0.1)' }, '& pre': { m: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all' } }}>
      <Typography variant="body1" component="strong" sx={{ fontFamily: 'monospace', display: 'block', mb: 1 }}>
        {before.qualified_path}
      </Typography>
      {severityChanged && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, my: 1.5, p:1, border: '1px solid', borderColor: 'divider', borderRadius: 1, background: theme => theme.palette.action.hover }}>
          <Typography variant="body2">Severity changed:</Typography>
          <SeverityBadge severity={before.severity} count={1} />
          <Typography>➡️</Typography>
          <SeverityBadge severity={after.severity} count={1} />
        </Box>
      )}
      {explanationChanged && (() => {
        const patch = createPatch('explanation.txt', before.explanation, after.explanation, '', '', { context: 0 });
        const [file] = parseDiff(patch);
        return (
          <Diff viewType="unified" diffType={file.type} hunks={file.hunks}>
            {(hunks: any[]) => hunks.map(hunk => <Hunk key={hunk.content} hunk={hunk} />)}
          </Diff>
        );
      })()}
    </Paper>
  );
};

const DriftCompare: FC<DriftCompareProps> = ({ diff }) => {
  const addedBySeverity = groupBySeverity(diff.added);
  const removedBySeverity = groupBySeverity(diff.removed);

  const hasChanges = diff.added.length > 0 || diff.removed.length > 0 || diff.changed.length > 0;

  if (!hasChanges) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h6">✅ No Differences Found</Typography>
        <Typography color="text.secondary">The log entries in both reports are identical.</Typography>
      </Paper>
    );
  }

  return (
    <Grid container spacing={4}>
      <Grid item xs={12} md={6}>
        <Typography variant="h5" gutterBottom sx={{color: 'success.main'}}>Added</Typography>
        {Object.keys(addedBySeverity).length > 0 ? (Object.entries(addedBySeverity).map(([sev, entries]: [string, DriftLogEntry[]]) => (
          <SeveritySection key={`added-${sev}`} severity={sev} entries={entries} />
        ))) : (
          <Typography color="text.secondary">No new entries.</Typography>
        )}
      </Grid>
      <Grid item xs={12} md={6}>
        <Typography variant="h5" gutterBottom sx={{color: 'error.main'}}>Removed</Typography>
        {Object.keys(removedBySeverity).length > 0 ? (Object.entries(removedBySeverity).map(([sev, entries]: [string, DriftLogEntry[]]) => (
          <SeveritySection key={`removed-${sev}`} severity={sev} entries={entries} />
        ))) : (
          <Typography color="text.secondary">No removed entries.</Typography>
        )}
      </Grid>
      {diff.changed.length > 0 && (
        <Grid item xs={12}>
          <Typography variant="h5" gutterBottom sx={{color: 'warning.main'}}>Changed</Typography>
          {diff.changed.map((change) => (
            <ChangedEntryDiff key={change.before.qualified_path} {...change} />
          ))}
        </Grid>
      )}
    </Grid>
  );
};

export default DriftCompare;