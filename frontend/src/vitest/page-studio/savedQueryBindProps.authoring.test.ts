import { describe, it, expect } from 'vitest';
import { savedQueryBindProps, subjectFromSavedQuery } from '../../features/analytical-subject';
import { cubeSubject } from '../../features/analytical-subject/types';

/** Authoring contract: PropertiesPanel / QueryBuilderModal must use these patches. */
describe('PR1b authoring bind contract', () => {
  const cubeQuery = {
    id: 'sq-cube-1',
    sourceKind: 'cube' as const,
    boId: 'cube-aaa',
    bindingId: '',
    subject: cubeSubject('cube-aaa', 3),
    name: 'Cube KPI',
  };

  it('cube bind writes savedQueryId + mirrored numeric subject', () => {
    const patch = savedQueryBindProps(cubeQuery);
    expect(patch.savedQueryId).toBe('sq-cube-1');
    expect(patch.subject).toEqual({ kind: 'cube', cubeId: 'cube-aaa', contractVersion: 3 });
    expect(patch.savedQueryParams).toBeUndefined();
  });

  it('cube bind with sourceKind only synthesizes latest pin (draft-legal)', () => {
    const patch = savedQueryBindProps({
      id: 'sq-2',
      sourceKind: 'cube',
      boId: 'cube-bbb',
    });
    expect(patch.subject).toEqual(cubeSubject('cube-bbb', 'latest'));
  });

  it('unbind clears both id and subject so checker no longer sees a cube pin', () => {
    const patch = savedQueryBindProps(null);
    expect(patch).toEqual({
      savedQueryId: undefined,
      subject: undefined,
      savedQueryParams: undefined,
    });
  });

  it('subjectFromSavedQuery prefers row.subject over state', () => {
    expect(
      subjectFromSavedQuery({
        subject: cubeSubject('a', 1),
        state: { subject: cubeSubject('b', 2) },
      }),
    ).toEqual(cubeSubject('a', 1));
  });
});
