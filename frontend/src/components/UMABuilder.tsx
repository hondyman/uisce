import React, { useState, useCallback, useMemo } from 'react';
import ReactFlow, {
  Background,
  Controls,
  Node,
  Edge,
  NodeProps,
  useNodesState,
  useEdgesState,
  Connection,
  addEdge,
  MarkerType,
} from 'reactflow';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Button,
  TextField,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Chip,
  Grid,
  Typography,
  Alert,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Tooltip,
  IconButton,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  PlayArrow as PlayArrowIcon,
  Info as InfoIcon,
} from '@mui/icons-material';
import { useMutation, useQuery } from '@tanstack/react-query';
import 'reactflow/dist/style.css';
import './UMABuilder.css';

interface UMASleeveNode {
  id: string;
  model: string;
  sleeveType: string;
  targetAllocation: number;
  currentAllocation: number;
  drift: number;
  minDriftThreshold: number;
  status: string;
}

interface UMAAccount {
  id: string;
  name: string;
  aum: number;
  status: string;
  sleeves: UMASleeveNode[];
}

interface RebalancePlan {
  id: string;
  driftSignal: number;
  trades: Trade[];
  approvalStatus: string;
  createdAt: string;
}

interface Trade {
  symbol: string;
  side: 'buy' | 'sell';
  quantity: number;
  estimatedPrice: number;
  estimatedValue: number;
  reason: string;
}

// Custom Sleeve Node Component
const SleeveNode = ({ data }: NodeProps) => {
  const driftColor = data.drift > data.minDriftThreshold ? '#d32f2f' : '#4caf50';
  const driftStyle = data.drift > data.minDriftThreshold ? 'error' : 'success';

  return (
    <Tooltip
      title={
        <Box>
          <Typography variant="caption" display="block">
            <strong>Model:</strong> {data.model}
          </Typography>
          <Typography variant="caption" display="block">
            <strong>Type:</strong> {data.sleeveType}
          </Typography>
          <Typography variant="caption" display="block">
            <strong>Target:</strong> {(data.targetAllocation * 100).toFixed(1)}%
          </Typography>
          <Typography variant="caption" display="block">
            <strong>Current:</strong> {(data.currentAllocation * 100).toFixed(1)}%
          </Typography>
          <Typography variant="caption" display="block" sx={{ color: driftColor }}>
            <strong>Drift:</strong> {(data.drift * 100).toFixed(2)}%
          </Typography>
          <Typography variant="caption" display="block">
            <strong>Threshold:</strong> {(data.minDriftThreshold * 100).toFixed(1)}%
          </Typography>
        </Box>
      }
      arrow
      placement="top"
    >
      <Card
        sx={{
          width: 180,
          border: data.drift > data.minDriftThreshold ? '2px solid #d32f2f' : '2px solid #4caf50',
          cursor: 'pointer',
          '&:hover': {
            boxShadow: 3,
          },
        }}
      >
        <CardContent sx={{ padding: 1 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 0.5 }}>
            {data.model}
          </Typography>
          <Typography variant="caption" sx={{ display: 'block', mb: 0.5 }}>
            {data.sleeveType}
          </Typography>
          <Box sx={{ display: 'flex', gap: 0.5, mb: 0.5 }}>
            <Chip
              label={`Target: ${(data.targetAllocation * 100).toFixed(0)}%`}
              size="small"
              variant="outlined"
            />
            <Chip
              label={`Current: ${(data.currentAllocation * 100).toFixed(0)}%`}
              size="small"
              variant="outlined"
            />
          </Box>
          <Chip
            label={`Drift: ${(data.drift * 100).toFixed(2)}%`}
            size="small"
            color={driftStyle}
            variant="filled"
          />
        </CardContent>
      </Card>
    </Tooltip>
  );
};

interface UMABuilderProps {
  umaId?: string;
  onRebalanceTriggered?: (workflowId: string) => void;
  readOnly?: boolean;
}

