import { useCallback, useEffect, useMemo, useState } from 'react';
import { CircularProgress } from '@mui/material';
import {
  ImpactReport,
  ScriptDetail,
  ScriptSearchFilters,
  ScriptSummary
} from '../../types/scripts';
import { ScriptSearchBar } from './ScriptSearchBar';
import { ScriptList } from './ScriptList';
import { ScriptDetailDrawer } from './ScriptDetailDrawer';
import { StewardshipModal } from './StewardshipModal';
import { ImpactPanel } from './ImpactPanel';
import './scripts.css';
import { devError } from '../../utils/devLogger';

export function ScriptCatalogPage() {
  const [query, setQuery] = useState('');
  const [filters, setFilters] = useState<ScriptSearchFilters>({});
  const [scripts, setScripts] = useState<ScriptSummary[]>([]);
  const [selected, setSelected] = useState<ScriptDetail | null>(null);
  const [impact, setImpact] = useState<ImpactReport | null>(null);
  const [showStewardship, setShowStewardship] = useState(false);
  const [loading, setLoading] = useState(true);

  const params = useMemo(() => {
    const search = new URLSearchParams();
    if (query.trim().length > 0) {
      search.set('query', query.trim());
    }
    if (filters.state) {
      search.set('state', filters.state);
    }
    if (filters.scope) {
      search.set('scope', filters.scope);
    }
    if (filters.tag) {
      search.set('tag', filters.tag);
    }
    if (filters.steward) {
      search.set('steward', filters.steward);
    }
    return search;
  }, [filters.scope, filters.state, filters.steward, filters.tag, query]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch(`/api/scripts?${params.toString()}`);
      if (!response.ok) {
        throw new Error(`Failed to load scripts (${response.status})`);
      }
      const data: ScriptSummary[] = await response.json();
      setScripts(data);
    } catch (error) {
      devError('Failed to fetch scripts', error);
      setScripts([]);
    } finally {
      setLoading(false);
    }
  }, [params]);

  const openDetail = useCallback(async (id: string) => {
    try {
      const response = await fetch(`/api/scripts/${id}`);
      if (!response.ok) {
        throw new Error(`Failed to load script ${id}`);
      }
      const detail: ScriptDetail = await response.json();
      setSelected(detail);
      setImpact(null);
    } catch (error) {
      devError('Failed to load script detail', error);
      setSelected(null);
    }
  }, []);

  const runImpact = useCallback(async () => {
    if (!selected) {
      return;
    }
    try {
      const response = await fetch(`/api/scripts/${selected.id}/impact?version=${selected.latestVersion}`, {
        method: 'POST'
      });
      if (!response.ok) {
        throw new Error('Failed to load impact report');
      }
      const report: ImpactReport = await response.json();
      setImpact(report);
    } catch (error) {
      devError('Failed to load impact report', error);
      setImpact(null);
    }
  }, [selected]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="page">
      <h1>Script Management</h1>
      <ScriptSearchBar query={query} setQuery={setQuery} filters={filters} setFilters={setFilters} />
      {loading ? <CircularProgress /> : <ScriptList scripts={scripts} onOpen={openDetail} />}
      {selected && (
        <ScriptDetailDrawer
          script={selected}
          onClose={() => setSelected(null)}
          onImpact={runImpact}
          onAssignSteward={() => setShowStewardship(true)}
          onRefresh={openDetail}
        />
      )}
      {impact && <ImpactPanel report={impact} onClose={() => setImpact(null)} />}
      {showStewardship && selected && (
        <StewardshipModal
          scriptId={selected.id}
          currentSteward={selected.steward}
          onClose={() => setShowStewardship(false)}
          onSaved={() => { setShowStewardship(false); openDetail(selected.id); }}
        />
      )}
    </div>
  );
}

export default ScriptCatalogPage;
