export interface RunData {
  run_id: string;
  timestamp: string; // Added timestamp property
  decision_a: string;
  decision_b?: string;
  violations_added: { rule_id: string }[];
  violations_removed: { rule_id: string }[];
}

export interface DiffResult {
  run_id: string;
  type: 'added' | 'removed' | 'changed' | 'unchanged';
  runA: RunData | null;
  runB: RunData | null;
  violationDelta: {
    added: string[];
    removed: string[];
  };
}

export function diffRuns(dataA: RunData[], dataB: RunData[]): DiffResult[] {
  const mapA = new Map(dataA.map((r) => [r.run_id, r]));
  const mapB = new Map(dataB.map((r) => [r.run_id, r]));
  const allRunIds = new Set([...mapA.keys(), ...mapB.keys()]);

  const diffs: DiffResult[] = [];

  for (const id of allRunIds) {
    const runA = mapA.get(id) || null;
    const runB = mapB.get(id) || null;

    if (!runA) {
      diffs.push({ run_id: id, type: 'added', runA, runB, violationDelta: getViolationDelta(runA, runB) });
    } else if (!runB) {
      diffs.push({ run_id: id, type: 'removed', runA, runB, violationDelta: getViolationDelta(runA, runB) });
    } else if (JSON.stringify(runA) !== JSON.stringify(runB)) {
      diffs.push({ run_id: id, type: 'changed', runA, runB, violationDelta: getViolationDelta(runA, runB) });
    }
  }
  return diffs;
}

function getViolationDelta(runA: RunData | null, runB: RunData | null): { added: string[]; removed: string[] } {
  const violationsA = new Set((runA?.violations_added || []).map(v => v.rule_id));
  const violationsB = new Set((runB?.violations_added || []).map(v => v.rule_id));

  const added = [...violationsB].filter(v => !violationsA.has(v));
  const removed = [...violationsA].filter(v => !violationsB.has(v));

  return { added, removed };
}