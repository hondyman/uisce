import { useState, useEffect } from 'react';
import { useNotification } from '../../hooks/useNotification';
import { devError } from '../../utils/devLogger';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogTrigger } from '@/components/ui/dialog';
import ModalHeader from '../ModalHeader';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
// TabsTrigger intentionally removed - not used
import {
  AlertTriangle, TrendingUp, TrendingDown, Play, Plus, Search, Filter,
  Calendar, Shield, CheckCircle, Clock as _Clock, Target as _Target,
  ArrowUpRight, ArrowDownRight, Minus, GitBranch as _GitBranch, Database as _Database, Users as _Users,
  FileText as _FileText, Hash as _Hash, Eye as _Eye
} from 'lucide-react';
import { PoPMetricForm } from './PoPMetricForm';
import { useNotification } from '../../hooks/useNotification';
import { PoPChart } from './PoPChart';

// Contract Schema Types (aligned with period_over_period.schema.json)
interface PoPContract {
  node_id: string;
  node_type: string;
  name: string;
  description: string;
  version: string;
  base_metric: string;
  period: 'day' | 'week' | 'month' | 'quarter' | 'year';
  comparison: 'day_ago' | 'week_ago' | 'month_ago' | 'quarter_ago' | 'year_ago';
  formula: string;
  dimensions: string[];
  time_dimension: string;
  granularity: 'day' | 'week' | 'month' | 'quarter' | 'year';
  tags: {
    domain?: string;
    category?: string;
    [key: string]: string | undefined;
  };
  status: 'active' | 'draft' | 'deprecated';
  last_updated: string;
  owner: string;
  steward_group: string;
  lineage: {
    upstream_sources: string[];
    downstream_consumers: string[];
  };
  data_quality_contract: {
    null_threshold_pct: number;
    latency_minutes: number;
    completeness_pct: number;
  };
  sla: {
    refresh_frequency: string;
    max_delay_minutes: number;
  };
  anomaly_detection: {
    method: 'zscore' | 'iqr' | 'prophet' | 'custom';
    threshold: number;
    enabled: boolean;
  };
  golden_path: boolean;
  schema_hash: string;
}

interface PoPMetricWithContract extends PoPContract {
  // Runtime data
  id: string;
  display_name: string;
  current_value?: number;
  previous_value?: number;
  delta?: number;
  percent_change?: number;
  period_start: string;
  period_end: string;
  last_computed_at: string;
  has_anomalies: boolean;
  anomaly_count: number;
  computation_status: 'idle' | 'computing' | 'completed' | 'error';
}

export type { PoPMetricWithContract, PoPContract };

interface PoPMetricExplorerProps {
  metrics: PoPMetricWithContract[];
  onRefresh: () => void;
  onContractUpdate?: (contract: PoPContract) => void;
}

type PoPPeriod = 'day' | 'week' | 'month' | 'quarter' | 'year';
type DeltaView = 'percentage' | 'absolute' | 'both';

