import React, { useState, useEffect } from 'react';
import { Sparkles, AlertTriangle, CheckCircle } from 'lucide-react';

interface TradeExplanationProps {
  tradeId: string;
  ticker: string;
  action: 'BUY' | 'SELL';
  reason: string; // e.g., "Tax Loss Harvesting", "Rebalancing"
}

export const GenerativeTradeExplanation: React.FC<TradeExplanationProps> = ({ tradeId, ticker, action, reason }) => {
  const [loading, setLoading] = useState(true);
  const [explanation, setExplanation] = useState<string | null>(null);

  useEffect(() => {
    // Simulate AI generation delay
    setLoading(true);
    const timer = setTimeout(() => {
      // Mock AI response based on reason
      let text = "";
      if (reason === "Tax Loss Harvesting") {
        text = `We sold ${ticker} to realize a loss of approximately $1,200, which can be used to offset gains elsewhere in the portfolio. To maintain market exposure, we simultaneously purchased a correlated ETF. This action is projected to increase after-tax returns by 0.15% this year.`;
      } else if (reason === "Drift") {
        text = `The portfolio's allocation to ${ticker} had drifted 5% above target due to recent market movements. We trimmed the position to bring the portfolio back within the risk tolerance bands defined in the Investment Policy Statement.`;
      } else {
        text = `Executed ${action} order for ${ticker} as part of the regular rebalancing schedule to align with the target index model.`;
      }
      setExplanation(text);
      setLoading(false);
    }, 1500);

    return () => clearTimeout(timer);
  }, [tradeId, reason, ticker, action]);

  return (
    <div className="p-4 bg-slate-50 border border-slate-200 rounded-lg shadow-sm">
      <div className="flex items-center gap-2 mb-2">
        <Sparkles className="w-5 h-5 text-purple-600" />
        <h3 className="font-semibold text-slate-800">AI Trade Analysis</h3>
      </div>
      
      {loading ? (
        <div className="animate-pulse flex space-x-4">
          <div className="flex-1 space-y-4 py-1">
            <div className="h-4 bg-slate-200 rounded w-3/4"></div>
            <div className="space-y-2">
              <div className="h-4 bg-slate-200 rounded"></div>
              <div className="h-4 bg-slate-200 rounded w-5/6"></div>
            </div>
          </div>
        </div>
      ) : (
        <div className="text-slate-700 text-sm leading-relaxed">
          {explanation}
        </div>
      )}
      
      <div className="mt-3 flex items-center gap-2 text-xs text-slate-500">
        <CheckCircle className="w-3 h-3" />
        <span>Verified by Compliance Engine</span>
      </div>
    </div>
  );
};
