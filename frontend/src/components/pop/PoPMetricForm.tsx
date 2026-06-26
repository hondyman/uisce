import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Plus, X, Database, Settings, Clock, Shield } from 'lucide-react';

interface PoPMetricFormProps {
  initialData?: Partial<PoPMetric>;
  onSubmit: (data: PoPMetricFormData) => void;
  onCancel: () => void;
}

interface PoPMetric {
  id: string;
  name: string;
  display_name: string;
  description: string;
  domain: string;
  category: string;
  metric_type: string;
  base_query: string;
  aggregation_function: string;
  date_column: string;
  value_column: string;
  granularity: string;
  comparison_periods: string[];
  owner_user_id: string;
  steward_group: string;
  data_source: string;
  schema_name: string;
  table_name: string;
  sla_freshness_hours: number;
  sla_completeness_threshold: number;
  data_quality_checks: Record<string, any>;
  golden_path: boolean;
}

interface PoPMetricFormData {
  name: string;
  display_name: string;
  description: string;
  domain: string;
  category: string;
  metric_type: string;
  base_query: string;
  aggregation_function: string;
  date_column: string;
  value_column: string;
  granularity: string;
  comparison_periods: string[];
  owner_user_id: string;
  steward_group: string;
  data_source: string;
  schema_name: string;
  table_name: string;
  sla_freshness_hours: number;
  sla_completeness_threshold: number;
  data_quality_checks: Record<string, any>;
  golden_path: boolean;
}

