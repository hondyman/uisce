import { getPool } from '../services/database.js';
import { getRedisClient } from '../services/redis.js';
import { logger } from '../utils/logger.js';
import { getEnv } from '../../internal/pkg/env/getEnv.js';
import { Client, Account, PersonalAccount, IRAAccount, TrustAccount } from '../entities/index.js';
export class EntityManager {
    static instance;
    cache = new Map();
    constructor() { }
    static getInstance() {
        if (!EntityManager.instance) {
            EntityManager.instance = new EntityManager();
        }
        return EntityManager.instance;
    }
    async loadEntity(entityId) {
        if (this.cache.has(entityId)) {
            return this.cache.get(entityId);
        }
        try {
            const redis = getRedisClient();
            const cached = await redis.get(`entity:${entityId}`);
            if (cached) {
                const entityData = JSON.parse(cached);
                const entity = this.deserializeEntity(entityData);
                this.cache.set(entityId, entity);
                return entity;
            }
        }
        catch (error) {
            logger.warn('Redis cache miss for entity:', entityId);
        }
        const entity = await this.loadEntityFromDatabase(entityId);
        if (entity) {
            this.cache.set(entityId, entity);
            try {
                const redis = getRedisClient();
                await redis.set(`entity:${entityId}`, JSON.stringify(entity.toJSON()), {
                    EX: 3600
                });
            }
            catch (error) {
                logger.warn('Failed to cache entity in Redis:', error);
            }
        }
        return entity;
    }
    async loadEntityFromDatabase(entityId) {
        const pool = getPool();
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
        }
        catch (error) {
            logger.error('Failed to load entity from database:', error);
            return null;
        }
    }
    async loadClientFromDatabase(clientId) {
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
            return new Client(row.id, row.tenant_id, row.datasource_id, row.first_name, row.last_name, row.email, new Date(row.date_of_birth), {
                ssn: row.ssn,
                kycStatus: row.kyc_status,
                sanctionsStatus: row.sanctions_status,
                pepStatus: row.pep_status,
                kycExpiresAt: row.kyc_expires_at ? new Date(row.kyc_expires_at) : undefined,
                kycLastCheckedAt: row.kyc_last_checked_at ? new Date(row.kyc_last_checked_at) : undefined,
                isAccreditedInvestor: row.is_accredited_investor,
                riskTolerance: row.risk_tolerance,
                netWorth: row.net_worth,
                annualIncome: row.annual_income,
                createdAt: new Date(row.created_at),
                updatedAt: new Date(row.updated_at)
            });
        }
        catch (error) {
            logger.error('Failed to load client from database:', error);
            return null;
        }
    }
    async loadAccountFromDatabase(accountId) {
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
                    return new PersonalAccount(row.id, row.tenant_id, row.datasource_id, row.account_number, row.name, row.owner_id, row.custodian_id, row.risk_tolerance, row.investment_objective, row.status, row.net_worth, new Date(row.created_at), new Date(row.updated_at));
                case 'ira':
                    return new IRAAccount(row.id, row.tenant_id, row.datasource_id, row.account_number, row.name, row.owner_id, row.custodian_id, row.ira_type, row.owner_age, row.contribution_limit, row.status, new Date(row.created_at), new Date(row.updated_at));
                case 'trust':
                    return new TrustAccount(row.id, row.tenant_id, row.datasource_id, row.account_number, row.name, row.owner_id, row.custodian_id, row.trust_type, row.trustee_id, row.beneficiaries || [], row.status, new Date(row.created_at), new Date(row.updated_at));
                default:
                    logger.error(`Unknown account type: ${row.specific_type}`);
                    return null;
            }
        }
        catch (error) {
            logger.error('Failed to load account from database:', error);
            return null;
        }
    }
    async saveEntity(entity) {
        const pool = getPool();
        try {
            if (entity instanceof Client) {
                await this.saveClientToDatabase(entity);
            }
            else if (entity instanceof Account) {
                await this.saveAccountToDatabase(entity);
            }
            this.cache.set(entity.id, entity);
            try {
                const redis = getRedisClient();
                await redis.set(`entity:${entity.id}`, JSON.stringify(entity.toJSON()), {
                    EX: 3600
                });
            }
            catch (error) {
                logger.warn('Failed to update Redis cache:', error);
            }
            logger.info(`Entity saved: ${entity.id}`);
        }
        catch (error) {
            logger.error('Failed to save entity:', error);
            throw error;
        }
    }
    async saveClientToDatabase(client) {
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
    async saveAccountToDatabase(account) {
        const pool = getPool();
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
        if (account instanceof PersonalAccount) {
            await this.savePersonalAccountData(account);
        }
        else if (account instanceof IRAAccount) {
            await this.saveIRAAccountData(account);
        }
        else if (account instanceof TrustAccount) {
            await this.saveTrustAccountData(account);
        }
    }
    async savePersonalAccountData(account) {
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
    async saveIRAAccountData(account) {
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
    async saveTrustAccountData(account) {
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
    async deleteEntity(entityId) {
        const pool = getPool();
        try {
            await pool.query('DELETE FROM entities WHERE id = $1 AND tenant_id = $2', [
                entityId, this.getCurrentTenantId()
            ]);
            this.cache.delete(entityId);
            try {
                const redis = getRedisClient();
                await redis.del(`entity:${entityId}`);
            }
            catch (error) {
                logger.warn('Failed to remove from Redis cache:', error);
            }
            logger.info(`Entity deleted: ${entityId}`);
        }
        catch (error) {
            logger.error('Failed to delete entity:', error);
            throw error;
        }
    }
    deserializeEntity(data) {
        switch (data.entityType) {
            case 'client':
                return new Client(data.id, data.tenantId, data.datasourceId, data.firstName, data.lastName, data.email, new Date(data.dateOfBirth), {
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
                });
            case 'account':
                throw new Error('Account deserialization not implemented');
            default:
                throw new Error(`Unknown entity type: ${data.entityType}`);
        }
    }
    getCurrentTenantId() {
        return getEnv('DEFAULT_TENANT_ID', 'VITE_DEFAULT_TENANT_ID', 'default-tenant');
    }
    clearCache() {
        this.cache.clear();
    }
}
//# sourceMappingURL=EntityManager.js.map