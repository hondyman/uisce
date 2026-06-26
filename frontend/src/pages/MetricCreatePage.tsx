/**
 * Metrics Console - Create Metric Page
 */

 
import { useNavigate } from 'react-router-dom';
import MetricForm from '../components/MetricForm';
import { useCreateMetric } from '../hooks/useMetricsConsole';
import { CreateMetricRequest } from '../types/metrics-console';

export default function MetricCreatePage() {
  const navigate = useNavigate();
  const { mutate: createMetric, isPending } = useCreateMetric();

  const handleSubmit = (data: CreateMetricRequest) => {
    createMetric(data, {
      onSuccess: (metric: any) => {
        navigate(`/metrics/${metric.metric_id}`);
      },
    });
  };

  return (
    <div className="p-8 max-w-4xl mx-auto">
      <h1 className="text-4xl font-black text-gray-900 dark:text-white mb-8">Create New Metric</h1>
      <MetricForm 
        onSubmit={handleSubmit}
        isLoading={isPending}
        onCancel={() => navigate('/metrics')}
      />
    </div>
  );
}
