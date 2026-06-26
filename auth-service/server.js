import express from 'express';
import cors from 'cors';
import jwt from 'jsonwebtoken';
import bcrypt from 'bcryptjs';
import pkg from 'pg';
import dotenv from 'dotenv';
import crypto from 'crypto';

dotenv.config();

const { Pool } = pkg;

const app = express();
const port = process.env.AUTH_SERVICE_PORT || 8001;
const jwtSecret = process.env.JWT_SECRET || 'dev-jwt-secret-key-change-in-production';
const jwtExpiry = process.env.JWT_EXPIRY || '1h';
const refreshTokenExpiry = process.env.REFRESH_TOKEN_EXPIRY || '24h';

// Database connection pool
const pool = new Pool({
  user: process.env.POSTGRES_USER || 'postgres',
  password: process.env.POSTGRES_PASSWORD || 'postgres',
  host: process.env.POSTGRES_HOST || 'localhost',
  port: parseInt(process.env.POSTGRES_PORT) || 5432,
  database: process.env.POSTGRES_DB || 'alpha',
});

// Test database connection on startup
pool.query('SELECT NOW()', (err, res) => {
  if (err) {
    console.error('❌ Database connection failed:', err.message);
    console.error('   Make sure PostgreSQL is running and accessible');
  } else {
    console.log('✅ Database connected successfully at', res.rows[0].now);
  }
});

// Middleware
app.use(cors({
  origin: process.env.ALLOWED_ORIGINS?.split(',') || ['http://localhost:5173', 'http://localhost:5174'],
  credentials: true
}));
app.use(express.json());

// Debug middleware to check body parsing
app.use((req, res, next) => {
  if (req.path.includes('/auth')) {
    console.log(`[DEBUG] ${req.method} ${req.path}`);
    console.log('[DEBUG] Headers:', JSON.stringify(req.headers));
    console.log('[DEBUG] Body:', JSON.stringify(req.body));
  }
  next();
});

// Logging middleware
app.use((req, res, next) => {
  const start = Date.now();
  res.on('finish', () => {
    const duration = Date.now() - start;
    console.log(`${req.method} ${req.path} ${res.statusCode} ${duration}ms`);
  });
  next();
});

// Helper function to parse JWT expiry to milliseconds
function expiryToMs(expiry) {
  const match = expiry.match(/^(\d+)([smhd])$/);
  if (!match) return 3600000; // default 1 hour
  const value = parseInt(match[1]);
  const unit = match[2];
  switch (unit) {
    case 's': return value * 1000;
    case 'm': return value * 60 * 1000;
    case 'h': return value * 60 * 60 * 1000;
    case 'd': return value * 24 * 60 * 60 * 1000;
    default: return 3600000;
  }
}

// Helper function to log auth events
async function logAuthEvent(userId, eventType, success, ipAddress, userAgent, errorMessage = null, metadata = {}) {
  try {
    await pool.query(
      `INSERT INTO auth_audit_log (user_id, event_type, success, ip_address, user_agent, error_message, metadata)
       VALUES ($1, $2, $3, $4, $5, $6, $7)`,
      [userId, eventType, success, ipAddress, userAgent, errorMessage, JSON.stringify(metadata)]
    );
  } catch (error) {
    console.error('Failed to log auth event:', error.message);
  }
}

// Health check
app.get('/health', async (req, res) => {
  try {
    await pool.query('SELECT 1');
    res.json({ status: 'ok', service: 'auth-service', database: 'connected' });
  } catch (error) {
    res.status(503).json({ status: 'error', service: 'auth-service', database: 'disconnected' });
  }
});

