import React, { useState, useEffect as _useEffect, useMemo as _useMemo } from 'react';
import { Code, Zap, Box, Settings as _Settings, Plus, Trash2, Eye as _Eye, Link2, Database, RefreshCw, Filter, Copy, AlertCircle } from 'lucide-react';
import { useCustomComponents } from '../../hooks/useCustomComponents';
import { useTenant } from '../../contexts/TenantContext';
import { useNotification } from '../../hooks/useNotification';
import styles from './CustomComponentManager.module.css';
import { devDebug } from '../../utils/devLogger';

// Types for Custom Components
export interface CustomComponent {
  id: string;
  name: string;
  type: 'web_component' | 'iframe' | 'api_integration' | 'custom_widget' | 'chart' | 'custom_code';
  config: ComponentConfig;
  events: ComponentEvent[];
  filters: ComponentFilter[];
  createdAt?: string;
  updatedAt?: string;
  datasourceId?: string;
  tenantId?: string;
}

export interface ComponentConfig {
  url?: string;
  apiEndpoint?: string;
  htmlTemplate?: string;
  jsCode?: string;
  cssCode?: string;
  dataSource?: string;
  refreshInterval?: number;
  width?: string;
  height?: string;
  tagName?: string;
}

export interface ComponentEvent {
  id: string;
  eventName: string;
  action: 'refresh' | 'filter' | 'navigate' | 'custom';
  targetComponentId?: string;
  customScript?: string;
}

export interface ComponentFilter {
  id: string;
  field: string;
  operator: string;
  listenToComponent?: string;
}

// Workday Custom Component Types
const CUSTOM_COMPONENT_TYPES = [
  {
    type: 'web_component',
    label: 'Web Component',
    icon: Box,
    description: 'Embed reusable web components (React, Vue, Angular)',
    color: '#3b82f6'
  },
  {
    type: 'iframe',
    label: 'iFrame Embed',
    icon: Link2,
    description: 'Embed external web pages or applications',
    color: '#8b5cf6'
  },
  {
    type: 'api_integration',
    label: 'API Integration',
    icon: Database,
    description: 'Fetch and display data from external APIs',
    color: '#10b981'
  },
  {
    type: 'custom_widget',
    label: 'Custom Widget',
    icon: Zap,
    description: 'Build custom visualizations with D3.js, Chart.js',
    color: '#f59e0b'
  },
  {
    type: 'chart',
    label: 'Interactive Chart',
    icon: RefreshCw,
    description: 'Create charts with cross-filtering capabilities',
    color: '#ef4444'
  },
  {
    type: 'custom_code',
    label: 'Custom Code',
    icon: Code,
    description: 'Write HTML/CSS/JavaScript directly',
    color: '#6366f1'
  }
];

// Custom Component Configurator
const CustomComponentConfigurator: React.FC<{
  component: CustomComponent;
  onUpdate: (component: CustomComponent) => void;
  onDelete: () => void;
  availableComponents: CustomComponent[];
  isLoading?: boolean;
}> = ({ component, onUpdate, onDelete, availableComponents, isLoading = false }) => {
  const [activeTab, setActiveTab] = useState<'config' | 'events' | 'filters'>('config');

  const componentType = CUSTOM_COMPONENT_TYPES.find(t => t.type === component.type);
  const Icon = componentType?.icon || Box;

  return (
    <div className={styles.componentCard} style={{
      borderColor: componentType?.color + '40',
      opacity: isLoading ? 0.6 : 1,
      pointerEvents: isLoading ? 'none' : 'auto'
    }}>
      {/* Header */}
      <div className={styles.cardHeader} style={{ background: `${componentType?.color}15` }}>
        <div className={styles.headerLeft}>
          <div className={styles.iconBox} style={{ background: 'white' }}>
            <Icon size={20} style={{ color: componentType?.color }} />
          </div>
          <div>
            <input
              type="text"
              value={component.name}
              onChange={(e) => onUpdate({ ...component, name: e.target.value })}
              className={styles.componentName}
              placeholder="Component Name"
            />
            <div className={styles.componentType}>
              {componentType?.label}
            </div>
          </div>
        </div>
        <button
          onClick={onDelete}
          className={styles.deleteBtn}
          title="Delete component"
        >
          <Trash2 size={16} />
        </button>
      </div>

      {/* Tabs */}
      <div className={styles.tabsContainer}>
        {['config', 'events', 'filters'].map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab as any)}
            className={`${styles.tabButton} ${activeTab === tab ? styles.tabActive : ''}`}
            style={{
              borderBottomColor: activeTab === tab ? componentType?.color : 'transparent',
              color: activeTab === tab ? componentType?.color : '#6b7280'
            }}
          >
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className={styles.tabContent}>
        {/* Configuration Tab */}
        {activeTab === 'config' && (
          <ConfigurationTab component={component} onUpdate={onUpdate} />
        )}

        {/* Events Tab */}
        {activeTab === 'events' && (
          <EventsTab 
            component={component} 
            onUpdate={onUpdate} 
            availableComponents={availableComponents}
          />
        )}

        {/* Filters Tab */}
        {activeTab === 'filters' && (
          <FiltersTab 
            component={component} 
            onUpdate={onUpdate} 
            availableComponents={availableComponents}
          />
        )}
      </div>
    </div>
  );
};

