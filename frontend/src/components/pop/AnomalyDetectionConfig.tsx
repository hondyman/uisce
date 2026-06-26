import React, { useState, useEffect } from 'react';
import { devError } from '../../utils/devLogger';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { Badge } from '@/components/ui/badge';
import { Settings, Play, RefreshCw } from 'lucide-react';
import { useNotification } from '../../hooks/useNotification';
import { AnomalyDetectionMethod, DetectionConfig, AnomalyDetectionResult } from '@/types';

interface AnomalyDetectionConfigProps {
  metricId: string;
  onDetectionComplete: (result: AnomalyDetectionResult) => void;
  onRefresh: () => void;
}

export const AnomalyDetectionConfig: React.FC<AnomalyDetectionConfigProps> = ({
  metricId,
  onDetectionComplete,
  onRefresh: _onRefresh
}) => {
  const [methods, setMethods] = useState<AnomalyDetectionMethod[]>([]);
  const [selectedMethod, setSelectedMethod] = useState<string>('');
  const [config, setConfig] = useState<DetectionConfig>({
    method: 'z_score',
    sensitivity: 0.8,
    window_size: 30,
    min_data_points: 7,
    custom_parameters: {}
  });
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const notification = useNotification();

  useEffect(() => {
    fetchDetectionMethods();
  }, []);

  const fetchDetectionMethods = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/pop/anomaly-methods');
      if (!response.ok) throw new Error('Failed to fetch detection methods');

      const data = await response.json();
      setMethods(data.methods || []);
    } catch (error) {
      devError('Error fetching detection methods:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleMethodChange = (methodId: string) => {
    setSelectedMethod(methodId);
    setConfig(prev => ({
      ...prev,
      method: methodId,
      custom_parameters: getDefaultParameters(methodId)
    }));
  };

  const getDefaultParameters = (methodId: string): Record<string, any> => {
    const method = methods.find(m => m.id === methodId);
    if (!method) return {};

    const defaults: Record<string, any> = {};

    switch (methodId) {
      case 'z_score':
        defaults.zscore_threshold = 2.5;
        break;
      case 'iqr':
        defaults.multiplier = 1.5;
        break;
      case 'mad':
        defaults.threshold = 3.0;
        break;
      case 'isolation_forest':
        defaults.contamination = 0.1;
        break;
      case 'prophet':
        defaults.changepoint_prior_scale = 0.05;
        break;
      case 'custom':
        defaults.upper_threshold = 100;
        defaults.lower_threshold = 0;
        break;
    }

    return defaults;
  };

  const handleParameterChange = (param: string, value: any) => {
    setConfig(prev => ({
      ...prev,
      custom_parameters: {
        ...prev.custom_parameters,
        [param]: value
      }
    }));
  };

  const runDetection = async () => {
    setRunning(true);
    try {
      const response = await fetch(`/api/pop/metrics/${metricId}/detect-anomalies`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      });

      if (!response.ok) throw new Error('Failed to run anomaly detection');

      const result: AnomalyDetectionResult = await response.json();
      onDetectionComplete(result);
    } catch (error) {
      devError('Error running anomaly detection:', error);
      notification.error('Failed to run anomaly detection');
    } finally {
      setRunning(false);
    }
  };

  const renderParameterControls = (method: AnomalyDetectionMethod) => {
    const params = method.parameters;

    return (
      <div className="space-y-4">
        {Object.entries(params).map(([key, description]) => (
          <div key={key} className="space-y-2">
            <Label htmlFor={key} className="text-sm font-medium">
              {key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
            </Label>
            <p className="text-xs text-gray-500">{description}</p>

            {key.includes('threshold') || key.includes('multiplier') || key.includes('contamination') ? (
              <Input
                id={key}
                type="number"
                step="0.1"
                value={config.custom_parameters[key] || ''}
                onChange={(e) => handleParameterChange(key, parseFloat(e.target.value))}
                className="w-full"
              />
            ) : key.includes('scale') ? (
              <Input
                id={key}
                type="number"
                step="0.01"
                value={config.custom_parameters[key] || ''}
                onChange={(e) => handleParameterChange(key, parseFloat(e.target.value))}
                className="w-full"
              />
            ) : (
              <Input
                id={key}
                type="number"
                value={config.custom_parameters[key] || ''}
                onChange={(e) => handleParameterChange(key, parseInt(e.target.value))}
                className="w-full"
              />
            )}
          </div>
        ))}
      </div>
    );
  };

  const selectedMethodData = methods.find(m => m.id === selectedMethod);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center">
          <Settings className="w-4 h-4 mr-2" />
          Advanced Anomaly Detection
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Method Selection */}
        <div className="space-y-2">
          <Label htmlFor="method-select">Detection Method</Label>
          <Select value={selectedMethod} onValueChange={handleMethodChange}>
            <SelectTrigger>
              <SelectValue placeholder="Select detection method" />
            </SelectTrigger>
            <SelectContent>
              {methods.map((method) => (
                <SelectItem key={method.id} value={method.id}>
                  <div className="flex items-center space-x-2">
                    <span className="font-medium">{method.name}</span>
                    <Badge variant="outline" className="text-xs">
                      {method.id}
                    </Badge>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {selectedMethodData && (
            <p className="text-sm text-gray-600">{selectedMethodData.description}</p>
          )}
        </div>

        {/* Global Parameters */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="space-y-2">
            <Label htmlFor="sensitivity">Sensitivity</Label>
            <Slider
              id="sensitivity"
              min={0.1}
              max={1.0}
              step={0.1}
              value={[config.sensitivity]}
              onValueChange={(value: number[]) => setConfig(prev => ({ ...prev, sensitivity: value[0] }))}
              className="w-full"
            />
            <div className="text-xs text-gray-500 text-center">
              {config.sensitivity.toFixed(1)}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="window-size">Window Size</Label>
            <Input
              id="window-size"
              type="number"
              min="7"
              max="365"
              value={config.window_size}
              onChange={(e) => setConfig(prev => ({ ...prev, window_size: parseInt(e.target.value) }))}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="min-data">Min Data Points</Label>
            <Input
              id="min-data"
              type="number"
              min="3"
              max="100"
              value={config.min_data_points}
              onChange={(e) => setConfig(prev => ({ ...prev, min_data_points: parseInt(e.target.value) }))}
            />
          </div>
        </div>

        {/* Method-Specific Parameters */}
        {selectedMethodData && (
          <div className="space-y-4">
            <h4 className="font-medium text-gray-900">Method Parameters</h4>
            {renderParameterControls(selectedMethodData)}
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex items-center justify-between pt-4 border-t">
          <Button
            variant="outline"
            onClick={fetchDetectionMethods}
            disabled={loading}
          >
            <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Refresh Methods
          </Button>

          <Button
            onClick={runDetection}
            disabled={running || !selectedMethod}
          >
            <Play className={`w-4 h-4 mr-2 ${running ? 'animate-pulse' : ''}`} />
            {running ? 'Running Detection...' : 'Run Detection'}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};
