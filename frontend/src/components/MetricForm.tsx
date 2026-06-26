/**
 * Metrics Console - Metric Form Component
 * Shared form for creating and editing metrics
 */

import React, { useMemo, useEffect, useState } from 'react';
import { MetricRegistry, CreateMetricRequest, UpdateMetricRequest } from '../types/metrics-console';
import DynamicEntityForm from './DynamicEntityForm';

interface MetricFormProps {
  initial?: Partial<MetricRegistry>;
  onSubmit: (data: CreateMetricRequest | UpdateMetricRequest) => void;
  isLoading?: boolean;
  onCancel?: () => void;
}

export default function MetricForm({ initial, onSubmit, isLoading, onCancel }: MetricFormProps) {
  // In a full implementation, we would fetch the "Metric" entity definition from the registry
  // and pass it to DynamicEntityForm. For now, we'll simulate that by adapting the props.
  
  const [open, setOpen] = useState(true);

  const handleSubmit = async (payload: any) => {
    // Transform dynamic form payload back to MetricRequest if needed
    // For now, assume payload matches
    onSubmit(payload.data);
    return Promise.resolve();
  };

  // We can reuse DynamicEntityForm if we treat "Metric" as just another entity type.
  // However, DynamicEntityForm is a Dialog. We might want an inline form here.
  // If we want to strictly follow "metadata-first", we should use the DynamicEntityForm
  // or extract its rendering logic.
  
  // Let's use DynamicEntityForm as a controlled component for now, 
  // but since it renders a Dialog, we might need to adjust it or wrap it.
  // Given the current architecture, let's render a simplified dynamic form here 
  // that mimics the metadata-driven approach but inline.

  // TODO: Fetch this from /api/metrics/definitions/schema or similar
  const metricSchema = {
    fields: [
      { key: 'name', label: 'Name', type: 'text', required: true },
      { key: 'display_name', label: 'Display Name', type: 'text' },
      { key: 'domain', label: 'Domain', type: 'text', required: true },
      { key: 'granularity', label: 'Granularity', type: 'select', options: ['day', 'month', 'quarter', 'year'], required: true },
      { key: 'aggregation_function', label: 'Aggregation', type: 'select', options: ['SUM', 'AVG', 'COUNT', 'MAX', 'MIN'], required: true },
      { key: 'base_query', label: 'Base Query', type: 'text' }, // Should be code editor
      { key: 'owner', label: 'Owner', type: 'text' },
    ]
  };

  const [formData, setFormData] = useState<any>(initial || {});

  const handleChange = (key: string, value: any) => {
    setFormData((prev: any) => ({ ...prev, [key]: value }));
  };

  return (
    <div className="space-y-6">
      <section className="bg-white dark:bg-gray-900 rounded-xl p-6 border border-gray-200 dark:border-gray-800">
        <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-4">Metric Configuration</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {metricSchema.fields.map((field) => (
            <div key={field.key} className={field.key === 'base_query' ? 'md:col-span-2' : ''}>
              <label className="block text-sm font-medium text-gray-900 dark:text-white mb-2">
                {field.label} {field.required && '*'}
              </label>
              {field.type === 'select' ? (
                <select
                  value={formData[field.key] || ''}
                  onChange={(e) => handleChange(field.key, e.target.value)}
                  className="w-full px-4 h-11 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary/50"
                >
                  <option value="">Select...</option>
                  {field.options?.map((opt) => (
                    <option key={opt} value={opt}>{opt}</option>
                  ))}
                </select>
              ) : field.key === 'base_query' ? (
                <textarea
                  value={formData[field.key] || ''}
                  onChange={(e) => handleChange(field.key, e.target.value)}
                  rows={4}
                  className="w-full px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary/50 font-mono text-sm"
                />
              ) : (
                <input
                  type={field.type}
                  value={formData[field.key] || ''}
                  onChange={(e) => handleChange(field.key, e.target.value)}
                  className="w-full px-4 h-11 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary/50"
                />
              )}
            </div>
          ))}
        </div>
      </section>

      <div className="flex justify-end gap-3">
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className="px-6 h-11 rounded-lg border border-gray-300 dark:border-gray-700 text-gray-900 dark:text-white hover:bg-gray-50 dark:hover:bg-gray-800 font-medium text-sm"
          >
            Cancel
          </button>
        )}
        <button
          type="button"
          onClick={() => onSubmit(formData)}
          disabled={isLoading}
          className="px-6 h-11 rounded-lg bg-primary text-white hover:bg-primary/90 font-medium text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isLoading ? 'Saving...' : 'Save Metric'}
        </button>
      </div>
    </div>
  );
}
