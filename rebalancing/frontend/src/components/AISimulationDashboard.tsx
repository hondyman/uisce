import React, { useState, useCallback } from 'react';
import { useSubscription } from '@apollo/client';
import { gql } from '@apollo/client';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { PlayCircle, Loader, BarChart2 } from 'lucide-react';

const PORTFOLIOS_SUB = gql`
  subscription Portfolios {
    portfolios(order_by: {aum: desc}) {
      id
      name
    }
  }
`;

const AISimulationDashboard = () => {
  const { data: portfoliosData } = useSubscription(PORTFOLIOS_SUB);
  const [selectedPortfolio, setSelectedPortfolio] = useState('');
  const [startDate, setStartDate] = useState('2022-01-01');
  const [endDate, setEndDate] = useState('2022-12-31');
  const [status, setStatus] = useState('idle');
  const [result, setResult] = useState(null);
  const [workflowId, setWorkflowId] = useState('');

  const pollWorkflowStatus = useCallback(async (wfId) => {
    const interval = setInterval(async () => {
      try {
        const response = await fetch(`/api/simulation/${wfId}`);
        const data = await response.json();

        if (data.status === 'Completed') {
          setResult(data.result);
          setStatus('completed');
          clearInterval(interval);
        } else if (data.status !== 'Running') {
          setStatus('failed');
          clearInterval(interval);
        }
      } catch (error) {
        console.error("Polling error:", error);
        setStatus('failed');
        clearInterval(interval);
      }
    }, 2000);
  }, []);

  const handleRunSimulation = async () => {
    if (!selectedPortfolio) return;
    setStatus('running');
    setResult(null);

    try {
      const response = await fetch(`/api/portfolio/${selectedPortfolio}/simulate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ start_date: startDate, end_date: endDate }),
      });
      const data = await response.json();
      if (response.status === 202) {
        setWorkflowId(data.workflow_id);
        pollWorkflowStatus(data.workflow_id);
      } else {
        setStatus('failed');
      }
    } catch (error) {
      console.error("Simulation trigger error:", error);
      setStatus('failed');
    }
  };

  return (
    <div>
      <h1 className="text-4xl font-bold mb-2 bg-gradient-to-r from-green-400 to-cyan-500 bg-clip-text text-transparent">
        AI Simulation Engine
      </h1>
      <p className="text-slate-400 mb-8">Backtest AI rebalancing strategies against historical market data.</p>

      {/* Configuration */}
      <div className="grid grid-cols-4 gap-6 bg-slate-800 border border-slate-700 rounded-xl p-6 mb-8">
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">Portfolio</label>
          <select
            title="Select a portfolio"
            value={selectedPortfolio}
            onChange={(e) => setSelectedPortfolio(e.target.value)}
            className="w-full bg-slate-700 border border-slate-600 rounded-md py-2 px-3 text-white"
          >
            <option value="" disabled>Select a portfolio</option>
            {portfoliosData?.portfolios.map(p => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">Start Date</label>
          <input title="Start Date" type="date" value={startDate} onChange={e => setStartDate(e.target.value)} className="w-full bg-slate-700 border border-slate-600 rounded-md py-2 px-3 text-white" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">End Date</label>
          <input title="End Date" type="date" value={endDate} onChange={e => setEndDate(e.target.value)} className="w-full bg-slate-700 border border-slate-600 rounded-md py-2 px-3 text-white" />
        </div>
        <div className="flex items-end">
          <button
            onClick={handleRunSimulation}
            disabled={!selectedPortfolio || status === 'running'}
            className="w-full py-3 rounded-lg font-medium transition flex items-center justify-center gap-2 bg-gradient-to-r from-green-600 to-cyan-600 hover:from-green-700 hover:to-cyan-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {status === 'running' ? <Loader className="w-5 h-5 animate-spin" /> : <PlayCircle className="w-5 h-5" />}
            Run Simulation
          </button>
        </div>
      </div>

      {/* Results */}
      <div className="bg-slate-800 border border-slate-700 rounded-xl p-6 min-h-[400px] flex items-center justify-center">
        {status === 'idle' && <div className="text-slate-500">Configure and run a simulation to see results.</div>}
        {status === 'running' && <div className="flex flex-col items-center gap-4 text-slate-400"><Loader className="w-12 h-12 animate-spin text-cyan-500" /><span>Running simulation... This may take a minute.</span></div>}
        {status === 'failed' && <div className="text-red-400">Simulation failed. Please check the console for errors.</div>}
        {status === 'completed' && result && (
          <div className="w-full">
            <h2 className="text-2xl font-bold mb-4">Simulation Results</h2>
            <ResponsiveContainer width="100%" height={400}>
              <LineChart data={result.performance_chart}>
                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                <XAxis dataKey="date" tickFormatter={ts => new Date(ts * 1000).toLocaleDateString()} stroke="#9ca3af" />
                <YAxis tickFormatter={val => `$${(val/1000000).toFixed(1)}M`} stroke="#9ca3af" />
                <Tooltip
                  contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #374151' }}
                  labelFormatter={ts => new Date(ts * 1000).toLocaleDateString()}
                />
                <Legend />
                <Line type="monotone" dataKey="portfolio" stroke="#22d3ee" strokeWidth={2} name="Simulated Portfolio" />
                <Line type="monotone" dataKey="benchmark" stroke="#4ade80" strokeWidth={2} name="Benchmark" />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>
    </div>
  );
};

export default AISimulationDashboard;