export const PoPMetricForm: React.FC<PoPMetricFormProps> = ({
  initialData,
  onSubmit,
  onCancel
}) => {
  const [formData, setFormData] = useState<PoPMetricFormData>({
    name: initialData?.name || '',
    display_name: initialData?.display_name || '',
    description: initialData?.description || '',
    domain: initialData?.domain || '',
    category: initialData?.category || '',
    metric_type: initialData?.metric_type || 'count',
    base_query: initialData?.base_query || '',
    aggregation_function: initialData?.aggregation_function || 'COUNT',
    date_column: initialData?.date_column || '',
    value_column: initialData?.value_column || '',
    granularity: initialData?.granularity || 'month',
    comparison_periods: initialData?.comparison_periods || ['previous_period'],
    owner_user_id: initialData?.owner_user_id || '',
    steward_group: initialData?.steward_group || '',
    data_source: initialData?.data_source || '',
    schema_name: initialData?.schema_name || '',
    table_name: initialData?.table_name || '',
    sla_freshness_hours: initialData?.sla_freshness_hours || 24,
    sla_completeness_threshold: initialData?.sla_completeness_threshold || 0.95,
    data_quality_checks: initialData?.data_quality_checks || {},
    golden_path: initialData?.golden_path || false,
  });

  const [newComparisonPeriod, setNewComparisonPeriod] = useState('');
  const [activeTab, setActiveTab] = useState('basic');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
  };

  const handleChange = (field: keyof PoPMetricFormData, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const addComparisonPeriod = () => {
    if (newComparisonPeriod && !formData.comparison_periods.includes(newComparisonPeriod)) {
      handleChange('comparison_periods', [...formData.comparison_periods, newComparisonPeriod]);
      setNewComparisonPeriod('');
    }
  };

  const removeComparisonPeriod = (period: string) => {
    handleChange('comparison_periods', formData.comparison_periods.filter(p => p !== period));
  };

  const tabs = [
    { id: 'basic', label: 'Basic Info', icon: Database },
    { id: 'query', label: 'Query Config', icon: Settings },
    { id: 'sla', label: 'SLA & Quality', icon: Clock },
    { id: 'governance', label: 'Governance', icon: Shield },
  ];

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Tab Navigation */}
      <div className="flex space-x-1 bg-gray-100 p-1 rounded-lg">
        {tabs.map((tab) => {
          const Icon = tab.icon;
          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center space-x-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                activeTab === tab.id
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              <Icon className="w-4 h-4" />
              <span>{tab.label}</span>
            </button>
          );
        })}
      </div>

      {/* Basic Info Tab */}
      {activeTab === 'basic' && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Metric Name *</label>
              <Input
                value={formData.name}
                onChange={(e) => handleChange('name', e.target.value)}
                placeholder="e.g., user_registrations"
                title="Metric Name"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Display Name *</label>
              <Input
                value={formData.display_name}
                onChange={(e) => handleChange('display_name', e.target.value)}
                placeholder="e.g., User Registrations"
                title="Display Name"
                required
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Description</label>
            <Textarea
              value={formData.description}
              onChange={(e) => handleChange('description', e.target.value)}
              rows={3}
              placeholder="Describe what this metric measures..."
              title="Description"
            />
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Domain *</label>
              <Select value={formData.domain} onValueChange={(value: string) => handleChange('domain', value)}>
                <SelectTrigger title="Domain">
                  <SelectValue placeholder="Select domain" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="sales">Sales</SelectItem>
                  <SelectItem value="marketing">Marketing</SelectItem>
                  <SelectItem value="product">Product</SelectItem>
                  <SelectItem value="finance">Finance</SelectItem>
                  <SelectItem value="operations">Operations</SelectItem>
                  <SelectItem value="customer_success">Customer Success</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Category *</label>
              <Input
                value={formData.category}
                onChange={(e) => handleChange('category', e.target.value)}
                placeholder="e.g., acquisition"
                title="Category"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Metric Type</label>
              <Select value={formData.metric_type} onValueChange={(value: string) => handleChange('metric_type', value)}>
                <SelectTrigger title="Metric Type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="count">Count</SelectItem>
                  <SelectItem value="sum">Sum</SelectItem>
                  <SelectItem value="average">Average</SelectItem>
                  <SelectItem value="percentage">Percentage</SelectItem>
                  <SelectItem value="ratio">Ratio</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Granularity</label>
              <Select value={formData.granularity} onValueChange={(value: string) => handleChange('granularity', value)}>
                <SelectTrigger title="Granularity">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="day">Daily</SelectItem>
                  <SelectItem value="week">Weekly</SelectItem>
                  <SelectItem value="month">Monthly</SelectItem>
                  <SelectItem value="quarter">Quarterly</SelectItem>
                  <SelectItem value="year">Yearly</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Golden Path</label>
              <div className="flex items-center space-x-2 mt-2">
                <input
                  type="checkbox"
                  checked={formData.golden_path}
                  onChange={(e) => handleChange('golden_path', e.target.checked)}
                  className="rounded"
                  title="Golden Path"
                />
                <span className="text-sm">Mark as golden path metric</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Query Config Tab */}
      {activeTab === 'query' && (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Base Query</CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                value={formData.base_query}
                onChange={(e) => handleChange('base_query', e.target.value)}
                rows={6}
                placeholder={`SELECT * FROM your_table WHERE condition`}
                className="font-mono text-sm"
                title="Base Query"
              />
              <p className="text-sm text-gray-500 mt-2">
                This query will be used as the foundation for computing the metric.
                Use placeholders for date filters.
              </p>
            </CardContent>
          </Card>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Aggregation Function</label>
              <Select value={formData.aggregation_function} onValueChange={(value: string) => handleChange('aggregation_function', value)}>
                <SelectTrigger title="Aggregation Function">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="COUNT">COUNT</SelectItem>
                  <SelectItem value="SUM">SUM</SelectItem>
                  <SelectItem value="AVG">AVG</SelectItem>
                  <SelectItem value="MIN">MIN</SelectItem>
                  <SelectItem value="MAX">MAX</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Value Column</label>
              <Input
                value={formData.value_column}
                onChange={(e) => handleChange('value_column', e.target.value)}
                placeholder="e.g., amount, quantity"
                title="Value Column"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Date Column *</label>
            <Input
              value={formData.date_column}
              onChange={(e) => handleChange('date_column', e.target.value)}
              placeholder="e.g., created_at, event_date"
              title="Date Column"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Comparison Periods</label>
            <div className="space-y-2">
              <div className="flex space-x-2">
                <Input
                  value={newComparisonPeriod}
                  onChange={(e) => setNewComparisonPeriod(e.target.value)}
                  placeholder="e.g., previous_period, same_period_last_year"
                  title="New Comparison Period"
                />
                <Button type="button" onClick={addComparisonPeriod} size="sm">
                  <Plus className="w-4 h-4" />
                </Button>
              </div>
              <div className="flex flex-wrap gap-2">
                {formData.comparison_periods.map((period) => (
                  <Badge key={period} variant="secondary" className="flex items-center space-x-1">
                    <span>{period}</span>
                    <button
                      type="button"
                      onClick={() => removeComparisonPeriod(period)}
                      className="ml-1 hover:bg-gray-300 rounded-full p-0.5"
                      title={`Remove ${period}`}
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* SLA & Quality Tab */}
      {activeTab === 'sla' && (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">SLA Configuration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Freshness SLA (hours)</label>
                  <Input
                    type="number"
                    value={formData.sla_freshness_hours}
                    onChange={(e) => handleChange('sla_freshness_hours', parseInt(e.target.value) || 24)}
                    min="1"
                    title="Freshness SLA Hours"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Completeness Threshold</label>
                  <Input
                    type="number"
                    value={formData.sla_completeness_threshold}
                    onChange={(e) => handleChange('sla_completeness_threshold', parseFloat(e.target.value) || 0.95)}
                    min="0"
                    max="1"
                    step="0.01"
                    title="Completeness Threshold"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    Expected completeness ratio (0.0 - 1.0)
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Data Quality Checks</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">Null Check Column</label>
                    <Input
                      value={formData.data_quality_checks?.null_check_column || ''}
                      onChange={(e) => handleChange('data_quality_checks', {
                        ...formData.data_quality_checks,
                        null_check_column: e.target.value
                      })}
                      placeholder="Column to check for nulls"
                      title="Null Check Column"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Max Null Percentage</label>
                    <Input
                      type="number"
                      value={formData.data_quality_checks?.max_null_percentage || ''}
                      onChange={(e) => handleChange('data_quality_checks', {
                        ...formData.data_quality_checks,
                        max_null_percentage: parseFloat(e.target.value) || 0
                      })}
                      min="0"
                      max="100"
                      step="0.1"
                      title="Max Null Percentage"
                    />
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Governance Tab */}
      {activeTab === 'governance' && (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Ownership & Stewardship</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Owner User ID *</label>
                  <Input
                    value={formData.owner_user_id}
                    onChange={(e) => handleChange('owner_user_id', e.target.value)}
                    placeholder="user@company.com"
                    title="Owner User ID"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Steward Group</label>
                  <Input
                    value={formData.steward_group}
                    onChange={(e) => handleChange('steward_group', e.target.value)}
                    placeholder="data-stewards"
                    title="Steward Group"
                  />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Data Source</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Data Source</label>
                  <Input
                    value={formData.data_source}
                    onChange={(e) => handleChange('data_source', e.target.value)}
                    placeholder="e.g., postgres, snowflake"
                    title="Data Source"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Schema Name</label>
                  <Input
                    value={formData.schema_name}
                    onChange={(e) => handleChange('schema_name', e.target.value)}
                    placeholder="e.g., public, analytics"
                    title="Schema Name"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Table Name</label>
                  <Input
                    value={formData.table_name}
                    onChange={(e) => handleChange('table_name', e.target.value)}
                    placeholder="e.g., users, events"
                    title="Table Name"
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Form Actions */}
      <div className="flex justify-end space-x-3 pt-6 border-t">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">
          {initialData ? 'Update Metric' : 'Create Metric'}
        </Button>
      </div>
    </form>
  );
};