export const PoPMetricExplorer: React.FC<PoPMetricExplorerProps> = ({ metrics, onRefresh, onContractUpdate: _onContractUpdate }) => {
  const [filteredMetrics, setFilteredMetrics] = useState<PoPMetricWithContract[]>(metrics);
  const [searchTerm, setSearchTerm] = useState('');
  const [domainFilter, setDomainFilter] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [selectedMetric, setSelectedMetric] = useState<PoPMetricWithContract | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [computingMetrics, setComputingMetrics] = useState<Set<string>>(new Set());

  // New state for PoP toggles and deltas
  const [selectedPeriod, setSelectedPeriod] = useState<PoPContract['period']>('day');
  const [deltaView, setDeltaView] = useState<DeltaView>('percentage');
  const [showGoldenPathOnly, setShowGoldenPathOnly] = useState(false);
  const [sortBy, setSortBy] = useState<'name' | 'change' | 'anomalies' | 'updated'>('name');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');

  // Get unique values for filters (using tags as proxy for domain/category)
  const domains = [...new Set(metrics.map(m => m.tags?.domain).filter(Boolean))];
  const categories = [...new Set(metrics.map(m => m.tags?.category).filter(Boolean))];
  const statuses = [...new Set(metrics.map(m => m.status))];

  useEffect(() => {
    let filtered = metrics;

    if (searchTerm) {
      filtered = filtered.filter(metric =>
        metric.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        metric.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
        metric.node_id.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }

    if (domainFilter) {
      filtered = filtered.filter(metric =>
        metric.tags?.domain === domainFilter
      );
    }

    if (categoryFilter) {
      filtered = filtered.filter(metric =>
        metric.tags?.category === categoryFilter
      );
    }

    if (statusFilter) {
      filtered = filtered.filter(metric => metric.status === statusFilter);
    }

    if (showGoldenPathOnly) {
      filtered = filtered.filter(metric => metric.golden_path);
    }

    // Sort metrics
    filtered.sort((a, b) => {
      let aValue: any, bValue: any;

      switch (sortBy) {
        case 'change':
          aValue = a.percent_change || 0;
          bValue = b.percent_change || 0;
          break;
        case 'anomalies':
          aValue = a.anomaly_count || 0;
          bValue = b.anomaly_count || 0;
          break;
        case 'updated':
          aValue = new Date(a.last_computed_at).getTime();
          bValue = new Date(b.last_computed_at).getTime();
          break;
        default:
          aValue = a.name.toLowerCase();
          bValue = b.name.toLowerCase();
      }

      if (sortOrder === 'asc') {
        return aValue > bValue ? 1 : -1;
      } else {
        return aValue < bValue ? 1 : -1;
      }
    });

    setFilteredMetrics(filtered);
  }, [metrics, searchTerm, domainFilter, categoryFilter, statusFilter, showGoldenPathOnly, sortBy, sortOrder]);

  const handleComputeMetric = async (metricId: string) => {
    setComputingMetrics(prev => new Set(prev).add(metricId));

    try {
      const response = await fetch(`/api/pop/metrics/${metricId}/compute`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) throw new Error('Failed to compute metric');

      onRefresh();
    } catch (error) {
      try { devError('Error computing metric:', error); } catch {}
    } finally {
      setComputingMetrics(prev => {
        const newSet = new Set(prev);
        newSet.delete(metricId);
        return newSet;
      });
    }
  };

  const handleBulkCompute = async () => {
      const notification = useNotification();
      notification.success(`Computed ${result.success_count} metrics successfully`);
      notification.error('Failed to compute metrics');
    try {
      const response = await fetch('/api/pop/compute-all', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) throw new Error('Failed to compute all metrics');

      const result = await response.json();
      const notification = useNotification();
      notification.success(`Computed ${result.success_count} metrics successfully`);
      onRefresh();
    } catch (error) {
      try { devError('Error computing all metrics:', error); } catch {}
      const notification = useNotification();
      notification.error('Failed to compute metrics');
    }
  };

  const formatValue = (value?: number) => {
    if (value === undefined || value === null) return 'N/A';
    return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
  };

  const formatPercentChange = (change?: number) => {
    if (change === undefined || change === null) return null;

    const isPositive = change >= 0;
    const color = isPositive ? 'text-green-600' : 'text-red-600';
    const icon = isPositive ? <TrendingUp className="w-3 h-3" /> : <TrendingDown className="w-3 h-3" />;

    return (
      <div className={`flex items-center ${color}`}>
        {icon}
        <span className="ml-1">{Math.abs(change).toFixed(1)}%</span>
      </div>
    );
  };

  const renderDeltaCell = (metric: PoPMetricWithContract, view: DeltaView) => {
    const percentChange = metric.percent_change;
    const absoluteChange = metric.delta;

    if (view === 'percentage') {
      return formatPercentChange(percentChange);
    } else if (view === 'absolute') {
      return (
        <div className="font-mono text-sm">
          {formatValue(absoluteChange)}
        </div>
      );
    } else {
      return (
        <div className="space-y-1">
          {formatPercentChange(percentChange)}
          <div className="font-mono text-xs text-gray-500">
            {formatValue(absoluteChange)}
          </div>
        </div>
      );
    }
  };

  const renderTrendIndicator = (metric: PoPMetricWithContract) => {
    const change = metric.percent_change;
    if (change === undefined || change === null) {
      return <Minus className="w-4 h-4 text-gray-400" />;
    }

    if (change > 0) {
      return <ArrowUpRight className="w-4 h-4 text-green-500" />;
    } else if (change < 0) {
      return <ArrowDownRight className="w-4 h-4 text-red-500" />;
    } else {
      return <Minus className="w-4 h-4 text-gray-400" />;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header and Controls */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">PoP Metrics Explorer</h2>
          <p className="text-gray-600">Browse and manage period-over-period metrics with governance</p>
        </div>
        <div className="flex items-center space-x-2">
          <Button onClick={handleBulkCompute} variant="outline">
            <Play className="w-4 h-4 mr-2" />
            Compute All
          </Button>
          <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="w-4 h-4 mr-2" />
                New Metric
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
              <ModalHeader title="Create New PoP Metric" onClose={() => setShowCreateDialog(false)} />
              <PoPMetricForm
                onSubmit={(_data) => {
                  setShowCreateDialog(false);
                  onRefresh();
                }}
                onCancel={() => setShowCreateDialog(false)}
              />
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* PoP Period Toggles */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <Calendar className="w-4 h-4 mr-2" />
            Period-over-Period Analysis
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <span className="text-sm font-medium">Period:</span>
              <div className="flex space-x-1">
                {(['day', 'week', 'month', 'quarter', 'year'] as PoPPeriod[]).map((period) => (
                  <Button
                    key={period}
                    variant={selectedPeriod === period ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setSelectedPeriod(period)}
                    className="capitalize"
                  >
                    {period}
                  </Button>
                ))}
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <span className="text-sm font-medium">Delta View:</span>
              <Select value={deltaView} onValueChange={(value: DeltaView) => setDeltaView(value)}>
                <SelectTrigger className="w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="percentage">Percentage</SelectItem>
                  <SelectItem value="absolute">Absolute</SelectItem>
                  <SelectItem value="both">Both</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Enhanced Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <Filter className="w-4 h-4 mr-2" />
            Filters & Sorting
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
            <div className="relative">
              <Search className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
              <Input
                placeholder="Search metrics..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-9"
              />
            </div>

            <Select value={domainFilter} onValueChange={setDomainFilter}>
              <SelectTrigger>
                <SelectValue placeholder="All Domains" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">All Domains</SelectItem>
                {domains.map(domain => (
                  <SelectItem key={domain} value={domain}>{domain}</SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select value={categoryFilter} onValueChange={setCategoryFilter}>
              <SelectTrigger>
                <SelectValue placeholder="All Categories" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">All Categories</SelectItem>
                {categories.map(category => (
                  <SelectItem key={category} value={category}>{category}</SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger>
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">All Statuses</SelectItem>
                {statuses.map(status => (
                  <SelectItem key={status} value={status}>
                    <Badge variant={status === 'active' ? 'default' : 'secondary'}>
                      {status}
                    </Badge>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="golden-path"
                checked={showGoldenPathOnly}
                onChange={(e) => setShowGoldenPathOnly(e.target.checked)}
                className="rounded"
              />
              <label htmlFor="golden-path" className="text-sm flex items-center">
                <Shield className="w-3 h-3 mr-1" />
                Golden Path Only
              </label>
            </div>

            <div className="flex items-center space-x-2">
              <Select value={sortBy} onValueChange={(value: any) => setSortBy(value)}>
                <SelectTrigger className="w-24">
                  <SelectValue placeholder="Sort by" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="name">Name</SelectItem>
                  <SelectItem value="change">Change</SelectItem>
                  <SelectItem value="anomalies">Anomalies</SelectItem>
                  <SelectItem value="updated">Updated</SelectItem>
                </SelectContent>
              </Select>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')}
              >
                {sortOrder === 'asc' ? '↑' : '↓'}
              </Button>
            </div>
          </div>

          <div className="mt-4 flex justify-end">
            <Button
              variant="outline"
              onClick={() => {
                setSearchTerm('');
                setDomainFilter('');
                setCategoryFilter('');
                setStatusFilter('');
                setShowGoldenPathOnly(false);
                setSortBy('name');
                setSortOrder('asc');
              }}
            >
              Clear All Filters
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Enhanced Metrics Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>
              Metrics ({filteredMetrics.length} of {metrics.length})
            </span>
            <div className="flex items-center space-x-2">
              <Badge variant="outline" className="text-xs">
                Period: {selectedPeriod}
              </Badge>
              <Badge variant="secondary" className="text-xs">
                View: {deltaView}
              </Badge>
            </div>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Metric</TableHead>
                <TableHead>Domain</TableHead>
                <TableHead>Current Value</TableHead>
                <TableHead>Delta ({selectedPeriod})</TableHead>
                <TableHead>Trend</TableHead>
                <TableHead>Governance</TableHead>
                <TableHead>Anomalies</TableHead>
                <TableHead>Last Updated</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredMetrics.map((metric) => (
                <TableRow key={metric.node_id}>
                  <TableCell>
                    <div>
                      <div className="font-medium flex items-center">
                        {metric.name}
                        {metric.golden_path && (
                          <span title="Golden Path">
                            <Shield className="w-3 h-3 ml-1 text-yellow-500" />
                          </span>
                        )}
                      </div>
                      <div className="text-sm text-gray-500">{metric.description}</div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center space-x-2">
                      <Badge variant="outline">{metric.tags?.domain || 'N/A'}</Badge>
                      <Badge variant="secondary">{metric.tags?.category || 'N/A'}</Badge>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="font-mono text-lg">
                      {formatValue(metric.current_value)}
                    </div>
                  </TableCell>
                  <TableCell>
                    {renderDeltaCell(metric, deltaView)}
                  </TableCell>
                  <TableCell>
                    {renderTrendIndicator(metric)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center space-x-1">
                      <Badge variant={metric.status === 'active' ? 'default' : 'secondary'} className="text-xs">
                        {metric.status}
                      </Badge>
                      {metric.golden_path && (
                        <span title="Golden Path Approved">
                          <CheckCircle className="w-3 h-3 text-green-500" />
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    {metric.anomaly_detection?.enabled && metric.has_anomalies ? (
                      <div className="flex items-center text-red-600">
                        <AlertTriangle className="w-4 h-4 mr-1" />
                        <span className="font-medium">{metric.anomaly_count}</span>
                      </div>
                    ) : (
                      <span className="text-gray-400 text-sm">None</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="text-sm text-gray-500">
                      {new Date(metric.last_computed_at).toLocaleDateString()}
                    </div>
                    <div className="text-xs text-gray-400">
                      {new Date(metric.last_computed_at).toLocaleTimeString()}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center space-x-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleComputeMetric(metric.node_id)}
                        disabled={computingMetrics.has(metric.node_id)}
                      >
                        {computingMetrics.has(metric.node_id) ? (
                          <div className="animate-spin rounded-full h-3 w-3 border-b border-current mr-1" />
                        ) : (
                          <Play className="w-3 h-3 mr-1" />
                        )}
                        Compute
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => setSelectedMetric(metric)}
                      >
                        View
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Metric Detail Dialog */}
      {selectedMetric && (
        <Dialog open={!!selectedMetric} onOpenChange={() => setSelectedMetric(null)}>
          <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto">
            <ModalHeader title={selectedMetric.name} onClose={() => setSelectedMetric(null)} />
            <div className="space-y-6">
              {/* Metric Details */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <h4 className="font-medium mb-2">Basic Information</h4>
                  <dl className="space-y-1 text-sm">
                    <div><dt className="inline font-medium">Name:</dt> <dd className="inline ml-2">{selectedMetric.name}</dd></div>
                    <div><dt className="inline font-medium">Description:</dt> <dd className="inline ml-2">{selectedMetric.description}</dd></div>
                    <div><dt className="inline font-medium">Type:</dt> <dd className="inline ml-2">{selectedMetric.node_type}</dd></div>
                    <div><dt className="inline font-medium">Granularity:</dt> <dd className="inline ml-2">{selectedMetric.granularity}</dd></div>
                    <div><dt className="inline font-medium">Owner:</dt> <dd className="inline ml-2">{selectedMetric.owner}</dd></div>
                    <div><dt className="inline font-medium">Steward Group:</dt> <dd className="inline ml-2">{selectedMetric.steward_group}</dd></div>
                  </dl>
                </div>
                <div>
                  <h4 className="font-medium mb-2">Current Values</h4>
                  <dl className="space-y-1 text-sm">
                    <div><dt className="inline font-medium">Current Value:</dt> <dd className="inline ml-2 font-mono">{formatValue(selectedMetric.current_value)}</dd></div>
                    <div><dt className="inline font-medium">Previous Value:</dt> <dd className="inline ml-2 font-mono">{formatValue(selectedMetric.previous_value)}</dd></div>
                    <div><dt className="inline font-medium">Delta:</dt> <dd className="inline ml-2 font-mono">{formatValue(selectedMetric.delta)}</dd></div>
                    <div><dt className="inline font-medium">Percent Change:</dt> <dd className="inline ml-2">{formatPercentChange(selectedMetric.percent_change)}</dd></div>
                    <div><dt className="inline font-medium">Period:</dt> <dd className="inline ml-2">{selectedMetric.period_start} to {selectedMetric.period_end}</dd></div>
                    <div><dt className="inline font-medium">Last Computed:</dt> <dd className="inline ml-2">{new Date(selectedMetric.last_computed_at).toLocaleString()}</dd></div>
                  </dl>
                </div>
              </div>

              {/* SLA Information */}
              <div>
                <h4 className="font-medium mb-2">SLA & Quality</h4>
                <div className="grid grid-cols-3 gap-4">
                  <div className="p-3 border rounded-lg">
                    <div className="text-sm text-gray-500">Freshness SLA</div>
                    <div className="text-lg font-medium">{selectedMetric.sla?.max_delay_minutes} minutes</div>
                  </div>
                  <div className="p-3 border rounded-lg">
                    <div className="text-sm text-gray-500">Golden Path</div>
                    <div className="text-lg font-medium">
                      <Badge variant={selectedMetric.golden_path ? 'default' : 'secondary'}>
                        {selectedMetric.golden_path ? 'Yes' : 'No'}
                      </Badge>
                    </div>
                  </div>
                  <div className="p-3 border rounded-lg">
                    <div className="text-sm text-gray-500">Anomalies</div>
                    <div className="text-lg font-medium">
                      {selectedMetric.has_anomalies ? (
                        <span className="text-red-600">{selectedMetric.anomaly_count}</span>
                      ) : (
                        <span className="text-green-600">None</span>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              {/* Chart */}
              <div>
                <h4 className="font-medium mb-2">Trend Chart</h4>
                <PoPChart metrics={[selectedMetric]} height={300} />
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
};
