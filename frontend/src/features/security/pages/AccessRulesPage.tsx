import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { accessRulesApi, AccessRule, RuleStatus } from '../../../api/accessRules';

const statuses: RuleStatus[] = ['DRAFT', 'REVIEW', 'APPROVED', 'DEPRECATED'];

export const AccessRulesPage: React.FC = () => {
  const navigate = useNavigate();
  const [rules, setRules] = useState<AccessRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [boId, setBoId] = useState('');
  const [groupDn, setGroupDn] = useState('');
  const [status, setStatus] = useState<RuleStatus | ''>('');

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await accessRulesApi.list({ businessObjectId: boId || undefined, groupDn: groupDn || undefined, status: status || undefined });
      setRules(data);
    } catch (e: any) {
      setError(e?.message || 'Failed to load access rules');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filtered = useMemo(() => rules, [rules]);

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">Access Rules</h1>
          <p className="text-sm text-gray-500">Manage row filters, column masks, and access levels per Business Object.</p>
        </div>
        <button
          className="px-3 py-2 rounded bg-blue-600 text-white text-sm"
          onClick={() => navigate('/security/access-rules/new')}
        >
          New Rule
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
        <div>
          <label className="text-xs text-gray-600">Business Object ID</label>
          <input value={boId} onChange={e => setBoId(e.target.value)} className="w-full border rounded px-2 py-1" placeholder="bo:portfolio" />
        </div>
        <div>
          <label className="text-xs text-gray-600">Group DN</label>
          <input value={groupDn} onChange={e => setGroupDn(e.target.value)} className="w-full border rounded px-2 py-1" placeholder="cn=group,ou=..." />
        </div>
        <div>
          <label htmlFor="status-filter" className="text-xs text-gray-600">Status</label>
          <select id="status-filter" value={status} onChange={e => setStatus(e.target.value as RuleStatus | '')} className="w-full border rounded px-2 py-1">
            <option value="">All</option>
            {statuses.map(s => (<option key={s} value={s}>{s}</option>))}
          </select>
        </div>
        <div className="flex gap-2">
          <button onClick={() => void load()} className="px-3 py-2 rounded bg-gray-200 text-sm">Filter</button>
          <button onClick={() => { setBoId(''); setGroupDn(''); setStatus(''); void load(); }} className="px-3 py-2 rounded bg-gray-100 text-sm">Reset</button>
        </div>
      </div>

      {loading && <div className="text-sm text-gray-600">Loading...</div>}
      {error && <div className="text-sm text-red-600">{error}</div>}

      <div className="overflow-auto border rounded">
        <table className="min-w-full text-sm">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-3 py-2 text-left">Rule ID</th>
              <th className="px-3 py-2 text-left">Business Object</th>
              <th className="px-3 py-2 text-left">Group DN</th>
              <th className="px-3 py-2 text-left">Access</th>
              <th className="px-3 py-2 text-left">Status</th>
              <th className="px-3 py-2 text-left">Masks</th>
              <th className="px-3 py-2 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(rule => (
              <tr key={rule.ruleId} className="border-t">
                <td className="px-3 py-2 font-mono text-xs">{rule.ruleId}</td>
                <td className="px-3 py-2">{rule.businessObjectId}</td>
                <td className="px-3 py-2 text-xs">{rule.groupDn}</td>
                <td className="px-3 py-2">{rule.accessLevel}</td>
                <td className="px-3 py-2">{rule.status}</td>
                <td className="px-3 py-2 text-xs">{rule.columnMasks?.length || 0}</td>
                <td className="px-3 py-2 space-x-2">
                  <Link className="text-blue-600" to={`/security/access-rules/${rule.ruleId}`}>Edit</Link>
                </td>
              </tr>
            ))}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={7} className="px-3 py-4 text-center text-sm text-gray-500">No access rules found.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default AccessRulesPage;
