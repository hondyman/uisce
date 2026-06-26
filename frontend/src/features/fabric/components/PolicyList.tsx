// React default import removed (not used as a value)
import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow, IconButton, Chip, Tooltip } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import { AccessControlPolicy } from '../../../types';

interface PolicyListProps {
  policies: AccessControlPolicy[];
  onEdit: (policy: AccessControlPolicy) => void;
}

export default function PolicyList({ policies, onEdit }: PolicyListProps) {
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Policy ID</TableCell>
            <TableCell>Scope</TableCell>
            <TableCell>Role</TableCell>
            <TableCell>Permissions</TableCell>
            <TableCell>Duration</TableCell>
            <TableCell>Certification</TableCell>
            <TableCell>Max Claims</TableCell>
            <TableCell>Approval Threshold</TableCell>
            <TableCell>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {policies.map((policy) => (
            <TableRow key={policy.id} hover>
              <TableCell sx={{ fontFamily: 'monospace' }}>{policy.policy_id}</TableCell>
              <TableCell>{policy.scope}</TableCell>
              <TableCell>{policy.role}</TableCell>
              <TableCell>
                {policy.permissions.map(p => <Chip key={p} label={p} size="small" sx={{ mr: 0.5 }} />)}
              </TableCell>
              <TableCell>{policy.duration_days} days</TableCell>
              <TableCell>{policy.requires_certification ? 'Yes' : 'No'}</TableCell>
              <TableCell>{policy.max_claims_per_user}</TableCell>
              <TableCell>{policy.approval_threshold}</TableCell>
              <TableCell>
                <Tooltip title="Edit Policy">
                  <IconButton onClick={() => onEdit(policy)} size="small">
                    <EditIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}