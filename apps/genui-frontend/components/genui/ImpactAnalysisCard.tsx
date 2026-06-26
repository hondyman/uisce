"use client";

import { motion } from 'framer-motion';
import { TrendingDown, TrendingUp, Minus } from 'lucide-react';

interface ImpactAnalysisCardProps {
  headline: string;
  affectedSector: string;
  impactScore: number; // 0-100
}

export function ImpactAnalysisCard({ headline, affectedSector, impactScore }: ImpactAnalysisCardProps) {
  const isHighImpact = impactScore > 70;
  const isPositive = impactScore > 50; // Simplified logic

  return (
    <motion.div 
      initial={{ x: -20, opacity: 0 }}
      animate={{ x: 0, opacity: 1 }}
      className="bg-white rounded-xl shadow-md border border-gray-200 overflow-hidden my-4"
    >
      <div className="p-4 border-b border-gray-100 flex justify-between items-center bg-gray-50">
        <span className="text-xs font-bold uppercase tracking-wider text-gray-500">Impact Analysis</span>
        <span className={`text-xs font-bold px-2 py-1 rounded-full ${isHighImpact ? 'bg-red-100 text-red-700' : 'bg-gray-100 text-gray-700'}`}>
          Score: {impactScore}/100
        </span>
      </div>
      
      <div className="p-5">
        <h3 className="text-lg font-bold text-gray-900 mb-2">{headline}</h3>
        
        <div className="flex items-center gap-4 mt-4">
          <div className="flex-1">
            <div className="text-sm text-gray-500 mb-1">Affected Sector</div>
            <div className="font-medium text-gray-900">{affectedSector}</div>
          </div>
          
          <div className="flex-1">
            <div className="text-sm text-gray-500 mb-1">Projected Impact</div>
            <div className="flex items-center gap-2">
              {isPositive ? (
                <TrendingUp className="w-5 h-5 text-green-500" />
              ) : (
                <TrendingDown className="w-5 h-5 text-red-500" />
              )}
              <span className={`font-medium ${isPositive ? 'text-green-600' : 'text-red-600'}`}>
                {isPositive ? 'Positive' : 'Negative'} Volatility
              </span>
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );
}
