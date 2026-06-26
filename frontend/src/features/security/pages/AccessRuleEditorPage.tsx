import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { accessRulesApi, AccessRuleInput, AccessLevel, RuleStatus, MaskType } from '../../../api/accessRules';
import { RulePreview } from '../components/RulePreview';
import { RuleTest } from '../components/RuleTest';

const accessLevels: AccessLevel[] = ['NONE', 'READ', 'WRITE'];
const statuses: RuleStatus[] = ['DRAFT', 'REVIEW', 'APPROVED', 'DEPRECATED'];
const maskTypes: MaskType[] = ['NONE', 'MASK', 'HIDE'];

interface MaskRow {
  semanticTermId: string;
  maskType: MaskType;
}

export const AccessRuleEditorPage: React.FC = () => {
  const { ruleId } = useParams();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<'edit' | 'preview' | 'test'>('edit');

  const [model, setModel] = useState<AccessRuleInput>({
    ruleId: '',
    tenantId: '',
    businessObjectId: '',
    groupDn: '',
    accessLevel: 'READ',
    status: 'DRAFT',
    rowFilterDsl: '',
    columnMasks: [],
    scope: { appliesToApis: true, appliesToBi: true, appliesToAi: true },
  });
  const [maskRows, setMaskRows] = useState<MaskRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validation, setValidation] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      if (!ruleId) return;
      setLoading(true);
      try {
        const data = await accessRulesApi.get(ruleId);
        setModel({ ...data });
        setMaskRows(data.columnMasks || []);
      } catch (e: any) {
        setError(e?.message || 'Failed to load rule');
      } finally {
        setLoading(false);
      }
    };
    void load();
  }, [ruleId]);

  const upsert = async () => {
    setLoading(true);
    setError(null);
    try {
      const payload: AccessRuleInput = { ...model, columnMasks: maskRows };
      if (ruleId) {
        await accessRulesApi.update(ruleId, payload);
      } else {
        await accessRulesApi.create(payload);
      }
      navigate('/security/access-rules');
    } catch (e: any) {
      setError(e?.message || 'Failed to save rule');
    } finally {
      setLoading(false);
    }
  };

  const validateDsl = async () => {
    if (!model.rowFilterDsl) {
      setValidation('No predicate set; rule will allow all rows for matched groups.');
      return;
    }
    try {
      const res = await accessRulesApi.validate({ rowFilterDsl: model.rowFilterDsl, businessObjectId: model.businessObjectId });
      if (res.valid) setValidation(res.sql ? `Valid. SQL: ${res.sql}` : 'Valid predicate');
      else setValidation(res.error || 'Invalid predicate');
    } catch (e: any) {
      setValidation(e?.message || 'Validation failed');
    }
  };

  const updateMaskRow = (idx: number, next: Partial<MaskRow>) => {
    setMaskRows(prev => prev.map((r, i) => i === idx ? { ...r, ...next } : r));
  };

  const addMaskRow = () => setMaskRows(prev => [...prev, { semanticTermId: '', maskType: 'MASK' }]);
  const removeMaskRow = (idx: number) => setMaskRows(prev => prev.filter((_, i) => i !== idx));

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">{ruleId ? 'Edit Access Rule' : 'New Access Rule'}</h1>
          <p className="text-sm text-gray-500">Bind LDAP groups to Business Objects with row filters and masks.</p>
        </div>
        <button className="px-3 py-2 rounded bg-gray-100 text-sm" onClick={() => navigate('/security/access-rules')}>Back</button>
      </div>

      {error && <div className="text-sm text-red-600">{error}</div>}
      {loading && <div className="text-sm text-gray-600">Loading...</div>}

      {/* Tab Navigation */}
      <div className="border-b">
        <nav className="flex gap-4">
          <button
            className={`px-4 py-2 text-sm font-medium border-b-2 ${
              activeTab === 'edit' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-600 hover:text-gray-800'
            }`}
            onClick={() => setActiveTab('edit')}
          >
            Edit
          </button>
          <button
            className={`px-4 py-2 text-sm font-medium border-b-2 ${
              activeTab === 'preview' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-600 hover:text-gray-800'
            }`}
            onClick={() => setActiveTab('preview')}
          >
            Preview
          </button>
          <button
            className={`px-4 py-2 text-sm font-medium border-b-2 ${
              activeTab === 'test' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-600 hover:text-gray-800'
            }`}
            onClick={() => setActiveTab('test')}
          >
            Test
          </button>
        </nav>
      </div>

      {/* Edit Tab */}
      {activeTab === 'edit' && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-3">
              <div>
                <label className="text-xs text-gray-600">Rule ID (optional)</label>
                <input value={model.ruleId || ''} onChange={e => setModel(m => ({ ...m, ruleId: e.target.value }))} className="w-full border rounded px-2 py-1" placeholder="auto-generate if blank" />
              </div>
              <div>
                <label className="text-xs text-gray-600">Tenant ID</label>
                <input value={model.tenantId} onChange={e => setModel(m => ({ ...m, tenantId: e.target.value }))} className="w-full border rounded px-2 py-1" placeholder="tenant guid" />
              </div>
              <div>
                <label className="text-xs text-gray-600">Business Object ID</label>
                <input value={model.businessObjectId} onChange={e => setModel(m => ({ ...m, businessObjectId: e.target.value }))} className="w-full border rounded px-2 py-1" placeholder="bo:portfolio" />
              </div>
              <div>
                <label className="text-xs text-gray-600">LDAP Group DN</label>
                <input value={model.groupDn} onChange={e => setModel(m => ({ ...m, groupDn: e.target.value }))} className="w-full border rounded px-2 py-1" placeholder="cn=group,ou=..." />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="access-level" className="text-xs text-gray-600">Access Level</label>
                  <select id="access-level" value={model.accessLevel} onChange={e => setModel(m => ({ ...m, accessLevel: e.target.value as AccessLevel }))} className="w-full border rounded px-2 py-1">
                    {accessLevels.map(l => (<option key={l} value={l}>{l}</option>))}
                  </select>
                </div>
                <div>
                  <label htmlFor="status" className="text-xs text-gray-600">Status</label>
                  <select id="status" value={model.status} onChange={e => setModel(m => ({ ...m, status: e.target.value as RuleStatus }))} className="w-full border rounded px-2 py-1">
                    {statuses.map(s => (<option key={s} value={s}>{s}</option>))}
                  </select>
                </div>
              </div>
            </div>

            <div className="space-y-3">
              <div>
                <label className="text-xs text-gray-600">Row Filter DSL</label>
                <textarea value={model.rowFilterDsl || ''} onChange={e => setModel(m => ({ ...m, rowFilterDsl: e.target.value }))} className="w-full border rounded px-2 py-2 h-28" placeholder="region = 'EMEA' AND client_type != 'internal'" />
                <div className="flex gap-2 mt-1">
                  <button onClick={() => void validateDsl()} type="button" className="px-2 py-1 text-xs rounded bg-gray-100">Validate</button>
                  {validation && <span className="text-xs text-gray-600">{validation}</span>}
                </div>
              </div>
              <div className="grid grid-cols-3 gap-2">
                <label className="flex items-center gap-2 text-xs">
                  <input type="checkbox" checked={!!model.scope?.appliesToApis} onChange={e => setModel(m => ({ ...m, scope: { ...m.scope, appliesToApis: e.target.checked } }))} /> APIs
                </label>
                <label className="flex items-center gap-2 text-xs">
                  <input type="checkbox" checked={!!model.scope?.appliesToBi} onChange={e => setModel(m => ({ ...m, scope: { ...m.scope, appliesToBi: e.target.checked } }))} /> BI
                </label>
                <label className="flex items-center gap-2 text-xs">
                  <input type="checkbox" checked={!!model.scope?.appliesToAi} onChange={e => setModel(m => ({ ...m, scope: { ...m.scope, appliesToAi: e.target.checked } }))} /> AI
                </label>
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h2 className="font-semibold">Column Masks</h2>
              <button className="px-2 py-1 text-xs rounded bg-gray-100" onClick={addMaskRow}>Add Mask</button>
            </div>
            <div className="overflow-auto border rounded">
              <table className="min-w-full text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-3 py-2 text-left">Semantic Term ID</th>
                    <th className="px-3 py-2 text-left">Mask</th>
                    <th className="px-3 py-2"></th>
                  </tr>
                </thead>
                <tbody>
                  {maskRows.map((row, idx) => (
                    <tr key={idx} className="border-t">
                      <td className="px-3 py-2">
                        <input value={row.semanticTermId} onChange={e => updateMaskRow(idx, { semanticTermId: e.target.value })} className="w-full border rounded px-2 py-1" placeholder="term:client_ssn" />
                      </td>
                      <td className="px-3 py-2">
                        <select aria-label="Mask Type" value={row.maskType} onChange={e => updateMaskRow(idx, { maskType: e.target.value as MaskType })} className="w-full border rounded px-2 py-1">
                          {maskTypes.map(m => (<option key={m} value={m}>{m}</option>))}
                        </select>
                      </td>
                      <td className="px-3 py-2 text-right">
                        <button className="text-xs text-red-600" onClick={() => removeMaskRow(idx)}>Remove</button>
                      </td>
                    </tr>
                  ))}
                  {maskRows.length === 0 && (
                    <tr><td colSpan={3} className="px-3 py-3 text-sm text-gray-500 text-center">No masks defined.</td></tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          <div className="flex gap-2">
            <button onClick={() => void upsert()} className="px-3 py-2 rounded bg-blue-600 text-white text-sm" disabled={loading}>Save</button>
            <button onClick={() => navigate('/security/access-rules')} className="px-3 py-2 rounded bg-gray-100 text-sm">Cancel</button>
          </div>
        </>
      )}

      {/* Preview Tab */}
      {activeTab === 'preview' && (
        <RulePreview rule={model} />
      )}

      {/* Test Tab */}
      {activeTab === 'test' && (
        <RuleTest rule={model} />
      )}
    </div>
  );
};

export default AccessRuleEditorPage;
