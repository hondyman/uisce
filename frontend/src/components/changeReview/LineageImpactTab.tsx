import React from 'react';
import { ImpactReport } from '../../types/changeReview';

interface LineageImpactTabProps {
    impacts: Record<string, ImpactReport>;
}

const LineageImpactTab: React.FC<LineageImpactTabProps> = ({ impacts }) => {
    if (!impacts || Object.keys(impacts).length === 0) {
        return <div className="p-4 text-gray-500">No lineage impact detected.</div>;
    }

    return (
        <div className="space-y-6">
            {Object.entries(impacts).map(([objectId, report]) => (
                <div key={objectId} className="border rounded-lg p-4 bg-white dark:bg-gray-800 shadow-sm">
                    <h3 className="text-lg font-semibold mb-2">Source: {objectId} <span className="text-sm font-normal text-gray-500">(Impact Score: {report.impact_score})</span></h3>
                    
                    {report.affected_nodes.length === 0 ? (
                        <p className="text-sm text-gray-500">No downstream dependencies affected.</p>
                    ) : (
                        <ul className="space-y-2">
                            {report.affected_nodes.map((node, idx) => (
                                <li key={idx} className="flex items-center space-x-2 p-2 rounded hover:bg-gray-50 dark:hover:bg-gray-700/50">
                                    <span className={`w-2 h-2 rounded-full ${node.is_direct ? 'bg-red-500' : 'bg-orange-400'}`}></span>
                                    <span className="font-medium text-gray-900 dark:text-gray-100">{node.node_id}</span>
                                    <span className="text-xs px-2 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 border border-gray-200 dark:border-gray-600">
                                        {node.node_type}
                                    </span>
                                    {node.is_direct && <span className="text-xs text-red-500 ml-auto">Direct Dependency</span>}
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            ))}
        </div>
    );
};

export default LineageImpactTab;
