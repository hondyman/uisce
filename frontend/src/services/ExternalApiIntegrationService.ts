import { devWarn, devError } from '../utils/devLogger';
import { getEnv } from '@internal/pkg/env/getEnv';

/**
 * ExternalApiIntegrationService.ts
 * 
 * Handles integration with external data providers and AI services:
 * - MSCI ESG Ratings API
 * - World-Check AML Screening API
 * - Bloomberg Benchmark Data API
 * - AWS SageMaker Risk Model Endpoint
 * - Refinitiv ESG and Performance Data
 * 
 * All integrations include:
 * - Request/response caching for performance
 * - Retry logic with exponential backoff
 * - Error handling and logging
 * - Credential management
 * - Request timeout handling
 */

interface ApiCredentials {
  apiKey?: string;
  username?: string;
  password?: string;
  token?: string;
  endpoint: string;
  timeout?: number;
}

interface CacheEntry<T> {
  data: T;
  timestamp: number;
  ttl: number;
}

export class ExternalApiIntegrationService {
  private cache: Map<string, CacheEntry<any>> = new Map();
  private credentials: Map<string, ApiCredentials> = new Map();
  private maxRetries = 3;
  private initialRetryDelay = 1000; // ms

  constructor() {
    this.initializeCredentials();
  }

  /**
   * Initialize API credentials from environment variables or secure storage
   * In production, these would come from secure vaults (e.g., AWS Secrets Manager)
   */
  private initializeCredentials(): void {
    // MSCI ESG API
    this.credentials.set('msci_api', {
      apiKey: getEnv('', 'VITE_MSCI_API_KEY', '') as string,
      endpoint: getEnv('', 'VITE_MSCI_ENDPOINT', 'https://api.msci.com/esg-ratings') as string,
      timeout: 10000
    });

    // World-Check AML API
    this.credentials.set('world_check_api', {
      username: getEnv('', 'VITE_WORLD_CHECK_USERNAME', '') as string,
      password: getEnv('', 'VITE_WORLD_CHECK_PASSWORD', '') as string,
      endpoint: getEnv('', 'VITE_WORLD_CHECK_ENDPOINT', 'https://api.world-check.com/screen') as string,
      timeout: 15000
    });

    // Bloomberg API
    this.credentials.set('bloomberg_api', {
      token: getEnv('', 'VITE_BLOOMBERG_TOKEN', '') as string,
      endpoint: getEnv('', 'VITE_BLOOMBERG_ENDPOINT', 'https://api.bloomberg.com/benchmark-data') as string,
      timeout: 12000
    });

    // AWS SageMaker Endpoint
    this.credentials.set('sagemaker_endpoint', {
      endpoint: getEnv('', 'VITE_SAGEMAKER_ENDPOINT', 'https://api.sagemaker.example.com/risk-model') as string,
      timeout: 30000
    });
  }

