import { getPool } from '../services/database.js';
import { getRedisClient } from '../services/redis.js';
import { logger } from '../utils/logger.js';
import { getEnv } from '../../internal/pkg/env/getEnv.js';
import {
  Entity,
  Client,
  Account,
  PersonalAccount,
  IRAAccount,
  TrustAccount,
  AccountStatus,
  RiskTolerance,
  InvestmentObjective,
  IRAType,
  TrustType,
  KycStatus,
  SanctionsStatus,
  PepStatus
} from '../entities/index.js';

/**
 * Entity Manager - Central hub for entity operations
 * Handles loading, caching, and CRUD operations
 */
export class EntityManager {
  private static instance: EntityManager;
  private cache: Map<string, Entity> = new Map();

  private constructor() {}

  static getInstance(): EntityManager {
    if (!EntityManager.instance) {
      EntityManager.instance = new EntityManager();
    }
    return EntityManager.instance;
  }

  /**
   * Load entity by ID with caching
   */
  async loadEntity(entityId: string): Promise<Entity | null> {
    // Check cache first
    if (this.cache.has(entityId)) {
      return this.cache.get(entityId)!;
    }

    // Try Redis cache
    try {
      const redis = getRedisClient();
      const cached = await redis.get(`entity:${entityId}`);
      if (cached) {
        const entityData = JSON.parse(cached);
        const entity = this.deserializeEntity(entityData);
        this.cache.set(entityId, entity);
        return entity;
      }
    } catch (error) {
      logger.warn('Redis cache miss for entity:', entityId);
    }

    // Load from database
    const entity = await this.loadEntityFromDatabase(entityId);
    if (entity) {
      this.cache.set(entityId, entity);

      // Cache in Redis
      try {
        const redis = getRedisClient();
        await redis.set(`entity:${entityId}`, JSON.stringify(entity.toJSON()), {
          EX: 3600 // 1 hour TTL
        });
      } catch (error) {
        logger.warn('Failed to cache entity in Redis:', error);
      }
    }

    return entity;
  }

  /**
   * Load entity from database
   */
  private async loadEntityFromDatabase(entityId: string): Promise<Entity | null> {
    const pool = getPool();

    // First, determine entity type
    const typeQuery = `
      SELECT entity_type FROM entities
      WHERE id = $1 AND tenant_id = $2
    `;

    try {
      const typeResult = await pool.query(typeQuery, [entityId, this.getCurrentTenantId()]);
      if (typeResult.rows.length === 0) {
        return null;
      }

      const entityType = typeResult.rows[0].entity_type;

      switch (entityType) {
        case 'client':
          return await this.loadClientFromDatabase(entityId);
        case 'account':
          return await this.loadAccountFromDatabase(entityId);
        default:
          logger.error(`Unknown entity type: ${entityType}`);
          return null;
      }
    } catch (error) {
      logger.error('Failed to load entity from database:', error);
      return null;
    }
  }

  /**
   * Load client from database
   */
  private async loadClientFromDatabase(clientId: string): Promise<Client | null> {
    const pool = getPool();

    const query = `
      SELECT * FROM clients
      WHERE id = $1 AND tenant_id = $2
    `;

    try {
      const result = await pool.query(query, [clientId, this.getCurrentTenantId()]);
      if (result.rows.length === 0) {
        return null;
      }

      const row = result.rows[0];
      return new Client(
        row.id,
        row.tenant_id,
        row.datasource_id,
        row.first_name,
        row.last_name,
        row.email,
        new Date(row.date_of_birth),
        {
          ssn: row.ssn,
          kycStatus: row.kyc_status as KycStatus,
          sanctionsStatus: row.sanctions_status as SanctionsStatus,
          pepStatus: row.pep_status as PepStatus,
          kycExpiresAt: row.kyc_expires_at ? new Date(row.kyc_expires_at) : undefined,
          kycLastCheckedAt: row.kyc_last_checked_at ? new Date(row.kyc_last_checked_at) : undefined,
          isAccreditedInvestor: row.is_accredited_investor,
          riskTolerance: row.risk_tolerance,
          netWorth: row.net_worth,
          annualIncome: row.annual_income,
          createdAt: new Date(row.created_at),
          updatedAt: new Date(row.updated_at)
        }
      );
    } catch (error) {
      logger.error('Failed to load client from database:', error);
      return null;
    }
  }

