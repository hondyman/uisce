
import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { 
  PlayArrow, 
  Add, 
  CompareArrows, 
  Assessment, 
  TrendingUp, 
  Warning,
  Balance,
  Search
} from '@mui/icons-material';
import { simulationApi, SimulationScenario } from '../../api/simulationApi';

const SimulationWorkspace: React.FC = () => {
  const [scenarios, setScenarios] = useState<SimulationScenario[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    loadScenarios();
  }, []);

  const loadScenarios = async () => {
    try {
      setIsLoading(true);
      const data = await simulationApi.listScenarios();
      setScenarios(data || []);
    } catch (e) {
      console.error("Failed to list scenarios", e);
    } finally {
      setIsLoading(false);
    }
  };

  const statusColors: Record<string, string> = {
    'DRAFT': 'bg-gray-700 text-gray-300',
    'RUNNING': 'bg-blue-900/50 text-blue-300 border-blue-700',
    'COMPLETED': 'bg-green-900/50 text-green-300 border-green-700',
    'FAILED': 'bg-red-900/50 text-red-300 border-red-700',
  };

  const filteredScenarios = scenarios.filter(s => 
    s.name.toLowerCase().includes(searchTerm.toLowerCase()) || 
    s.description?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="p-8 bg-gray-900 min-h-screen text-gray-100 font-sans">
      
      {/* Header */}
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500">
            What-If Intelligence
          </h1>
          <p className="text-gray-400 mt-2">Simulate, Stress-Test, and optimize your portfolio strategies.</p>
        </div>
        <div className="flex gap-4">
           <button className="flex items-center gap-2 px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg hover:bg-gray-700 transition-colors">
              <CompareArrows fontSize="small" /> Compare
           </button>
           <button 
              onClick={() => window.location.href = '/simulation/rebalance'}
              className="flex items-center gap-2 px-4 py-2 bg-purple-600 rounded-lg hover:bg-purple-500 transition-shadow shadow-lg shadow-purple-900/50"
           >
              <Balance fontSize="small" /> Rebalance
           </button>
           <button 
              onClick={() => window.location.href = '/simulation/new'} // TODO: Implement Wizard
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 rounded-lg hover:bg-blue-500 transition-shadow shadow-lg shadow-blue-900/50"
           >
              <Add fontSize="small" /> New Scenario
           </button>
        </div>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-12">
        {[
          { label: 'Active Scenarios', value: scenarios.length, icon: <Assessment />, color: 'text-blue-400' },
          { label: 'Risk Models', value: 'MC-VaR', icon: <Warning />, color: 'text-orange-400' },
          { label: 'Recent Runs', value: scenarios.filter(s => s.status === 'COMPLETED').length, icon: <PlayArrow />, color: 'text-green-400' },
          { label: 'Simulated AUM', value: '$4.2B', icon: <TrendingUp />, color: 'text-purple-400' },
        ].map((stat, i) => (
          <motion.div 
            key={i}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: i * 0.1 }}
            className="p-6 bg-gray-800/50 border border-gray-700/50 rounded-xl backdrop-blur-sm"
          >
            <div className={`mb-4 ${stat.color}`}>{stat.icon}</div>
            <div className="text-2xl font-bold mb-1">{stat.value}</div>
            <div className="text-sm text-gray-500">{stat.label}</div>
          </motion.div>
        ))}
      </div>

      {/* Scenario List */}
      <div className="bg-gray-800/30 border border-gray-700 rounded-xl overflow-hidden">
        <div className="p-6 border-b border-gray-700 flex justify-between items-center">
          <h2 className="text-xl font-semibold">Recent Scenarios</h2>
          <div className="relative">
            <Search className="absolute left-3 top-2.5 text-gray-500" fontSize="small" />
            <input 
                type="text" 
                placeholder="Search..." 
                className="pl-10 pr-4 py-2 bg-gray-900 border border-gray-700 rounded-lg focus:outline-none focus:border-blue-500 text-sm"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
        </div>
        
        {isLoading ? (
          <div className="p-12 text-center text-gray-500">Loading simulations...</div>
        ) : (
          <table className="w-full text-left">
            <thead className="bg-gray-800/80 text-gray-400 uppercase text-xs">
              <tr>
                <th className="px-6 py-4">Name</th>
                <th className="px-6 py-4">Type</th>
                <th className="px-6 py-4">Status</th>
                <th className="px-6 py-4">Created At</th>
                <th className="px-6 py-4">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-700/50">
              {filteredScenarios.map((s) => (
                <tr key={s.id} className="hover:bg-gray-800/50 transition-colors cursor-pointer" onClick={() => window.location.href = `/simulation/${s.id}`}>
                  <td className="px-6 py-4">
                    <div className="font-medium text-gray-200">{s.name}</div>
                    <div className="text-xs text-gray-500 truncate max-w-xs">{s.description}</div>
                  </td>
                  <td className="px-6 py-4">
                    <span className="px-2 py-1 bg-gray-700 rounded text-xs text-gray-300 font-mono">{s.scenarioType}</span>
                  </td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded text-xs font-semibold border ${statusColors[s.status] || 'bg-gray-700'}`}>
                      {s.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-gray-400 text-sm">
                    {new Date(s.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4">
                    <ArrowForward className="text-gray-600 hover:text-white" fontSize="small"  />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};

// Helper for Arrow
import { ArrowForward } from '@mui/icons-material';

export default SimulationWorkspace;
