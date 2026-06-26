import { Entity } from '../entities/index.js';
export declare class EntityManager {
    private static instance;
    private cache;
    private constructor();
    static getInstance(): EntityManager;
    loadEntity(entityId: string): Promise<Entity | null>;
    private loadEntityFromDatabase;
    private loadClientFromDatabase;
    private loadAccountFromDatabase;
    saveEntity(entity: Entity): Promise<void>;
    private saveClientToDatabase;
    private saveAccountToDatabase;
    private savePersonalAccountData;
    private saveIRAAccountData;
    private saveTrustAccountData;
    deleteEntity(entityId: string): Promise<void>;
    private deserializeEntity;
    private getCurrentTenantId;
    clearCache(): void;
}
//# sourceMappingURL=EntityManager.d.ts.map