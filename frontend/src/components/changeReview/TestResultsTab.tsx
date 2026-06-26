import React from 'react';
import { TestResult } from '../../types/changeReview';

interface TestResultsTabProps {
    results: TestResult[]; // TestResult is actually just a string in the JSON? No, type def is TestResult object.
}

const TestResultsTab: React.FC<TestResultsTabProps> = ({ results }) => {
    // Backend creates TestResult struct but might unmarshal to string if stored as JSONB string?
    // Type definition expects object. If we get strings, we might need parsing.
    // Assuming API client parses JSON to objects.
    
    // Check if results is actually a string (double JSON encoding issue common in prototypes)
    const validResults = typeof results === 'string' ? JSON.parse(results) : (results || []);

    if (validResults.length === 0) {
        return <div className="p-4 text-gray-500">No semantic tests executed.</div>;
    }

    return (
        <div className="space-y-4">
             <div className="flex items-center space-x-4 mb-4">
                <div className="text-sm font-medium text-gray-500">
                    Total: <span className="text-gray-900 dark:text-gray-100">{validResults.length}</span>
                </div>
                <div className="text-sm font-medium text-green-600">
                    Passed: {validResults.filter((r: TestResult) => r.passed).length}
                </div>
                <div className="text-sm font-medium text-red-600">
                    Failed: {validResults.filter((r: TestResult) => !r.passed).length}
                </div>
            </div>

            <div className="divide-y divide-gray-200 dark:divide-gray-700 border rounded-lg bg-white dark:bg-gray-800">
                {validResults.map((result: TestResult, idx: number) => (
                    <div key={idx} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                        <div className="flex items-start justify-between">
                            <div className="flex-1">
                                <div className="flex items-center space-x-2">
                                    {result.passed ? (
                                        <svg className="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                                        </svg>
                                    ) : (
                                        <svg className="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                        </svg>
                                    )}
                                    <h4 className="font-medium text-gray-900 dark:text-gray-100">{result.test_name}</h4>
                                    <span className="text-xs text-gray-400">({result.duration_ms}ms)</span>
                                </div>
                                {result.error && (
                                    <div className="mt-2 text-sm text-red-600 bg-red-50 dark:bg-red-900/20 p-2 rounded">
                                        Error: {result.error}
                                    </div>
                                )}
                                {result.logs && result.logs.length > 0 && (
                                     <div className="mt-2 text-xs font-mono text-gray-500 bg-gray-50 dark:bg-gray-900 p-2 rounded max-h-32 overflow-y-auto">
                                        {result.logs.map((log, i) => <div key={i}>{log}</div>)}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default TestResultsTab;
