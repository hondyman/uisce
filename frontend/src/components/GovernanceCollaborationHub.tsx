import React, { useState } from 'react';
import {
  Alert,
  Avatar,
  Box,
  Button,
  ButtonGroup,
  Card,
  CardContent,
  CardHeader,
  Chip,
  Divider,
  IconButton,
  InputAdornment,
  Paper,
  Stack,
  Step,
  StepContent,
  StepLabel,
  Stepper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import Grid from '@mui/material/Grid';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CheckIcon from '@mui/icons-material/Check';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import EditNoteIcon from '@mui/icons-material/EditNote';
import CancelIcon from '@mui/icons-material/Cancel';
import SendIcon from '@mui/icons-material/Send';
import AttachFileIcon from '@mui/icons-material/AttachFile';
import ForumIcon from '@mui/icons-material/Forum';
import PriorityHighIcon from '@mui/icons-material/PriorityHigh';
import AccountTreeIcon from '@mui/icons-material/AccountTree';

// ── Types ──────────────────────────────────────────────────────────────────────

interface OverrideRequest {
  id: string;
  requester: string;
  priority: 'High' | 'Medium' | 'Low';
  submitted: string;
  entityId: string;
  status: string;
  workflowStep: number;
  originalValue: Record<string, string>;
  proposedValue: Record<string, string>;
  changeReason: string;
  stepTimestamps: Array<string | undefined>;
}

interface Message {
  id: string;
  author: string;
  initials: string;
  content: string;
  timestamp: string;
  isOwn: boolean;
  isSystem?: boolean;
}

interface Props {
  overrideId?: string;
}

const WORKFLOW_STEPS = ['Draft Created', 'Simulation Passed', 'Manager Approval', 'Compliance Review'];

const DEMO_REQUEST: OverrideRequest = {
  id: 'OR-789',
  requester: 'John Doe',
  priority: 'High',
  submitted: 'Oct 24, 2:45 PM',
  entityId: 'EDM-USR-9921',
  status: 'Pending Approval',
  workflowStep: 3,
  originalValue: { 'Market Cap Rating': 'Standard High-Cap', 'Risk Weight': '0.024' },
  proposedValue: { 'Market Cap Rating': 'Premium Enterprise', 'Risk Weight': '0.018' },
  changeReason: "The adjustment reflects the recent credit upgrade from Moody's on Oct 22.",
  stepTimestamps: ['Oct 24, 2:45 PM', 'Oct 24, 3:12 PM', 'Oct 24, 4:20 PM', undefined],
};

const DEMO_MESSAGES: Message[] = [
  { id: '0', author: 'System', initials: '', content: 'Simulation Passed', timestamp: '', isOwn: false, isSystem: true },
  { id: '1', author: 'Sarah Chen', initials: 'SC', content: "I've reviewed the simulation results. Everything looks compliant with the new Tier 1 standards. @James_Compliance, can you give the final sign-off?", timestamp: '4:25 PM', isOwn: false },
  { id: '2', author: 'You', initials: 'ME', content: 'Checking the exposure limits now. Will update in a minute.', timestamp: '4:32 PM', isOwn: true },
];

// ── Main Component ─────────────────────────────────────────────────────────────

const GovernanceCollaborationHub: React.FC<Props> = ({ overrideId = 'OR-789' }) => {
  const [request] = useState<OverrideRequest>(DEMO_REQUEST);
  const [messages, setMessages] = useState<Message[]>(DEMO_MESSAGES);
  const [draft, setDraft] = useState('');
  const [actionResult, setActionResult] = useState<{ label: string; color: 'success' | 'error' | 'warning' } | null>(null);

  const sendMessage = () => {
    if (!draft.trim()) return;
    setMessages(m => [...m, {
      id: Date.now().toString(),
      author: 'You', initials: 'ME', content: draft,
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      isOwn: true,
    }]);
    setDraft('');
  };

  const handleAction = (action: 'approve' | 'changes' | 'reject') => {
    const map = {
      approve: { label: 'Override Approved', color: 'success' as const },
      changes: { label: 'Changes Requested', color: 'warning' as const },
      reject:  { label: 'Override Rejected', color: 'error'   as const },
    };
    setActionResult(map[action]);
  };

  const priorityColor = request.priority === 'High' ? 'error' : request.priority === 'Medium' ? 'warning' : 'default';

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      {/* ── Page Header ──────────────────────────────────────────────────── */}
      <Box sx={{ px: 4, pt: 3, pb: 2 }}>
        <Typography variant="caption" color="text.secondary">
          Overrides › <strong>{overrideId}</strong>
        </Typography>
        <Stack direction="row" alignItems="center" justifyContent="space-between" flexWrap="wrap" gap={2} mt={0.5}>
          <Stack direction="row" alignItems="center" spacing={2}>
            <Typography variant="h5" fontWeight={700}>Override Request: {overrideId}</Typography>
            {actionResult
              ? <Alert severity={actionResult.color} sx={{ py: 0 }}>{actionResult.label}</Alert>
              : <Chip label={request.status} color="warning" variant="outlined" />}
          </Stack>
          <ButtonGroup variant="contained" disableElevation>
            <Button color="success" startIcon={<ThumbUpIcon />} onClick={() => handleAction('approve')}>
              Approve
            </Button>
            <Button color="warning" startIcon={<EditNoteIcon />} onClick={() => handleAction('changes')}>
              Request Changes
            </Button>
            <Button color="error" startIcon={<CancelIcon />} onClick={() => handleAction('reject')}>
              Reject
            </Button>
          </ButtonGroup>
        </Stack>
      </Box>

      <Divider />

      <Box sx={{ display: 'flex', height: 'calc(100vh - 140px)' }}>
        {/* ── Left Panel ────────────────────────────────────────────────── */}
        <Box sx={{ flex: 1, overflowY: 'auto', p: 3, display: 'flex', flexDirection: 'column', gap: 3 }}>
          {/* Metadata */}
          <Card elevation={2}>
            <CardHeader
              avatar={<AccountTreeIcon color="primary" />}
              title="Request Details"
              titleTypographyProps={{ fontWeight: 700, variant: 'body1' }}
            />
            <Divider />
            <CardContent>
              <Grid container spacing={3}>
                <Grid size={{ xs: 6, sm: 3 }}>
                  <Typography variant="caption" color="text.secondary">Requester</Typography>
                  <Typography variant="body2" fontWeight={600} mt={0.5}>{request.requester}</Typography>
                </Grid>
                <Grid size={{ xs: 6, sm: 3 }}>
                  <Typography variant="caption" color="text.secondary">Priority</Typography>
                  <Box mt={0.5}>
                    <Chip icon={<PriorityHighIcon />} label={`${request.priority} Priority`} color={priorityColor as any} size="small" />
                  </Box>
                </Grid>
                <Grid size={{ xs: 6, sm: 3 }}>
                  <Typography variant="caption" color="text.secondary">Submitted</Typography>
                  <Typography variant="body2" fontWeight={600} mt={0.5}>{request.submitted}</Typography>
                </Grid>
                <Grid size={{ xs: 6, sm: 3 }}>
                  <Typography variant="caption" color="text.secondary">Entity ID</Typography>
                  <Typography variant="body2" fontWeight={600} color="primary.main" mt={0.5}>{request.entityId}</Typography>
                </Grid>
              </Grid>
            </CardContent>
          </Card>

          <Grid container spacing={3}>
            {/* Workflow Stepper */}
            <Grid size={{ xs: 12, md: 4 }}>
              <Card elevation={2} sx={{ height: '100%' }}>
                <CardHeader title="Approval Workflow" titleTypographyProps={{ fontWeight: 700, variant: 'body1' }} />
                <Divider />
                <CardContent>
                  <Stepper activeStep={request.workflowStep} orientation="vertical">
                    {WORKFLOW_STEPS.map((label, i) => (
                      <Step key={label} completed={i < request.workflowStep}>
                        <StepLabel
                          StepIconComponent={i < request.workflowStep ? () => (
                            <CheckCircleIcon color="success" sx={{ fontSize: 22 }} />
                          ) : undefined}
                        >
                          <Typography variant="body2" fontWeight={i === request.workflowStep ? 700 : 400}>
                            {label}
                          </Typography>
                        </StepLabel>
                        <StepContent>
                          <Typography variant="caption" color="text.secondary">
                            {request.stepTimestamps[i] ?? 'In Progress'}
                          </Typography>
                        </StepContent>
                      </Step>
                    ))}
                  </Stepper>
                </CardContent>
              </Card>
            </Grid>

            {/* Override comparison */}
            <Grid size={{ xs: 12, md: 8 }}>
              <Card elevation={2} sx={{ height: '100%' }}>
                <CardHeader title="Override Comparison" titleTypographyProps={{ fontWeight: 700, variant: 'body1' }} />
                <Divider />
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow sx={{ bgcolor: 'action.hover' }}>
                        <TableCell sx={{ fontWeight: 700 }}>Field</TableCell>
                        <TableCell sx={{ fontWeight: 700, color: 'text.secondary' }}>Original Value</TableCell>
                        <TableCell sx={{ fontWeight: 700, color: 'primary.main' }}>Proposed Override</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {Object.keys(request.originalValue).map(field => (
                        <TableRow key={field} hover>
                          <TableCell>
                            <Typography variant="body2" fontWeight={600}>{field}</Typography>
                          </TableCell>
                          <TableCell>
                            <Typography variant="body2" color="text.secondary" sx={{ textDecoration: 'line-through' }}>
                              {request.originalValue[field]}
                            </Typography>
                          </TableCell>
                          <TableCell>
                            <Typography variant="body2" color="primary.main" fontWeight={700}>
                              {request.proposedValue[field]}
                            </Typography>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
                <Box sx={{ p: 2, bgcolor: 'action.hover', borderTop: '1px solid', borderColor: 'divider' }}>
                  <Typography variant="body2" color="text.secondary" fontStyle="italic">
                    "{request.changeReason}"
                  </Typography>
                </Box>
              </Card>
            </Grid>
          </Grid>
        </Box>

        {/* ── Chat Panel ────────────────────────────────────────────────── */}
        <Paper
          elevation={0}
          sx={{
            width: 400,
            flexShrink: 0,
            borderLeft: '1px solid',
            borderColor: 'divider',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Box sx={{ p: 2, borderBottom: '1px solid', borderColor: 'divider' }}>
            <Stack direction="row" alignItems="center" justifyContent="space-between">
              <Stack direction="row" alignItems="center" spacing={1}>
                <ForumIcon color="primary" fontSize="small" />
                <Typography variant="subtitle2" fontWeight={700}>Collaboration Thread</Typography>
              </Stack>
              <Typography variant="caption" color="text.secondary">3 Participants</Typography>
            </Stack>
          </Box>

          {/* Messages */}
          <Box sx={{ flex: 1, overflowY: 'auto', p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
            {messages.map(msg => {
              if (msg.isSystem) {
                return (
                  <Box key={msg.id} sx={{ display: 'flex', justifyContent: 'center' }}>
                    <Chip label={`System: ${msg.content}`} size="small" variant="outlined" color="default" sx={{ fontSize: '0.7rem' }} />
                  </Box>
                );
              }
              return (
                <Box
                  key={msg.id}
                  sx={{
                    display: 'flex',
                    flexDirection: msg.isOwn ? 'row-reverse' : 'row',
                    gap: 1,
                    maxWidth: '85%',
                    alignSelf: msg.isOwn ? 'flex-end' : 'flex-start',
                  }}
                >
                  <Avatar sx={{ width: 32, height: 32, fontSize: '0.75rem', bgcolor: msg.isOwn ? 'primary.main' : 'grey.400', flexShrink: 0 }}>
                    {msg.initials}
                  </Avatar>
                  <Box>
                    <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 0.5, justifyContent: msg.isOwn ? 'flex-end' : 'flex-start' }}>
                      <Typography variant="caption" fontWeight={700} color={msg.isOwn ? 'primary.main' : 'text.primary'}>
                        {msg.isOwn ? 'You' : msg.author}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">{msg.timestamp}</Typography>
                    </Stack>
                    <Paper
                      elevation={0}
                      sx={{
                        p: 1.5,
                        bgcolor: msg.isOwn ? 'primary.main' : 'action.hover',
                        color: msg.isOwn ? 'primary.contrastText' : 'text.primary',
                        borderRadius: msg.isOwn ? '12px 2px 12px 12px' : '2px 12px 12px 12px',
                      }}
                    >
                      <Typography variant="body2">{msg.content}</Typography>
                    </Paper>
                    {msg.isOwn && (
                      <Typography variant="caption" color="text.secondary" sx={{ float: 'right', mt: 0.3 }}>
                        <CheckIcon sx={{ fontSize: 11 }} /> Read
                      </Typography>
                    )}
                  </Box>
                </Box>
              );
            })}
            <Typography variant="caption" color="text.secondary" fontStyle="italic" sx={{ alignSelf: 'flex-start', pl: 5 }}>
              Sarah is typing…
            </Typography>
          </Box>

          {/* Input */}
          <Box sx={{ p: 2, borderTop: '1px solid', borderColor: 'divider' }}>
            <TextField
              fullWidth
              multiline
              rows={3}
              value={draft}
              onChange={e => setDraft(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) sendMessage(); }}
              placeholder="Type '@' to mention team members…"
              size="small"
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end" sx={{ alignSelf: 'flex-end', mb: 1 }}>
                    <Tooltip title="Attach file">
                      <IconButton size="small"><AttachFileIcon fontSize="small" /></IconButton>
                    </Tooltip>
                    <Tooltip title="Send (⌘+Enter)">
                      <IconButton size="small" color="primary" onClick={sendMessage}>
                        <SendIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </InputAdornment>
                ),
              }}
            />
            <Typography variant="caption" color="text.disabled" sx={{ mt: 0.5, display: 'block' }}>
              ⌘ + Enter to send
            </Typography>
          </Box>
        </Paper>
      </Box>
    </Box>
  );
};

export default GovernanceCollaborationHub;
