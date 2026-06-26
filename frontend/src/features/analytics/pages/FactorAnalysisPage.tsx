import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { FactorExposureChart } from '../../../components/analytics/FactorExposureChart';
import { AttributionTable } from '../../../components/analytics/AttributionTable';

// Mock API calls for now - in real app, use fetch/axios calling the Go backend
const fetchExposure = async (portfolioID: string) => {
  // Simulate API delay
  await new Promise(resolve => setTimeout(resolve, 500));
  // In production: const res = await fetch(`/api/analytics/factors/exposure/${portfolioID}`); return res.json();
  return {
    portfolio_id: portfolioID,
    betas: {
      "Market": 1.1,
      "Size": 0.4,
      "Value": -0.2,
    },
    r_squared: 0.85,
  };
};

const fetchAttribution = async (portfolioID: string) => {
  await new Promise(resolve => setTimeout(resolve, 500));
  // In production: const res = await fetch(`/api/analytics/factors/attribution/${portfolioID}`); return res.json();
  return {
    TotalReturn: 0.045,
    AlphaContribution: 0.012,
    FactorContributions: {
      "Market": 0.025,
      "Size": 0.005,
      "Value": 0.003,
    },
    Residual: 0.000,
  };
};

export const FactorAnalysisPage: React.FC = () => {
  const { portfolioID } = useParams<{ portfolioID: string }>();
  const [exposure, setExposure] = useState<any>(null);
  const [attribution, setAttribution] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Default to a demo portfolio if no ID provided
    const pid = portfolioID || "demo-portfolio-001";
    
    Promise.all([
      fetchExposure(pid),
      fetchAttribution(pid)
    ]).then(([expData, attrData]) => {
      setExposure(expData);
      setAttribution(attrData);
      setLoading(false);
    });
  }, [portfolioID]);

  if (loading) return <div className="p-8">Loading analytics...</div>;

  return (
    <div className="p-8 bg-gray-50 min-h-screen">
      <h1 className="text-2xl font-bold mb-6">Factor Analytics: {portfolioID || "Demo Portfolio"}</h1>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        <div className="bg-white p-6 rounded-lg shadow">
          <FactorExposureChart betas={exposure.betas} />
          <div className="mt-4 text-sm text-gray-500 text-center">
            R-Squared: {(exposure.r_squared * 100).toFixed(1)}%
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow">
          <AttributionTable data={attribution} />
        </div>
      </div>
    </div>
  );
};
