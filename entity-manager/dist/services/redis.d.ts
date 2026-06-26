import { createClient } from 'redis';
export declare function connectRedis(): Promise<void>;
export declare function getRedisClient(): ReturnType<typeof createClient>;
export declare function disconnectRedis(): Promise<void>;
//# sourceMappingURL=redis.d.ts.map