  /**
   * Load account from database
   */
  private async loadAccountFromDatabase(accountId: string): Promise<Account | null> {
    const pool = getPool();

    const query = `
      SELECT a.*, at.account_type as specific_type
      FROM accounts a
      JOIN account_types at ON a.account_type = at.id
      WHERE a.id = $1 AND a.tenant_id = $2
    `;

    try {
      const result = await pool.query(query, [accountId, this.getCurrentTenantId()]);
      if (result.rows.length === 0) {
        return null;
      }

      const row = result.rows[0];

      switch (row.specific_type) {
        case 'personal':
          return new PersonalAccount(
            row.id,
            row.tenant_id,
            row.datasource_id,
            row.account_number,
            row.name,
            row.owner_id,
            row.custodian_id,
            row.risk_tolerance as RiskTolerance,
            row.investment_objective as InvestmentObjective,
            row.status as AccountStatus,
            row.net_worth,
            new Date(row.created_at),
            new Date(row.updated_at)
          );

        case 'ira':
          return new IRAAccount(
            row.id,
            row.tenant_id,
            row.datasource_id,
            row.account_number,
            row.name,
            row.owner_id,
            row.custodian_id,
            row.ira_type as IRAType,
            row.owner_age,
            row.contribution_limit,
            row.status as AccountStatus,
            new Date(row.created_at),
            new Date(row.updated_at)
          );

        case 'trust':
          return new TrustAccount(
            row.id,
            row.tenant_id,
            row.datasource_id,
            row.account_number,
            row.name,
            row.owner_id,
            row.custodian_id,
            row.trust_type as TrustType,
            row.trustee_id,
            row.beneficiaries || [],
            row.status as AccountStatus,
            new Date(row.created_at),
            new Date(row.updated_at)
          );

        default:
          logger.error(`Unknown account type: ${row.specific_type}`);
          return null;
      }
    } catch (error) {
      logger.error('Failed to load account from database:', error);
      return null;
    }
  }

  /**
   * Save entity to database
   */
  async saveEntity(entity: Entity): Promise<void> {
    const pool = getPool();

    try {
      if (entity instanceof Client) {
        await this.saveClientToDatabase(entity);
      } else if (entity instanceof Account) {
        await this.saveAccountToDatabase(entity);
      }

      // Update cache
      this.cache.set(entity.id, entity);

      // Update Redis cache
      try {
        const redis = getRedisClient();
        await redis.set(`entity:${entity.id}`, JSON.stringify(entity.toJSON()), {
          EX: 3600
        });
      } catch (error) {
        logger.warn('Failed to update Redis cache:', error);
      }

      logger.info(`Entity saved: ${entity.id}`);
    } catch (error) {
      logger.error('Failed to save entity:', error);
      throw error;
    }
  }

  /**
   * Save client to database
   */
  private async saveClientToDatabase(client: Client): Promise<void> {
    const pool = getPool();

    const query = `
      INSERT INTO clients (
        id, tenant_id, datasource_id, first_name, last_name, email, date_of_birth,
        ssn, kyc_status, sanctions_status, pep_status, kyc_expires_at, kyc_last_checked_at,
        is_accredited_investor, risk_tolerance, net_worth, annual_income, created_at, updated_at
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
      ON CONFLICT (id) DO UPDATE SET
        first_name = EXCLUDED.first_name,
        last_name = EXCLUDED.last_name,
        email = EXCLUDED.email,
        date_of_birth = EXCLUDED.date_of_birth,
        ssn = EXCLUDED.ssn,
        kyc_status = EXCLUDED.kyc_status,
        sanctions_status = EXCLUDED.sanctions_status,
        pep_status = EXCLUDED.pep_status,
        kyc_expires_at = EXCLUDED.kyc_expires_at,
        kyc_last_checked_at = EXCLUDED.kyc_last_checked_at,
        is_accredited_investor = EXCLUDED.is_accredited_investor,
        risk_tolerance = EXCLUDED.risk_tolerance,
        net_worth = EXCLUDED.net_worth,
        annual_income = EXCLUDED.annual_income,
        updated_at = EXCLUDED.updated_at
    `;

    await pool.query(query, [
      client.id, client.tenantId, client.datasourceId,
      client.firstName, client.lastName, client.email, client.dateOfBirth,
      client.ssn, client.kycStatus, client.sanctionsStatus, client.pepStatus,
      client.kycExpiresAt, client.kycLastCheckedAt,
      client.isAccreditedInvestor, client.riskTolerance, client.netWorth, client.annualIncome,
      client.createdAt, client.updatedAt
    ]);
  }

  /**
   * Save account to database
   */
  private async saveAccountToDatabase(account: Account): Promise<void> {
    const pool = getPool();

    // Base account data
    const baseQuery = `
      INSERT INTO accounts (
        id, tenant_id, datasource_id, account_number, name, owner_id, custodian_id,
        account_type, status, created_at, updated_at
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
      ON CONFLICT (id) DO UPDATE SET
        account_number = EXCLUDED.account_number,
        name = EXCLUDED.name,
        status = EXCLUDED.status,
        updated_at = EXCLUDED.updated_at
    `;

    await pool.query(baseQuery, [
      account.id, account.tenantId, account.datasourceId,
      account.accountNumber, account.name, account.ownerId, account.custodianId,
      account.accountType, account.status, account.createdAt, account.updatedAt
    ]);

    // Account-specific data
    if (account instanceof PersonalAccount) {
      await this.savePersonalAccountData(account);
    } else if (account instanceof IRAAccount) {
      await this.saveIRAAccountData(account);
    } else if (account instanceof TrustAccount) {
      await this.saveTrustAccountData(account);
    }
  }