// Configuration Tab Component
const ConfigurationTab: React.FC<{
  component: CustomComponent;
  onUpdate: (component: CustomComponent) => void;
}> = ({ component, onUpdate }) => {
  return (
    <div className={styles.configFields}>
      {/* Web Component */}
      {component.type === 'web_component' && (
        <>
          <div className={styles.formGroup}>
            <label>Component URL</label>
            <input
              type="text"
              value={component.config.url || ''}
              onChange={(e) => onUpdate({
                ...component,
                config: { ...component.config, url: e.target.value }
              })}
              placeholder="https://cdn.example.com/my-component.js"
              className={styles.input}
            />
            <div className={styles.helper}>URL to your web component bundle (ES Module)</div>
          </div>
          <div className={styles.formGroup}>
            <label>Custom Tag Name</label>
            <input
              type="text"
              value={component.config.tagName || ''}
              onChange={(e) => onUpdate({
                ...component,
                config: { ...component.config, tagName: e.target.value }
              })}
              placeholder="my-custom-component"
              className={styles.input}
            />
          </div>
        </>
      )}

      {/* iFrame */}
      {component.type === 'iframe' && (
        <>
          <div className={styles.formGroup}>
            <label>iFrame URL</label>
            <input
              type="text"
              value={component.config.url || ''}
              onChange={(e) => onUpdate({
                ...component,
                config: { ...component.config, url: e.target.value }
              })}
              placeholder="https://external-app.com/dashboard"
              className={styles.input}
            />
          </div>
          <div className={styles.gridForm}>
            <div className={styles.formGroup}>
              <label>Width</label>
              <input
                type="text"
                value={component.config.width || '100%'}
                onChange={(e) => onUpdate({
                  ...component,
                  config: { ...component.config, width: e.target.value }
                })}
                placeholder="100%"
                className={styles.input}
              />
            </div>
            <div className={styles.formGroup}>
              <label>Height</label>
              <input
                type="text"
                value={component.config.height || '600px'}
                onChange={(e) => onUpdate({
                  ...component,
                  config: { ...component.config, height: e.target.value }
                })}
                placeholder="600px"
                className={styles.input}
              />
            </div>
          </div>
        </>
      )}

      {/* API Integration */}
      {component.type === 'api_integration' && (
        <>
          <div className={styles.formGroup}>
            <label>API Endpoint</label>
            <input
              type="text"
              value={component.config.apiEndpoint || ''}
              onChange={(e) => onUpdate({
                ...component,
                config: { ...component.config, apiEndpoint: e.target.value }
              })}
              placeholder="https://api.example.com/v1/data"
              className={styles.input}
            />
          </div>
          <div className={styles.formGroup}>
            <label htmlFor={`refresh-${component.id}`}>Refresh Interval (seconds)</label>
            <input
              id={`refresh-${component.id}`}
              type="number"
              value={component.config.refreshInterval || 30}
              onChange={(e) => onUpdate({
                ...component,
                config: { ...component.config, refreshInterval: parseInt(e.target.value) }
              })}
              min="0"
              placeholder="30"
              title="Refresh Interval in seconds"
              className={styles.input}
            />
            <div className={styles.helper}>Set to 0 for no auto-refresh</div>
          </div>
        </>
      )}

      {/* Custom Code */}
      {component.type === 'custom_code' && (
        <>
          <div className={styles.formGroup}>
            <label>HTML Template</label>
            <textarea
              value={component.config.htmlTemplate || ''}
              onChange={(e) => onUpdate({
                ...component,
                config: { ...component.config, htmlTemplate: e.target.value }
              })}
              rows={6}
              placeholder="<div id='my-component'>\n  <!-- Your HTML here -->\n</div>"
              className={styles.textarea}
            />
          </div>
          <div className={styles.formGroup}>
            <label>JavaScript</label>
            <textarea
              value={component.config.jsCode || ''}
              onChange={(e) => onUpdate({
                ...component,
                config: { ...component.config, jsCode: e.target.value }
              })}
              rows={8}
              placeholder="// Your JavaScript code here\n// Access Workday API: window.WorkdayAPI\n// Emit events: window.emitEvent('filter', data)"
              className={`${styles.textarea} ${styles.codeEditor}`}
            />
          </div>
          <div className={styles.formGroup}>
            <label>CSS Styles</label>
            <textarea
              value={component.config.cssCode || ''}
              onChange={(e) => onUpdate({
                ...component,
                config: { ...component.config, cssCode: e.target.value }
              })}
              rows={4}
              placeholder="/* Your CSS styles here */"
              className={styles.textarea}
            />
          </div>
        </>
      )}

      {/* Chart/Widget */}
      {(component.type === 'chart' || component.type === 'custom_widget') && (
        <>
          <div className={styles.formGroup}>
            <label>Data Source (API or BO Query)</label>
            <input
              type="text"
              value={component.config.dataSource || ''}
              onChange={(e) => onUpdate({
                ...component,
                config: { ...component.config, dataSource: e.target.value }
              })}
              placeholder="BO:Customer.orders or API:/api/analytics"
              className={styles.input}
            />
          </div>
          <div className={styles.infoBox}>
            <div className={styles.infoTitle}>💡 Chart Configuration</div>
            <div className={styles.infoText}>
              This component supports cross-filtering with other charts. Configure filters in the "Filters" tab.
            </div>
          </div>
        </>
      )}
    </div>
  );
};