// Register endpoint
app.post('/api/auth/register', async (req, res) => {
  const client = await pool.connect();
  try {
    const { email, password, name, organization, tenant_id } = req.body;

    // Validation
    if (!email || !password) {
      return res.status(400).json({ error: 'Email and password are required' });
    }

    if (password.length < 8) {
      return res.status(400).json({ error: 'Password must be at least 8 characters' });
    }

    // Email format validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
      return res.status(400).json({ error: 'Invalid email format' });
    }

    // Check if user already exists
    const existingUser = await client.query(
      'SELECT id FROM users WHERE email = $1',
      [email.toLowerCase()]
    );

    if (existingUser.rows.length > 0) {
      await logAuthEvent(null, 'register', false, req.ip, req.get('user-agent'), 'Email already exists');
      return res.status(409).json({ error: 'Email already exists' });
    }

    // Hash password
    const saltRounds = 10;
    const passwordHash = await bcrypt.hash(password, saltRounds);

    // Get default tenant if one isn't specified
    let assignedTenantId = tenant_id;
    // Automatic assignment removed for security reasons.
    // Users without a tenant_id will have null tenant_id and must be assigned manually or have global access.

    // Insert new user with tenant assignment
    const result = await client.query(
      `INSERT INTO users (email, password_hash, name, organization, role, permissions, is_active, email_verified, tenant_scope, username, tenant_id)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
       RETURNING id, email, name, role, organization, permissions, is_active, is_core_admin, tenant_scope, tenant_id, org_id, created_at`,
      [email.toLowerCase(), passwordHash, name || 'User', organization || 'Default Org', 'user', JSON.stringify(['read']), true, false, 'single', email.split('@')[0], assignedTenantId]
    );

    const newUser = result.rows[0];

    // Generate tokens with multi-tenant claims
    const jti = crypto.randomUUID();
    
    // Build tenant claims based on tenant_scope
    const tenantClaims = {};
    if (newUser.tenant_scope === 'single' && newUser.tenant_id) {
      tenantClaims.tenant_id = newUser.tenant_id;
    } else if (newUser.tenant_scope === 'multi') {
      // Fetch tenant assignments
      const assignments = await client.query(
        'SELECT tenant_id FROM tenant_assignments WHERE user_id = $1',
        [newUser.id]
      );
      tenantClaims.tenant_ids = assignments.rows.map(r => r.tenant_id);
    }
    if (newUser.org_id) {
      tenantClaims.org_id = newUser.org_id;
    }

    // Build Hasura-specific claims
    const allowedRoles = ['user'];
    let defaultRole = 'user';
    const hasuraClaims = {
      'x-hasura-allowed-roles': allowedRoles,
      'x-hasura-default-role': defaultRole,
      'x-hasura-user-id': newUser.id,
    };
    if (newUser.tenant_id) {
      hasuraClaims['x-hasura-tenant-id'] = newUser.tenant_id;
    }
    
    const accessToken = jwt.sign(
      {
        sub: newUser.id,
        user_id: newUser.id,
        email: newUser.email,
        role: newUser.role,
        roles: [newUser.role],
        scopes: Array.isArray(newUser.permissions) ? newUser.permissions : JSON.parse(newUser.permissions || '[]'),
        tenant_scope: newUser.tenant_scope,
        ...tenantClaims,
        jti: jti,
        'https://hasura.io/jwt/claims': hasuraClaims
      },
      jwtSecret,
      { expiresIn: jwtExpiry }
    );

    const refreshToken = crypto.randomBytes(32).toString('hex');
    const refreshExpiresAt = new Date(Date.now() + expiryToMs(refreshTokenExpiry));

    // Store refresh token
    await client.query(
      `INSERT INTO refresh_tokens (user_id, token, expires_at, ip_address, user_agent)
       VALUES ($1, $2, $3, $4, $5)`,
      [newUser.id, refreshToken, refreshExpiresAt, req.ip, req.get('user-agent')]
    );

    // Log successful registration
    await logAuthEvent(newUser.id, 'register', true, req.ip, req.get('user-agent'));

    res.status(201).json({
      user: {
        id: newUser.id,
        email: newUser.email,
        name: newUser.name,
        role: newUser.role,
        organization: newUser.organization,
        permissions: Array.isArray(newUser.permissions) ? newUser.permissions : JSON.parse(newUser.permissions || '[]'),
        is_active: newUser.is_active,
        is_core_admin: newUser.is_core_admin,
        tenant_scope: newUser.tenant_scope,
        tenant_id: newUser.tenant_id,
        org_id: newUser.org_id
      },
      access_token: accessToken,
      refresh_token: refreshToken,
      token_type: 'Bearer',
      expires_in: Math.floor(expiryToMs(jwtExpiry) / 1000)
    });
  } catch (error) {
    console.error('Register error:', error);
    await logAuthEvent(null, 'register', false, req.ip, req.get('user-agent'), error.message);
    res.status(500).json({ error: 'Internal server error' });
  } finally {
    client.release();
  }
});

