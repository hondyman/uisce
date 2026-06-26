import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import axios from 'axios';

interface Incident {
    cluster_id: string;
    root_cause: string;
    error_count: number;
    first_seen: string;
    last_seen: string;
    examples: string[];
    suggested_fix: string;
    can_auto_heal: boolean;
    ai_analysis?: {
        summary: string;
        remediation_plan: string[];
        blast_radius: string[];
    };
}

const IncidentPage: React.FC = () => {
    const { id } = useParams<{ id: string }>(); 
    const navigate = useNavigate();
    const [incident, setIncident] = useState<Incident | null>(null);
    const [loading, setLoading] = useState(true);
    const [healing, setHealing] = useState(false);

    useEffect(() => {
        // Mock fetch - assume we have an endpoint /api/autonomous-runtime/incidents/{id}
        // or we fetch from ai_requests for the specific cluster ID
        // For demonstration, we'll simulate the data fetch + AI enrichment
        const fetchIncident = async () => {
            setLoading(true);
            try {
                // In a real app: await axios.get(`/api/incidents/${id}`);
                // Simulating response with AI data included
                setTimeout(() => {
                    setIncident({
                        cluster_id: id || 'mock-id',
                        root_cause: 'Stale pre-aggregation: positions_daily',
                        error_count: 17,
                        first_seen: '2026-01-16 09:30:00',
                        last_seen: '2026-01-16 09:45:00',
                        examples: [
                            'Query timeout: Pre-agg positions_daily is stale',
                            'Data freshness violation: positions_daily last refreshed 4 hours ago'
                        ],
                        suggested_fix: 'Force refresh positions_daily pre-aggregation',
                        can_auto_heal: true,
                        ai_analysis: {
                            summary: "The `positions_daily` pre-aggregation has fallen behind the real-time data ingestion rate, causing query timeouts in the `PortfolioOverview` dashboard. This appears to be caused by a lock contention issue in the underlying data warehouse.",
                            remediation_plan: [
                                "1. Terminate blocking queries on `raw_trades` table.",
                                "2. Trigger immediate refresh of `positions_daily`.",
                                "3. Scale up compute cluster `shared-warehouse-1` to handle backlog."
                            ],
                            blast_radius: [
                                "PortfolioOverview Dashboard (High Impact)",
                                "RiskReport API (Medium Impact)"
                            ]
                        }
                    });
                    setLoading(false);
                }, 800);
            } catch (err) {
                console.error(err);
                setLoading(false);
            }
        };

        if (id) fetchIncident();
    }, [id]);

    const handleAutoHeal = async () => {
        setHealing(true);
        // await axios.post(`/api/incidents/${id}/heal`);
        setTimeout(() => {
            alert('Auto-healing triggered successfully. Incident resolving...');
            setHealing(false);
            navigate('/scheduler'); // Redirect back to console
        }, 1500);
    };

    if (loading) return <div className="p-8 text-center text-gray-500">Analyze Incident...</div>;
    if (!incident) return <div className="p-8 text-center text-red-500">Incident not found</div>;

    return (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <div className="mb-8">
                <button 
                    onClick={() => navigate(-1)}
                    className="mb-4 text-sm text-gray-500 hover:text-gray-700"
                >
                    &larr; Back to Scheduler Console
                </button>
                <div className="flex md:items-center md:justify-between">
                    <div className="flex-1 min-w-0">
                        <h2 className="text-2xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:text-3xl sm:truncate">
                            Incident Analysis: <span className="font-mono text-red-600">{incident.root_cause}</span>
                        </h2>
                        <div className="mt-1 flex flex-col sm:flex-row sm:flex-wrap sm:mt-0 sm:space-x-6">
                            <div className="mt-2 flex items-center text-sm text-gray-500">
                                Cluster ID: {incident.cluster_id}
                            </div>
                            <div className="mt-2 flex items-center text-sm text-gray-500">
                                Errors: {incident.error_count}
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                {/* Left Column: AI Analysis */}
                <div className="lg:col-span-2 space-y-6">
                    <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                        <div className="flex items-center mb-4">
                            <span className="text-2xl mr-2">🤖</span>
                            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">AI Narrative</h3>
                        </div>
                        <p className="text-gray-700 dark:text-gray-300 mb-4 whitespace-pre-wrap">
                            {incident.ai_analysis?.summary}
                        </p>
                        
                        <h4 className="font-semibold text-sm uppercase text-gray-500 mb-2">Remediation Plan</h4>
                        <div className="bg-gray-50 dark:bg-gray-900 rounded-md p-4 mb-4">
                            <ul className="space-y-2">
                                {incident.ai_analysis?.remediation_plan.map((step, i) => (
                                    <li key={i} className="flex items-start">
                                        <span className="flex-shrink-0 h-5 w-5 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center text-xs font-bold mr-3 mt-0.5">
                                            {i + 1}
                                        </span>
                                        <span className="text-sm text-gray-800 dark:text-gray-200">{step}</span>
                                    </li>
                                ))}
                            </ul>
                        </div>

                        {incident.can_auto_heal && (
                            <button
                                onClick={handleAutoHeal}
                                disabled={healing}
                                className={`w-full flex justify-center items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 ${healing ? 'opacity-50' : ''}`}
                            >
                                {healing ? 'Healing System...' : 'Execute Remediation Plan'}
                            </button>
                        )}
                    </div>

                    <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Blast Radius</h3>
                        <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                            {incident.ai_analysis?.blast_radius.map((item, i) => (
                                <li key={i} className="py-3 flex items-center">
                                    <span className="text-red-500 mr-3">⚠️</span>
                                    <span className="text-sm text-gray-700 dark:text-gray-300">{item}</span>
                                </li>
                            ))}
                        </ul>
                    </div>
                </div>

                {/* Right Column: Technical Details */}
                <div className="space-y-6">
                    <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Error Samples</h3>
                        <div className="space-y-3">
                            {incident.examples.map((ex, i) => (
                                <div key={i} className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs font-mono text-gray-800 dark:text-gray-200 break-all">
                                    {ex}
                                </div>
                            ))}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default IncidentPage;