  /**
   * Save personal account specific data
   */
  private async savePersonalAccountData(account: PersonalAccount): Promise<void> {
    const pool = getPool();

    const query = `
      INSERT INTO personal_accounts (account_id, risk_tolerance, investment_objective, net_worth)
      VALUES ($1, $2, $3, $4)
      ON CONFLICT (account_id) DO UPDATE SET
        risk_tolerance = EXCLUDED.risk_tolerance,
        investment_objective = EXCLUDED.investment_objective,
        net_worth = EXCLUDED.net_worth
    `;

    await pool.query(query, [
      account.id, account.riskTolerance, account.investmentObjective, account.netWorth
    ]);
  }

  /**
   * Save IRA account specific data
   */
  private async saveIRAAccountData(account: IRAAccount): Promise<void> {
    const pool = getPool();

    const query = `
      INSERT INTO ira_accounts (account_id, ira_type, owner_age, contribution_limit)
      VALUES ($1, $2, $3, $4)
      ON CONFLICT (account_id) DO UPDATE SET
        ira_type = EXCLUDED.ira_type,
        owner_age = EXCLUDED.owner_age,
        contribution_limit = EXCLUDED.contribution_limit
    `;

    await pool.query(query, [
      account.id, account.iraType, account.ownerAge, account.contributionLimit
    ]);
  }

  /**
   * Save trust account specific data
   */
  private async saveTrustAccountData(account: TrustAccount): Promise<void> {
    const pool = getPool();

    const query = `
      INSERT INTO trust_accounts (account_id, trust_type, trustee_id, beneficiaries)
      VALUES ($1, $2, $3, $4)
      ON CONFLICT (account_id) DO UPDATE SET
        trust_type = EXCLUDED.trust_type,
        trustee_id = EXCLUDED.trustee_id,
        beneficiaries = EXCLUDED.beneficiaries
    `;

    await pool.query(query, [
      account.id, account.trustType, account.trusteeId, JSON.stringify(account.beneficiaries)
    ]);
  }

  /**
   * Delete entity
   */
  async deleteEntity(entityId: string): Promise<void> {
    const pool = getPool();

    try {
      await pool.query('DELETE FROM entities WHERE id = $1 AND tenant_id = $2', [
        entityId, this.getCurrentTenantId()
      ]);

      // Remove from cache
      this.cache.delete(entityId);

      // Remove from Redis
      try {
        const redis = getRedisClient();
        await redis.del(`entity:${entityId}`);
      } catch (error) {
        logger.warn('Failed to remove from Redis cache:', error);
      }

      logger.info(`Entity deleted: ${entityId}`);
    } catch (error) {
      logger.error('Failed to delete entity:', error);
      throw error;
    }
  }

  /**
   * Deserialize entity from JSON data
   */
  private deserializeEntity(data: any): Entity {
    switch (data.entityType) {
      case 'client':
        return new Client(
          data.id, data.tenantId, data.datasourceId,
          data.firstName, data.lastName, data.email, new Date(data.dateOfBirth),
          {
            ssn: data.ssn,
            kycStatus: data.kycStatus,
            sanctionsStatus: data.sanctionsStatus,
            pepStatus: data.pepStatus,
            kycExpiresAt: data.kycExpiresAt ? new Date(data.kycExpiresAt) : undefined,
            kycLastCheckedAt: data.kycLastCheckedAt ? new Date(data.kycLastCheckedAt) : undefined,
            isAccreditedInvestor: data.isAccreditedInvestor,
            riskTolerance: data.riskTolerance,
            netWorth: data.netWorth,
            annualIncome: data.annualIncome,
            createdAt: new Date(data.createdAt),
            updatedAt: new Date(data.updatedAt)
          }
        );

      case 'account':
        // This would need more complex logic to instantiate the correct account type
        // For now, return a basic account
        throw new Error('Account deserialization not implemented');

      default:
        throw new Error(`Unknown entity type: ${data.entityType}`);
    }
  }

  /**
   * Get current tenant ID (would come from request context)
   */
  private getCurrentTenantId(): string {
    // This should come from the current request context
    // For now, return a placeholder
    return getEnv('DEFAULT_TENANT_ID', 'VITE_DEFAULT_TENANT_ID', 'default-tenant') as string;
  }

  /**
   * Clear cache (useful for testing)
   */
  clearCache(): void {
    this.cache.clear();
  }
}