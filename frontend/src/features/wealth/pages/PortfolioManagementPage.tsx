import React, { useState, useEffect } from 'react';
import { MetaRenderer } from '../../../components/metadata/MetaRenderer';
import { devDebug } from '../../../utils/devLogger';

// Sample metadata-driven Portfolio view definition
const portfolioViewMetadata = {
  id: 'view_portfolio_form',
  type: 'Form' as const,
  sections: [
    [
      { label: 'Portfolio Name', attr: 'name', component: 'Text', required: true },
      { label: 'Portfolio Type', attr: 'portfolio_type', component: 'Select', required: true },
    ],
    [
      { label: 'Description', attr: 'description', component: 'Text', required: false },
    ],
    [
      { label: 'Base Currency', attr: 'base_currency', component: 'Text', required: true, helpText: 'Default: USD' },
      { label: 'Inception Date', attr: 'inception_date', component: 'Date', required: true },
    ],
    [
      { label: 'Benchmark Symbol', attr: 'benchmark_symbol', component: 'Text', required: false, helpText: 'e.g., SPY for S&P 500' },
    ],
  ],
  dataSource: 'bo_portfolio',
  actions: ['Save Portfolio', 'Cancel'],
  theme: {
    font: 'Inter, system-ui, sans-serif',
    textColor: '#1f2937',
    gap: '16px',
  },
};

export const PortfolioManagementPage: React.FC = () => {
  const [portfolioData, setPortfolioData] = useState<Record<string, any>>({
    name: '',
    portfolio_type: 'INVESTMENT',
    description: '',
    base_currency: 'USD',
    inception_date: new Date().toISOString().split('T')[0],
    benchmark_symbol: 'SPY',
  });

  const [portfolios, setPortfolios] = useState<any[]>([]);
  const [selectedPortfolioId, setSelectedPortfolioId] = useState<string | null>(null);

  useEffect(() => {
    // Mock: Fetch portfolios from API
    setPortfolios([
      { id: '1', name: 'Growth Portfolio', portfolio_type: 'INVESTMENT', base_currency: 'USD', total_value: 1250000 },
      { id: '2', name: 'Retirement Fund', portfolio_type: 'RETIREMENT', base_currency: 'USD', total_value: 850000 },
      { id: '3', name: 'Trust Account', portfolio_type: 'TRUST', base_currency: 'USD', total_value: 2500000 },
    ]);
  }, []);

  const handleFieldChange = (attr: string, value: any) => {
    setPortfolioData((prev) => ({ ...prev, [attr]: value }));
  };

  const handleAction = async (action: string) => {
    if (action === 'Save Portfolio') {
      devDebug('Saving portfolio:', portfolioData);
      alert('Portfolio saved successfully! (Mock)');
      
      // In production:
      // const response = await fetch('/api/wealth/portfolios', {
      //   method: 'POST',
      //   headers: { 'Content-Type': 'application/json' },
      //   body: JSON.stringify(portfolioData),
      // });
    } else if (action === 'Cancel') {
      setPortfolioData({
        name: '',
        portfolio_type: 'INVESTMENT',
        description: '',
        base_currency: 'USD',
        inception_date: new Date().toISOString().split('T')[0],
        benchmark_symbol: 'SPY',
      });
    }
  };

  const handleSelectPortfolio = (portfolio: any) => {
    setSelectedPortfolioId(portfolio.id);
    setPortfolioData({
      name: portfolio.name,
      portfolio_type: portfolio.portfolio_type,
      description: portfolio.description || '',
      base_currency: portfolio.base_currency,
      inception_date: portfolio.inception_date || new Date().toISOString().split('T')[0],
      benchmark_symbol: portfolio.benchmark_symbol || 'SPY',
    });
  };

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold text-gray-900 mb-6">Portfolio Management</h1>
        
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Portfolio List */}
          <div className="bg-white rounded-lg shadow p-4">
            <h2 className="text-lg font-semibold mb-4 text-gray-800">Portfolios</h2>
            <div className="space-y-2">
              {portfolios.map((portfolio) => (
                <div
                  key={portfolio.id}
                  onClick={() => handleSelectPortfolio(portfolio)}
                  className={`p-3 rounded cursor-pointer transition-colors ${
                    selectedPortfolioId === portfolio.id
                      ? 'bg-blue-100 border-blue-500 border'
                      : 'bg-gray-50 hover:bg-gray-100'
                  }`}
                >
                  <p className="font-medium text-gray-900">{portfolio.name}</p>
                  <p className="text-sm text-gray-600">{portfolio.portfolio_type}</p>
                  <p className="text-xs text-gray-500 mt-1">
                    ${portfolio.total_value?.toLocaleString()}
                  </p>
                </div>
              ))}
            </div>
            <button className="mt-4 w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700">
              + New Portfolio
            </button>
          </div>

          {/* Metadata-Driven Form */}
          <div className="lg:col-span-2">
            <MetaRenderer
              view={portfolioViewMetadata}
              data={portfolioData}
              onChange={handleFieldChange}
              onAction={handleAction}
            />

            {/* Holdings Preview (Mock) */}
            {selectedPortfolioId && (
              <div className="mt-6 bg-white rounded-lg shadow p-6">
                <h3 className="text-lg font-semibold mb-4 text-gray-800">Holdings</h3>
                <div className="space-y-2">
                  <div className="flex justify-between items-center p-3 bg-gray-50 rounded">
                    <div>
                      <p className="font-medium">AAPL - Apple Inc.</p>
                      <p className="text-sm text-gray-600">100 shares @ $175.50</p>
                    </div>
                    <p className="font-bold text-green-600">$17,550</p>
                  </div>
                  <div className="flex justify-between items-center p-3 bg-gray-50 rounded">
                    <div>
                      <p className="font-medium">MSFT - Microsoft Corp.</p>
                      <p className="text-sm text-gray-600">50 shares @ $335.75</p>
                    </div>
                    <p className="font-bold text-green-600">$16,787.50</p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};