// libs/jwt-middleware-node.ts
// JWT Middleware for Node.js/Express services

import jwt from 'jsonwebtoken';
import { Request, Response, NextFunction } from 'express';

export interface JWTClaims {
  user_id: string;
  email: string;
  tenant_id: string;
  tenant_ids: string[];
  roles: string[];
  is_active: boolean;
  is_core_admin: boolean;
  org_id?: string;
  iat: number;
  exp: number;
}

// Extend Express Request to include JWT claims
declare global {
  namespace Express {
    interface Request {
      jwtClaims?: JWTClaims;
    }
  }
}

const JWT_SECRET = process.env.JWT_SECRET || 'dev-jwt-secret-key-change-in-production';

/**
 * Extract JWT token from Authorization header
 */
export function extractToken(req: Request): string | null {
  const authHeader = req.headers.authorization;
  if (!authHeader) return null;

  const parts = authHeader.split(' ');
  if (parts.length !== 2 || parts[0].toLowerCase() !== 'bearer') return null;

  return parts[1];
}

/**
 * Validate JWT token and return claims
 */
export function validateToken(token: string): JWTClaims | null {
  try {
    const decoded = jwt.verify(token, JWT_SECRET) as JWTClaims;
    return decoded;
  } catch (error) {
    console.error('JWT validation error:', error);
    return null;
  }
}

/**
 * Validate tenant access - ensure user has access to requested tenant
 */
export function validateTenantAccess(claims: JWTClaims, tenantId: string): boolean {
  if (claims.is_core_admin) return true;
  if (claims.tenant_id === tenantId) return true;
  if (claims.tenant_ids?.includes(tenantId)) return true;
  return false;
}

/**
 * Check if user has required role
 */
export function hasRole(claims: JWTClaims, role: string): boolean {
  return claims.roles?.includes(role) || false;
}

/**
 * JWT Middleware Factory
 * Creates middleware that validates JWT on all routes except public paths
 */
export function jwtMiddleware(publicPaths: string[] = ['/health', '/ready', '/docs']) {
  return (req: Request, res: Response, next: NextFunction) => {
    // Skip JWT validation for public paths
    if (publicPaths.some(path => req.path.startsWith(path))) {
      return next();
    }

    const token = extractToken(req);
    if (!token) {
      return res.status(401).json({
        error: 'Unauthorized',
        message: 'Missing or invalid Authorization header'
      });
    }

    const claims = validateToken(token);
    if (!claims) {
      return res.status(401).json({
        error: 'Unauthorized',
        message: 'Invalid or expired token'
      });
    }

    // Attach claims to request for use in handlers
    req.jwtClaims = claims;
    next();
  };
}

/**
 * Optional JWT Middleware - doesn't fail if token missing, but validates if present
 */
export function optionalJwtMiddleware(publicPaths: string[] = []) {
  return (req: Request, res: Response, next: NextFunction) => {
    if (publicPaths.some(path => req.path.startsWith(path))) {
      return next();
    }

    const token = extractToken(req);
    if (token) {
      const claims = validateToken(token);
      if (claims) {
        req.jwtClaims = claims;
      }
    }

    next();
  };
}

/**
 * Require specific tenant access
 */
export function requireTenant(tenantIdSource: 'header' | 'param' | 'query' = 'header') {
  return (req: Request, res: Response, next: NextFunction) => {
    if (!req.jwtClaims) {
      return res.status(401).json({
        error: 'Unauthorized',
        message: 'JWT claims not found'
      });
    }

    let tenantId: string | undefined;

    if (tenantIdSource === 'header') {
      tenantId = req.headers['x-tenant-id'] as string;
    } else if (tenantIdSource === 'param') {
      tenantId = req.params.tenantId;
    } else if (tenantIdSource === 'query') {
      tenantId = req.query.tenantId as string;
    }

    if (!tenantId) {
      return res.status(400).json({
        error: 'Bad Request',
        message: 'Tenant ID required'
      });
    }

    if (!validateTenantAccess(req.jwtClaims, tenantId)) {
      return res.status(403).json({
        error: 'Forbidden',
        message: 'Access denied to this tenant'
      });
    }

    next();
  };
}

/**
 * Require specific role
 */
export function requireRole(...allowedRoles: string[]) {
  return (req: Request, res: Response, next: NextFunction) => {
    if (!req.jwtClaims) {
      return res.status(401).json({
        error: 'Unauthorized',
        message: 'JWT claims not found'
      });
    }

    const hasRequiredRole = allowedRoles.some(role => hasRole(req.jwtClaims!, role));

    if (!hasRequiredRole) {
      return res.status(403).json({
        error: 'Forbidden',
        message: `Requires role: ${allowedRoles.join(' or ')}`
      });
    }

    next();
  };
}

/**
 * Get JWT claims from request
 */
export function getClaims(req: Request): JWTClaims | null {
  return req.jwtClaims || null;
}

/**
 * Middleware to inject tenant ID from claims into request if not provided
 */
export function injectTenantFromClaims() {
  return (req: Request, res: Response, next: NextFunction) => {
    if (req.jwtClaims && !req.headers['x-tenant-id']) {
      req.headers['x-tenant-id'] = req.jwtClaims.tenant_id;
    }
    next();
  };
}
