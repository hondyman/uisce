import React, { useEffect, useState } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

interface Metrics {
    avgCycleTime: number;
    instancesInFlight: number;
    slaBreaches: number;
    completionRate: number;
    stepDwellTime: Array<{ step: string; hours: number }>;
}

export const WorkflowHealthDashboard: React.FC<{ bpKey: string }> = ({ bpKey }) => {
  const [metrics, setMetrics] = useState<Metrics | null>(null);

  useEffect(() => {
    // Poll for metrics
    const fetchMetrics = async () => {
        try {
             // Mock response for visual verification without backend
             const mock: Metrics = {
                 avgCycleTime: 24.5,
                 instancesInFlight: 12,
                 slaBreaches: 2,
                 completionRate: 98.5,
                 stepDwellTime: [
                     { step: "Review", hours: 4 },
                     { step: "Approval", hours: 12 },
                     { step: "Compliance", hours: 8.5 }
                 ]
             };
             // const res = await fetch(`/api/analytics/cycle-time?bp_key=${bpKey}`);
             // const data = await res.json();
             setMetrics(mock); 
        } catch(e) { console.error(e); }
    };
    
    fetchMetrics();
    const poll = setInterval(fetchMetrics, 30000);
    return () => clearInterval(poll);
  }, [bpKey]);

  return (
    <div className="p-6 bg-gray-50 rounded-xl">
      <div className="flex justify-between items-center mb-6">
          <h2 className="text-xl font-bold text-gray-800">Workflow Health: <span className="text-blue-600">{bpKey}</span></h2>
          <span className="text-xs text-gray-500">Live Updates (30s)</span>
      </div>
      
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <MetricCard title="Avg Cycle Time" value={metrics?.avgCycleTime} unit="h" />
        <MetricCard title="In Flight" value={metrics?.instancesInFlight} unit="" highlight={true} />
        <MetricCard title="SLA Breaches" value={metrics?.slaBreaches} unit="" color="text-red-600" />
        <MetricCard title="Completion Rate" value={metrics?.completionRate} unit="%" />
      </div>

      <div className="bg-white p-4 rounded-lg shadow-sm border h-80">
        <h3 className="text-sm font-semibold text-gray-600 mb-4">Step Dwell Time (Avg Hours)</h3>
         {metrics && (
             <ResponsiveContainer width="100%" height="100%">
                <BarChart data={metrics.stepDwellTime}>
                    <XAxis dataKey="step" tick={{fontSize: 12}} />
                    <YAxis tick={{fontSize: 12}} />
                    <Tooltip />
                    <Bar dataKey="hours" fill="#4f46e5" radius={[4, 4, 0, 0]} />
                </BarChart>
             </ResponsiveContainer>
         )}
      </div>
    </div>
  );
};

const MetricCard = ({ title, value, unit, highlight, color }: any) => (
    <div className="bg-white p-4 rounded-lg shadow-sm border flex flex-col">
        <span className="text-xs text-gray-500 uppercase tracking-wide">{title}</span>
        <div className={`text-2xl font-bold mt-1 ${color ? color : 'text-gray-900'} ${highlight ? 'animate-pulse' : ''}`}>
            {value !== undefined ? value : '-'}
            <span className="text-sm text-gray-400 font-normal ml-1">{unit}</span>
        </div>
    </div>
);
