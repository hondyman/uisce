
import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { ArrowForward, CheckCircle, Warning, PlayArrow, Refresh, Publish } from '@mui/icons-material';
import { simulationApi, SimulationScenario, SimulationDelta, SimulationResult } from '../../api/simulationApi';

const ScenarioDetail: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const [scenario, setScenario] = useState<SimulationScenario | null>(null);
    const [deltas, setDeltas] = useState<SimulationDelta[]>([]);
    const [result, setResult] = useState<SimulationResult | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [isRunning, setIsRunning] = useState(false);

    useEffect(() => {
        if (id) {
            loadData(id);
        }
    }, [id]);

    const loadData = async (scenarioId: string) => {
        setIsLoading(true);
        try {
            const [scn, dlts, res] = await Promise.all([
                simulationApi.getScenario(scenarioId),
                simulationApi.getDeltas(scenarioId),
                simulationApi.getLatestResult(scenarioId).catch(() => null) // Ignore 404 if no result yet
            ]);
            setScenario(scn);
            setDeltas(dlts || []);
            setResult(res);
        } catch (e) {
            console.error("Failed to load scenario data", e);
        } finally {
            setIsLoading(false);
        }
    };

    const handleRun = async () => {
        if (!id) return;
        setIsRunning(true);
        try {
            const res = await simulationApi.runSimulation(id);
            setResult(res);
        } catch (e) {
            console.error("Failed to run simulation", e);
            alert("Simulation failed. See console.");
        } finally {
            setIsRunning(false);
        }
    };

    const handleCreateChangeSet = async () => {
        if (!id) return;
        if (!confirm("This will lock the scenario result and create a governed ChangeSet. Proceed?")) return;
        try {
            const res = await simulationApi.createChangeSet(id);
            alert(`ChangeSet Created: ${res.changeset_id}`);
            // Navigate to ChangeSet or Governance view
            // navigate(`/governance/changesets/${res.changeset_id}`);
        } catch (e) {
            console.error("Failed to create changeset", e);
            alert("Failed to create ChangeSet.");
        }
    };

    if (isLoading) return <div className="p-8 text-gray-400">Loading...</div>;
    if (!scenario) return <div className="p-8 text-red-500">Scenario not found</div>;

    const navDelta = result?.summary?.nav_delta || 0;
    const varDelta = result?.summary?.var_delta || 0;
    const newIssues = result?.complianceSummary?.newIssues || [];

    return (
        <div className="p-8 bg-gray-900 min-h-screen text-gray-100 font-sans">
            <div className="mb-8 flex justify-between items-start">
                <div>
                     <button onClick={() => navigate('/simulation')} className="text-gray-500 hover:text-white mb-2 text-sm">
                        ← Back to Workspace
                    </button>
                    <h1 className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-green-400">
                        {scenario.name}
                    </h1>
                    <p className="text-gray-400 mt-2 max-w-2xl">{scenario.description}</p>
                    <div className="mt-4 flex gap-2">
                         <span className="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400 border border-gray-700">{scenario.scenarioType}</span>
                         <span className="px-2 py-1 bg-gray-800 rounded text-xs text-gray-400 border border-gray-700">{scenario.status}</span>
                    </div>
                </div>
                <div className="flex gap-4">
                     <button 
                        onClick={handleRun}
                        disabled={isRunning}
                        className={`flex items-center gap-2 px-6 py-2 rounded-lg font-semibold transition-all ${isRunning ? 'bg-gray-700 cursor-not-allowed' : 'bg-green-600 hover:bg-green-500 shadow-lg shadow-green-900/50'}`}
                     >
                        {isRunning ? <Refresh className="animate-spin" /> : <PlayArrow />}
                        Run Simulation
                    </button>
                    {result && (
                         <button 
                            onClick={handleCreateChangeSet}
                            className="flex items-center gap-2 px-6 py-2 bg-purple-600 rounded-lg font-semibold hover:bg-purple-500 shadow-lg shadow-purple-900/50"
                        >
                            <Publish fontSize="small" />
                            Convert to ChangeSet
                        </button>
                    )}
                </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                {/* Deltas Panel */}
                <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6 h-fit">
                    <h3 className="text-lg font-semibold mb-4 text-blue-300">Applied Deltas</h3>
                    <div className="space-y-3">
                        {deltas.length === 0 && <div className="text-gray-500 italic">No deltas defined.</div>}
                        {deltas.map(d => (
                            <div key={d.scenarioId + d.boId} className="flex justify-between items-center p-3 bg-gray-900 rounded border border-gray-700/50">
                                <div>
                                    <div className="font-mono text-gray-200 text-sm">{d.boId}</div>
                                    <div className="text-xs text-blue-400 tracking-wider">{d.deltaType}</div>
                                </div>
                                <div className="flex items-center gap-2 text-gray-400">
                                    <ArrowForward fontSize="small" className="text-gray-600" />
                                    {/* Visual summary of changes JSON? */}
                                    <span className="text-gray-300 text-xs truncate max-w-[100px]" title={JSON.stringify(d.changes)}>
                                        {JSON.stringify(d.changes).substring(0, 15)}...
                                    </span>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>

                {/* Results Panel */}
                <div className="col-span-2 space-y-8">
                    {!result ? (
                        <div className="p-12 border-2 border-dashed border-gray-800 rounded-xl text-center text-gray-600">
                            No simulation results yet. Run the simulation to see impact.
                        </div>
                    ) : (
                        <>
                        {/* Key Metrics */}
                        <div className="grid grid-cols-2 gap-4">
                            <motion.div initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} className="p-6 bg-gray-800/80 rounded-xl border border-gray-700 relative overflow-hidden">
                                <div className="absolute top-0 right-0 p-4 opacity-10 text-red-500 font-bold text-6xl">$</div>
                                <div className="text-gray-400 text-sm uppercase tracking-wider">NAV Impact</div>
                                <div className={`text-3xl font-mono mt-2 ${navDelta >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                                    {navDelta > 0 ? '+' : ''}{navDelta?.toLocaleString('en-US', { style: 'currency', currency: 'USD' })}
                                </div>
                            </motion.div>
                            <motion.div initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} transition={{ delay: 0.1 }} className="p-6 bg-gray-800/80 rounded-xl border border-gray-700 relative overflow-hidden">
                                <div className="absolute top-0 right-0 p-4 opacity-10 text-orange-500 font-bold text-6xl">Var</div>
                                <div className="text-gray-400 text-sm uppercase tracking-wider">VaR (95%) Delta</div>
                                <div className={`text-3xl font-mono mt-2 ${varDelta <= 0 ? 'text-green-400' : 'text-red-400'}`}>
                                    {varDelta > 0 ? '+' : ''}{varDelta?.toLocaleString('en-US', { style: 'currency', currency: 'USD' })}
                                </div>
                                <div className="text-xs text-gray-500 mt-1">Lower is better</div>
                            </motion.div>
                        </div>

                        {/* Compliance */}
                        <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                            <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
                                <CheckCircle fontSize="small" className={newIssues.length > 0 ? 'text-orange-500' : 'text-green-500'} />
                                Compliance Check
                            </h3>
                            <div className="space-y-3">
                                {newIssues.length === 0 ? (
                                    <div className="p-4 bg-green-900/10 border border-green-800/30 rounded text-green-400 text-sm">
                                        All checks passed. No new compliance violations.
                                    </div>
                                ) : (
                                    newIssues.map((issue: any, i: number) => (
                                        <div key={i} className="p-4 bg-red-900/10 border border-red-800/30 rounded flex gap-4 items-start">
                                            <Warning className="text-red-500 mt-1" fontSize="small" />
                                            <div>
                                                <div className="font-medium text-red-300">{issue.ruleId}</div>
                                                <div className="text-sm text-red-400/70">{issue.description}</div>
                                            </div>
                                        </div>
                                    ))
                                )}
                            </div>
                        </div>
                        </>
                    )}
                </div>
            </div>
        </div>
    );
};

export default ScenarioDetail;
