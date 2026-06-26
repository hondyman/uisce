/**
 * Metrics Console - Edit Metric Page
 */

 
import { useParams, useNavigate } from 'react-router-dom';
import MetricForm from '../components/MetricForm';
import { useMetric, useUpdateMetric } from '../hooks/useMetricsConsole';
import { UpdateMetricRequest } from '../types/metrics-console';

export default function MetricEditPage() {
  const { metricId } = useParams<{ metricId: string }>();
  const navigate = useNavigate();
  const { data: metric, isLoading: metricsLoading } = useMetric(metricId);
  const { mutate: updateMetric, isPending } = useUpdateMetric(metricId!);

  const handleSubmit = (data: UpdateMetricRequest) => {
    updateMetric(data, {
      onSuccess: () => {
        navigate(`/metrics/${metricId}`);
      },
    });
  };

  if (metricsLoading) return <div className="p-8 text-center">Loading...</div>;
  if (!metric) return <div className="p-8 text-center text-red-500">Metric not found</div>;

  return (
    <div className="p-8 max-w-4xl mx-auto">
      <h1 className="text-4xl font-black text-gray-900 dark:text-white mb-8">
        Edit Metric: {metric.display_name || metric.name}
      </h1>
      <MetricForm 
        initial={metric}
        onSubmit={handleSubmit}
        isLoading={isPending}
        onCancel={() => navigate(`/metrics/${metricId}`)}
      />
    </div>
  );
}
