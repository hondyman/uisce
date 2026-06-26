import { useState, useEffect, useCallback } from 'react';
import { getEffectiveClaims, revokeDirectClaim, listSemanticViews, requestClaimRenewal } from '../../api';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import { SemanticModelClaim } from './types';
import './ClaimsTable.css';

interface ClaimsTableProps {
  domain?: string;
  statusFilter?: string;
}

export default function ClaimsTable({ domain, statusFilter }: ClaimsTableProps) {
  const [claims, setClaims] = useState<SemanticModelClaim[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterUser, setFilterUser] = useState('');
  const [renewDialogOpen, setRenewDialogOpen] = useState(false);
  const [renewTargetClaim, setRenewTargetClaim] = useState<string | null>(null);
  const [renewReason, setRenewReason] = useState('');
  const [modelDomainMap, setModelDomainMap] = useState<Map<string, string>>(new Map());

  // Fetch all models to map their domains
  useEffect(() => {
    listSemanticViews("mock-datasource-id").then((models: any[]) => {
      const newMap = new Map<string, string>();
      models.forEach((m: any) => {
        if (m && m.domain) {
          newMap.set(m.id, m.domain);
        }
      });
      setModelDomainMap(newMap);
    }).catch((e: unknown) => {
      // Log in dev environment
      const { devError } = require('../utils/devLogger');
      devError(e);
    });
  }, []);

  const fetchClaims = useCallback(async (userId: string) => {
    if (!userId) {
      setClaims([]);
      return;
    }
    try {
      setLoading(true);
      const data = await getEffectiveClaims(userId);
      // Filter for direct claims only for this table
  const directClaims = data.filter((c: SemanticModelClaim) => !c.granted_by.startsWith('role:'));

  let filteredByDomain = domain ? directClaims.filter((c: SemanticModelClaim) => modelDomainMap.get(c.model_id) === domain) : directClaims;

      // NEW: Filter by status
      if (statusFilter && statusFilter !== 'all') {
  filteredByDomain = filteredByDomain.filter((c: SemanticModelClaim) => c.status === statusFilter);
      }

      setClaims(filteredByDomain);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch claims');
    } finally {
      setLoading(false);
    }
  }, [domain, modelDomainMap, statusFilter]);

  useEffect(() => {
    if (filterUser) {
      fetchClaims(filterUser);
    } else {
      setClaims([]);
    }
  }, [filterUser, fetchClaims]);

  const handleRevoke = async (claimId: string) => {
    const confirm = useConfirm();
    const notification = useNotification();
    if (!(await confirm({ title: 'Revoke claim', description: 'Are you sure you want to revoke this claim?' }))) return;
    try {
      await revokeDirectClaim(claimId);
      // Refresh list
      fetchClaims(filterUser);
      notification.success('Claim revoked');
    } catch (err) {
      notification.error('Failed to revoke claim');
    }
  };

  const handleRenew = async (claimId: string) => {
    // Show a dialog to capture a short reason for renewal
    setRenewDialogOpen(true);
    setRenewTargetClaim(claimId);
  };

  const submitRenewal = async () => {
    if (!renewTargetClaim) return;
    try {
      await requestClaimRenewal(renewTargetClaim, renewReason);
      notification.success('Renewal requested');
      setRenewDialogOpen(false);
      setRenewReason('');
      fetchClaims(filterUser);
    } catch (err) {
      notification.error('Failed to request renewal.');
    }
  };

  const getExpiryStatus = (claim: SemanticModelClaim): { text: string; statusClass?: string; isActionable: boolean } => {
    if (!claim.expires_at) return { text: 'Never', statusClass: undefined, isActionable: false };

    const expiryDate = new Date(claim.expires_at);
    const now = new Date();
    const daysUntilExpiry = (expiryDate.getTime() - now.getTime()) / (1000 * 3600 * 24);

    if (daysUntilExpiry < 0) return { text: 'Expired', statusClass: 'expired', isActionable: false };
    if (daysUntilExpiry <= 7) return { text: `in ${Math.ceil(daysUntilExpiry)} days`, statusClass: 'expiring', isActionable: true };

    return { text: new Date(claim.expires_at).toLocaleDateString(), statusClass: undefined, isActionable: false };
  };

  // TODO: Add a form to grant new claims.

  return (
    <div>
      <h2>Direct User Claims</h2>
      {!statusFilter && ( // Only show user filter if not part of the lifecycle dashboard
  <div className="filter-bar">
          <label htmlFor="user-filter">User ID: </label>
          <input
            id="user-filter"
            type="text"
            value={filterUser}
            onChange={(e) => setFilterUser(e.target.value)}
            placeholder="Enter user ID to view claims"
          />
        </div>
      )}

      {loading && filterUser && <p>Loading...</p>}
  {error && <div className="error-text">Error: {error}</div>}

      <table className="governance-table">
        <thead>
          <tr>
            <th>User</th>
            <th>Model ID</th>
            <th>Permission</th>
            <th>Status / Expires</th>
            <th>Granted By</th>
            <th>Granted At</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {claims.map((claim: SemanticModelClaim) => (
            (() => {
              const statusDisplay = getExpiryStatus(claim);
              return (
                <tr key={claim.id}>
                  <td>{claim.user_id}</td>
                  <td><code>{claim.model_id}</code></td>
                  <td>{claim.permission}</td>
                  <td className={`status-text ${statusDisplay.statusClass ?? ''}`}>
                    {statusDisplay.text}
                  </td>
                  <td>{claim.granted_by}</td>
                  <td>{new Date(claim.granted_at).toLocaleString()}</td>
                  <td className="actions">
                    {statusDisplay.isActionable && !claim.renewal_requested && (
                      <button className="approve" onClick={() => handleRenew(claim.id)}>Renew</button>
                    )}
                    {claim.renewal_requested && (<span className="pending-text">Pending</span>)}
                    <button className="reject" onClick={() => handleRevoke(claim.id)}>Revoke</button>
                  </td>
                </tr>
              );
            })()
          ))}
        </tbody>
      </table>
      {/* Renewal Reason Dialog */}
      {renewDialogOpen && (
        <div className="claims-dialog">
          <div className="claims-dialog-content">
            <h3>Request renewal</h3>
            <p>Please provide a short reason for the renewal:</p>
            <label htmlFor="renew-reason" className="sr-only">Reason for renewal</label>
            <textarea id="renew-reason" aria-label="Reason for renewal" placeholder="e.g., Continued work on project" value={renewReason} onChange={(e) => setRenewReason(e.target.value)} rows={3} />
            <div className="claims-dialog-actions">
              <button onClick={() => setRenewDialogOpen(false)}>Cancel</button>
              <button onClick={submitRenewal} disabled={!renewReason.trim()}>Submit</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}