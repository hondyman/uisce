import { devDebug } from '../../utils/devLogger';
// Local fallback type for CustomComponent to avoid namespace/type collisions in TS config
type CustomComponent = any;

/**
 * Pre-built component templates for common use cases
 * Use these as starting points for new components
 */

export const ComponentTemplates = {
  /**
   * Sales by Region - Interactive Bar Chart
   */
  SalesChart: (): CustomComponent => ({
    id: `comp_${Date.now()}`,
    name: 'Sales by Region - Interactive Chart',
    type: 'chart',
    config: {
      dataSource: 'API:/api/analytics/sales-by-region',
      refreshInterval: 60,
      width: '100%',
      height: '400px'
    },
    events: [
      {
        id: 'evt_1',
        eventName: 'onBarClick',
        action: 'filter',
        customScript: `
          const region = event.data.region;
          window.WorkdayAPI.emitEvent('filter', {
            field: 'region',
            value: region
          });
        `
      }
    ],
    filters: [],
  }),

  /**
   * Order Details List - Responds to cross-filters
   */
  OrdersList: (): CustomComponent => ({
    id: `comp_${Date.now()}`,
    name: 'Order Details - Filterable List',
    type: 'api_integration',
    config: {
      apiEndpoint: 'API:/api/orders',
      refreshInterval: 30,
    },
    events: [],
    filters: [
      {
        id: 'filter_1',
        field: 'region',
        operator: 'equals',
        listenToComponent: 'comp_1' // Reference to Sales Chart
      },
      {
        id: 'filter_2',
        field: 'status',
        operator: 'equals'
      }
    ],
  }),

  /**
   * Real-time Metrics Widget
   */
  MetricsWidget: (): CustomComponent => ({
    id: `comp_${Date.now()}`,
    name: 'Real-time Metrics Widget',
    type: 'custom_widget',
    config: {
      dataSource: 'API:/api/metrics/live',
      refreshInterval: 10,
    },
    events: [
      {
        id: 'evt_alert',
        eventName: 'onThresholdExceeded',
        action: 'custom',
        customScript: `
          if (metric.value > 1000) {
            window.WorkdayAPI.showNotification('Alert: Threshold exceeded!');
            window.WorkdayAPI.emitEvent('filter', {
              field: 'alert_status',
              value: 'critical'
            });
          }
        `
      }
    ],
    filters: [],
  }),

  /**
   * Custom HTML Dashboard
   */
  CustomHTMLDashboard: (): CustomComponent => ({
    id: `comp_${Date.now()}`,
    name: 'Custom HTML Dashboard',
    type: 'custom_code',
    config: {
      htmlTemplate: `
        <div id="dashboard-container" style="padding: 20px;">
          <h2>Performance Dashboard</h2>
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-top: 20px;">
            <div style="border: 1px solid #ddd; padding: 15px; border-radius: 8px;">
              <h3>Revenue</h3>
              <div id="revenue-value" style="font-size: 24px; font-weight: bold; color: #10b981;">$0</div>
            </div>
            <div style="border: 1px solid #ddd; padding: 15px; border-radius: 8px;">
              <h3>Orders</h3>
              <div id="orders-value" style="font-size: 24px; font-weight: bold; color: #3b82f6;">0</div>
            </div>
          </div>
          <button id="refresh-btn" style="margin-top: 20px; padding: 10px 20px; background: #6366f1; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Refresh Data
          </button>
        </div>
      `,
      cssCode: `
        #dashboard-container {
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
          color: #111827;
        }
        
        #dashboard-container h2 {
          margin: 0 0 10px 0;
          font-size: 20px;
        }
        
        #refresh-btn:hover {
          background: #4f46e5;
          transform: translateY(-2px);
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        }
      `,
      jsCode: `
        // Load initial data
        async function loadDashboardData() {
          try {
            const response = await fetch('/api/dashboard/metrics', {
              headers: {
                'Authorization': 'Bearer ' + window.WorkdayAPI.getAuthToken()
              }
            });
            
            if (response.ok) {
              const data = await response.json();
              document.getElementById('revenue-value').textContent = '$' + data.revenue;
              document.getElementById('orders-value').textContent = data.orders;
              
              // Emit event that data was loaded
              window.WorkdayAPI.emitEvent('refresh', {
                dashboard: 'loaded',
                timestamp: Date.now()
              });
            }
          } catch (error) {
            console.error('Error loading dashboard:', error);
            window.WorkdayAPI.showNotification('Error loading data');
          }
        }
        
        // Set up refresh button
        document.getElementById('refresh-btn').addEventListener('click', loadDashboardData);
        
        // Listen for external filters
        window.WorkdayAPI.onFilter((filter) => {
          devDebug('Filter applied to dashboard:', filter);
          loadDashboardData();
        });
        
        // Auto-refresh every 60 seconds
        setInterval(loadDashboardData, 60000);
        
        // Load on mount
        loadDashboardData();
      `
    },
    events: [
      {
        id: 'evt_refresh',
        eventName: 'onDataLoad',
        action: 'refresh'
      }
    ],
    filters: [
      {
        id: 'filter_status',
        field: 'status',
        operator: 'equals'
      }
    ],
  }),

  /**
   * External iFrame App
   */
  ExternalApp: (): CustomComponent => ({
    id: `comp_${Date.now()}`,
    name: 'External Dashboard App',
    type: 'iframe',
    config: {
      url: 'https://external-dashboard.example.com/dashboard',
      width: '100%',
      height: '800px',
    },
    events: [],
    filters: [
      {
        id: 'filter_1',
        field: 'tenant_id',
        operator: 'equals'
      },
      {
        id: 'filter_2',
        field: 'user_id',
        operator: 'equals'
      }
    ],
  }),

  /**
   * Web Component from NPM
   */
  WebComponentChart: (): CustomComponent => ({
    id: `comp_${Date.now()}`,
    name: 'Chart.js Web Component',
    type: 'web_component',
    config: {
      url: 'https://cdn.jsdelivr.net/npm/chart-component@latest/dist/chart-component.js',
      tagName: 'chart-component',
    },
    events: [
      {
        id: 'evt_click',
        eventName: 'datapoint-clicked',
        action: 'filter'
      }
    ],
    filters: [],
  }),

  /**
   * Real-time Data Stream
   */
  RealtimeStream: (): CustomComponent => ({
    id: `comp_${Date.now()}`,
    name: 'Real-time Data Stream',
    type: 'api_integration',
    config: {
      apiEndpoint: 'API:/api/stream/transactions',
      refreshInterval: 5, // Update every 5 seconds
    },
    events: [
      {
        id: 'evt_alert',
        eventName: 'anomaly_detected',
        action: 'custom',
        customScript: `
          if (transaction.amount > threshold) {
            window.WorkdayAPI.showNotification('⚠️ Large transaction: $' + transaction.amount);
          }
        `
      },
      {
        id: 'evt_filter',
        eventName: 'transaction_selected',
        action: 'filter',
        targetComponentId: 'comp_details'
      }
    ],
    filters: [
      {
        id: 'filter_type',
        field: 'transaction_type',
        operator: 'equals'
      }
    ],
  }),

  /**
   * KPI Dashboard
   */
  KPIDashboard: (): CustomComponent => ({
    id: `comp_${Date.now()}`,
    name: 'KPI Dashboard',
    type: 'custom_code',
    config: {
      htmlTemplate: `
        <div id="kpi-dashboard" style="padding: 20px;">
          <h1 style="margin: 0 0 30px 0;">Key Performance Indicators</h1>
          <div id="kpi-grid" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px;">
            <!-- KPIs will be inserted here -->
          </div>
        </div>
      `,
      cssCode: `
        .kpi-card {
          background: white;
          border: 1px solid #e5e7eb;
          border-radius: 8px;
          padding: 20px;
          text-align: center;
          box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
          transition: all 0.3s;
        }
        
        .kpi-card:hover {
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
          transform: translateY(-4px);
          cursor: pointer;
        }
        
        .kpi-label {
          font-size: 12px;
          color: #6b7280;
          margin-bottom: 8px;
          text-transform: uppercase;
          font-weight: 600;
        }
        
        .kpi-value {
          font-size: 32px;
          font-weight: 700;
          color: #111827;
          margin-bottom: 8px;
        }
        
        .kpi-change {
          font-size: 12px;
          font-weight: 600;
        }
        
        .kpi-change.up {
          color: #10b981;
        }
        
        .kpi-change.down {
          color: #ef4444;
        }
      `,
      jsCode: `
        const kpis = [
          { label: 'Revenue', value: 125000, change: 12, unit: '$' },
          { label: 'Orders', value: 3450, change: 8, unit: '' },
          { label: 'Conversion', value: 3.2, change: -1.5, unit: '%' },
          { label: 'Customers', value: 892, change: 5, unit: '' }
        ];
        
        function renderKPIs() {
          const grid = document.getElementById('kpi-grid');
          grid.innerHTML = kpis.map(kpi => \`
            <div class="kpi-card" onclick="handleKPIClick('\${kpi.label}')">
              <div class="kpi-label">\${kpi.label}</div>
              <div class="kpi-value">\${kpi.unit}\${kpi.value}</div>
              <div class="kpi-change \${kpi.change >= 0 ? 'up' : 'down'}">
                \${kpi.change >= 0 ? '↑' : '↓'} \${Math.abs(kpi.change)}% vs last month
              </div>
            </div>
          \`).join('');
        }
        
        function handleKPIClick(label) {
          window.WorkdayAPI.emitEvent('filter', {
            field: 'kpi_type',
            value: label
          });
        }
        
        window.WorkdayAPI.onRefresh(() => {
          renderKPIs();
        });
        
        renderKPIs();
      `
    },
    events: [
      {
        id: 'evt_filter',
        eventName: 'kpi_selected',
        action: 'filter'
      }
    ],
    filters: [],
  }),
};

/**
 * Get template by name
 */
export function getTemplate(name: keyof typeof ComponentTemplates): CustomComponent {
  const templateFn = ComponentTemplates[name];
  if (!templateFn) {
    throw new Error(`Template not found: ${name}`);
  }
  return templateFn();
}

/**
 * List all available templates
 */
export function listTemplates(): Array<{ name: string; description: string }> {
  return [
    { name: 'SalesChart', description: 'Interactive bar chart with region filtering' },
    { name: 'OrdersList', description: 'Orders list that responds to cross-filters' },
    { name: 'MetricsWidget', description: 'Real-time metrics with threshold alerts' },
    { name: 'CustomHTMLDashboard', description: 'Custom HTML/CSS/JS dashboard' },
    { name: 'ExternalApp', description: 'iFrame embed of external application' },
    { name: 'WebComponentChart', description: 'Chart.js web component' },
    { name: 'RealtimeStream', description: 'Real-time transaction stream' },
    { name: 'KPIDashboard', description: 'KPI dashboard with grid layout' },
  ];
}