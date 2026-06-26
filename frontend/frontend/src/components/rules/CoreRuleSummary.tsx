import React from 'react';
import { Box, Typography, Button, Card, CardContent } from '@mui/material';

interface CoreRuleSummaryProps {
  core: { id: string; name?: string; severity?: string; script_content?: string } | null;
}

export const CoreRuleSummary: React.FC<CoreRuleSummaryProps> = ({ core }) => {
  if (!core) return null;
  return (
    <Card variant="outlined" sx={{ p: 1, mb: 2 }}>
      <CardContent>
        <Typography variant="h6">Core Rule Summary</Typography>
        <Typography variant="body2"><strong>Name:</strong> {core.name}</Typography>
        <Typography variant="body2"><strong>Severity:</strong> {core.severity}</Typography>
        <details style={{ marginTop: 8 }}>
          <summary style={{ cursor: 'pointer' }}>View Core DSL</summary>
          <pre style={{ whiteSpace: 'pre-wrap', background: '#fafafa', padding: 8 }}>{core.script_content}</pre>
        </details>
        <Box sx={{ mt: 1 }}>
          <Button size="small" href={`/rules/${core.id}`}>Open Core Rule</Button>
        </Box>
      </CardContent>
    </Card>
  );
};

export default CoreRuleSummary;