// Events Tab Component
const EventsTab: React.FC<{
  component: CustomComponent;
  onUpdate: (component: CustomComponent) => void;
  availableComponents: CustomComponent[];
}> = ({ component, onUpdate, availableComponents }) => {
  return (
    <div className={styles.eventsContainer}>
      <div className={styles.tabHeader}>
        <div>
          <div className={styles.tabTitle}>Component Events</div>
          <div className={styles.tabDescription}>Define what happens when this component fires events</div>
        </div>
        <button
          onClick={() => {
            const newEvent: ComponentEvent = {
              id: `event_${Date.now()}`,
              eventName: 'onClick',
              action: 'filter'
            };
            onUpdate({
              ...component,
              events: [...component.events, newEvent]
            });
          }}
          className={styles.primaryBtn}
        >
          <Plus size={16} />
          Add Event
        </button>
      </div>

      {component.events.length === 0 ? (
        <div className={styles.emptyState}>
          No events configured
        </div>
      ) : (
        <div className={styles.eventsList}>
          {component.events.map((event, idx) => (
            <div key={event.id} className={styles.eventItem}>
              <div className={styles.eventGrid}>
                <div className={styles.formGroup}>
                  <label>Event Name</label>
                  <input
                    type="text"
                    value={event.eventName}
                    onChange={(e) => {
                      const updated = [...component.events];
                      updated[idx] = { ...event, eventName: e.target.value };
                      onUpdate({ ...component, events: updated });
                    }}
                    placeholder="onClick, onFilter"
                    className={styles.input}
                  />
                </div>
                <div className={styles.formGroup}>
                  <label>Action</label>
                  <select
                    value={event.action}
                    title="Select action for this event"
                    onChange={(e) => {
                      const updated = [...component.events];
                      updated[idx] = { ...event, action: e.target.value as any };
                      onUpdate({ ...component, events: updated });
                    }}
                    className={styles.input}
                  >
                    <option value="refresh">Refresh</option>
                    <option value="filter">Filter</option>
                    <option value="navigate">Navigate</option>
                    <option value="custom">Custom Script</option>
                  </select>
                </div>
                <div className={styles.formGroup}>
                  <label>Target Component</label>
                  <select
                    value={event.targetComponentId || ''}
                    onChange={(e) => {
                      const updated = [...component.events];
                      updated[idx] = { ...event, targetComponentId: e.target.value };
                      onUpdate({ ...component, events: updated });
                    }}
                    title="Select target component"
                    className={styles.input}
                  >
                    <option value="">Select target...</option>
                    {availableComponents
                      .filter(c => c.id !== component.id)
                      .map(c => (
                        <option key={c.id} value={c.id}>{c.name}</option>
                      ))}
                  </select>
                </div>
                <button
                  onClick={() => {
                    onUpdate({
                      ...component,
                      events: component.events.filter((_, i) => i !== idx)
                    });
                  }}
                  title="Delete event"
                  className={styles.dangerBtn}
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className={styles.exampleBox}>
        <div className={styles.exampleTitle}>
          <Zap size={16} style={{ display: 'inline', marginRight: '0.5rem' }} />
          Event Examples
        </div>
        <div className={styles.exampleText}>
          • <strong>onClick → Filter:</strong> When user clicks chart bar, filter related list<br/>
          • <strong>onHover → Refresh:</strong> When hovering, refresh API data<br/>
          • <strong>onSelect → Navigate:</strong> Navigate to detail page with context
        </div>
      </div>
    </div>
  );
};

// Filters Tab Component
const FiltersTab: React.FC<{
  component: CustomComponent;
  onUpdate: (component: CustomComponent) => void;
  availableComponents: CustomComponent[];
}> = ({ component, onUpdate, availableComponents }) => {
  return (
    <div className={styles.filtersContainer}>
      <div className={styles.tabHeader}>
        <div>
          <div className={styles.tabTitle}>Cross-Filtering Configuration</div>
          <div className={styles.tabDescription}>Define how this component responds to filters from other components</div>
        </div>
        <button
          onClick={() => {
            const newFilter: ComponentFilter = {
              id: `filter_${Date.now()}`,
              field: '',
              operator: 'equals'
            };
            onUpdate({
              ...component,
              filters: [...component.filters, newFilter]
            });
          }}
          className={styles.secondaryBtn}
        >
          <Filter size={16} />
          Add Filter
        </button>
      </div>

      {component.filters.length === 0 ? (
        <div className={styles.emptyState}>
          No filters configured - this component won't respond to cross-filtering
        </div>
      ) : (
        <div className={styles.filtersList}>
          {component.filters.map((filter, idx) => (
            <div key={filter.id} className={styles.filterItem}>
              <div className={styles.filterGrid}>
                <div className={styles.formGroup}>
                  <label>Field to Filter</label>
                  <input
                    type="text"
                    value={filter.field}
                    onChange={(e) => {
                      const updated = [...component.filters];
                      updated[idx] = { ...filter, field: e.target.value };
                      onUpdate({ ...component, filters: updated });
                    }}
                    placeholder="customer_id, category"
                    className={styles.input}
                  />
                </div>
                <div className={styles.formGroup}>
                  <label>Operator</label>
                  <select
                    title="Select operator for this filter"
                    value={filter.operator}
                    onChange={(e) => {
                      const updated = [...component.filters];
                      updated[idx] = { ...filter, operator: e.target.value };
                      onUpdate({ ...component, filters: updated });
                    }}
                    className={styles.input}
                  >
                    <option value="equals">Equals</option>
                    <option value="contains">Contains</option>
                    <option value="in">In List</option>
                    <option value="between">Between</option>
                  </select>
                </div>
                <div className={styles.formGroup}>
                  <label>Listen to Component</label>
                  <select
                    value={filter.listenToComponent || ''}
                    onChange={(e) => {
                      const updated = [...component.filters];
                      updated[idx] = { ...filter, listenToComponent: e.target.value };
                      onUpdate({ ...component, filters: updated });
                    }}
                    title="Select component to listen to"
                    className={styles.input}
                  >
                    <option value="">All components...</option>
                    {availableComponents
                      .filter(c => c.id !== component.id)
                      .map(c => (
                        <option key={c.id} value={c.id}>{c.name}</option>
                      ))}
                  </select>
                </div>
                <button
                  onClick={() => {
                    onUpdate({
                      ...component,
                      filters: component.filters.filter((_, i) => i !== idx)
                    });
                  }}
                  title="Delete filter"
                  className={styles.dangerBtn}
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// Main Component
export const CustomComponentManager: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const { 
    components, 
    loading, 
    addComponent, 
    updateComponent, 
    deleteComponent,
    saveComponent: _saveComponent 
  } = useCustomComponents();

  const [showCode, setShowCode] = useState(false);
  const [_selectedComponent, _setSelectedComponent] = useState<string | null>(null);

  // Check for tenant scope
  const hasTenantScope = tenant && datasource;

  if (!hasTenantScope) {
    return (
      <div className={styles.container}>
        <div className={styles.scopeWarning}>
          <AlertCircle size={24} />
          <div>
            <h3>Select a Tenant & Datasource</h3>
            <p>Custom components require a tenant and datasource scope to load. Use the tenant picker in the header.</p>
          </div>
        </div>
      </div>
    );
  }

  const generateCodeExample = (component: CustomComponent) => {
    switch (component.type) {
      case 'web_component':
        return `<!-- Web Component Integration -->
<script type="module" src="${component.config.url || 'https://cdn.example.com/component.js'}"></script>
<${component.config.tagName || 'my-custom-component'} 
  data-workday-context="true"
  data-bo-id="{{businessObject.id}}"
  data-record-id="{{record.id}}">
</${component.config.tagName || 'my-custom-component'}>

<script>
  // Component can access Workday API
  const component = document.querySelector('${component.config.tagName || 'my-custom-component'}');
  component.addEventListener('filter-changed', (e) => {
    window.WorkdayAPI.emitEvent('filter', {
      field: e.detail.field,
      value: e.detail.value
    });
  });
</script>`;

      case 'iframe':
        return `<!-- iFrame Integration -->
<iframe 
  src="${component.config.url || 'https://external-app.com/dashboard'}"
  width="${component.config.width || '100%'}"
  height="${component.config.height || '600px'}"
  frameborder="0"
  id="external-app-frame">
</iframe>

<script>
  // Post messages to iframe
  const iframe = document.getElementById('external-app-frame');
  window.WorkdayAPI.onFilter((filterData) => {
    iframe.contentWindow.postMessage({
      type: 'WORKDAY_FILTER',
      data: filterData
    }, '*');
  });

  // Receive messages from iframe
  window.addEventListener('message', (event) => {
    if (event.data.type === 'FILTER_APPLIED') {
      window.WorkdayAPI.emitEvent('filter', event.data.filter);
    }
  });
</script>`;

      case 'api_integration':
        return `// API Integration Code
const API_ENDPOINT = '${component.config.apiEndpoint || 'https://api.example.com/v1/data'}';
const REFRESH_INTERVAL = ${component.config.refreshInterval || 30} * 1000;

async function fetchData(filters = {}) {
  const url = new URL(API_ENDPOINT);
  Object.entries(filters).forEach(([key, value]) => {
    url.searchParams.append(key, value);
  });

  const response = await fetch(url, {
    headers: {
      'Authorization': 'Bearer ' + window.WorkdayAPI.getAuthToken(),
      'Content-Type': 'application/json'
    }
  });

  const data = await response.json();
  renderData(data);
}

// Auto-refresh
setInterval(() => fetchData(), REFRESH_INTERVAL);

// Listen to cross-filters
window.WorkdayAPI.onFilter((filterData) => {
  fetchData(filterData);
});

// Initial load
fetchData();`;

      case 'custom_code':
        return `<!-- Custom Component HTML -->
${component.config.htmlTemplate || '<div id="custom-component"></div>'}

<style>
${component.config.cssCode || '/* Your CSS */'}
</style>

<script>
${component.config.jsCode || `
// Access Workday API
const api = window.WorkdayAPI;

// Get current business object data
const boData = api.getBusinessObjectData();

// Listen for filter events
api.onFilter((filterData) => {
  devDebug('Filter applied:', filterData);
  updateComponent(filterData);
});

// Emit filter event to other components
function applyFilter(field, value) {
  api.emitEvent('filter', { field, value });
}

// Refresh data
api.onRefresh(() => {
  refreshComponent();
});
`}
</script>`;

      case 'chart':
        return `// Interactive Chart with Cross-Filtering
import Chart from 'chart.js/auto';

const ctx = document.getElementById('myChart').getContext('2d');
const chart = new Chart(ctx, {
  type: 'bar',
  data: {
    labels: [],
    datasets: [{
      label: 'Sales',
      data: [],
      backgroundColor: 'rgba(59, 130, 246, 0.5)'
    }]
  },
  options: {
    onClick: (event, elements) => {
      if (elements.length > 0) {
        const index = elements[0].index;
        const label = chart.data.labels[index];
        
        // Emit filter event to other components
        window.WorkdayAPI.emitEvent('filter', {
          field: 'category',
          value: label
        });
      }
    }
  }
});

// Load data from data source
async function loadChartData(filters = {}) {
  const data = await fetch('${component.config.dataSource || 'API:/api/analytics'}', {
    method: 'POST',
    body: JSON.stringify(filters)
  }).then(r => r.json());
  
  chart.data.labels = data.labels;
  chart.data.datasets[0].data = data.values;
  chart.update();
}

// Listen for filters from other components
window.WorkdayAPI.onFilter((filterData) => {
  loadChartData(filterData);
});

// Initial load
loadChartData();`;

      default:
        return '// No code example available';
    }
  };

  return (
    <div className={styles.page}>
      <div className={styles.container}>
        {/* Header */}
        <div className={styles.header}>
          <div>
            <h1>Custom Component Manager</h1>
            <p>Build custom visualizations, integrations, and interactive components for your datasource</p>
          </div>
          <button
            onClick={() => setShowCode(!showCode)}
            className={styles.headerBtn}
          >
            <Code size={20} />
            {showCode ? 'Hide' : 'Show'} Integration Code
          </button>
        </div>

        {/* Info Banner */}
        <div className={styles.infoBanner}>
          <div className={styles.infoTitle}>✨ Workday Custom Component Features</div>
          <div className={styles.infoContent}>
            • <strong>Web Components:</strong> Use any framework (React, Vue, Angular, Web Components)<br/>
            • <strong>Event System:</strong> Cross-component communication with filters, refresh, navigation<br/>
            • <strong>API Access:</strong> Full access to Workday API, business object data, and user context<br/>
            • <strong>Cross-Filtering:</strong> Click chart bar → filter related lists automatically<br/>
            • <strong>Real-time Updates:</strong> Auto-refresh with configurable intervals<br/>
            • <strong>Custom Code:</strong> Write HTML/CSS/JavaScript directly in the platform
          </div>
        </div>

        {/* Add Component Palette */}
        <div className={styles.paletteCard}>
          <h2>Add Custom Component</h2>
          <div className={styles.palette}>
            {CUSTOM_COMPONENT_TYPES.map(type => {
              const Icon = type.icon;
              return (
                <button
                  key={type.type}
                  onClick={() => addComponent(type.type)}
                  className={styles.paletteItem}
                  onMouseOver={(e) => {
                    (e.currentTarget as HTMLElement).style.borderColor = type.color;
                    (e.currentTarget as HTMLElement).style.background = `${type.color}10`;
                  }}
                  onMouseOut={(e) => {
                    (e.currentTarget as HTMLElement).style.borderColor = '#e5e7eb';
                    (e.currentTarget as HTMLElement).style.background = 'white';
                  }}
                >
                  <div className={styles.paletteIcon} style={{ background: `${type.color}20` }}>
                    <Icon size={20} style={{ color: type.color }} />
                  </div>
                  <span className={styles.paletteLabel}>{type.label}</span>
                  <p className={styles.paletteDesc}>{type.description}</p>
                </button>
              );
            })}
          </div>
        </div>

        {/* Component List */}
        <div className={styles.componentList}>
          <h2>Custom Components ({components.length})</h2>
          {components.length === 0 ? (
            <div className={styles.emptyCard}>
              <Box size={48} style={{ color: '#d1d5db', margin: '0 auto 1rem' }} />
              <p>No custom components yet</p>
              <p className={styles.emptyHint}>Click a component type above to get started</p>
            </div>
          ) : (
            components.map(component => (
              <CustomComponentConfigurator
                key={component.id}
                component={component}
                onUpdate={(updated) => updateComponent(component.id, updated)}
                onDelete={() => deleteComponent(component.id)}
                availableComponents={components.filter(c => c.id !== component.id)}
                isLoading={loading}
              />
            ))
          )}
        </div>

        {/* Integration Code Examples */}
        {showCode && components.length > 0 && (
          <div className={styles.codeExamplesCard}>
            <h2>Integration Code Examples</h2>
            {components.map(component => (
              <div key={component.id} className={styles.codeExample}>
                <div className={styles.codeHeader}>
                  <Code size={16} style={{ color: '#6366f1' }} />
                  <span>{component.name}</span>
                  <button
                    onClick={() => {
                      navigator.clipboard.writeText(generateCodeExample(component));
                      const notification = useNotification();
                      notification.success('Code copied to clipboard!');
                    }}
                    className={styles.copyBtn}
                  >
                    <Copy size={12} />
                    Copy
                  </button>
                </div>
                <pre className={styles.codeBlock}>
                  {generateCodeExample(component)}
                </pre>
              </div>
            ))}
          </div>
        )}

        {/* API Reference */}
        <div className={styles.apiReferenceCard}>
          <h2>Workday Component API Reference</h2>
          <div className={styles.apiGrid}>
            {[
              {
                method: 'window.WorkdayAPI.getBusinessObjectData()',
                desc: 'Get current business object data and context'
              },
              {
                method: 'window.WorkdayAPI.emitEvent(type, data)',
                desc: 'Emit events to other components (filter, refresh, navigate)'
              },
              {
                method: 'window.WorkdayAPI.onFilter(callback)',
                desc: 'Listen for filter events from other components'
              },
              {
                method: 'window.WorkdayAPI.onRefresh(callback)',
                desc: 'Listen for refresh events'
              },
              {
                method: 'window.WorkdayAPI.getAuthToken()',
                desc: 'Get authentication token for API calls'
              },
              {
                method: 'window.WorkdayAPI.navigate(url, context)',
                desc: 'Navigate to other pages with context'
              },
              {
                method: 'window.WorkdayAPI.showNotification(msg)',
                desc: 'Display user notifications'
              },
              {
                method: 'window.WorkdayAPI.queryBusinessObject(bo, filter)',
                desc: 'Query business objects directly'
              }
            ].map((api, idx) => (
              <div key={idx} className={styles.apiItem}>
                <code className={styles.apiMethod}>{api.method}</code>
                <div className={styles.apiDesc}>{api.desc}</div>
              </div>
            ))}
          </div>
        </div>

        {/* Cross-Filtering Example */}
        <div className={styles.exampleFlowCard}>
          <h3>💡 Cross-Filtering Example Flow</h3>
          <div className={styles.flowText}>
            <strong>Scenario:</strong> Sales Dashboard with Chart and Related Orders List<br/><br/>
            
            1. User clicks on "West Region" bar in the chart<br/>
            2. Chart component emits: <code className={styles.inline}>
              WorkdayAPI.emitEvent('filter', &#123;region: 'West'&#125;)
            </code><br/>
            3. Orders list listens for filter and automatically refreshes<br/>
            4. Orders list now shows only West region orders<br/>
            5. User can click "Clear Filter" button to reset<br/><br/>
            
            <strong>Result:</strong> Seamless, interactive dashboard like Tableau or Power BI!
          </div>
        </div>
      </div>
    </div>
  );
};

export default CustomComponentManager;