// Login endpoint
app.post('/api/auth/login', async (req, res) => {
  const client = await pool.connect();
  try {
    const { email, password } = req.body;

    if (!email || !password) {
      return res.status(400).json({ error: 'Email and password are required' });
    }

    // Fetch user from database with tenant fields
    const result = await client.query(
      `SELECT id, email, password_hash, name, role, organization, permissions, is_active, is_core_admin, tenant_scope, tenant_id, org_id
       FROM users WHERE email = $1`,
      [email.toLowerCase()]
    );

    if (result.rows.length === 0) {
      await logAuthEvent(null, 'login', false, req.ip, req.get('user-agent'), 'User not found');
      return res.status(401).json({ error: 'Invalid credentials' });
    }

    const user = result.rows[0];

    // Check if user is active
    if (!user.is_active) {
      await logAuthEvent(user.id, 'login', false, req.ip, req.get('user-agent'), 'Account disabled');
      return res.status(403).json({ error: 'Account is disabled' });
    }

    // Verify password
    const passwordValid = await bcrypt.compare(password, user.password_hash);
    if (!passwordValid) {
      await logAuthEvent(user.id, 'login', false, req.ip, req.get('user-agent'), 'Invalid password');
      return res.status(401).json({ error: 'Invalid credentials' });
    }

    // Update last login timestamp
    await client.query('UPDATE users SET last_login_at = NOW() WHERE id = $1', [user.id]);

    // Generate tokens with multi-tenant claims
    const jti = crypto.randomUUID();
    
    // Build tenant claims based on tenant_scope
    const tenantClaims = {};
    
    if (user.tenant_scope === 'single') {
      // For single-scope users, try user.tenant_id first, then check assignments
      if (user.tenant_id) {
        tenantClaims.tenant_id = user.tenant_id;
      } else {
        // Fall back to checking assignments table
        const assignments = await client.query(
          'SELECT tenant_id FROM tenant_assignments WHERE user_id = $1 LIMIT 1',
          [user.id]
        );
        if (assignments.rows.length > 0) {
          tenantClaims.tenant_id = assignments.rows[0].tenant_id;
        }
      }
    } else if (user.tenant_scope === 'multi') {
      // Fetch tenant assignments
      const assignments = await client.query(
        'SELECT tenant_id FROM tenant_assignments WHERE user_id = $1',
        [user.id]
      );
      tenantClaims.tenant_ids = assignments.rows.map(r => r.tenant_id);
    }
    
    if (user.org_id) {
      tenantClaims.org_id = user.org_id;
    }

    // Build Hasura-specific claims (required for Hasura GraphQL authorization)
    const allowedRoles = ['user'];
    let defaultRole = 'user';

    // Global admins (uisce organization) get global_admin role
    if (user.organization === 'uisce' && user.is_core_admin) {
      allowedRoles.push('global_admin');
      defaultRole = 'global_admin';
    }

    // Add user's role if it's not already in the list
    if (user.role && user.role !== 'user' && user.role !== 'global_admin') {
      allowedRoles.push(user.role);
    }

    const hasuraClaims = {
      'x-hasura-allowed-roles': allowedRoles,
      'x-hasura-default-role': defaultRole,
      'x-hasura-user-id': user.id,
    };

    // Add tenant_id to Hasura claims for RLS filtering (use from tenantClaims if available)
    if (tenantClaims.tenant_id) {
      hasuraClaims['x-hasura-tenant-id'] = tenantClaims.tenant_id;
    } else if (user.tenant_id) {
      hasuraClaims['x-hasura-tenant-id'] = user.tenant_id;
    }
    
    const accessToken = jwt.sign(
      {
        sub: user.id,
        user_id: user.id,
        email: user.email,
        role: user.role,
        roles: [user.role],
        scopes: Array.isArray(user.permissions) ? user.permissions : JSON.parse(user.permissions || '[]'),
        tenant_scope: user.tenant_scope,
        ...tenantClaims,
        jti: jti,
        'https://hasura.io/jwt/claims': hasuraClaims
      },
      jwtSecret,
      { expiresIn: jwtExpiry }
    );

    const refreshToken = crypto.randomBytes(32).toString('hex');
    const refreshExpiresAt = new Date(Date.now() + expiryToMs(refreshTokenExpiry));

    // Store refresh token
    await client.query(
      `INSERT INTO refresh_tokens (user_id, token, expires_at, ip_address, user_agent)
       VALUES ($1, $2, $3, $4, $5)`,
      [user.id, refreshToken, refreshExpiresAt, req.ip, req.get('user-agent')]
    );

    // Log successful login
    await logAuthEvent(user.id, 'login', true, req.ip, req.get('user-agent'));

    res.json({
      user: {
        id: user.id,
        email: user.email,
        name: user.name,
        role: user.role,
        organization: user.organization,
        permissions: Array.isArray(user.permissions) ? user.permissions : JSON.parse(user.permissions || '[]'),
        is_active: user.is_active,
        is_core_admin: user.is_core_admin,
        tenant_scope: user.tenant_scope,
        tenant_id: user.tenant_id,
        org_id: user.org_id
      },
      access_token: accessToken,
      refresh_token: refreshToken,
      token_type: 'Bearer',
      expires_in: Math.floor(expiryToMs(jwtExpiry) / 1000)
    });
  } catch (error) {
    console.error('Login error:', error);
    await logAuthEvent(null, 'login', false, req.ip, req.get('user-agent'), error.message);
    res.status(500).json({ error: 'Internal server error' });
  } finally {
    client.release();
  }
});

