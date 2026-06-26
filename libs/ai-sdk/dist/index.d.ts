export interface AIRequest {
    model: string;
    messages: Array<{
        role: 'user' | 'assistant' | 'system';
        content: string;
    }>;
    temperature?: number;
    max_tokens?: number;
}
export interface AIResponse {
    choices: Array<{
        message: {
            role: string;
            content: string;
        };
        finish_reason: string;
    }>;
    usage: {
        prompt_tokens: number;
        completion_tokens: number;
        total_tokens: number;
    };
}
export interface TaxOptimizationRequest {
    umaId: string;
    holdings: Array<{
        symbol: string;
        quantity: number;
        basis: number;
        currentPrice: number;
    }>;
    taxYear: number;
    strategy: 'harvest' | 'defer' | 'optimize';
}
export interface AttributionRequest {
    portfolioId: string;
    returns: number[];
    benchmarkReturns: number[];
    factors: string[];
    period: string;
}
export interface IndexOptimizationRequest {
    indexId: string;
    holdings: Array<{
        symbol: string;
        weight: number;
        drift: number;
    }>;
    constraints: {
        maxDrift: number;
        taxSensitivity: number;
        esgFocus: boolean;
    };
}
export declare class AIService {
    private client;
    private cache;
    private defaultCacheTTL;
    private retryConfig;
    constructor(apiKey: string, baseURL?: string);
    chat(request: AIRequest): Promise<AIResponse>;
    private getCacheKey;
    private getCachedResult;
    private setCachedResult;
    private retryWithBackoff;
    optimizeTax(request: TaxOptimizationRequest): Promise<any>;
    analyzeAttribution(request: AttributionRequest): Promise<any>;
    optimizeIndex(request: IndexOptimizationRequest): Promise<any>;
    private buildTaxOptimizationPrompt;
    private buildAttributionPrompt;
    private buildIndexOptimizationPrompt;
    private parseTaxOptimizationResponse;
    private parseAttributionResponse;
    private parseIndexOptimizationResponse;
    clearExpiredCache(): void;
    getCacheStats(): {
        size: number;
        hitRate: number;
    };
    healthCheck(): Promise<boolean>;
    setRetryConfig(config: Partial<typeof this.retryConfig>): void;
    setCacheTTL(ttl: number): void;
}
export declare function createAIService(apiKey: string): AIService;
