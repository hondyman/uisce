import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ChangeReviewApi } from '../api/changeReview';
import { ChangeReview } from '../types/changeReview';
import DiffTab from '../components/changeReview/DiffTab';
import LineageImpactTab from '../components/changeReview/LineageImpactTab';
import TestResultsTab from '../components/changeReview/TestResultsTab';
import ASOImpactTab from '../components/changeReview/ASOImpactTab';

// Tabs
const TABS = [
    { id: 'diff', label: 'Semantic Diff' },
    { id: 'lineage', label: 'Lineage Impact' },
    { id: 'tests', label: 'Test Results' },
    { id: 'aso', label: 'ASO Impact' },
    { id: 'ai_impact', label: 'AI Impact Analysis' },
];

const ChangeReviewPage: React.FC = () => {
    const { id } = useParams<{ id: string }>(); // This is the ChangeSet ID
    const navigate = useNavigate();
    const [review, setReview] = useState<ChangeReview | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [activeTab, setActiveTab] = useState('diff');
    const [promoting, setPromoting] = useState(false);

    useEffect(() => {
        if (!id) return;
        loadReview();
    }, [id]);

    const loadReview = async () => {
        setLoading(true);
        try {
            // Fetch review by ChangeSet ID
            const data = await ChangeReviewApi.getReview(id!);
            setReview(data);
            setError(null);
        } catch (err: any) {
             // If not found, maybe create one? Or show "Analysis Pending" state?
             // For now, assume flow starts with creation.
             // If 404, we could offer "Start Review" button.
            setError(err.response?.status === 404 
                ? 'Review not found. Analysis might not have started.' 
                : 'Failed to load review.');
        } finally {
            setLoading(false);
        }
    };

    const handleStartReview = async () => {
        if (!id) return;
        setLoading(true);
        try {
             await ChangeReviewApi.createReview(id);
             // Poll or wait? Assuming async workflow, it returns "pending" review immediately.
             await loadReview();
        } catch (err) {
            setError('Failed to start review.');
            setLoading(false);
        }
    };

    const handlePromote = async () => {
        if (!id) return;
        if (!window.confirm('Are you sure you want to promote these changes to production?')) return;
        
        setPromoting(true);
        try {
            await ChangeReviewApi.promote(id);
            alert('Promotion started successfully.');
            // Reload to show updated status? Status update might be async.
            loadReview();
        } catch (err) {
            alert('Failed to promote changes.');
        } finally {
            setPromoting(false);
        }
    };

    if (loading) return <div className="p-8 text-center text-gray-500">Loading Change Review...</div>;

    if (error) {
        return (
            <div className="p-8 text-center">
                <div className="text-red-500 mb-4">{error}</div>
                {error.includes('not found') && (
                    <button 
                        onClick={handleStartReview}
                        className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition"
                    >
                        Start Semantic Analysis
                    </button>
                )}
            </div>
        );
    }

    if (!review) return null;

    return (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            {/* Header */}
            <div className="md:flex md:items-center md:justify-between mb-8">
                <div className="flex-1 min-w-0">
                    <h2 className="text-2xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:text-3xl sm:truncate">
                        Change Review: <span className="font-mono text-lg text-gray-500">{id}</span>
                    </h2>
                    <div className="mt-1 flex flex-col sm:flex-row sm:flex-wrap sm:mt-0 sm:space-x-6">
                        <div className="mt-2 flex items-center text-sm text-gray-500 dark:text-gray-400">
                            Status: 
                            <span className={`ml-2 px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                review.status === 'approved' ? 'bg-green-100 text-green-800' : 
                                review.status === 'promoted' ? 'bg-purple-100 text-purple-800' :
                                review.status === 'rejected' ? 'bg-red-100 text-red-800' :
                                'bg-yellow-100 text-yellow-800'
                            }`}>
                                {review.status.toUpperCase()}
                            </span>
                        </div>
                    </div>
                </div>
                <div className="mt-4 flex md:mt-0 md:ml-4">
                     {/* Action Buttons */}
                     {review.status === 'approved' && (
                        <button
                            type="button"
                            onClick={handlePromote}
                            disabled={promoting}
                            className={`inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 ${promoting ? 'opacity-50 cursor-not-allowed' : ''}`}
                        >
                            {promoting ? 'Promoting...' : 'Promote to Production'}
                        </button>
                     )}
                     {review.status === 'pending' && (
                         <span className="text-sm text-gray-500 italic">Analysis in progress or awaiting approval...</span>
                     )}
                </div>
            </div>

            {/* Tabs */}
            <div className="border-b border-gray-200 dark:border-gray-700 mb-6">
                <nav className="-mb-px flex space-x-8" aria-label="Tabs">
                    {TABS.map((tab) => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id)}
                            className={`${
                                activeTab === tab.id
                                    ? 'border-blue-500 text-blue-600 dark:text-blue-500'
                                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                            } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm transition-colors`}
                        >
                            {tab.label}
                        </button>
                    ))}
                </nav>
            </div>

            {/* Tab Content */}
            <div className="mt-4">
                {activeTab === 'diff' && <DiffTab diffs={JSON.parse(review.diff_summary as any || '{}')} />}
                {activeTab === 'lineage' && <LineageImpactTab impacts={JSON.parse(review.lineage_impact as any || '{}')} />}
                {activeTab === 'tests' && <TestResultsTab results={review.test_results} />}
                {activeTab === 'aso' && <ASOImpactTab review={review} />}
                {activeTab === 'ai_impact' && (
                    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
                        <div className="flex items-center mb-6">
                            <span className="text-2xl mr-2">🚀</span>
                            <h3 className="text-xl font-bold">AI Change Impact Summary</h3>
                        </div>
                        {review.ai_summary ? (
                            <div className="prose prose-blue dark:prose-invert max-w-none">
                                {/* Markdown Rendering handled by CSS for now, or just raw text with newlines */}
                                <div className="whitespace-pre-wrap text-lg leading-relaxed mb-4">
                                    {review.ai_summary}
                                </div>
                                
                                <div className="mt-8 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-100">
                                    <div className="flex items-start">
                                        <span className="text-xl mr-3">💡</span>
                                        <div>
                                            <div className="font-bold text-blue-800 dark:text-blue-300">
                                                AI Risk Assessment: {review.ai_risk_level} ({review.ai_risk_score}/10)
                                            </div>
                                            <p className="text-sm text-blue-700 dark:text-blue-400">Analysis based on lineage graph and historical SLO metadata.</p>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        ) : (
                            <div className="text-gray-500 italic">
                                AI analysis is pending or unavailable for this review.
                            </div>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
};

export default ChangeReviewPage;
