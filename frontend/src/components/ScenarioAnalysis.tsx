
import React, { useState } from 'react';
import { gql, useSubscription } from '@apollo/client';
import styles from './ScenarioAnalysis.module.css';

const PORTFOLIOS_SUBSCRIPTION = gql`
  subscription { 
    portfolios {
      id
      aum
      sharpe
      risk
      status
    }
  }
`;

const ScenarioAnalysis: React.FC = () => {
  const { data: portfolioData, loading: portfolioLoading, error: portfolioError } = useSubscription(PORTFOLIOS_SUBSCRIPTION);
  const [selectedPortfolio, setSelectedPortfolio] = useState<string>("");
  const [selectedScenario, setSelectedScenario] = useState<string>("");
  const [analysisResult, setAnalysisResult] = useState<any>(null);
  const [loadingAnalysis, setLoadingAnalysis] = useState(false);

  const scenarios = [
    "Market Crash (-20%)",
    "Interest Rate Hike (+2%)",
    "High Inflation (+5%)",
    "Tech Bubble Burst (-30% on tech stocks)",
    "Geopolitical Crisis"
  ];

  const handleRunAnalysis = async () => {
    if (!selectedPortfolio || !selectedScenario) return;

    setLoadingAnalysis(true);
    setAnalysisResult(null);

    try {
      const response = await fetch(`/api/portfolio/${selectedPortfolio}/scenario`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ scenario: selectedScenario })
      });

      if (!response.ok) {
        throw new Error('Failed to run analysis');
      }

      const result = await response.json();
      setAnalysisResult(result);
    } catch (error) {
      console.error(error);
    } finally {
      setLoadingAnalysis(false);
    }
  };

  const basePortfolio = portfolioData?.portfolios.find((p: any) => p.id === selectedPortfolio);

  return (
    <div className={styles.container}>
      <div className={styles.panel}>
        <h2>Scenario Analysis</h2>
        
        <div style={{ marginBottom: '1rem' }}>
          <label htmlFor="portfolio-select">Portfolio:</label>
          <select id="portfolio-select" value={selectedPortfolio} onChange={(e) => setSelectedPortfolio(e.target.value)} className="w-full p-2">
            <option value="">Select a portfolio</option>
            {portfolioData?.portfolios.map((p: any) => (
              <option key={p.id} value={p.id}>Portfolio {p.id} - ${p.aum.toLocaleString()}</option>
            ))}
          </select>
        </div>

        <div style={{ marginBottom: '1rem' }}>
          <label htmlFor="scenario-select">Scenario:</label>
          <select id="scenario-select" value={selectedScenario} onChange={(e) => setSelectedScenario(e.target.value)} className="w-full p-2">
            <option value="">Select a scenario</option>
            {scenarios.map(s => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
        </div>

        <button onClick={handleRunAnalysis} disabled={!selectedPortfolio || !selectedScenario || loadingAnalysis} className="p-2 cursor-pointer">
          {loadingAnalysis ? 'Running...' : 'Run Analysis'}
        </button>
      </div>
      <div className={styles.results}>
        <h3>Analysis Results: {selectedScenario}</h3>
        {loadingAnalysis && <p>Loading analysis...</p>}
        {analysisResult && (
          <div className={styles.resultsContainer}>
            <div className={styles.resultCard}>
              <h4>Base Case</h4>
              {basePortfolio && (
                <>
                  <p>AUM: <strong>${basePortfolio.aum.toLocaleString()}</strong></p>
                  <p>Sharpe: <strong>{basePortfolio.sharpe}</strong></p>
                  <p>Risk: <strong>{basePortfolio.risk}%</strong></p>
                  <p>Status: <strong>{basePortfolio.status}</strong></p>
                </>
              )}
            </div>
            <div className={styles.resultCard}>
              <h4>Scenario Case</h4>
              <p>Projected AUM: <strong>${analysisResult.aum.toLocaleString()}</strong></p>
              <p>Projected Sharpe: <strong>{analysisResult.sharpe}</strong></p>
              <p>Projected Risk: <strong>{analysisResult.risk}%</strong></p>
              <p>Projected Status: <strong>{analysisResult.status}</strong></p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ScenarioAnalysis;