  /**
   * Get ESG ratings for a security from MSCI API
   */
  async getESGRating(securityId: string, securityType: 'ticker' | 'isin' | 'cusip' = 'ticker'): Promise<{
    securityId: string;
    esgScore: number;
    esgRating: string;
    environmentScore: number;
    socialScore: number;
    governanceScore: number;
    controversies: any[];
    lastUpdated: string;
  } | null> {
    const cacheKey = `esg_rating_${securityId}`;
    
    // Check cache
    const cached = this.getFromCache(cacheKey);
    if (cached) return cached as {
      securityId: string;
      esgScore: number;
      esgRating: string;
      environmentScore: number;
      socialScore: number;
      governanceScore: number;
      controversies: any[];
      lastUpdated: string;
    };

    try {
      const credentials = this.credentials.get('msci_api');
      if (!credentials?.apiKey) {
        devWarn('MSCI API key not configured');
        return null;
      }

      const params = new URLSearchParams({
        [securityType]: securityId,
        format: 'json'
      });

      const response = await this.retryableRequest(
        `${credentials.endpoint}?${params}`,
        {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${credentials.apiKey}`,
            'Content-Type': 'application/json'
          }
        },
        credentials.timeout
      );

      const data = await response.json();
      const result = {
        securityId,
        esgScore: data.esgScore || 0,
        esgRating: data.esgRating || 'N/A',
        environmentScore: data.envScore || 0,
        socialScore: data.socScore || 0,
        governanceScore: data.govScore || 0,
        controversies: data.controversies || [],
        lastUpdated: new Date().toISOString()
      };

      // Cache for 24 hours
      this.setInCache(cacheKey, result, 24 * 60 * 60 * 1000);
      return result;
    } catch (error) {
      devError('Failed to fetch ESG rating:', error);
      return null;
    }
  }

  /**
   * Screen a person or entity against AML watchlists via World-Check
   */
  async screenAML(
    name: string,
    entityType: 'INDIVIDUAL' | 'ORGANIZATION' = 'INDIVIDUAL'
  ): Promise<{
    screeningId: string;
    entityName: string;
    riskLevel: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
    matches: any[];
    screeningDate: string;
  } | null> {
    const cacheKey = `aml_screening_${name.toLowerCase()}`;
    
    // Check cache
    const cached = this.getFromCache(cacheKey);
    if (cached) return cached as {
      screeningId: string;
      entityName: string;
      riskLevel: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
      matches: any[];
      screeningDate: string;
    };

    try {
      const credentials = this.credentials.get('world_check_api');
      if (!credentials?.username || !credentials?.password) {
        devWarn('World-Check API credentials not configured');
        return null;
      }

      const response = await this.retryableRequest(
        credentials.endpoint,
        {
          method: 'POST',
          headers: {
            'Authorization': `Basic ${btoa(`${credentials.username}:${credentials.password}`)}`,
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            entityName: name,
            entityType,
            screeningDate: new Date().toISOString()
          })
        },
        credentials.timeout
      );

      const data = await response.json();
      const result = {
        screeningId: data.screeningId || `screen_${Date.now()}`,
        entityName: name,
        riskLevel: data.riskLevel || 'LOW',
        matches: data.matches || [],
        screeningDate: new Date().toISOString()
      };

      // Cache for 7 days
      this.setInCache(cacheKey, result, 7 * 24 * 60 * 60 * 1000);
      return result;
    } catch (error) {
      devError('Failed to screen AML:', error);
      return null;
    }
  }

  /**
   * Get benchmark performance data from Bloomberg
   */
  async getBenchmarkPerformance(
    benchmarkIndex: string,
    startDate: string,
    endDate: string
  ): Promise<{
    benchmark: string;
    returns: number;
    volatility: number;
    sharpeRatio: number;
    dataPoints: Array<{ date: string; value: number }>;
    lastUpdated: string;
  } | null> {
    const cacheKey = `benchmark_${benchmarkIndex}_${startDate}_${endDate}`;
    
    // Check cache
    const cached = this.getFromCache(cacheKey);
    if (cached) return cached as {
      benchmark: string;
      returns: number;
      volatility: number;
      sharpeRatio: number;
      dataPoints: { date: string; value: number; }[];
      lastUpdated: string;
    };

    try {
      const credentials = this.credentials.get('bloomberg_api');
      if (!credentials?.token) {
        devWarn('Bloomberg API token not configured');
        return null;
      }

      const response = await this.retryableRequest(
        `${credentials.endpoint}?index=${benchmarkIndex}&startDate=${startDate}&endDate=${endDate}`,
        {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${credentials.token}`,
            'Content-Type': 'application/json'
          }
        },
        credentials.timeout
      );

      const data = await response.json();
      const result = {
        benchmark: benchmarkIndex,
        returns: data.returns || 0,
        volatility: data.volatility || 0,
        sharpeRatio: data.sharpeRatio || 0,
        dataPoints: data.dataPoints || [],
        lastUpdated: new Date().toISOString()
      };

      // Cache for 1 day
      this.setInCache(cacheKey, result, 24 * 60 * 60 * 1000);
      return result;
    } catch (error) {
      devError('Failed to fetch benchmark performance:', error);
      return null;
    }
  }

  /**
   * Call AI risk model endpoint (AWS SageMaker) for portfolio risk assessment
   */
  async assessPortfolioRisk(portfolioData: {
    holdings: Array<{ ticker: string; weight: number; price: number; volatility: number }>;
    correlationMatrix?: number[][];
    historicalReturns?: number[];
  }): Promise<{
    var95: number;
    var99: number;
    conditionalVar: number;
    stressTestResults: Record<string, number>;
    riskLevel: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
    recommendations: string[];
    generatedAt: string;
  } | null> {
    const cacheKey = `portfolio_risk_${JSON.stringify(portfolioData).substring(0, 50)}`;
    
    // Check cache (shorter TTL for AI results - 1 hour)
    const cached = this.getFromCache(cacheKey);
    if (cached) return cached as {
      var95: number;
      var99: number;
      conditionalVar: number;
      stressTestResults: Record<string, number>;
      riskLevel: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
      recommendations: string[];
      generatedAt: string;
    };

    try {
      const credentials = this.credentials.get('sagemaker_endpoint');
      if (!credentials?.endpoint) {
        devWarn('SageMaker endpoint not configured');
        return null;
      }

      const response = await this.retryableRequest(
        credentials.endpoint,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(portfolioData)
        },
        credentials.timeout
      );

      const data = await response.json();
      const result = {
        var95: data.var95 || 0.05,
        var99: data.var99 || 0.08,
        conditionalVar: data.cvar || 0.12,
        stressTestResults: data.stressTests || {},
        riskLevel: this.calculateRiskLevel(data.var95),
        recommendations: data.recommendations || [],
        generatedAt: new Date().toISOString()
      };

      // Cache for 1 hour
      this.setInCache(cacheKey, result, 60 * 60 * 1000);
      return result;
    } catch (error) {
      devError('Failed to assess portfolio risk:', error);
      return null;
    }
  }

  /**
   * Retryable HTTP request with exponential backoff
   */
  private async retryableRequest(
    url: string,
    options: RequestInit,
    timeout: number = 10000
  ): Promise<Response> {
    let lastError: Error | null = null;

    for (let attempt = 0; attempt < this.maxRetries; attempt++) {
      try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), timeout);

        const response = await fetch(url, {
          ...options,
          signal: controller.signal
        });

        clearTimeout(timeoutId);

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        return response;
      } catch (error) {
        lastError = error as Error;
        
        if (attempt < this.maxRetries - 1) {
          const delay = this.initialRetryDelay * Math.pow(2, attempt);
          await new Promise(resolve => setTimeout(resolve, delay));
        }
      }
    }

    throw lastError || new Error('Request failed after retries');
  }

  /**
   * Cache management
   */
  private getFromCache<T>(key: string): T | null {
    const entry = this.cache.get(key);
    if (!entry) return null;

    if (Date.now() - entry.timestamp > entry.ttl) {
      this.cache.delete(key);
      return null;
    }

    return entry.data as T;
  }

  private setInCache<T>(key: string, data: T, ttl: number): void {
    this.cache.set(key, {
      data,
      timestamp: Date.now(),
      ttl
    });
  }

  /**
   * Clear cache for specific key or all
   */
  clearCache(key?: string): void {
    if (key) {
      this.cache.delete(key);
    } else {
      this.cache.clear();
    }
  }

  /**
   * Calculate risk level based on VaR
   */
  private calculateRiskLevel(var95: number): 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL' {
    if (var95 < 0.02) return 'LOW';
    if (var95 < 0.05) return 'MEDIUM';
    if (var95 < 0.10) return 'HIGH';
    return 'CRITICAL';
  }

  /**
   * Validate API credentials are configured
   */
  validateCredentials(): { valid: boolean; missingServices: string[] } {
    const missingServices: string[] = [];

    if (!getEnv('', 'VITE_MSCI_API_KEY')) missingServices.push('MSCI ESG API');
    if (!getEnv('', 'VITE_WORLD_CHECK_USERNAME')) missingServices.push('World-Check AML');
    if (!getEnv('', 'VITE_BLOOMBERG_TOKEN')) missingServices.push('Bloomberg API');
    if (!getEnv('', 'VITE_SAGEMAKER_ENDPOINT')) missingServices.push('AWS SageMaker');

    return {
      valid: missingServices.length === 0,
      missingServices
    };
  }

  /**
   * Get health check status for all integrated services
   */
  async getHealthStatus(): Promise<Record<string, { healthy: boolean; lastChecked: string; error?: string }>> {
    const status: Record<string, { healthy: boolean; lastChecked: string; error?: string }> = {};

    // Check each service
    const services = ['msci_api', 'world_check_api', 'bloomberg_api', 'sagemaker_endpoint'];
    
    for (const service of services) {
      try {
        const credentials = this.credentials.get(service);
        if (!credentials) {
          status[service] = {
            healthy: false,
            lastChecked: new Date().toISOString(),
            error: 'Credentials not configured'
          };
          continue;
        }

        // Try a simple request to check health
        const response = await Promise.race([
          fetch(credentials.endpoint, { method: 'HEAD' }),
          new Promise((_, reject) => 
            setTimeout(() => reject(new Error('Timeout')), 5000)
          )
        ]) as Response;

        status[service] = {
          healthy: response.ok || response.status === 405, // 405 Method Not Allowed is OK for HEAD
          lastChecked: new Date().toISOString()
        };
      } catch (error) {
        status[service] = {
          healthy: false,
          lastChecked: new Date().toISOString(),
          error: (error as Error).message
        };
      }
    }

    return status;
  }
}

// Export singleton instance
export const externalApiService = new ExternalApiIntegrationService();

export default ExternalApiIntegrationService;
