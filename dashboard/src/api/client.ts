import axios, { AxiosInstance, AxiosError } from 'axios'
import { 
  SLAComplianceTrend, 
  ConflictResolutionTrend, 
  ChainExecutionStats, 
  ChainHealthReport, 
  ChainPrediction,
  Chain,
  DashboardFilters,
  ApiError
} from '../types'

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080'

class ApiClient {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      }
    })

    // Add auth token to requests if available
    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('auth_token')
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }
      return config
    })

    // Handle errors globally
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError<ApiError>) => {
        console.error('API Error:', error.response?.data?.message || error.message)
        return Promise.reject(error)
      }
    )
  }

  // SLA Analytics
  async getSLAComplianceTrends(tenantId: string, limit = 30): Promise<SLAComplianceTrend[]> {
    const response = await this.client.get('/admin/ops/analytics/sla-trends', {
      params: { tenant_id: tenantId, limit }
    })
    return response.data
  }

  async getConflictResolutionTrend(tenantId: string, periodStart: string): Promise<ConflictResolutionTrend> {
    const response = await this.client.get('/admin/ops/analytics/conflict-trends', {
      params: { tenant_id: tenantId, period_start: periodStart }
    })
    return response.data
  }

  // Chain Statistics
  async getChainExecutionStats(chainId: string): Promise<ChainExecutionStats> {
    const response = await this.client.get(`/admin/ops/chains/${chainId}/stats`)
    return response.data
  }

  async getChainHealth(chainId: string): Promise<ChainHealthReport> {
    const response = await this.client.get(`/admin/ops/chains/${chainId}/health`)
    return response.data
  }

  // Predictions
  async getChainPredictions(tenantId: string): Promise<ChainPrediction[]> {
    const response = await this.client.get('/admin/ops/chains/predictions', {
      params: { tenant_id: tenantId }
    })
    return response.data
  }

  // Search and Filter
  async searchChains(tenantId: string, query: string, limit = 50): Promise<Chain[]> {
    const response = await this.client.get('/admin/ops/chains/search', {
      params: { tenant_id: tenantId, q: query, limit }
    })
    return response.data
  }

  async filterChains(filters: DashboardFilters): Promise<Chain[]> {
    const response = await this.client.post('/admin/ops/chains/filter', filters)
    return response.data
  }

  // Batch Operations
  async batchResolveConflicts(
    tenantId: string,
    conflictIds: string[],
    rule: 'priority' | 'first_win' | 'serial_execute'
  ): Promise<{ batch_id: string }> {
    const response = await this.client.post('/admin/ops/batch/conflicts/resolve', {
      tenant_id: tenantId,
      conflict_ids: conflictIds,
      resolution_rule: rule
    })
    return response.data
  }

  async getBatchOperation(batchId: string): Promise<any> {
    const response = await this.client.get(`/admin/ops/batch/conflicts/${batchId}`)
    return response.data
  }

  // Reports
  async getScheduledReports(tenantId: string): Promise<any[]> {
    const response = await this.client.get('/admin/ops/reports', {
      params: { tenant_id: tenantId }
    })
    return response.data
  }

  async generateReport(tenantId: string, reportType: string, startDate: string, endDate: string): Promise<any> {
    const response = await this.client.post('/admin/ops/reports/generate', {
      tenant_id: tenantId,
      report_type: reportType,
      start_date: startDate,
      end_date: endDate
    })
    return response.data
  }
}

export const apiClient = new ApiClient()
