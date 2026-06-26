import { useState } from 'react';
import type { FC } from 'react';
import {
  DataGrid,
  GridColDef,
} from '@mui/x-data-grid';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Select,
  Stack,
  Button,
  Box,
} from '@mui/material';
import { LocalizationProvider, DatePicker } from '@mui/x-date-pickers';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import ActionButton from '../ui/ActionButton';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNotification } from '../../hooks/useNotification';
import type { Dayjs } from 'dayjs';

/**
 * DelegationManager Component
 * 
 * Manages temporary role delegations for ABAC policies.
 * Allows managers to temporarily grant their access to another user.
 * 
 * Features:
 * - View active delegations
 * - Revoke delegations
 * - Create new delegations with expiry
 * - Audit trail of all delegations
 */

interface Delegation {
  id: string;
  from_user_id: string;
  from_user_name: string;
  to_user_id: string;
  to_user_name: string;
  policy_id: string;
  policy_name: string;
  expires_at: string;
  created_at: string;
  reason?: string;
}

interface DelegationManagerProps {
  tenantId: string;
  baseUrl?: string;
}

export const DelegationManager: FC<DelegationManagerProps> = ({
  tenantId,
  baseUrl = '/api',
}) => {
  const notification = useNotification();
  const queryClient = useQueryClient();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedUsers, setSelectedUsers] = useState<{
    toUserId: string;
    policyId: string;
    expiresAt: Dayjs | null;
  }>({ toUserId: '', policyId: '', expiresAt: null });

  // Fetch active delegations
  const { data: delegations = [], isLoading } = useQuery<Delegation[]>({
    queryKey: ['delegations', tenantId],
    queryFn: async () => {
      const response = await fetch(`${baseUrl}/abac/delegations`, {
        headers: {
          'X-Tenant-ID': tenantId,
        },
      });
      if (!response.ok) throw new Error('Failed to load delegations');
      return response.json();
    },
  });

  // Create delegation
  const createMutation = useMutation({
    mutationFn: (data: {
      to_user_id: string;
      policy_id: string;
      expires_at: string;
      reason?: string;
    }) =>
      fetch(`${baseUrl}/abac/delegations`, {
        method: 'POST',
        headers: {
          'X-Tenant-ID': tenantId,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      })
        .then((r) => {
          if (!r.ok) throw new Error('Failed to create delegation');
          return r.json();
        }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['delegations', tenantId] });
      notification.success('Delegation created');
      setIsModalOpen(false);
    },
  });

  // Revoke delegation
  const revokeMutation = useMutation({
    mutationFn: (id: string) =>
      fetch(`${baseUrl}/abac/delegations/${id}`, {
        method: 'DELETE',
        headers: {
          'X-Tenant-ID': tenantId,
        },
      })
        .then((r) => {
          if (!r.ok) throw new Error('Failed to revoke delegation');
        }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['delegations', tenantId] });
      notification.success('Delegation revoked');
    },
  });

  const columns: GridColDef[] = [
    {
      field: 'from_user_name',
      headerName: 'From User',
      flex: 1,
      minWidth: 120,
    },
    {
      field: 'to_user_name',
      headerName: 'To User',
      flex: 1,
      minWidth: 120,
    },
    {
      field: 'policy_name',
      headerName: 'Policy',
      flex: 1,
      minWidth: 120,
    },
    {
      field: 'expires_at',
      headerName: 'Expires',
      flex: 1,
      minWidth: 180,
      renderCell: (params: any) => new Date(params.value).toLocaleString(),
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: 120,
      renderCell: (params: any) => (
        <ActionButton
          size="sm"
          variant="danger"
          iconName="close"
          onClick={() => revokeMutation.mutate(params.row.id)}
          disabled={revokeMutation.isPending}
        >
          Revoke
        </ActionButton>
      ),
    },
  ];

  return (
    <Box className="delegation-manager">
      <Box sx={{ marginBottom: 2 }}>
        <ActionButton variant="primary" onClick={() => setIsModalOpen(true)}>
          Create Delegation
        </ActionButton>
      </Box>

      <Box sx={{ height: 400, width: '100%' }}>
        <DataGrid
          columns={columns}
          rows={delegations}
          getRowId={(row) => row.id}
          loading={isLoading}
          pageSizeOptions={[10, 25, 50]}
          initialState={{
            pagination: {
              paginationModel: { pageSize: 10, page: 0 },
            },
          }}
        />
      </Box>

      <Dialog open={isModalOpen} onClose={() => setIsModalOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create Delegation</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 2 }}>
            <Box>
              <label>Delegate To User</label>
              <Select
                fullWidth
                value={selectedUsers.toUserId}
                onChange={(e) =>
                  setSelectedUsers({ ...selectedUsers, toUserId: e.target.value })
                }
                placeholder="Select user"
              />
            </Box>

            <Box>
              <label>Policy</label>
              <Select
                fullWidth
                value={selectedUsers.policyId}
                onChange={(e) =>
                  setSelectedUsers({ ...selectedUsers, policyId: e.target.value })
                }
                placeholder="Select policy"
              />
            </Box>

            <Box>
              <label>Expires At</label>
              <LocalizationProvider dateAdapter={AdapterDayjs}>
                <DatePicker
                  value={selectedUsers.expiresAt}
                  onChange={(date: any) =>
                    setSelectedUsers({ ...selectedUsers, expiresAt: date })
                  }
                  slotProps={{
                    textField: { fullWidth: true },
                  }}
                />
              </LocalizationProvider>
            </Box>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setIsModalOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={() => {
              if (!selectedUsers.toUserId || !selectedUsers.policyId) {
                notification.error('Please select a user and policy');
                return;
              }
              createMutation.mutate({
                to_user_id: selectedUsers.toUserId,
                policy_id: selectedUsers.policyId,
                expires_at: selectedUsers.expiresAt?.toISOString() || '',
              });
            }}
            disabled={createMutation.isPending}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default DelegationManager;
