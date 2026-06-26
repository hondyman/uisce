import React from 'react';

interface AttributionResult {
  TotalReturn: number;
  AlphaContribution: number;
  FactorContributions: Record<string, number>;
  Residual: number;
}

interface AttributionTableProps {
  data: AttributionResult;
}

export const AttributionTable: React.FC<AttributionTableProps> = ({ data }) => {
  const formatPercent = (val: number) => (val * 100).toFixed(2) + '%';

  return (
    <div className="overflow-x-auto">
      <h3 className="text-lg font-semibold mb-2">Performance Attribution</h3>
      <table className="min-w-full bg-white border border-gray-200">
        <thead>
          <tr className="bg-gray-50">
            <th className="px-4 py-2 text-left text-sm font-medium text-gray-500">Source</th>
            <th className="px-4 py-2 text-right text-sm font-medium text-gray-500">Contribution</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200">
          <tr>
            <td className="px-4 py-2 text-sm font-medium text-gray-900">Alpha (Idiosyncratic)</td>
            <td className="px-4 py-2 text-right text-sm text-green-600 font-bold">
              {formatPercent(data.AlphaContribution)}
            </td>
          </tr>
          {Object.entries(data.FactorContributions).map(([factor, contribution]) => (
            <tr key={factor}>
              <td className="px-4 py-2 text-sm text-gray-700">Factor: {factor}</td>
              <td className={`px-4 py-2 text-right text-sm ${contribution >= 0 ? 'text-gray-900' : 'text-red-600'}`}>
                {formatPercent(contribution)}
              </td>
            </tr>
          ))}
          <tr className="bg-gray-50 font-bold">
            <td className="px-4 py-2 text-sm text-gray-900">Total Return</td>
            <td className="px-4 py-2 text-right text-sm text-gray-900">
              {formatPercent(data.TotalReturn)}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  );
};
