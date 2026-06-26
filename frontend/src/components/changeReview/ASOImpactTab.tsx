import React from 'react';
import { ChangeReview } from '../../types/changeReview';

interface ASOImpactTabProps {
    review: ChangeReview;
}

const ASOImpactTab: React.FC<ASOImpactTabProps> = ({ review }) => {
    // ASO Impact is simpler for now, maybe just "Invalidated caches" message
    // If backend returns specific ASO details later, update this.
    // For now, based on lineage impact, we infer ASO invalidation.
    
    // In future: parse detailed ASO impact report if available.
    
    return (
        <div className="p-4 bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
            <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">Automated Semantic Optimization (ASO) Impact</h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                The following optimization artifacts will be invalidated and rebuilt upon promotion.
            </p>
            
            <div className="space-y-4">
                 <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 p-4 rounded-md">
                    <h4 className="text-sm font-semibold text-blue-800 dark:text-blue-300">Cache Invalidation Strategy</h4>
                    <p className="text-sm text-blue-700 dark:text-blue-400 mt-1">
                        All downstream pre-aggregations for affected Business Objects will be marked as STALE.
                    </p>
                </div>

                {/* Potentially list affected ASO candidates if lineage report includes them */}
                {review.lineage_impact && Object.keys(review.lineage_impact).length > 0 && (
                    <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
                        <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Affected Objects Requiring Re-Optimization:</h4>
                        <ul className="list-disc list-inside text-sm text-gray-600 dark:text-gray-400 space-y-1">
                             {Object.values(review.lineage_impact).flatMap(report => report.affected_nodes).map((node, i) => (
                                <li key={i}>{node.node_id} ({node.node_type})</li>
                             ))}
                        </ul>
                    </div>
                )}
            </div>
        </div>
    );
};

export default ASOImpactTab;