// Refresh token endpoint
app.post('/api/auth/refresh', async (req, res) => {
  const client = await pool.connect();
  try {
    const { refresh_token } = req.body;

    if (!refresh_token) {
      return res.status(400).json({ error: 'Refresh token is required' });
    }

    // Fetch refresh token from database
    const result = await client.query(
      `SELECT rt.id, rt.user_id, rt.expires_at, rt.revoked,
              u.id, u.email, u.role, u.is_active, u.tenant_id, u.organization, u.permissions, u.is_core_admin, u.tenant_scope, u.org_id
       FROM refresh_tokens rt
       JOIN users u ON rt.user_id = u.id
       WHERE rt.token = $1`,
      [refresh_token]
    );

    if (result.rows.length === 0) {
      await logAuthEvent(null, 'token_refresh', false, req.ip, req.get('user-agent'), 'Invalid refresh token');
      return res.status(401).json({ error: 'Invalid refresh token' });
    }

    const tokenData = result.rows[0];

    // Check if token is revoked
    if (tokenData.revoked) {
      await logAuthEvent(tokenData.user_id, 'token_refresh', false, req.ip, req.get('user-agent'), 'Token revoked');
      return res.status(401).json({ error: 'Refresh token has been revoked' });
    }

    // Check if token is expired
    if (new Date(tokenData.expires_at) < new Date()) {
      await logAuthEvent(tokenData.user_id, 'token_refresh', false, req.ip, req.get('user-agent'), 'Token expired');
      return res.status(401).json({ error: 'Refresh token has expired' });
    }

    // Check if user is still active
    if (!tokenData.is_active) {
      await logAuthEvent(tokenData.user_id, 'token_refresh', false, req.ip, req.get('user-agent'), 'Account disabled');
      return res.status(403).json({ error: 'Account is disabled' });
    }

    // Build tenant claims
    const tenantClaims = {};
    if (tokenData.tenant_scope === 'single') {
      if (tokenData.tenant_id) {
        tenantClaims.tenant_id = tokenData.tenant_id;
      } else {
        // Fall back to checking assignments table
        const assignments = await client.query(
          'SELECT tenant_id FROM tenant_assignments WHERE user_id = $1 LIMIT 1',
          [tokenData.user_id]
        );
        if (assignments.rows.length > 0) {
          tenantClaims.tenant_id = assignments.rows[0].tenant_id;
        }
      }
    } else if (tokenData.tenant_scope === 'multi') {
      // Fetch tenant assignments
      const assignments = await client.query(
        'SELECT tenant_id FROM tenant_assignments WHERE user_id = $1',
        [tokenData.user_id]
      );
      tenantClaims.tenant_ids = assignments.rows.map(r => r.tenant_id);
    }

    // Build Hasura-specific claims (required for Hasura GraphQL authorization)
    const allowedRoles = ['user'];
    let defaultRole = 'user';

    // Global admins (uisce organization) get global_admin role
    if (tokenData.organization === 'uisce' && tokenData.is_core_admin) {
      allowedRoles.push('global_admin');
      defaultRole = 'global_admin';
    }

    // Add user's role if it's not already in the list
    if (tokenData.role && tokenData.role !== 'user' && tokenData.role !== 'global_admin') {
      allowedRoles.push(tokenData.role);
    }

    const hasuraClaims = {
      'x-hasura-allowed-roles': allowedRoles,
      'x-hasura-default-role': defaultRole,
      'x-hasura-user-id': tokenData.id,
    };

    // Add tenant_id to Hasura claims for RLS filtering
    if (tenantClaims.tenant_id) {
      hasuraClaims['x-hasura-tenant-id'] = tenantClaims.tenant_id;
    }

    // Generate new access token
    const jti = crypto.randomUUID();
    const accessToken = jwt.sign(
      {
        sub: tokenData.user_id,
        user_id: tokenData.user_id,
        email: tokenData.email,
        role: tokenData.role,
        roles: [tokenData.role],
        tenant_scope: tokenData.tenant_scope,
        ...tenantClaims,
        jti: jti,
        'https://hasura.io/jwt/claims': hasuraClaims
      },
      jwtSecret,
      { expiresIn: jwtExpiry }
    );

    // Optional: Rotate refresh token (uncomment for enhanced security)
    // const newRefreshToken = crypto.randomBytes(32).toString('hex');
    // await client.query('UPDATE refresh_tokens SET revoked = true WHERE id = $1', [tokenData.id]);
    // await client.query(
    //   'INSERT INTO refresh_tokens (user_id, token, expires_at, ip_address, user_agent) VALUES ($1, $2, $3, $4, $5)',
    //   [tokenData.user_id, newRefreshToken, new Date(Date.now() + expiryToMs(refreshTokenExpiry)), req.ip, req.get('user-agent')]
    // );

    await logAuthEvent(tokenData.user_id, 'token_refresh', true, req.ip, req.get('user-agent'));

    res.json({
      access_token: accessToken,
      // refresh_token: newRefreshToken, // Uncomment if rotating
      token_type: 'Bearer',
      expires_in: Math.floor(expiryToMs(jwtExpiry) / 1000)
    });
  } catch (error) {
    console.error('Refresh error:', error);
    await logAuthEvent(null, 'token_refresh', false, req.ip, req.get('user-agent'), error.message);
    res.status(500).json({ error: 'Internal server error' });
  } finally {
    client.release();
  }
});

