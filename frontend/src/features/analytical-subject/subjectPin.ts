import type { CubeSubject, QuerySubject } from './types';
import {
  businessObjectSubject,
  cubeSubject,
  isCubeSubject,
} from './types';

export type SubjectPinCheck =
  | { ok: true }
  | { ok: false; reason: string };

/** Fields needed to recover a QuerySubject from a saved-query row. */
export type SavedQuerySubjectSource = {
  subject?: QuerySubject | null;
  sourceKind?: string;
  /** For cube rows, boId carries source_id (= cubeId). */
  boId?: string;
  bindingId?: string;
  relatedBoIds?: string[];
  state?: { subject?: QuerySubject | null };
};

/**
 * Authoring helper (PR1b): derive the mirrored page `subject` from a saved
 * query when binding it onto a tile. Prefers explicit subject, then
 * state.subject, then synthesizes from sourceKind / boId+bindingId.
 */
export function subjectFromSavedQuery(
  sq: SavedQuerySubjectSource | null | undefined,
): QuerySubject | undefined {
  if (!sq) return undefined;
  const explicit = sq.subject ?? sq.state?.subject;
  if (explicit && (explicit.kind === 'cube' || explicit.kind === 'business_object')) {
    return explicit;
  }
  if (sq.sourceKind === 'cube' && sq.boId?.trim()) {
    return cubeSubject(sq.boId, 'latest');
  }
  if (sq.boId?.trim() && sq.bindingId?.trim()) {
    return businessObjectSubject(sq.boId, sq.bindingId, sq.relatedBoIds);
  }
  return undefined;
}

/**
 * Props patch when binding or clearing a saved query on a page tile.
 * Always writes `subject` alongside `savedQueryId` so publish checks stay sync.
 */
export function savedQueryBindProps(
  sq: (SavedQuerySubjectSource & { id?: string }) | null | undefined,
): { savedQueryId: string | undefined; subject: QuerySubject | undefined; savedQueryParams: undefined } {
  if (!sq?.id) {
    return { savedQueryId: undefined, subject: undefined, savedQueryParams: undefined };
  }
  return {
    savedQueryId: sq.id,
    subject: subjectFromSavedQuery(sq),
    savedQueryParams: undefined,
  };
}

/** True when a publishable cube pin has a positive numeric contractVersion. */
export function isNumericCubePin(subject: QuerySubject | null | undefined): subject is CubeSubject & {
  contractVersion: number;
} {
  return (
    isCubeSubject(subject) &&
    typeof subject.contractVersion === 'number' &&
    subject.contractVersion > 0
  );
}

/**
 * Runtime fail-closed compare: mirrored page `subject` vs the loaded saved-query
 * subject (or synthesized from sourceKind + boId/cube id).
 * Non-cube mirrors always pass (BO tiles unchanged).
 */
export function assertCubeSubjectMirror(
  mirrored: QuerySubject | null | undefined,
  saved: {
    subject?: QuerySubject | null;
    sourceKind?: string;
    /** For cube rows, boId carries source_id (= cubeId). */
    boId?: string;
  },
): SubjectPinCheck {
  if (!mirrored || mirrored.kind !== 'cube') {
    return { ok: true };
  }
  if (!mirrored.cubeId?.trim()) {
    return { ok: false, reason: 'page cube subject is missing cubeId' };
  }

  let savedCube: CubeSubject | null = null;
  if (isCubeSubject(saved.subject)) {
    savedCube = saved.subject;
  } else if (saved.sourceKind === 'cube' && saved.boId) {
    savedCube = {
      kind: 'cube',
      cubeId: saved.boId,
      contractVersion: 'latest',
    };
  }

  if (!savedCube) {
    return {
      ok: false,
      reason: 'page pins a cube but the saved query is not cube-backed',
    };
  }
  if (savedCube.cubeId !== mirrored.cubeId) {
    return {
      ok: false,
      reason: `cubeId mismatch: page=${mirrored.cubeId} saved=${savedCube.cubeId}`,
    };
  }
  if (savedCube.contractVersion !== mirrored.contractVersion) {
    return {
      ok: false,
      reason: `contractVersion mismatch: page=${String(mirrored.contractVersion)} saved=${String(savedCube.contractVersion)}`,
    };
  }
  return { ok: true };
}
