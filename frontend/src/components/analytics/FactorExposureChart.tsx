import React from 'react';
import {
  Radar,
  RadarChart,
  PolarGrid,
  PolarAngleAxis,
  PolarRadiusAxis,
  ResponsiveContainer,
  Tooltip,
} from 'recharts';

interface FactorExposureProps {
  betas: Record<string, number>;
}

export const FactorExposureChart: React.FC<FactorExposureProps> = ({ betas }) => {
  const data = Object.entries(betas).map(([factor, beta]) => ({
    factor,
    beta,
    fullMark: 1.5, // Scale max
  }));

  return (
    <div className="h-64 w-full">
      <h3 className="text-lg font-semibold mb-2">Factor Exposures (Beta)</h3>
      <ResponsiveContainer width="100%" height="100%">
        <RadarChart cx="50%" cy="50%" outerRadius="80%" data={data}>
          <PolarGrid />
          <PolarAngleAxis dataKey="factor" />
          <PolarRadiusAxis angle={30} domain={[-0.5, 1.5]} />
          <Radar
            name="Portfolio"
            dataKey="beta"
            stroke="#8884d8"
            fill="#8884d8"
            fillOpacity={0.6}
          />
          <Tooltip />
        </RadarChart>
      </ResponsiveContainer>
    </div>
  );
};
