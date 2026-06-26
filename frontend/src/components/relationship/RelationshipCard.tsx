import type { FC } from 'react';
import LightbulbIcon from '@mui/icons-material/Lightbulb';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import RelationshipActionButton from './RelationshipActionButton';

interface RelationshipCardProps {
  rel: any;
  appearance: {
    badgeClassName: string;
    iconName: string;
    label: string;
    tailwindBadgeClass?: string;
  };
  isPending: (id: string) => boolean;
  handleLink: (rel: any) => Promise<void>;
  handleUnlink: (rel: any) => Promise<void>;
  setSelectedObject: (obj: { type: 'node' | 'edge'; data: any } | null) => void;
}

const RelationshipCard: FC<RelationshipCardProps> = ({ rel, appearance, isPending, handleLink, handleUnlink, setSelectedObject }) => {
  // Get display name or fallback to ID
  const targetName = rel.targetName || rel.targetEntity;
  
  return (
    <div
      key={rel.id}
      className={`relationship-card flex flex-col gap-4 rounded-xl border ${rel.isSuggestion ? 'border-yellow-400' : 'border-slate-200 dark:border-slate-700'} bg-white dark:bg-slate-800 p-4 shadow-sm hover:shadow-md transition-shadow`}
      onClick={() => setSelectedObject({ type: 'node', data: rel })}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className={`relationship-badge ${appearance.badgeClassName}`}>
            {/* The SVGIcon is not being replaced here as per user request to only change link/unlink icons */}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 flex-wrap">
              <h3 className="truncate font-semibold text-slate-900 dark:text-slate-100">{targetName}</h3>
              <span className={`${appearance.tailwindBadgeClass || 'inline-flex items-center rounded-full bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-800'}`}>
                {rel.cardinality || appearance.label}
              </span>
              {rel.isSuggestion && (
                <span className="inline-flex items-center gap-1 rounded-full bg-yellow-100 dark:bg-yellow-900/30 px-2 py-0.5 text-xs font-semibold text-yellow-800 dark:text-yellow-300">
                  <LightbulbIcon className="h-3 w-3" />
                  Suggestion
                </span>
              )}
            </div>
            {rel.description && (
              <p className="text-sm text-slate-600 dark:text-slate-400 mt-1 line-clamp-2">{rel.description}</p>
            )}
            {rel.confidence !== undefined && (
              <p className="text-xs text-slate-500 dark:text-slate-500 mt-1">
                Confidence: <span className="font-semibold">{(rel.confidence * 100).toFixed(0)}%</span>
              </p>
            )}
          </div>
        </div>
      </div>

      <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-200 dark:border-slate-700">
        {rel.isApplied ? (
          <>
            <div className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full bg-green-100 dark:bg-green-900/30 border border-green-300 dark:border-green-700">
              <CheckCircleIcon className="h-4 w-4 text-green-600 dark:text-green-400" />
              <span className="text-xs font-semibold text-green-700 dark:text-green-300">Linked</span>
            </div>
            {!rel.isSuggestion && <RelationshipActionButton variant="unlink" pending={isPending(rel.id)} onClick={() => handleUnlink(rel)} />}
          </>
        ) : (
          <>
            {rel.isSuggestion ? (
              <RelationshipActionButton variant="link" label="Accept" pending={isPending(rel.id)} onClick={() => handleLink(rel)} />
            ) : (
              <RelationshipActionButton variant="link" pending={isPending(rel.id)} onClick={() => handleLink(rel)} />
            )}
          </>
        )}
      </div>
    </div>
  );
};

export default RelationshipCard;
