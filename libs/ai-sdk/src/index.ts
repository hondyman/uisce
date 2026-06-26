// AI SDK for xAI integration
// Provides unified interface for AI-powered financial analysis

import axios, { AxiosInstance } from 'axios';

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

export class AIService {
  private client: AxiosInstance;
  private cache: Map<string, { data: any; timestamp: number; ttl: number }> = new Map();
  private defaultCacheTTL = 5 * 60 * 1000; // 5 minutes
  private retryConfig = {
    maxRetries: 3,
    baseDelay: 1000, // 1 second
    maxDelay: 10000, // 10 seconds
  };

  constructor(apiKey: string, baseURL: string = 'https://api.x.ai/v1') {
    this.client = axios.create({
      baseURL,
      headers: {
        'Authorization': `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      timeout: 30000, // 30 second timeout
    });

    // Add request interceptor for logging
    this.client.interceptors.request.use(
      (config: any) => {
        console.log(`AI Service Request: ${config.method?.toUpperCase()} ${config.url}`);
        return config;
      },
      (error: any) => Promise.reject(error)
    );

    // Enhanced response interceptor for error handling
    this.client.interceptors.response.use(
      (response: any) => {
        console.log(`AI Service Response: ${response.status} ${response.statusText}`);
        return response;
      },
      (error: any) => {
        console.error('AI Service Error:', error.response?.data || error.message);

        if (error.response?.status === 429) {
          throw new Error('AI service rate limit exceeded. Please retry later.');
        }
        if (error.response?.status === 401) {
          throw new Error('Invalid AI service API key. Please check your credentials.');
        }
        if (error.response?.status === 403) {
          throw new Error('AI service access forbidden. Please check your permissions.');
        }
        if (error.response?.status >= 500) {
          throw new Error('AI service internal error. Please try again later.');
        }
        if (error.code === 'ECONNABORTED') {
          throw new Error('AI service request timeout. Please try again.');
        }

        throw new Error(`AI service error: ${error.message}`);
      }
    );
  }

  async chat(request: AIRequest): Promise<AIResponse> {
    return this.retryWithBackoff(async () => {
      try {
        const response = await this.client.post('/chat/completions', request);
        return response.data;
      } catch (error) {
        console.error('AI chat request failed:', error);
        throw error;
      }
    });
  }

  private getCacheKey(request: any): string {
    return JSON.stringify(request);
  }

  private getCachedResult(key: string): any | null {
    const cached = this.cache.get(key);
    if (cached && Date.now() - cached.timestamp < cached.ttl) {
      console.log('Returning cached result for key:', key);
      return cached.data;
    }
    if (cached) {
      this.cache.delete(key);
    }
    return null;
  }

  private setCachedResult(key: string, data: any, ttl?: number): void {
    const cacheTTL = ttl || this.defaultCacheTTL;
    this.cache.set(key, { data, timestamp: Date.now(), ttl: cacheTTL });
  }

  private async retryWithBackoff<T>(operation: () => Promise<T>): Promise<T> {
    let lastError: Error;

    for (let attempt = 0; attempt <= this.retryConfig.maxRetries; attempt++) {
      try {
        return await operation();
      } catch (error) {
        lastError = error as Error;

        if (attempt === this.retryConfig.maxRetries) {
          break;
        }

        // Calculate delay with exponential backoff
        const delay = Math.min(
          this.retryConfig.baseDelay * Math.pow(2, attempt),
          this.retryConfig.maxDelay
        );

        console.log(`AI service retry attempt ${attempt + 1} after ${delay}ms delay`);
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }

    throw lastError!;
  }

  async optimizeTax(request: TaxOptimizationRequest): Promise<any> {
    const cacheKey = this.getCacheKey({ type: 'tax', ...request });
    const cached = this.getCachedResult(cacheKey);
    if (cached) return cached;

    const prompt = this.buildTaxOptimizationPrompt(request);

    const aiRequest: AIRequest = {
      model: 'grok-beta',
      messages: [{
        role: 'user',
        content: prompt,
      }],
      temperature: 0.1, // Low temperature for consistent financial analysis
      max_tokens: 2000,
    };

    const response = await this.chat(aiRequest);
    const result = this.parseTaxOptimizationResponse(response.choices[0].message.content);

    // Cache tax optimization results for longer (30 minutes) since they change less frequently
    this.setCachedResult(cacheKey, result, 30 * 60 * 1000);
    return result;
  }

  async analyzeAttribution(request: AttributionRequest): Promise<any> {
    const prompt = this.buildAttributionPrompt(request);

    const aiRequest: AIRequest = {
      model: 'grok-beta',
      messages: [{
        role: 'user',
        content: prompt,
      }],
      temperature: 0.1,
      max_tokens: 1500,
    };

    const response = await this.chat(aiRequest);
    return this.parseAttributionResponse(response.choices[0].message.content);
  }

  async optimizeIndex(request: IndexOptimizationRequest): Promise<any> {
    const prompt = this.buildIndexOptimizationPrompt(request);

    const aiRequest: AIRequest = {
      model: 'grok-beta',
      messages: [{
        role: 'user',
        content: prompt,
      }],
      temperature: 0.1,
      max_tokens: 2000,
    };

    const response = await this.chat(aiRequest);
    return this.parseIndexOptimizationResponse(response.choices[0].message.content);
  }

  private buildTaxOptimizationPrompt(request: TaxOptimizationRequest): string {
    return `
Analyze tax optimization opportunities for UMA ${request.umaId}:

Holdings:
${request.holdings.map(h => `${h.symbol}: ${h.quantity} shares @ $${h.basis} basis, current $${h.currentPrice}`).join('\n')}

Tax Year: ${request.taxYear}
Strategy: ${request.strategy}

Provide:
1. Optimal tax lots to harvest
2. Estimated tax savings
3. Wash sale risk assessment
4. ESG impact analysis
5. Recommended actions with timing

Format as JSON with keys: lots_selected, tax_saved, wash_sale_risk, esg_score, recommendations
`;
  }

  private buildAttributionPrompt(request: AttributionRequest): string {
    return `
Perform Brinson-Fachler attribution analysis for portfolio ${request.portfolioId}:

Portfolio Returns: [${request.returns.join(', ')}]
Benchmark Returns: [${request.benchmarkReturns.join(', ')}]
Factors: ${request.factors.join(', ')}
Period: ${request.period}

Calculate:
1. Total attribution effect
2. Factor contributions
3. Alpha generation
4. Risk-adjusted performance

Format as JSON with keys: total_attribution, factor_contributions, alpha, sharpe_ratio
`;
  }

  private buildIndexOptimizationPrompt(request: IndexOptimizationRequest): string {
    return `
Optimize direct index ${request.indexId} for tax efficiency and tracking:

Current Holdings:
${request.holdings.map(h => `${h.symbol}: ${h.weight}% weight, ${h.drift}% drift`).join('\n')}

Constraints:
- Max Drift: ${request.constraints.maxDrift}%
- Tax Sensitivity: ${request.constraints.taxSensitivity}/10
- ESG Focus: ${request.constraints.esgFocus ? 'Yes' : 'No'}

Provide:
1. Rebalancing recommendations
2. Tax-efficient trades
3. Expected drift reduction
4. ESG score impact

Format as JSON with keys: trades, drift_reduction, tax_impact, esg_impact
`;
  }

  private parseTaxOptimizationResponse(content: string): any {
    try {
      return JSON.parse(content);
    } catch {
      // Fallback parsing for non-JSON responses
      return {
        lots_selected: [],
        tax_saved: 0,
        wash_sale_risk: 0,
        esg_score: 0,
        recommendations: content,
      };
    }
  }

  private parseAttributionResponse(content: string): any {
    try {
      return JSON.parse(content);
    } catch {
      return {
        total_attribution: 0,
        factor_contributions: {},
        alpha: 0,
        sharpe_ratio: 0,
        analysis: content,
      };
    }
  }

  private parseIndexOptimizationResponse(content: string): any {
    try {
      return JSON.parse(content);
    } catch {
      return {
        trades: [],
        drift_reduction: 0,
        tax_impact: 0,
        esg_impact: 0,
        recommendations: content,
      };
    }
  }

  // Clear expired cache entries
  clearExpiredCache(): void {
    const now = Date.now();
    for (const [key, cached] of this.cache.entries()) {
      if (now - cached.timestamp >= cached.ttl) {
        this.cache.delete(key);
      }
    }
  }

  // Get cache statistics
  getCacheStats(): { size: number; hitRate: number } {
    return {
      size: this.cache.size,
      hitRate: 0, // Would need to track hits/misses for accurate rate
    };
  }

  // Health check for AI service
  async healthCheck(): Promise<boolean> {
    try {
      const testRequest: AIRequest = {
        model: 'grok-beta',
        messages: [{ role: 'user', content: 'Hello' }],
        max_tokens: 10,
      };
      await this.chat(testRequest);
      return true;
    } catch (error) {
      console.error('AI service health check failed:', error);
      return false;
    }
  }

  // Configure retry settings
  setRetryConfig(config: Partial<typeof this.retryConfig>): void {
    this.retryConfig = { ...this.retryConfig, ...config };
  }

  // Configure cache TTL
  setCacheTTL(ttl: number): void {
    this.defaultCacheTTL = ttl;
  }
}

// Factory function for creating AI service instances
export function createAIService(apiKey: string): AIService {
  return new AIService(apiKey);
}