import { describe, it, expect } from 'vitest';
import { buildSavePayload } from '../../../components/reporting/builderSerialization';
import { cubeSubject, businessObjectSubject } from '../../../features/analytical-subject';

describe('buildSavePayload subject pin (PR7)', () => {
  it('persists cube subject on metadata and layout without legacy cube name', () => {
    const pin = cubeSubject('cube-1', 3);
    const payload = buildSavePayload(
      { elements: [], reportTitle: 'T' },
      null,
      'rep-1',
      'tenant-1',
      { subject: pin },
    );
    expect((payload.metadata as any).subject).toEqual(pin);
    expect((payload.layout_config as any).subject).toEqual(pin);
    expect((payload.metadata as any).data_bindings).toEqual([{ subject: pin }]);
  });

  it('persists BO subject with bo_path binding', () => {
    const pin = businessObjectSubject('bo-1', 'bind-1');
    const payload = buildSavePayload(
      { elements: [], reportTitle: 'T' },
      { qualifiedPath: 'oms.account/institutional', boId: 'bo-1' },
      undefined,
      'tenant-1',
      { subject: pin },
    );
    expect((payload.metadata as any).subject).toEqual(pin);
    expect((payload.metadata as any).data_bindings[0].bo_path).toBe('oms.account/institutional');
    expect(payload.primary_business_object_id).toBe('bo-1');
  });
});
