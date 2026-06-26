import Ajv from 'ajv';
import schema from '../../../schemas/upgrade-artifacts-data.schema.json';
import { UpgradeArtifacts } from '../types/upgrade-generated';

const ajv = new Ajv({ allErrors: true });
const validate = ajv.compile(schema);

export function assertUpgradeArtifacts(data: unknown): UpgradeArtifacts {
  const valid = validate(data);
  if (!valid) {
    const errors = validate.errors?.map(err => `${(err as any).instancePath || (err as any).dataPath || ''}: ${err.message || 'Unknown error'}`).join(', ') || 'Unknown validation error';
    throw new Error(`Invalid upgrade artifact: ${errors}`);
  }
  return data as UpgradeArtifacts;
}

export function isValidUpgradeArtifacts(data: unknown): data is UpgradeArtifacts {
  return validate(data) as boolean;
}

export function getValidationErrors(data: unknown): string[] | null {
  const valid = validate(data);
  if (valid) {
    return null;
  }
  return validate.errors?.map(err => `${(err as any).instancePath || (err as any).dataPath || ''}: ${err.message || 'Unknown error'}`) || [];
}
