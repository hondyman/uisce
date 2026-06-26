export interface User {
  id: string;
  name: string;
  role: 'lp' | 'gp' | 'fof' | 'steward';
  organization: string;
  permissions: string[];
}

import { PrivateMarketsBundle } from '../../types/bundles';
export type Bundle = PrivateMarketsBundle;
