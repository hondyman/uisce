import React from 'react';
import { SemanticDiffDTO } from '../../types/changeReview';
import CodeBlock from '../../components/CodeBlock'; // Assuming this exists or use simple pre

interface DiffTabProps {
    diffs: Record<string, SemanticDiffDTO>;
}

const DiffTab: React.FC<DiffTabProps> = ({ diffs }) => {
    if (!diffs || Object.keys(diffs).length === 0) {
        return <div className="p-4 text-gray-500">No semantic differences detected.</div>;
    }

    return (
        <div className="space-y-4">
            {Object.entries(diffs).map(([objectId, diff]) => (
                <div key={objectId} className="border rounded-lg p-4 bg-white dark:bg-gray-800 shadow-sm">
                    <h3 className="text-lg font-semibold mb-2">Object: {objectId}</h3>
                    <div className="space-y-2">
                        {Object.entries(diff).map(([key, changeData]) => (
                            <div key={key}>
                                {/* Assuming SemanticDiffDTO structure: map[id]struct{Changes []SemanticDiffChange} */}
                                {/* Actually, backend returns map[string]SemanticDiffDTO where DTO is map[string]{changes: []} */}
                                {/* Wait, the type definition:
                                    export interface SemanticDiffDTO {
                                        [key: string]: {
                                            changes: SemanticDiffChange[];
                                        };
                                    }
                                    If diffs is Record<string, SemanticDiffDTO>, then for each objectId, 
                                    we have a DTO which maps IDs (usually same ID) to changes.
                                */}
                                {(changeData.changes || []).map((change, idx) => (
                                    <div key={idx} className={`p-2 rounded text-sm font-mono ${
                                        change.type === 'added' ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300' :
                                        change.type === 'removed' ? 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300' :
                                        'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300'
                                    }`}>
                                        <div className="flex justify-between">
                                            <span className="font-bold">{change.type.toUpperCase()}</span>
                                            <span className="opacity-75">{change.path}</span>
                                        </div>
                                        {change.old !== undefined && (
                                            <div className="mt-1">
                                                <span className="text-xs text-gray-500 mr-2">Old:</span>
                                                <span>{JSON.stringify(change.old)}</span>
                                            </div>
                                        )}
                                        {change.new !== undefined && (
                                            <div className="mt-1">
                                                <span className="text-xs text-gray-500 mr-2">New:</span>
                                                <span>{JSON.stringify(change.new)}</span>
                                            </div>
                                        )}
                                    </div>
                                ))}
                            </div>
                        ))}
                    </div>
                </div>
            ))}
        </div>
    );
};

export default DiffTab;
