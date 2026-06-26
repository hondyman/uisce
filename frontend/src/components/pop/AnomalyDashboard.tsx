import { useState, useEffect } from 'react';
import { devError } from '../../utils/devLogger';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent } from '@/components/ui/dialog';
import ModalHeader from '../ModalHeader';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { AlertTriangle, CheckCircle, XCircle, Clock, Filter, RefreshCw, Settings } from 'lucide-react';
import { AnomalyDetectionConfig } from './AnomalyDetectionConfig';
import { AnomalyDetectionResult, PoPAnomaly } from '@/types';
import { useNotification } from '../../hooks/useNotification';
import { Download, TrendingUp, History, Zap } from 'lucide-react';
import { useAuthFetch } from '../../utils/authFetch';

interface AnomalySummary {
  domain: string;
  category: string;
  severity: string;
  anomaly_type: string;
  anomaly_count: number;
  latest_detection: string;
  affected_metrics: string[];
}

interface PoPMetricWithLatest {
  id: string;
  name: string;
  display_name: string;
  domain: string;
  category: string;
  current_value?: number;
  previous_value?: number;
  delta?: number;
  percent_change?: number;
  period_start: string;
  period_end: string;
  last_computed_at: string;
  has_anomalies: boolean;
  anomaly_count: number;
}

interface AnomalyDashboardProps {
  anomalySummary: AnomalySummary[];
  metrics: PoPMetricWithLatest[];
  onRefresh: () => void;
}

interface AnomalyDetail extends PoPAnomaly {}

