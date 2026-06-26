// Wealth Management API Service
// This service provides methods to interact with wealth management metrics

export interface WealthManagementMetric {
  node_id: string;
  category: string;
  description: string;
  governance_status: 'golden' | 'draft';
  formula_type: string;
  formula?: string;
  arguments?: Record<string, string>;
  audience: string[];
  tags: string[];
  created_at?: string;
  updated_at?: string;
}

export interface MetricCalculation {
  metric_id: string;
  value: number;
  timestamp: string;
  grain_values?: Record<string, any>;
}

import { devError } from '../utils/devLogger';

class WealthManagementService {
  private baseUrl: string;

  constructor(baseUrl: string = '/api') {
    this.baseUrl = baseUrl;
  }

  /**
   * Fetch all wealth management metrics for a tenant
   */
  async getMetrics(tenantId: string): Promise<WealthManagementMetric[]> {
    try {
  const response = await fetch(`${this.baseUrl}/wealth-management/metrics?tenantId=${tenantId}`, { credentials: 'include' });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data.metrics || [];
    } catch (error) {
      devError('Error fetching wealth management metrics:', error);
      throw error; // Let the component handle the error
    }
  }

  /**
   * Fetch calculation results for specific metrics
   */
  async getMetricCalculations(
    tenantId: string,
    metricIds: string[],
    clientId?: string,
    dateRange?: { start: string; end: string }
  ): Promise<MetricCalculation[]> {
    try {
      const params = new URLSearchParams({
        tenantId,
        metricIds: metricIds.join(','),
        ...(clientId && { clientId }),
        ...(dateRange && {
          startDate: dateRange.start,
          endDate: dateRange.end
        })
      });

  const response = await fetch(`${this.baseUrl}/wealth-management/calculations?${params}`, { credentials: 'include' });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data.calculations || [];
    } catch (error) {
      devError('Error fetching metric calculations:', error);
      return [];
    }
  }

  /**
   * Get metric metadata by ID
   */
  async getMetricById(tenantId: string, metricId: string): Promise<WealthManagementMetric | null> {
    try {
  const response = await fetch(`${this.baseUrl}/wealth-management/metrics/${metricId}?tenantId=${tenantId}`, { credentials: 'include' });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return data.metric || null;
    } catch (error) {
      devError('Error fetching metric by ID:', error);
      return null;
    }
  }

  /**
   * Refresh metric calculations
   */
  async refreshMetrics(tenantId: string, metricIds?: string[]): Promise<boolean> {
    try {
      const body = {
        tenantId,
        ...(metricIds && { metricIds })
      };

      const response = await fetch(`${this.baseUrl}/wealth-management/refresh`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });

      return response.ok;
    } catch (error) {
      devError('Error refreshing metrics:', error);
      return false;
    }
  }
}

// Export singleton instance
export const wealthManagementService = new WealthManagementService();
export default wealthManagementService;
