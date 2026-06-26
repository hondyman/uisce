
import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { 
  ArrowForward, 
  CheckCircle, 
  Warning, 
  TrendingUp, 
  AttachMoney, 
  Balance 
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { simulationApi } from '../../api/simulationApi';
import { v4 as uuidv4 } from 'uuid'; // Assuming uuid is available or use random string

// Types
interface RebalanceConfig {
  portfolioId: string;
  type: 'TO_TARGET_WEIGHTS' | 'TO_NEW_TARGET';
  constraints: {
    avoidShortTermGains: boolean;
    capSector: boolean;
    sectorCapValue: number;
    sectorCapName: string;
  }
}

const RebalancingWizard: React.FC = () => {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  
  const [config, setConfig] = useState<RebalanceConfig>({
    portfolioId: 'pf-123',
    type: 'TO_TARGET_WEIGHTS',
    constraints: {
      avoidShortTermGains: false,
      capSector: false,
      sectorCapValue: 20,
      sectorCapName: 'Technology'
    }
  });

  // Mock Data for Step 2 (Trades)
  const mockTrades = [
    { id: 't1', symbol: 'TSLA', side: 'SELL', qty: 50, price: 250, value: 12500, reason: 'Overweight 2.5%' },
    { id: 't2', symbol: 'MSFT', side: 'BUY', qty: 15, price: 300, value: 4500, reason: 'Underweight 0.8%' },
    { id: 't3', symbol: 'AAPL', side: 'BUY', qty: 20, price: 150, value: 3000, reason: 'Underweight 0.5%' },
  ];

  // Mock Data for Step 3 (Impact)
  const mockImpact = {
    originalNav: 1500000,
    newNav: 1499500, // slightly less due to costs
    navDelta: -500,
    originalVar: 45000, 
    newVar: 38000, // Reduced risk
    varDelta: -7000,
    sectorChanges: [
        { name: 'Technology', from: 28, to: 20 },
        { name: 'Healthcare', from: 12, to: 15 }
    ]
  };

  const handleSimulate = () => {
      setLoading(true);
      // Simulate API call delay
      setTimeout(() => {
          setLoading(false);
          setStep(2);
      }, 1500);
  };

  const handleExecute = () => {
    setLoading(true);
    setTimeout(() => {
        setLoading(false);
        setStep(3);
    }, 1500);
  };

  const handleFinish = async () => {
      try {
          setLoading(true);
          // 1. Create a Scenario container
          const scenario = await simulationApi.createScenario({
              name: `Rebalance - ${config.portfolioId} - ${new Date().toISOString().split('T')[0]}`,
              description: `Strategy: ${config.type}. Constraint: ${config.constraints.capSector ? 'Sector Cap' : 'None'}`,
              scenarioType: 'PORTFOLIO',
              status: 'DRAFT',
              tenantId: '910638ba-a459-4a3f-bb2d-78391b0595f6' // Hardcoded for wizard
          });

          // 2. Add the Rebalance Rule as a Delta
          await simulationApi.addDelta(scenario.id, {
              boId: config.portfolioId,
              deltaType: 'REBALANCE_RULE',
              changes: {
                  type: config.type,
                  constraints: config.constraints
              }
          });

          // 3. Run Simulation (to generate result)
          await simulationApi.runSimulation(scenario.id);

          // 4. Create ChangeSet
          const { changeset_id } = await simulationApi.createChangeSet(scenario.id);
          
          alert(`ChangeSet Proposed Successfully! ID: ${changeset_id}`);
          navigate('/simulation'); // Ideally to the changeset view
      } catch (e) {
          console.error(e);
          alert('Failed to propose changeset. See console.');
      } finally {
          setLoading(false);
      }
  };

  return (
    <div className="p-8 bg-gray-900 min-h-screen text-gray-100 font-sans flex flex-col items-center">
      
      <div className="w-full max-w-4xl">
        {/* Header */}
        <div className="mb-8 text-center">
            <h1 className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500">
                Portfolio Rebalancing Wizard
            </h1>
            <p className="text-gray-400 mt-2">
                Align portfolios with strategic targets based on risk and constraints.
            </p>
            
            {/* Stepper */}
            <div className="flex justify-center mt-8 gap-4">
                {[1, 2, 3].map(s => (
                    <div key={s} className={`flex items-center gap-2 ${step >= s ? 'text-blue-400' : 'text-gray-600'}`}>
                        <div className={`w-8 h-8 rounded-full flex items-center justify-center border ${step >= s ? 'border-blue-400 bg-blue-900/30' : 'border-gray-600'}`}>
                            {s}
                        </div>
                        <span className="text-sm font-medium">
                            {s === 1 ? 'Configuration' : s === 2 ? 'Review Trades' : 'Impact Analysis'}
                        </span>
                        {s < 3 && <div className="w-12 h-px bg-gray-700 mx-2" />}
                    </div>
                ))}
            </div>
        </div>

        {/* Content Area */}
        <div className="bg-gray-800/40 border border-gray-700 rounded-2xl p-8 min-h-[400px]">
            
            {/* STEP 1: CONFIG */}
            {step === 1 && (
                <motion.div initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }}>
                    <h2 className="text-xl font-semibold mb-6 flex items-center gap-2">
                        <Balance /> Define Scenario
                    </h2>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                        <div>
                            <label className="block text-sm text-gray-400 mb-2">Target Portfolio</label>
                            <select 
                                className="w-full bg-gray-900 border border-gray-700 rounded p-3 text-white focus:ring-2 focus:ring-blue-500 outline-none"
                                value={config.portfolioId}
                                onChange={e => setConfig({...config, portfolioId: e.target.value})}
                            >
                                <option value="pf-123">Global Growth Strategy (PF-123)</option>
                                <option value="pf-456">Income Conservative (PF-456)</option>
                            </select>
                        </div>
                        <div>
                            <label className="block text-sm text-gray-400 mb-2">Rebalance Strategy</label>
                            <select 
                                className="w-full bg-gray-900 border border-gray-700 rounded p-3 text-white focus:ring-2 focus:ring-blue-500 outline-none"
                                value={config.type}
                                onChange={e => setConfig({...config, type: e.target.value as any})}
                            >
                                <option value="TO_TARGET_WEIGHTS">Rebalance to Strategic Targets</option>
                                <option value="TO_NEW_TARGET">Shift to New Target Model</option>
                            </select>
                        </div>
                    </div>

                    <div className="mt-8">
                        <h3 className="text-sm font-semibold text-gray-300 mb-4">Constraints</h3>
                        <div className="space-y-4">
                            <label className="flex items-center gap-3 p-3 bg-gray-900/50 rounded border border-gray-700/50 cursor-pointer hover:bg-gray-700/30">
                                <input 
                                    type="checkbox" 
                                    checked={config.constraints.avoidShortTermGains}
                                    onChange={e => setConfig({...config, constraints: {...config.constraints, avoidShortTermGains: e.target.checked}})}
                                    className="w-5 h-5 rounded border-gray-600 text-blue-500 focus:ring-blue-500 bg-gray-800"
                                />
                                <div>
                                    <div className="font-medium text-gray-200">Avoid Short-Term Capital Gains</div>
                                    <div className="text-xs text-gray-500">Only sell positions held {'>'} 12 months</div>
                                </div>
                            </label>
                            
                            <label className="flex items-center gap-3 p-3 bg-gray-900/50 rounded border border-gray-700/50">
                                <input 
                                    type="checkbox" 
                                    checked={config.constraints.capSector}
                                    onChange={e => setConfig({...config, constraints: {...config.constraints, capSector: e.target.checked}})}
                                    className="w-5 h-5 rounded border-gray-600 text-blue-500 focus:ring-blue-500 bg-gray-800"
                                />
                                <div className="flex-1">
                                    <div className="font-medium text-gray-200">Sector Cap Constraint</div>
                                    <div className="text-xs text-gray-500">Limit exposure to specific sectors</div>
                                </div>
                                {config.constraints.capSector && (
                                    <div className="flex items-center gap-2">
                                        <select 
                                            value={config.constraints.sectorCapName}
                                            onChange={e => setConfig({...config, constraints: {...config.constraints, sectorCapName: e.target.value}})}
                                            className="bg-gray-800 border border-gray-700 rounded p-1 text-sm text-gray-300"
                                        >
                                            <option>Technology</option>
                                            <option>Financials</option>
                                            <option>Energy</option>
                                        </select>
                                        <span className="text-gray-400">≤</span>
                                        <input 
                                            type="number"
                                            value={config.constraints.sectorCapValue}
                                            onChange={e => setConfig({...config, constraints: {...config.constraints, sectorCapValue: Number(e.target.value)}})}
                                            className="bg-gray-800 border border-gray-700 rounded p-1 w-16 text-sm text-gray-300 text-right"
                                        />
                                        <span className="text-gray-400">%</span>
                                    </div>
                                )}
                            </label>
                        </div>
                    </div>

                    <div className="mt-8 flex justify-end">
                        <button 
                            onClick={handleSimulate}
                            disabled={loading}
                            className="flex items-center gap-2 px-6 py-3 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition-all shadow-lg shadow-blue-900/40"
                        >
                            {loading ? 'Calculating...' : 'Generate Trades'} <ArrowForward fontSize="small" />
                        </button>
                    </div>
                </motion.div>
            )}

            {/* STEP 2: REVIEW TRADES */}
            {step === 2 && (
                <motion.div initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }}>
                    <h2 className="text-xl font-semibold mb-6 flex items-center gap-2">
                        <AttachMoney /> Proposed Trades
                    </h2>
                    
                    <div className="mb-6 bg-blue-900/20 border border-blue-800 rounded-lg p-4 flex gap-8">
                        <div>
                            <span className="block text-xs uppercase text-blue-300 mb-1">Total Trades</span>
                            <span className="text-xl font-bold">{mockTrades.length}</span>
                        </div>
                        <div>
                            <span className="block text-xs uppercase text-blue-300 mb-1">Buy Value</span>
                            <span className="text-xl font-bold text-green-400">+$7,500</span>
                        </div>
                        <div>
                            <span className="block text-xs uppercase text-blue-300 mb-1">Sell Value</span>
                            <span className="text-xl font-bold text-red-400">-$12,500</span>
                        </div>
                    </div>

                    <div className="overflow-hidden rounded-lg border border-gray-700 mb-8">
                         <table className="w-full text-left">
                            <thead className="bg-gray-900 text-gray-400 text-xs uppercase">
                                <tr>
                                    <th className="px-4 py-3">Side</th>
                                    <th className="px-4 py-3">Asset</th>
                                    <th className="px-4 py-3 text-right">Qty</th>
                                    <th className="px-4 py-3 text-right">Price</th>
                                    <th className="px-4 py-3 text-right">Value</th>
                                    <th className="px-4 py-3">Reason</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-700">
                                {mockTrades.map(t => (
                                    <tr key={t.id} className="bg-gray-800/50 hover:bg-gray-800/80">
                                         <td className="px-4 py-3">
                                            <span className={`px-2 py-0.5 rounded text-xs font-bold ${t.side === 'BUY' ? 'bg-green-900/40 text-green-400' : 'bg-red-900/40 text-red-400'}`}>
                                                {t.side}
                                            </span>
                                         </td>
                                         <td className="px-4 py-3 font-medium">{t.symbol}</td>
                                         <td className="px-4 py-3 text-right font-mono">{t.qty}</td>
                                         <td className="px-4 py-3 text-right text-gray-400">${t.price}</td>
                                         <td className="px-4 py-3 text-right font-mono">${t.value.toLocaleString()}</td>
                                         <td className="px-4 py-3 text-sm text-gray-400">{t.reason}</td>
                                    </tr>
                                ))}
                            </tbody>
                         </table>
                    </div>

                    <div className="flex justify-between">
                         <button 
                            onClick={() => setStep(1)}
                            className="text-gray-400 hover:text-white"
                        >
                            Back to Config
                        </button>
                        <button 
                            onClick={handleExecute}
                            disabled={loading}
                            className="flex items-center gap-2 px-6 py-3 bg-purple-600 hover:bg-purple-500 text-white rounded-lg font-medium transition-all shadow-lg shadow-purple-900/40"
                        >
                            {loading ? 'Analyzing...' : 'Analyze Impact'} <TrendingUp fontSize="small" />
                        </button>
                    </div>
                </motion.div>
            )}

            {/* STEP 3: IMPACT ANALYSIS */}
            {step === 3 && (
                <motion.div initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }}>
                    <h2 className="text-xl font-semibold mb-6 flex items-center gap-2">
                        <TrendingUp /> Impact Analysis
                    </h2>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
                        <div className="bg-gray-900/50 border border-gray-700 p-6 rounded-xl">
                            <h3 className="text-gray-400 text-sm mb-4">Risk & Return</h3>
                            <div className="space-y-4">
                                <div className="flex justify-between items-center">
                                    <span className="text-gray-300">Nav Impact</span>
                                    <div className="text-right">
                                        <div className="text-red-400 font-mono">-${Math.abs(mockImpact.navDelta).toLocaleString()}</div>
                                        <div className="text-xs text-gray-600">due to trans. costs</div>
                                    </div>
                                </div>
                                <div className="h-px bg-gray-700/50" />
                                <div className="flex justify-between items-center">
                                    <span className="text-gray-300">VaR (95%) Reduction</span>
                                    <div className="text-right">
                                        <div className="text-green-400 font-mono">-${Math.abs(mockImpact.varDelta).toLocaleString()}</div>
                                        <div className="text-xs text-gray-600">Risk Mitigation</div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div className="bg-gray-900/50 border border-gray-700 p-6 rounded-xl">
                            <h3 className="text-gray-400 text-sm mb-4">Compliance Checks</h3>
                            <div className="space-y-3">
                                {config.constraints.capSector && (
                                    <div className="flex gap-3 items-start">
                                        <CheckCircle className="text-green-500 mt-0.5" fontSize="small" />
                                        <div>
                                            <div className="text-green-300 text-sm font-medium">Sector Cap Satisfied</div>
                                            <div className="text-xs text-gray-500">{config.constraints.sectorCapName} sector reduced to {config.constraints.sectorCapValue}%</div>
                                        </div>
                                    </div>
                                )}
                                <div className="flex gap-3 items-start">
                                    <CheckCircle className="text-green-500 mt-0.5" fontSize="small" />
                                    <div>
                                        <div className="text-green-300 text-sm font-medium">Cash Buffer Maintained</div>
                                        <div className="text-xs text-gray-500">Portfolio cash {'>'} 2%</div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div className="bg-blue-900/20 border border-blue-800 p-6 rounded-xl mb-8">
                         <div className="flex gap-4 items-center">
                             <div className="bg-blue-500/20 p-3 rounded-full text-blue-300"><Balance fontSize="large"/></div>
                             <div>
                                 <h4 className="font-semibold text-blue-200">Ready to Propose?</h4>
                                 <p className="text-sm text-blue-300/70">This will create a ChangeSet (CS-1024) for approval by the Investment Committee.</p>
                             </div>
                             <div className="ml-auto">
                                <button 
                                    onClick={handleFinish}
                                    className="px-6 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium shadow-md"
                                >
                                    Create ChangeSet
                                </button>
                             </div>
                         </div>
                    </div>
                    
                    <button 
                        onClick={() => setStep(2)}
                        className="text-gray-400 hover:text-white"
                    >
                        Back to Trades
                    </button>
                    
                </motion.div>
            )}

        </div>
      </div>
    </div>
  );
};

export default RebalancingWizard;