export const AnomalyDashboard: React.FC<AnomalyDashboardProps> = ({
  anomalySummary,
  metrics,
  onRefresh
}) => {
  const { authFetch } = useAuthFetch();
  const [anomalies, setAnomalies] = useState<AnomalyDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const [severityFilter, setSeverityFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [selectedAnomaly, setSelectedAnomaly] = useState<AnomalyDetail | null>(null);
  const [showResolveDialog, setShowResolveDialog] = useState(false);
  const [showDetectionConfig, setShowDetectionConfig] = useState(false);
  const [selectedMetricForDetection, setSelectedMetricForDetection] = useState<string>('');
  const [detectionResults, setDetectionResults] = useState<AnomalyDetectionResult | null>(null);
  const [showBulkOperations, setShowBulkOperations] = useState(false);
  const [selectedMetricsForBulk, setSelectedMetricsForBulk] = useState<string[]>([]);
  const [exportLoading, setExportLoading] = useState(false);
  const [showTrends, setShowTrends] = useState(false);
  const [trendsData, setTrendsData] = useState<any[]>([]);
  const [showExport, setShowExport] = useState(false);
  const [exportFormat, setExportFormat] = useState('csv');
  const [exportStartDate, setExportStartDate] = useState('');
  const [exportEndDate, setExportEndDate] = useState('');

  useEffect(() => {
    fetchAnomalies();
  }, []);

  const notification = useNotification();

  const fetchAnomalies = async () => {
    setLoading(true);
    try {
  const response = await authFetch('/api/pop/anomalies');
  if (!response.ok) throw new Error('Failed to fetch anomalies');

  const data = (response && (response as any).data !== undefined) ? (response as any).data : await (response as any).json?.();
  setAnomalies(data.anomalies || []);
    } catch (error) {
      devError('Error fetching anomalies:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleBulkAnomalyDetection = async () => {
    if (selectedMetricsForBulk.length === 0) {
      notification.error('Please select metrics for bulk detection');
      return;
    }

    try {
  const response = await authFetch('/api/pop/bulk-detect-anomalies', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          metric_ids: selectedMetricsForBulk,
          config: {
            method: 'z_score',
            sensitivity: 0.8,
            window_size: 30,
            min_data_points: 7,
            custom_parameters: { zscore_threshold: 2.5 }
          }
        }),
      });

  if (!response.ok) throw new Error('Failed to run bulk anomaly detection');

  const result = (response && (response as any).data !== undefined) ? (response as any).data : await (response as any).json?.();
      notification.success(`Bulk detection completed! Processed ${result.metrics_processed} metrics, found ${result.total_anomalies} anomalies`);
      onRefresh();
      fetchAnomalies();
      setShowBulkOperations(false);
    } catch (error) {
      devError('Error running bulk anomaly detection:', error);
      notification.error('Failed to run bulk anomaly detection');
    }
  };

  const handleDetectionComplete = (result: AnomalyDetectionResult) => {
    setDetectionResults(result);
    // Add new anomalies to the existing list
    setAnomalies(prev => [...prev, ...result.anomalies]);
    setShowDetectionConfig(false);
    onRefresh();
    fetchAnomalies();
  };

  const handleOpenDetectionConfig = (metricId: string) => {
    setSelectedMetricForDetection(metricId);
    setShowDetectionConfig(true);
  };

  const handleExportAnomalies = async (format: string) => {
    setExportLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('format', format);
      if (severityFilter) params.append('severity', severityFilter);
      if (statusFilter) params.append('status', statusFilter);
      if (exportStartDate) params.append('start_date', exportStartDate);
      if (exportEndDate) params.append('end_date', exportEndDate);

  const response = await authFetch(`/api/pop/export-anomalies?${params}`);
  if (!response.ok) throw new Error('Failed to export anomalies');

  // authFetch may return a Response-like wrapper; try to get blob either via .blob() or data
  const blob = (response && (response as any).blob) ? await (response as any).blob() : await (response as any).json?.();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `anomalies_export.${format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      setShowExport(false);
    } catch (error) {
      devError('Error exporting anomalies:', error);
      notification.error('Failed to export anomalies');
    } finally {
      setExportLoading(false);
    }
  };

  const handleFetchTrends = async () => {
    try {
  const response = await authFetch('/api/pop/anomaly-trends?days=30');
  if (!response.ok) throw new Error('Failed to fetch trends');

  const data = (response && (response as any).data !== undefined) ? (response as any).data : await (response as any).json?.();
  setTrendsData(data.trends || []);
      setShowTrends(true);
    } catch (error) {
      devError('Error fetching trends:', error);
      notification.error('Failed to fetch anomaly trends');
    }
  };

  const handleResolveAnomaly = async (anomalyId: string, resolutionNotes: string) => {
    try {
  const response = await authFetch(`/api/pop/anomalies/${anomalyId}/resolve`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          resolution_notes: resolutionNotes,
          resolved_by: 'current_user' // TODO: Get from auth context
        }),
      });

  if (!response.ok) throw new Error('Failed to resolve anomaly');

      fetchAnomalies();
      setShowResolveDialog(false);
      setSelectedAnomaly(null);
    } catch (error) {
      devError('Error resolving anomaly:', error);
      notification.error('Failed to resolve anomaly');
    }
  };

  const filteredAnomalies = anomalies.filter(anomaly => {
    if (severityFilter && anomaly.severity !== severityFilter) return false;
    if (statusFilter && anomaly.status !== statusFilter) return false;
    return true;
  });

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'high': return 'text-red-600 bg-red-50 border-red-200';
      case 'medium': return 'text-orange-600 bg-orange-50 border-orange-200';
      case 'low': return 'text-yellow-600 bg-yellow-50 border-yellow-200';
      default: return 'text-gray-600 bg-gray-50 border-gray-200';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'resolved': return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'open': return <AlertTriangle className="w-4 h-4 text-red-500" />;
      case 'investigating': return <Clock className="w-4 h-4 text-blue-500" />;
      default: return <XCircle className="w-4 h-4 text-gray-500" />;
    }
  };

  const formatValue = (value?: number) => {
    if (value === undefined || value === null) return 'N/A';
    return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Anomaly Dashboard</h2>
          <p className="text-gray-600">Monitor and manage detected anomalies</p>
        </div>
        <div className="flex items-center space-x-2">
          <Button onClick={fetchAnomalies} variant="outline" disabled={loading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <Button onClick={handleBulkAnomalyDetection}>
            <Zap className="w-4 h-4 mr-2" />
            Bulk Detection
          </Button>
          <Button
            variant="outline"
            onClick={() => setShowBulkOperations(true)}
          >
            <TrendingUp className="w-4 h-4 mr-2" />
            Bulk Operations
          </Button>
          <Button
            variant="outline"
            onClick={handleFetchTrends}
          >
            <History className="w-4 h-4 mr-2" />
            Trends
          </Button>
          <Button
            variant="outline"
            onClick={() => setShowExport(true)}
            disabled={exportLoading}
          >
            <Download className="w-4 h-4 mr-2" />
            Export
          </Button>
          <Button
            variant="outline"
            onClick={() => setShowDetectionConfig(true)}
          >
            <Settings className="w-4 h-4 mr-2" />
            Advanced Detection
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Anomalies</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{anomalies.length}</div>
            <p className="text-xs text-muted-foreground">
              Across all metrics
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Open Anomalies</CardTitle>
            <Clock className="h-4 w-4 text-orange-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-orange-600">
              {anomalies.filter(a => a.status === 'open').length}
            </div>
            <p className="text-xs text-muted-foreground">
              Require attention
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Resolved Today</CardTitle>
            <CheckCircle className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              {anomalies.filter(a =>
                a.status === 'resolved' &&
                new Date(a.resolved_at || '').toDateString() === new Date().toDateString()
              ).length}
            </div>
            <p className="text-xs text-muted-foreground">
              Today's resolutions
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Detection Results */}
      {detectionResults && (
        <Card className="border-green-200 bg-green-50">
          <CardHeader>
            <CardTitle className="text-green-800">Detection Results</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-green-700">
                  Found {detectionResults.count} anomalies using {detectionResults.method_used}
                </p>
                <p className="text-sm text-green-600">
                  Detection completed at {new Date(detectionResults.detection_time).toLocaleString()}
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setDetectionResults(null)}
              >
                Dismiss
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Anomaly Summary by Domain/Category */}
      <Card>
        <CardHeader>
          <CardTitle>Anomaly Summary</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {anomalySummary.map((summary, index) => (
              <div key={index} className="flex items-center justify-between p-4 border rounded-lg">
                <div>
                  <p className="font-medium">{summary.domain} • {summary.category}</p>
                  <p className="text-sm text-gray-500">
                    {summary.anomaly_type} • Latest: {new Date(summary.latest_detection).toLocaleDateString()}
                  </p>
                </div>
                <div className="flex items-center space-x-4">
                  <Badge className={getSeverityColor(summary.severity)}>
                    {summary.severity}
                  </Badge>
                  <div className="text-right">
                    <div className="text-lg font-bold">{summary.anomaly_count}</div>
                    <div className="text-sm text-gray-500">anomalies</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <Filter className="w-4 h-4 mr-2" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Select value={severityFilter} onValueChange={setSeverityFilter}>
              <SelectTrigger>
                <SelectValue placeholder="All Severities" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">All Severities</SelectItem>
                <SelectItem value="high">High</SelectItem>
                <SelectItem value="medium">Medium</SelectItem>
                <SelectItem value="low">Low</SelectItem>
              </SelectContent>
            </Select>

            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger>
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">All Statuses</SelectItem>
                <SelectItem value="open">Open</SelectItem>
                <SelectItem value="investigating">Investigating</SelectItem>
                <SelectItem value="resolved">Resolved</SelectItem>
              </SelectContent>
            </Select>

            <Button
              variant="outline"
              onClick={() => {
                setSeverityFilter('');
                setStatusFilter('');
              }}
            >
              Clear Filters
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Anomalies Table */}
      <Card>
        <CardHeader>
          <CardTitle>
            Anomalies ({filteredAnomalies.length} of {anomalies.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Metric</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Detection Method</TableHead>
                <TableHead>Severity</TableHead>
                <TableHead>Confidence</TableHead>
                <TableHead>Expected vs Actual</TableHead>
                <TableHead>Detected</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredAnomalies.map((anomaly) => {
                const metric = metrics.find(m => m.id === anomaly.metric_id);
                return (
                  <TableRow key={anomaly.id}>
                    <TableCell>
                      <div>
                        <div className="font-medium">{metric?.display_name || 'Unknown'}</div>
                        <div className="text-sm text-gray-500">{metric?.domain} • {metric?.category}</div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{anomaly.anomaly_type}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="text-xs">
                        {anomaly.detection_method.replace(/_/g, ' ')}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge className={getSeverityColor(anomaly.severity)}>
                        {anomaly.severity}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {anomaly.confidence ? (
                        <div className="text-sm">
                          {(anomaly.confidence * 100).toFixed(1)}%
                        </div>
                      ) : (
                        <span className="text-gray-400">N/A</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">
                        <div>Expected: {formatValue(anomaly.expected_value)}</div>
                        <div>Actual: {formatValue(anomaly.actual_value)}</div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm text-gray-500">
                        {new Date(anomaly.detected_at).toLocaleDateString()}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center">
                        {getStatusIcon(anomaly.status)}
                        <span className="ml-2 capitalize">{anomaly.status}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center space-x-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setSelectedAnomaly(anomaly)}
                        >
                          View
                        </Button>
                        {anomaly.status === 'open' && (
                          <Button
                            size="sm"
                            onClick={() => {
                              setSelectedAnomaly(anomaly);
                              setShowResolveDialog(true);
                            }}
                          >
                            Resolve
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Anomaly Detail Dialog */}
      {selectedAnomaly && !showResolveDialog && (
        <Dialog open={!!selectedAnomaly} onOpenChange={() => setSelectedAnomaly(null)}>
          <DialogContent className="max-w-2xl">
            <ModalHeader title="Anomaly Details" onClose={() => setSelectedAnomaly(null)} />

            <div className="space-y-4">
              <div>
                <h4 className="font-medium mb-2">Anomaly Information</h4>
                <dl className="space-y-1 text-sm">
                  <div>
                    <dt className="inline font-medium">Type:</dt>
                    <dd className="inline ml-2">{selectedAnomaly.anomaly_type}</dd>
                  </div>
                  <div>
                    <dt className="inline font-medium">Severity:</dt>
                    <dd className="inline ml-2">
                      <Badge className={getSeverityColor(selectedAnomaly.severity)}>
                        {selectedAnomaly.severity}
                      </Badge>
                    </dd>
                  </div>
                  <div>
                    <dt className="inline font-medium">Status:</dt>
                    <dd className="inline ml-2 capitalize">{selectedAnomaly.status}</dd>
                  </div>
                  <div>
                    <dt className="inline font-medium">Detection Method:</dt>
                    <dd className="inline ml-2">{selectedAnomaly.detection_method.replace(/_/g, ' ')}</dd>
                  </div>
                  <div>
                    <dt className="inline font-medium">Detected At:</dt>
                    <dd className="inline ml-2">{new Date(selectedAnomaly.detected_at).toLocaleString()}</dd>
                  </div>
                </dl>
              </div>

              <div>
                <h4 className="font-medium mb-2">Values</h4>
                <dl className="space-y-1 text-sm">
                  <div>
                    <dt className="inline font-medium">Expected:</dt>
                    <dd className="inline ml-2 font-mono">{formatValue(selectedAnomaly.expected_value)}</dd>
                  </div>
                  <div>
                    <dt className="inline font-medium">Actual:</dt>
                    <dd className="inline ml-2 font-mono">{formatValue(selectedAnomaly.actual_value)}</dd>
                  </div>
                  <div>
                    <dt className="inline font-medium">Z-Score:</dt>
                    <dd className="inline ml-2 font-mono">{selectedAnomaly.z_score?.toFixed(2) || 'N/A'}</dd>
                  </div>
                  <div>
                    <dt className="inline font-medium">Confidence:</dt>
                    <dd className="inline ml-2">{selectedAnomaly.confidence ? `${(selectedAnomaly.confidence * 100).toFixed(1)}%` : 'N/A'}</dd>
                  </div>
                </dl>
              </div>

              <div>
                <h4 className="font-medium mb-2">Detection Parameters</h4>
                <dl className="space-y-1 text-sm">
                  {selectedAnomaly.detection_params && Object.entries(selectedAnomaly.detection_params).map(([key, value]) => (
                    <div key={key}>
                      <dt className="inline font-medium">{key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}:</dt>
                      <dd className="inline ml-2 font-mono">{String(value)}</dd>
                    </div>
                  ))}
                </dl>
              </div>

              {selectedAnomaly.resolution_notes && (
                <div>
                  <h4 className="font-medium mb-2">Resolution Notes</h4>
                  <p className="text-sm text-gray-600 bg-gray-50 p-3 rounded">
                    {selectedAnomaly.resolution_notes}
                  </p>
                </div>
              )}
            </div>

          </DialogContent>
        </Dialog>
      )}

      {/* Advanced Detection Config Dialog */}
      {showDetectionConfig && (
        <Dialog open={showDetectionConfig} onOpenChange={setShowDetectionConfig}>
          <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
            <ModalHeader title="Advanced Anomaly Detection Configuration" onClose={() => setShowDetectionConfig(false)} />
            <div className="space-y-4">
              {selectedMetricForDetection ? (
                <AnomalyDetectionConfig
                  metricId={selectedMetricForDetection}
                  onDetectionComplete={handleDetectionComplete}
                  onRefresh={onRefresh}
                />
              ) : (
                <div className="space-y-4">
                  <p className="text-gray-600">Select a metric to configure anomaly detection:</p>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {metrics.map((metric) => (
                      <Card
                        key={metric.id}
                        className="cursor-pointer hover:bg-gray-50 transition-colors"
                        onClick={() => handleOpenDetectionConfig(metric.id)}
                      >
                        <CardContent className="p-4">
                          <h4 className="font-medium">{metric.display_name}</h4>
                          <p className="text-sm text-gray-500">{metric.domain} • {metric.category}</p>
                          <div className="mt-2 flex items-center justify-between">
                            <span className="text-sm text-gray-600">
                              Current: {formatValue(metric.current_value)}
                            </span>
                            {metric.has_anomalies && (
                              <Badge variant="destructive" className="text-xs">
                                {metric.anomaly_count} anomalies
                              </Badge>
                            )}
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Resolve Anomaly Dialog */}
      {selectedAnomaly && showResolveDialog && (
        <ResolveAnomalyDialog
          anomaly={selectedAnomaly}
          onResolve={handleResolveAnomaly}
          onCancel={() => {
            setShowResolveDialog(false);
            setSelectedAnomaly(null);
          }}
        />
      )}

      {/* Bulk Operations Dialog */}
      {showBulkOperations && (
        <Dialog open={showBulkOperations} onOpenChange={setShowBulkOperations}>
          <DialogContent className="max-w-4xl">
            <ModalHeader title="Bulk Operations" onClose={() => setShowBulkOperations(false)} />
            <div className="space-y-4">
              <div>
                <h4 className="font-medium mb-2">Select Metrics for Bulk Detection</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-h-60 overflow-y-auto">
                  {metrics.map((metric) => (
                    <div key={metric.id} className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        id={metric.id}
                        checked={selectedMetricsForBulk.includes(metric.id)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedMetricsForBulk(prev => [...prev, metric.id]);
                          } else {
                            setSelectedMetricsForBulk(prev => prev.filter(id => id !== metric.id));
                          }
                        }}
                        className="rounded"
                      />
                      <label htmlFor={metric.id} className="text-sm">
                        <div className="font-medium">{metric.display_name}</div>
                        <div className="text-gray-500">{metric.domain} • {metric.category}</div>
                      </label>
                    </div>
                  ))}
                </div>
              </div>
              <div className="flex justify-end space-x-2">
                <Button variant="outline" onClick={() => setShowBulkOperations(false)}>
                  Cancel
                </Button>
                <Button
                  onClick={handleBulkAnomalyDetection}
                  disabled={selectedMetricsForBulk.length === 0}
                >
                  Run Bulk Detection ({selectedMetricsForBulk.length} metrics)
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Trends Dialog */}
      {showTrends && (
        <Dialog open={showTrends} onOpenChange={setShowTrends}>
          <DialogContent className="max-w-6xl">
            <ModalHeader title="Anomaly Trends (Last 30 Days)" onClose={() => setShowTrends(false)} />
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {trendsData.map((trend, index) => (
                  <Card key={index}>
                    <CardContent className="p-4">
                      <div className="text-sm text-gray-500">{trend.date}</div>
                      <div className="text-2xl font-bold">{trend.total_count}</div>
                      <div className="text-xs text-gray-500">Total Anomalies</div>
                      <div className="mt-2 space-y-1">
                        <div className="flex justify-between text-xs">
                          <span>High:</span>
                          <Badge variant="destructive">{trend.severity_high}</Badge>
                        </div>
                        <div className="flex justify-between text-xs">
                          <span>Medium:</span>
                          <Badge variant="secondary">{trend.severity_med}</Badge>
                        </div>
                        <div className="flex justify-between text-xs">
                          <span>Low:</span>
                          <Badge variant="outline">{trend.severity_low}</Badge>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
              <div className="flex justify-end">
                <Button variant="outline" onClick={() => setShowTrends(false)}>
                  Close
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Export Dialog */}
      {showExport && (
        <Dialog open={showExport} onOpenChange={setShowExport}>
          <DialogContent className="max-w-md">
            <ModalHeader title="Export Anomalies" onClose={() => setShowExport(false)} />
            <div className="space-y-4">
              <div>
                <label htmlFor="export-format" className="block text-sm font-medium mb-2">Export Format</label>
                <select
                  id="export-format"
                  value={exportFormat}
                  onChange={(e) => setExportFormat(e.target.value)}
                  className="w-full p-2 border rounded"
                >
                  <option value="csv">CSV</option>
                  <option value="json">JSON</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Date Range</label>
                <div className="grid grid-cols-2 gap-2">
                  <input
                    type="date"
                    id="export-start-date"
                    value={exportStartDate}
                    onChange={(e) => setExportStartDate(e.target.value)}
                    className="p-2 border rounded"
                    placeholder="Start date"
                  />
                  <input
                    type="date"
                    id="export-end-date"
                    value={exportEndDate}
                    onChange={(e) => setExportEndDate(e.target.value)}
                    className="p-2 border rounded"
                    placeholder="End date"
                  />
                </div>
              </div>
              <div className="flex justify-end space-x-2">
                <Button variant="outline" onClick={() => setShowExport(false)}>
                  Cancel
                </Button>
                <Button onClick={() => handleExportAnomalies(exportFormat)}>
                  Export
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}

    </div>
  );
};

// Resolve Anomaly Dialog Component
interface ResolveAnomalyDialogProps {
  anomaly: AnomalyDetail;
  onResolve: (anomalyId: string, resolutionNotes: string) => void;
  onCancel: () => void;
}

const ResolveAnomalyDialog: React.FC<ResolveAnomalyDialogProps> = ({
  anomaly,
  onResolve,
  onCancel
}) => {
  const [resolutionNotes, setResolutionNotes] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onResolve(anomaly.id, resolutionNotes);
  };

  return (
    <Dialog open={true} onOpenChange={onCancel}>
        <DialogContent>
        <ModalHeader title="Resolve Anomaly" onClose={onCancel} />
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              Resolution Notes
            </label>
            <textarea
              value={resolutionNotes}
              onChange={(e) => setResolutionNotes(e.target.value)}
              className="w-full p-3 border border-gray-300 rounded-md"
              rows={4}
              placeholder="Explain how this anomaly was resolved..."
              required
            />
          </div>
          <div className="flex justify-end space-x-2">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit">
              Resolve Anomaly
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};
