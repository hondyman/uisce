export interface DriftLogEntry {
  id: string;
  severity: 'breaking' | 'medium' | 'low';
  qualified_path: string;
  explanation: string;
}

export interface DriftReportForCompare {
  id: string;
  schema_hash: string;
  drift_log_entries: DriftLogEntry[];
}

export type ChangedEntry = {
  before: DriftLogEntry;
  after: DriftLogEntry;
  severityChanged: boolean;
  explanationChanged: boolean;
};

export type DiffResult = {
  added: DriftLogEntry[];
  removed: DriftLogEntry[];
  changed: ChangedEntry[];
};

export function diffReports(a: DriftReportForCompare, b: DriftReportForCompare): DiffResult {
  const mapA = new Map(a.drift_log_entries.map(e => [e.qualified_path, e]));
  const mapB = new Map(b.drift_log_entries.map(e => [e.qualified_path, e]));

  const added: DriftLogEntry[] = [];
  const removed: DriftLogEntry[] = [];
  const changed: ChangedEntry[] = [];

  for (const [path, entryA] of mapA.entries()) {
    const entryB = mapB.get(path);
    if (!entryB) {
      removed.push(entryA);
    } else {
      const sevChanged = entryA.severity !== entryB.severity;
      const expChanged = entryA.explanation !== entryB.explanation;
      if (sevChanged || expChanged) {
        changed.push({
          before: entryA,
          after: entryB,
          severityChanged: sevChanged,
          explanationChanged: expChanged,
        });
      }
    }
  }

  for (const [path, entryB] of mapB.entries()) {
    if (!mapA.has(path)) {
      added.push(entryB);
    }
  }

  return { added, removed, changed };
}

export function groupBySeverity(entries: DriftLogEntry[]): Record<string, DriftLogEntry[]> {
  return entries.reduce<Record<string, DriftLogEntry[]>>((acc, e) => {
    if (!acc[e.severity]) {
      acc[e.severity] = [];
    }
    acc[e.severity].push(e);
    return acc;
  }, {});
}