// Logout endpoint
app.post('/api/auth/logout', async (req, res) => {
  const client = await pool.connect();
  try {
    const token = req.headers.authorization?.replace('Bearer ', '');
    const { refresh_token } = req.body;

    if (!token) {
      return res.status(401).json({ error: 'No token provided' });
    }

    // Decode token to get jti and user_id (without verification)
    const decoded = jwt.decode(token);
    
    if (decoded && decoded.jti) {
      // Add access token to revoked list
      const expiresAt = new Date(decoded.exp * 1000);
      await client.query(
        `INSERT INTO revoked_tokens (jti, user_id, expires_at, reason)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (jti) DO NOTHING`,
        [decoded.jti, decoded.user_id || decoded.sub, expiresAt, 'logout']
      );
    }

    // Revoke refresh token if provided
    if (refresh_token) {
      await client.query(
        'UPDATE refresh_tokens SET revoked = true, revoked_at = NOW() WHERE token = $1',
        [refresh_token]
      );
    }

    await logAuthEvent(decoded?.user_id || decoded?.sub, 'logout', true, req.ip, req.get('user-agent'));

    res.json({ message: 'Logged out successfully' });
  } catch (error) {
    console.error('Logout error:', error);
    res.status(500).json({ error: 'Internal server error' });
  } finally {
    client.release();
  }
});