export const UMABuilder: React.FC<UMABuilderProps> = ({
  umaId,
  onRebalanceTriggered,
  readOnly = false,
}) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [selectedSleeve, setSelectedSleeve] = useState<UMASleeveNode | null>(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [rebalanceDialogOpen, setRebalanceDialogOpen] = useState(false);
  const [rebalancePlan, setRebalancePlan] = useState<RebalancePlan | null>(null);
  const [approvalSignal, setApprovalSignal] = useState<string>('');
  const [sleeveFormData, setSleeveFormData] = useState<Partial<UMASleeveNode>>({});

  // Fetch UMA Account
  const { data: umaAccount, isLoading } = useQuery({
    queryKey: ['uma', umaId],
    queryFn: async () => {
      const response = await fetch(
        `/api/uma/${umaId}?tenant_id=${localStorage.getItem('selected_tenant')}&tenant_instance_id=${localStorage.getItem('selected_datasource')}`,
        {
          headers: {
            'X-Tenant-ID': localStorage.getItem('selected_tenant') || '',
            'X-Tenant-Datasource-ID': localStorage.getItem('selected_datasource') || '',
          },
        }
      );
      return response.json() as Promise<UMAAccount>;
    },
    enabled: !!umaId,
  });

  // Initialize nodes from UMA account
  React.useEffect(() => {
    if (umaAccount?.sleeves) {
      const sleeveNodes = umaAccount.sleeves.map((sleeve, idx) => ({
        id: sleeve.id,
        data: sleeve,
        position: { x: idx * 250, y: 0 },
      }));

      setNodes(sleeveNodes);

      // Create edges to show relationships
      const edges = sleeveNodes.slice(0, -1).map((node, idx) => ({
        id: `edge-${idx}`,
        source: node.id,
        target: sleeveNodes[idx + 1].id,
        markerEnd: { type: MarkerType.ArrowClosed },
        animated: false,
      }));

      setEdges(edges);
    }
  }, [umaAccount, setNodes, setEdges]);

  // Update UMA Sleeve
  const updateSleeveMutation = useMutation({
    mutationFn: async (updatedSleeve: UMASleeveNode) => {
      const response = await fetch(
        `/api/uma/sleeves/${updatedSleeve.id}?tenant_id=${localStorage.getItem('selected_tenant')}&tenant_instance_id=${localStorage.getItem('selected_datasource')}`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': localStorage.getItem('selected_tenant') || '',
            'X-Tenant-Datasource-ID': localStorage.getItem('selected_datasource') || '',
          },
          body: JSON.stringify(updatedSleeve),
        }
      );
      return response.json();
    },
    onSuccess: () => {
      setEditDialogOpen(false);
      setSelectedSleeve(null);
    },
  });

  // Trigger Rebalance
  const rebalanceMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch(
        `/api/uma/rebalance/request?tenant_id=${localStorage.getItem('selected_tenant')}&tenant_instance_id=${localStorage.getItem('selected_datasource')}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': localStorage.getItem('selected_tenant') || '',
            'X-Tenant-Datasource-ID': localStorage.getItem('selected_datasource') || '',
          },
          body: JSON.stringify({
            uma_account_id: umaId,
            request_type: 'manual',
          }),
        }
      );
      const data = await response.json();
      return data;
    },
    onSuccess: (data) => {
      setRebalancePlan(data.plan);
      onRebalanceTriggered?.(data.workflow_id);
    },
  });

  // Approve Rebalance
  const approveMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch(
        `/api/uma/rebalance/${rebalancePlan?.id}/approve?tenant_id=${localStorage.getItem('selected_tenant')}&tenant_instance_id=${localStorage.getItem('selected_datasource')}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': localStorage.getItem('selected_tenant') || '',
            'X-Tenant-Datasource-ID': localStorage.getItem('selected_datasource') || '',
          },
          body: JSON.stringify({
            approval_signal: approvalSignal,
          }),
        }
      );
      return response.json();
    },
    onSuccess: () => {
      setRebalanceDialogOpen(false);
      setApprovalSignal('');
    },
  });

  const handleEditSleeve = (sleeve: UMASleeveNode) => {
    setSelectedSleeve(sleeve);
    setSleeveFormData(sleeve);
    setEditDialogOpen(true);
  };

  const handleSaveSleeve = () => {
    if (selectedSleeve && sleeveFormData) {
      updateSleeveMutation.mutate({
        ...selectedSleeve,
        ...sleeveFormData,
      } as UMASleeveNode);
    }
  };

  const handleTriggerRebalance = () => {
    rebalanceMutation.mutate();
  };

  const handleApproveRebalance = () => {
    approveMutation.mutate();
  };

  const totalAllocation = umaAccount?.sleeves.reduce(
    (sum, sleeve) => sum + sleeve.currentAllocation,
    0
  ) || 0;

  const hasNegativeDrift = umaAccount?.sleeves.some(
    (sleeve) => sleeve.drift > sleeve.minDriftThreshold
  ) || false;

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 400 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!umaAccount) {
    return (
      <Alert severity="error">
        UMA Account not found. Please select a valid UMA account.
      </Alert>
    );
  }

  return (
    <Box sx={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Card sx={{ mb: 2 }}>
        <CardHeader
          title={`UMA Builder: ${umaAccount.name}`}
          subtitle={`AUM: $${(umaAccount.aum / 1000000).toFixed(2)}M | Status: ${umaAccount.status}`}
          action={
            !readOnly && (
              <Button
                variant="contained"
                color={hasNegativeDrift ? 'error' : 'primary'}
                startIcon={<PlayArrowIcon />}
                onClick={handleTriggerRebalance}
                disabled={rebalanceMutation.isPending}
              >
                {rebalanceMutation.isPending ? 'Calculating...' : 'Suggest Rebalance'}
              </Button>
            )
          }
        />
      </Card>

      {/* Drift Warning */}
      {hasNegativeDrift && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          <InfoIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          One or more sleeves have exceeded drift threshold. Rebalancing recommended.
        </Alert>
      )}

      {/* Allocation Summary */}
      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6}>
              <Typography variant="subtitle2" gutterBottom>
                Total Current Allocation
              </Typography>
              <Typography variant="h5" color={totalAllocation === 1 ? 'success.main' : 'error.main'}>
                {(totalAllocation * 100).toFixed(2)}%
              </Typography>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Typography variant="subtitle2" gutterBottom>
                Number of Sleeves
              </Typography>
              <Typography variant="h5">{umaAccount.sleeves.length}</Typography>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* ReactFlow Canvas */}
      <Box sx={{ flex: 1, border: '1px solid #ccc', borderRadius: 1, mb: 2 }}>
        <ReactFlow nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}>
          <Background />
          <Controls />
        </ReactFlow>
      </Box>

      {/* Sleeves Table */}
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
              <TableCell><strong>Model</strong></TableCell>
              <TableCell><strong>Type</strong></TableCell>
              <TableCell align="right"><strong>Target %</strong></TableCell>
              <TableCell align="right"><strong>Current %</strong></TableCell>
              <TableCell align="right"><strong>Drift %</strong></TableCell>
              <TableCell align="right"><strong>Threshold %</strong></TableCell>
              <TableCell><strong>Status</strong></TableCell>
              {!readOnly && <TableCell align="center"><strong>Actions</strong></TableCell>}
            </TableRow>
          </TableHead>
          <TableBody>
            {umaAccount.sleeves.map((sleeve) => (
              <TableRow key={sleeve.id}>
                <TableCell>{sleeve.model}</TableCell>
                <TableCell>{sleeve.sleeveType}</TableCell>
                <TableCell align="right">{(sleeve.targetAllocation * 100).toFixed(2)}%</TableCell>
                <TableCell align="right">{(sleeve.currentAllocation * 100).toFixed(2)}%</TableCell>
                <TableCell
                  align="right"
                  sx={{
                    color:
                      sleeve.drift > sleeve.minDriftThreshold ? '#d32f2f' : '#4caf50',
                    fontWeight: 'bold',
                  }}
                >
                  {(sleeve.drift * 100).toFixed(2)}%
                </TableCell>
                <TableCell align="right">{(sleeve.minDriftThreshold * 100).toFixed(2)}%</TableCell>
                <TableCell>
                  <Chip
                    label={sleeve.status}
                    size="small"
                    color={sleeve.status === 'active' ? 'success' : 'default'}
                  />
                </TableCell>
                {!readOnly && (
                  <TableCell align="center">
                    <Tooltip title="Edit Sleeve">
                      <IconButton
                        size="small"
                        onClick={() => handleEditSleeve(sleeve)}
                      >
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Edit Sleeve Dialog */}
      <Dialog open={editDialogOpen} onClose={() => setEditDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Sleeve</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <TextField
            fullWidth
            label="Model"
            value={sleeveFormData.model || ''}
            onChange={(e) => setSleeveFormData({ ...sleeveFormData, model: e.target.value })}
            margin="normal"
            disabled
          />
          <TextField
            fullWidth
            label="Sleeve Type"
            value={sleeveFormData.sleeveType || ''}
            onChange={(e) => setSleeveFormData({ ...sleeveFormData, sleeveType: e.target.value })}
            margin="normal"
            disabled
          />
          <TextField
            fullWidth
            label="Target Allocation (%)"
            type="number"
            value={(sleeveFormData.targetAllocation || 0) * 100}
            onChange={(e) =>
              setSleeveFormData({
                ...sleeveFormData,
                targetAllocation: parseFloat(e.target.value) / 100,
              })
            }
            margin="normal"
            disabled={readOnly}
          />
          <TextField
            fullWidth
            label="Min Drift Threshold (%)"
            type="number"
            value={(sleeveFormData.minDriftThreshold || 0) * 100}
            onChange={(e) =>
              setSleeveFormData({
                ...sleeveFormData,
                minDriftThreshold: parseFloat(e.target.value) / 100,
              })
            }
            margin="normal"
            disabled={readOnly}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditDialogOpen(false)}>Cancel</Button>
          {!readOnly && (
            <Button onClick={handleSaveSleeve} variant="contained">
              Save
            </Button>
          )}
        </DialogActions>
      </Dialog>

      {/* Rebalance Plan Dialog */}
      <Dialog
        open={rebalanceDialogOpen || !!rebalancePlan}
        onClose={() => {
          setRebalanceDialogOpen(false);
          setRebalancePlan(null);
        }}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Rebalance Plan</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {rebalanceMutation.isPending ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
              <CircularProgress />
            </Box>
          ) : rebalancePlan ? (
            <>
              <Typography variant="subtitle1" gutterBottom>
                Suggested Trades
              </Typography>
              <TableContainer sx={{ mb: 2 }}>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell><strong>Symbol</strong></TableCell>
                      <TableCell align="right"><strong>Side</strong></TableCell>
                      <TableCell align="right"><strong>Quantity</strong></TableCell>
                      <TableCell align="right"><strong>Price</strong></TableCell>
                      <TableCell align="right"><strong>Value</strong></TableCell>
                      <TableCell><strong>Reason</strong></TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {rebalancePlan.trades.map((trade, idx) => (
                      <TableRow key={idx}>
                        <TableCell>{trade.symbol}</TableCell>
                        <TableCell align="right">
                          <Chip
                            label={trade.side.toUpperCase()}
                            color={trade.side === 'buy' ? 'success' : 'error'}
                            size="small"
                          />
                        </TableCell>
                        <TableCell align="right">{trade.quantity.toFixed(2)}</TableCell>
                        <TableCell align="right">${trade.estimatedPrice.toFixed(2)}</TableCell>
                        <TableCell align="right">
                          ${trade.estimatedValue.toFixed(2)}
                        </TableCell>
                        <TableCell>{trade.reason}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>

              <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                <TextField
                  fullWidth
                  label="Approval Notes (optional)"
                  multiline
                  rows={2}
                  value={approvalSignal}
                  onChange={(e) => setApprovalSignal(e.target.value)}
                  disabled={readOnly}
                />
              </Box>

              <Typography variant="caption" color="textSecondary">
                Status: {rebalancePlan.approvalStatus}
              </Typography>
            </>
          ) : null}
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => {
              setRebalanceDialogOpen(false);
              setRebalancePlan(null);
            }}
          >
            Close
          </Button>
          {!readOnly && rebalancePlan?.approvalStatus === 'pending_approval' && (
            <Button
              onClick={handleApproveRebalance}
              variant="contained"
              disabled={approveMutation.isPending}
            >
              {approveMutation.isPending ? 'Approving...' : 'Approve & Execute'}
            </Button>
          )}
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default UMABuilder;
