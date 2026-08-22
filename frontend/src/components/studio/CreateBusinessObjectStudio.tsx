import React, { useState } from 'react';
import {
  Sparkles,
  Layers,
  Send,
  Star,
  CheckCircle2,
  Plus,
} from 'lucide-react';

interface SelectedField {
  termNodeId: string;
  termKey: string;
  fieldName: string;
  fieldRole: string; // DIMENSION, MEASURE, KEY
  bindingRequirement: string; // REQUIRED, OPTIONAL, BACKEND_SPECIFIC
  sourceNodeId?: string;
  sourceType: string;
  transformationType: string;
  transformationSql?: string;
  overrideReason?: string;
}

interface BindingCardState {
  backendId: string;
  drivingNodeId: string;
  isDefault: boolean;
  tableName: string;
  discoveredPK?: string;
  relatedTables: string[];
  fields: SelectedField[];
}

export const CreateBusinessObjectStudio: React.FC<{
  tenantId: string;
  modelId: string;
  onSuccess?: (boId: string) => void;
}> = ({ tenantId, modelId, onSuccess }) => {
  // 1. Definition State
  const [boName, setBoName] = useState('Customer');
  const [boKey, setBoKey] = useState('customer');
  const [boType, setBoType] = useState('ENTITY');
  const [businessKeyNodeId] = useState('');
  const [semanticIdNodeId] = useState('');

  // 2. Multi-Backend Binding Panels
  const [bindings, setBindings] = useState<BindingCardState[]>([
    {
      backendId: 'postgres-alpha-id',
      drivingNodeId: '',
      isDefault: true,
      tableName: '',
      relatedTables: [],
      fields: [],
    },
  ]);

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [statusMsg, setStatusMsg] = useState<string | null>(null);

  // Auto-Discovery Invocation
  const handleSelectDrivingTable = async (bindingIndex: number, drivingNodeId: string) => {
    try {
      const res = await fetch('/api/v1/business-objects/binding-context', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
          backendId: bindings[bindingIndex].backendId,
          drivingNodeId,
        }),
      });

      if (res.ok) {
        const data = await res.json();
        const updated = [...bindings];
        updated[bindingIndex].drivingNodeId = drivingNodeId;
        updated[bindingIndex].tableName = data.drivingTable.tableName;
        updated[bindingIndex].relatedTables = data.relatedTables.map((r: any) => r.tableName);

        // Auto-select discovered terms and auto-create field bindings
        updated[bindingIndex].fields = data.eligibleTerms.map((t: any) => ({
          termNodeId: t.termNodeId,
          termKey: t.termKey,
          fieldName: t.termKey,
          fieldRole: t.identityRole === 'BUSINESS_KEY' ? 'KEY' : 'DIMENSION',
          bindingRequirement: 'REQUIRED',
          sourceNodeId: t.mappings[0]?.columnNodeId,
          sourceType: 'COLUMN',
          transformationType: 'NONE',
        }));

        setBindings(updated);
      }
    } catch (err) {
      console.error('Failed auto-discovery:', err);
    }
  };

  const handleSaveAndPublish = async () => {
    setIsSubmitting(true);
    try {
      const payload = {
        tenantId,
        modelId,
        publish: true,
        businessObject: {
          boKey,
          boname: boName,
          boType,
          classificationNodeId: 'c0000000-0000-0000-0000-000000000003', // Level 3 Classification
          businessKeyNodeId: businessKeyNodeId || bindings[0]?.fields[0]?.termNodeId,
          semanticIdNodeId: semanticIdNodeId || bindings[0]?.fields[0]?.termNodeId,
          grainNodeId: businessKeyNodeId || bindings[0]?.fields[0]?.termNodeId,
        },
        bindings: bindings.map((b) => ({
          backendId: b.backendId,
          drivingNodeId: b.drivingNodeId,
          isDefault: b.isDefault,
          temporalOverride: 'NONE',
          fields: b.fields,
          relationships: [],
        })),
      };

      const res = await fetch('/api/v1/business-objects/save', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        const result = await res.json();
        setStatusMsg('Business Object & Bindings Published Successfully!');
        if (onSuccess) onSuccess(result.boId);
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="flex flex-col h-full bg-[#030914] text-slate-100 border border-slate-800 rounded-xl overflow-hidden font-sans">
      {/* Studio Header */}
      <div className="p-6 bg-[#071526] border-b border-slate-800 flex items-center justify-between">
        <div>
          <h2 className="text-base font-bold text-slate-100 flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-cyan-400" />
            Single-Screen Business Object Studio
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Auto-discover tables, primary keys, relationships, and mapped semantic terms in one unified canvas.
          </p>
        </div>
        <button
          onClick={handleSaveAndPublish}
          disabled={isSubmitting}
          className="px-5 py-2.5 bg-gradient-to-r from-cyan-500 to-emerald-400 text-slate-950 font-bold rounded-lg shadow hover:opacity-95 text-xs flex items-center gap-2 disabled:opacity-50 transition"
        >
          <Send className="w-4 h-4" /> Save & Publish Business Object
        </button>
      </div>

      {statusMsg && (
        <div className="bg-emerald-500/20 border-b border-emerald-500/40 px-6 py-2 text-xs text-emerald-300 flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 text-emerald-400" /> {statusMsg}
        </div>
      )}

      {/* Main Studio Canvas */}
      <div className="p-6 space-y-6 flex-1 overflow-y-auto">
        {/* Top Definition Section */}
        <div className="p-5 bg-slate-900/60 border border-slate-800 rounded-xl space-y-4">
          <span className="text-xs font-bold text-slate-300 uppercase tracking-wider block">
            1. Semantic Contract Definition
          </span>
          <div className="grid grid-cols-4 gap-4">
            <div>
              <label className="text-xs font-semibold text-slate-400 mb-1 block">BO Name</label>
              <input
                type="text"
                value={boName}
                onChange={(e) => setBoName(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 font-semibold"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-400 mb-1 block">BO Key</label>
              <input
                type="text"
                value={boKey}
                onChange={(e) => setBoKey(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 font-mono"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-400 mb-1 block">BO Type</label>
              <select
                value={boType}
                onChange={(e) => setBoType(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100"
              >
                <option value="ENTITY">ENTITY</option>
                <option value="FACT">FACT</option>
                <option value="DIMENSION">DIMENSION</option>
              </select>
            </div>
            <div>
              <label className="text-xs font-semibold text-slate-400 mb-1 block">Level 3 Classification</label>
              <input
                type="text"
                disabled
                value="Sales > Client > Client Entity"
                className="w-full bg-slate-950/60 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-400"
              />
            </div>
          </div>
        </div>

        {/* Multi-Backend Binding Cards */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center gap-2">
              <Layers className="w-4 h-4 text-cyan-400" />
              2. Backend Bindings & Scoped Term Discovery
            </span>
            <button
              onClick={() =>
                setBindings([
                  ...bindings,
                  {
                    backendId: 'starrocks-olap-id',
                    drivingNodeId: '',
                    isDefault: false,
                    tableName: '',
                    relatedTables: [],
                    fields: [],
                  },
                ])
              }
              className="p-2 bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded-lg text-xs text-slate-300 flex items-center gap-1.5 transition"
            >
              <Plus className="w-3.5 h-3.5" /> Add Backend Binding
            </button>
          </div>

          {bindings.map((b, idx) => (
            <div
              key={idx}
              className="p-5 bg-slate-900/40 border border-slate-800 rounded-xl space-y-4"
            >
              <div className="flex items-center justify-between pb-3 border-b border-slate-800">
                <div className="flex items-center gap-3">
                  <span
                    className={`p-1.5 rounded-lg flex items-center gap-1 text-xs font-bold ${
                      b.isDefault
                        ? 'bg-amber-500/20 text-amber-400 border border-amber-500/40'
                        : 'bg-slate-800 text-slate-400'
                    }`}
                  >
                    <Star className="w-3.5 h-3.5" />
                    {b.isDefault ? 'Default Golden Binding' : `Binding ${idx + 1}`}
                  </span>
                  <span className="text-xs font-mono text-slate-300">{b.backendId}</span>
                </div>

                <div className="flex items-center gap-2">
                  <select
                    onChange={(e) => handleSelectDrivingTable(idx, e.target.value)}
                    className="bg-slate-950 border border-slate-800 rounded-lg px-3 py-1.5 text-xs text-slate-100"
                  >
                    <option value="">Select Driving Table...</option>
                    <option value="tbl-customers-node">Customers (Postgres Alpha)</option>
                    <option value="tbl-orders-node">Orders (Postgres Alpha)</option>
                  </select>
                </div>
              </div>

              {b.tableName && (
                <div className="space-y-3">
                  <div className="flex items-center justify-between text-xs text-slate-400">
                    <span>
                      Driving Table: <strong className="text-cyan-400 font-mono">{b.tableName}</strong>
                      {' '}| Related Discovered: <strong className="text-slate-200">{b.relatedTables.join(', ') || 'None'}</strong>
                    </span>
                    <span className="text-emerald-400 font-bold">{b.fields.length} Terms Auto-Mapped</span>
                  </div>

                  {/* Scoped Field Mapping Grid */}
                  <div className="divide-y divide-slate-800/80 border border-slate-800 rounded-lg bg-slate-950/60 overflow-hidden">
                    {b.fields.map((f, fIdx) => (
                      <div
                        key={fIdx}
                        className="p-3 flex items-center justify-between text-xs font-mono"
                      >
                        <div className="flex items-center gap-2">
                          <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                          <span className="text-slate-200 font-semibold">{f.fieldName}</span>
                          <span className="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400">
                            {f.fieldRole}
                          </span>
                        </div>

                        <div className="flex items-center gap-4">
                          <span className="text-slate-400">{b.tableName}.{f.fieldName}</span>
                          <span className="text-[10px] text-cyan-400 font-sans font-bold">
                            {f.bindingRequirement}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
