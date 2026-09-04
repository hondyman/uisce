#!/usr/bin/env node
/**
 * E2E JWT generator for Playwright a11y baseline.
 * Reads JWT_SECRET from backend/.env and emits a signed JWT to stdout.
 * Tokens are valid for 24h. Run this before each test session to refresh.
 *
 * Usage:
 *   node scripts/generate-e2e-jwt.mjs
 *   node scripts/generate-e2e-jwt.mjs a11y-fixture admin,user 00000000-0000-0000-0000-000000000000
 */
import { createHmac } from 'crypto';
import { readFileSync, existsSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const USER_ID = process.argv[2] || 'a11y-fixture';
const ROLES = (process.argv[3] || 'admin,user').split(',');
const TENANT_ID = process.argv[4] || '00000000-0000-0000-0000-000000000000';

function getJWTSecret() {
  const envPath = resolve(__dirname, '../backend/.env');
  if (!existsSync(envPath)) {
    throw new Error(`backend/.env not found at ${envPath}`);
  }
  const content = readFileSync(envPath, 'utf8');
  const match = content.match(/^JWT_SECRET=(.+)$/m);
  if (!match) {
    throw new Error('JWT_SECRET not found in backend/.env');
  }
  return match[1].trim();
}

function generateJWT(secret, userId, roles, tenantId) {
  const now = Math.floor(Date.now() / 1000);
  const exp = now + 86400;
  const payload = {
    sub: userId,
    user_id: userId,
    tenant_id: tenantId,
    roles,
    tenant_ids: [tenantId],
    is_active: true,
    iss: 'dev://local',
    exp,
    iat: now,
  };
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const payloadB64 = Buffer.from(JSON.stringify(payload)).toString('base64url');
  const sig = createHmac('sha256', secret)
    .update(`${header}.${payloadB64}`)
    .digest('base64url');
  return `${header}.${payloadB64}.${sig}`;
}

const secret = getJWTSecret();
const token = generateJWT(secret, USER_ID, ROLES, TENANT_ID);
console.log(token);
