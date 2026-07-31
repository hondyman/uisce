import React, { useState } from 'react';
import { Lightbulb, ArrowRight, Sparkles, AlertCircle } from 'lucide-react';
import ArrowForward from '@mui/icons-material/ArrowForward';
import Close from '@mui/icons-material/Close';
import ThumbDown from '@mui/icons-material/ThumbDown';
import ThumbUp from '@mui/icons-material/ThumbUp';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

export interface RecommendationItem {
  type: 'FOLLOW_UP_QUERY' | 'INSIGHT_CHECK' | 'GRAPH_TRAVERSAL';
  label: string;
  confidence_score: number;
  semantic_intent: Record<string, any>;
}

interface AIRecommendationBarProps {
  prompt?: string;
  boKeys?: string[];
  recommendations?: RecommendationItem[];
  onSelectRecommendation?: (rec: RecommendationItem) => void;
}

const ERROR_CATEGORIES = [
  { id: 'WRONG_TABLE', label: 'Wrong Table / Business Object' },
  { id: 'INCORRECT_FORMULA', label: 'Incorrect Calculation / Formula' },
  { id: 'MISSING_DATA', label: 'Missing or Incorrectly Filtered Data' },
  { id: 'HALLUCINATED_SCHEMA', label: 'Hallucinated Schema or Columns' },
];

export const AIRecommendationBar: React.FC<AIRecommendationBarProps> = ({
  prompt,
  boKeys = ['Customer'],
  recommendations: initialRecommendations,
  onSelectRecommendation,
}) => {
  const [recommendations] = useState<RecommendationItem[]>(
    initialRecommendations || [
      {
        type: 'FOLLOW_UP_QUERY',
        label: 'Show me top orders by freight amount for these customers',
        confidence_score: 0.92,
        semantic_intent: { bo: 'Order', metrics: ['freight_amount'], dimensions: ['customer_company_name'] },
      },
      {
        type: 'INSIGHT_CHECK',
        label: 'Detect anomalies in customer discount percentages',
        confidence_score: 0.85,
        semantic_intent: { bo: 'OrderLine', metrics: ['avg_discount'] },
      },
    ]
  );
  const [feedbackSent, setFeedbackSent] = useState<Record<string, 'up' | 'down'>>({});
  const [activeSurveyLabel, setActiveSurveyLabel] = useState<string | null>(null);

  const handleFeedback = async (
    label: string,
    action: 'THUMBS_UP' | 'THUMBS_DOWN' | 'CLICKED',
    errorCategory?: string
  ) => {
    setFeedbackSent((prev) => ({
      ...prev,
      [label]: action === 'THUMBS_UP' || action === 'CLICKED' ? 'up' : 'down',
    }));

    if (action === 'THUMBS_DOWN' && !errorCategory) {
      setActiveSurveyLabel(label);
    } else {
      setActiveSurveyLabel(null);
    }

    try {
      await fetch('/api/ai/feedback', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          bo_key: boKeys[0] || 'General',
          recommendation_label: label,
          action,
          error_category: errorCategory,
        }),
      });
    } catch (e) {
      console.warn('Feedback submission error:', e);
    }
  };

  const handleSelect = (rec: RecommendationItem) => {
    handleFeedback(rec.label, 'CLICKED');
    if (onSelectRecommendation) {
      onSelectRecommendation(rec);
    }
  };

  return (
    <div className="bg-slate-900 border border-indigo-500/30 rounded-lg p-4 shadow-lg text-slate-100 my-3 relative">
      <div className="flex items-center justify-between mb-3 border-b border-slate-800 pb-2">
        <div className="flex items-center space-x-2">
          <Sparkles className="w-5 h-5 text-indigo-400 animate-pulse" />
          <span className="font-semibold text-sm tracking-wide text-indigo-200">
            AI Proactive Recommendations & Contextual Insights
          </span>
        </div>
        <span className="text-xs bg-indigo-950 text-indigo-300 border border-indigo-800 px-2 py-0.5 rounded-full font-mono">
          Two-Stage Feedback Active
        </span>
      </div>

      <div className="space-y-2">
        {recommendations.map((rec, idx) => (
          <div key={idx} className="flex flex-col space-y-2">
            <div
              className="flex items-center justify-between p-2.5 rounded-md bg-slate-800/80 hover:bg-slate-800 border border-slate-700/60 transition-all duration-150 group cursor-pointer"
              onClick={() => handleSelect(rec)}
            >
              <div className="flex items-center space-x-3">
                <Lightbulb className="w-4 h-4 text-amber-400 flex-shrink-0" />
                <span className="text-sm text-slate-200 group-hover:text-indigo-300 font-medium">
                  {rec.label}
                </span>
              </div>

              <div className="flex items-center space-x-3">
                <span className="text-xs font-mono text-slate-400 bg-slate-900 px-2 py-0.5 rounded">
                  {(rec.confidence_score * 100).toFixed(0)}% match
                </span>

                <div className="flex items-center space-x-1" onClick={(e) => e.stopPropagation()}>
                  <button
                    onClick={() => handleFeedback(rec.label, 'THUMBS_UP')}
                    className={`p-1 rounded hover:bg-slate-700 transition-colors ${
                      feedbackSent[rec.label] === 'up' ? 'text-emerald-400' : 'text-slate-400'
                    }`}
                    title="Helpful recommendation (Thumbs Up)"
                  >
                    <ThumbUp className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={() => handleFeedback(rec.label, 'THUMBS_DOWN')}
                    className={`p-1 rounded hover:bg-slate-700 transition-colors ${
                      feedbackSent[rec.label] === 'down' ? 'text-rose-400' : 'text-slate-400'
                    }`}
                    title="Unhelpful recommendation (Thumbs Down)"
                  >
                    <ThumbDown className="w-3.5 h-3.5" />
                  </button>
                </div>

                <ArrowForward className="w-4 h-4 text-slate-500 group-hover:text-indigo-400 transition-transform group-hover:translate-x-1" />
              </div>
            </div>

            {/* Stage 2: Contextual Micro-Survey on Negative Feedback */}
            {activeSurveyLabel === rec.label && (
              <div className="bg-slate-950 border border-rose-500/40 rounded-md p-3 ml-6 text-xs space-y-2 animate-fadeIn">
                <div className="flex items-center justify-between text-rose-300 font-medium">
                  <div className="flex items-center space-x-1.5">
                    <ErrorOutlineIcon className="w-4 h-4 text-rose-400" />
                    <span>Help us improve: What went wrong with this recommendation?</span>
                  </div>
                  <button
                    onClick={() => setActiveSurveyLabel(null)}
                    className="text-slate-400 hover:text-slate-200"
                  >
                    <Close className="w-3.5 h-3.5" />
                  </button>
                </div>
                <div className="grid grid-cols-2 gap-1.5 pt-1">
                  {ERROR_CATEGORIES.map((cat) => (
                    <button
                      key={cat.id}
                      onClick={() => handleFeedback(rec.label, 'THUMBS_DOWN', cat.id)}
                      className="text-left px-2 py-1 rounded bg-slate-900 hover:bg-rose-950/60 border border-slate-800 hover:border-rose-800 text-slate-300 transition-colors"
                    >
                      {cat.label}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};
