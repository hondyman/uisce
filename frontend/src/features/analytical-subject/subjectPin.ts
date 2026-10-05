import type { CubeSubject, QuerySubject } from './types';
import { isCubeSubject } from './types';

export type SubjectPinCheck =
  | { ok: true }
  | { ok: false; reason: string };

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