// Verify token endpoint
app.post('/api/auth/verify', async (req, res) => {
  const client = await pool.connect();
  try {
    const token = req.headers.authorization?.replace('Bearer ', '');

    if (!token) {
      return res.status(401).json({ error: 'No token provided' });
    }

    // Verify JWT
    const decoded = jwt.verify(token, jwtSecret);

    // Check if token is revoked
    if (decoded.jti) {
      const result = await client.query(
        'SELECT 1 FROM revoked_tokens WHERE jti = $1',
        [decoded.jti]
      );
      
      if (result.rows.length > 0) {
        return res.status(401).json({ error: 'Token has been revoked' });
      }
    }

    res.json({ valid: true, decoded });
  } catch (error) {
    if (error.name === 'TokenExpiredError') {
      res.status(401).json({ error: 'Token has expired' });
    } else if (error.name === 'JsonWebTokenError') {
      res.status(401).json({ error: 'Invalid token' });
    } else {
      res.status(500).json({ error: 'Internal server error' });
    }
  } finally {
    client.release();
  }
});

// Get current user info
app.get('/api/auth/me', async (req, res) => {
  const client = await pool.connect();
  try {
    const token = req.headers.authorization?.replace('Bearer ', '');

    if (!token) {
      return res.status(401).json({ error: 'No token provided' });
    }

    const decoded = jwt.verify(token, jwtSecret);

    // Fetch current user data
    const result = await client.query(
      `SELECT id, email, name, role, organization, permissions, is_active, is_core_admin, created_at, last_login_at
       FROM users WHERE id = $1`,
      [decoded.user_id || decoded.sub]
    );

    if (result.rows.length === 0) {
      return res.status(404).json({ error: 'User not found' });
    }

    res.json({ user: result.rows[0] });
  } catch (error) {
    res.status(401).json({ error: 'Invalid token' });
  } finally {
    client.release();
  }
});

// Cleanup expired tokens periodically (run every hour)
setInterval(async () => {
  try {
    await pool.query('SELECT cleanup_expired_tokens()');
    console.log('✅ Cleaned up expired tokens');
  } catch (error) {
    console.error('❌ Failed to cleanup expired tokens:', error.message);
  }
}, 60 * 60 * 1000); // 1 hour

// Start server
app.listen(port, '0.0.0.0', () => {
  console.log('');
  console.log('🚀 Auth Service Started');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log(`📍 URL: http://localhost:${port}`);
  console.log(`🔐 JWT Expiry: ${jwtExpiry}`);
  console.log(`🔄 Refresh Token Expiry: ${refreshTokenExpiry}`);
  console.log(`🗄️  Database: ${process.env.POSTGRES_HOST}:${process.env.POSTGRES_PORT}/${process.env.POSTGRES_DB}`);
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log('');
});
