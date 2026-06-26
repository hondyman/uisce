import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, ReferenceLine } from 'recharts';

type Exposure = {
  factor_name: string;
  beta: number;
  t_stat: number;
};

type Props = {
  data: Exposure[];
};

export const FactorExposureChart: React.FC<Props> = ({ data }) => {
  return (
    <div className="h-80 w-full bg-white dark:bg-gray-800 p-4 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
      <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">Factor Exposures (Betas)</h3>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart
          data={data}
          layout="vertical"
          margin={{ top: 5, right: 30, left: 40, bottom: 5 }}
        >
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis type="number" domain={[-1, 1]} />
          <YAxis dataKey="factor_name" type="category" width={80} />
          <Tooltip 
            contentStyle={{ backgroundColor: '#1f2937', borderColor: '#374151', color: '#f3f4f6' }}
            itemStyle={{ color: '#f3f4f6' }}
            cursor={{fill: 'transparent'}}
          />
          <Legend />
          <ReferenceLine x={0} stroke="#9ca3af" />
          <Bar dataKey="beta" fill="#3b82f6" name="Beta" />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
};
