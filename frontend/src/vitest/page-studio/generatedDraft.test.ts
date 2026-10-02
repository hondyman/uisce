import { beforeEach, describe, expect, it } from 'vitest';
import { GENERATED_DRAFT_TTL_MS, handOverGeneratedDraft, takeGeneratedDraft } from '../../pages/page-studio/app/generatedDraft';

beforeEach(() => sessionStorage.clear());

describe('generated draft hand-over', () => {
  it('survives being read more than once (the editor mounts more than once on the way)', () => {
    handOverGeneratedDraft({ name: 'X' }, 1000);
    expect(takeGeneratedDraft(2000)?.name).toBe('X');
    expect(takeGeneratedDraft(3000)?.name).toBe('X');
  });
  it('goes stale, so an old draft never reappears', () => {
    handOverGeneratedDraft({ name: 'X' }, 1000);
    expect(takeGeneratedDraft(1000 + GENERATED_DRAFT_TTL_MS + 1)).toBeUndefined();
    expect(takeGeneratedDraft(1001)).toBeUndefined(); // and was discarded
  });
  it('is empty when nothing was handed over or storage holds junk', () => {
    expect(takeGeneratedDraft()).toBeUndefined();
    sessionStorage.setItem('page-studio.generated-draft', 'not json');
    expect(takeGeneratedDraft()).toBeUndefined();
  });
});
