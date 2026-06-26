/**
 * Marketplace Components Data
 * Contains component definitions, categories, and configuration for the component marketplace
 */

export interface ComponentConfig {
  type: string;
  props: Record<string, any>;
}

export interface Component {
  id: string;
  name: string;
  description: string;
  category: string;
  author: string;
  downloads: number;
  rating: number;
  reviews: number;
  version: string;
  tags: string[];
  icon: string;
  price: 'free' | string;
  featured: boolean;
  preview?: string;
  dependencies: string[];
  config: ComponentConfig;
}

export interface Category {
  id: string;
  label: string;
  count: number;
}

export const categories: Category[] = [
  { id: 'all', label: 'All Components', count: 24 },
  { id: 'analytics', label: 'Analytics', count: 8 },
  { id: 'trading', label: 'Trading', count: 5 },
  { id: 'visualization', label: 'Visualization', count: 7 },
  { id: 'collaboration', label: 'Collaboration', count: 4 }
];

export const components: Component[] = [
  {
    id: 'attribution-analysis',
    name: 'Attribution Analysis',
    description: 'Brinson model performance attribution with waterfall charts',
    category: 'analytics',
    author: 'Portfolio Team',
    downloads: 1243,
    rating: 4.8,
    reviews: 45,
    version: '2.1.0',
    tags: ['performance', 'attribution', 'brinson'],
    icon: '📊',
    price: 'free',
    featured: true,
    preview: 'https://example.com/preview.png',
    dependencies: ['recharts', 'lodash'],
    config: {
      type: 'AttributionAnalysis',
      props: {
        portfolioData: 'positions',
        benchmarkData: 'benchmark',
        model: 'brinson'
      }
    }
  },
  {
    id: 'risk-heatmap',
    name: 'Risk Correlation Heatmap',
    description: 'Interactive correlation matrix with color-coded risk levels',
    category: 'analytics',
    author: 'Risk Analytics',
    downloads: 892,
    rating: 4.6,
    reviews: 32,
    version: '1.5.2',
    tags: ['risk', 'correlation', 'heatmap'],
    icon: '🔥',
    price: 'free',
    featured: true,
    dependencies: ['d3'],
    config: {
      type: 'RiskHeatmap',
      props: {
        dataSource: 'correlationMatrix',
        colorScale: { min: '#ef4444', mid: '#fbbf24', max: '#10b981' }
      }
    }
  },
  {
    id: 'trade-blotter-pro',
    name: 'Trade Blotter Pro',
    description: 'Real-time trade execution monitor with FIX protocol support',
    category: 'trading',
    author: 'Trading Solutions',
    downloads: 2156,
    rating: 4.9,
    reviews: 78,
    version: '3.0.1',
    tags: ['trading', 'execution', 'real-time'],
    icon: '⚡',
    price: '$99/mo',
    featured: true,
    dependencies: ['websocket', 'fix-protocol'],
    config: {
      type: 'TradeBlotterPro',
      props: {
        omsIntegration: 'charles-river',
        autoRefresh: true
      }
    }
  },
  {
    id: 'esg-scorecard',
    name: 'ESG Scorecard',
    description: 'Environmental, Social, Governance metrics with peer comparison',
    category: 'analytics',
    author: 'ESG Analytics',
    downloads: 567,
    rating: 4.7,
    reviews: 23,
    version: '1.2.0',
    tags: ['esg', 'sustainability', 'impact'],
    icon: '🌱',
    price: 'free',
    featured: false,
    dependencies: ['msci-api'],
    config: {
      type: 'ESGScorecard',
      props: {
        provider: 'msci',
        metrics: ['esg-score', 'carbon-intensity']
      }
    }
  },
  {
    id: 'candlestick-chart-pro',
    name: 'Candlestick Chart Pro',
    description: 'Advanced OHLC chart with 50+ technical indicators',
    category: 'visualization',
    author: 'Chart Masters',
    downloads: 3421,
    rating: 4.9,
    reviews: 156,
    version: '4.2.0',
    tags: ['charting', 'technical', 'indicators'],
    icon: '📈',
    price: '$49/mo',
    featured: true,
    dependencies: ['tradingview-lightweight-charts'],
    config: {
      type: 'CandlestickChartPro',
      props: {
        indicators: ['sma', 'ema', 'rsi', 'macd'],
        drawingTools: true
      }
    }
  },
  {
    id: 'investor-portal',
    name: 'Investor Portal',
    description: 'White-labeled client dashboard with document management',
    category: 'collaboration',
    author: 'Client Solutions',
    downloads: 445,
    rating: 4.5,
    reviews: 18,
    version: '2.0.0',
    tags: ['client', 'portal', 'white-label'],
    icon: '👥',
    price: '$199/mo',
    featured: false,
    dependencies: ['auth0', 'docusign'],
    config: {
      type: 'InvestorPortal',
      props: {
        whiteLabel: true,
        messaging: true
      }
    }
  },
  {
    id: 'scenario-simulator',
    name: 'Scenario Simulator',
    description: 'Monte Carlo simulations and stress testing',
    category: 'analytics',
    author: 'Quant Team',
    downloads: 789,
    rating: 4.8,
    reviews: 41,
    version: '1.8.0',
    tags: ['simulation', 'stress-test', 'monte-carlo'],
    icon: '🎲',
    price: 'free',
    featured: false,
    dependencies: ['mathjs'],
    config: {
      type: 'ScenarioSimulator',
      props: {
        simulations: 10000,
        confidenceLevel: 0.95
      }
    }
  },
  {
    id: 'nlp-query-engine',
    name: 'NLP Query Engine',
    description: 'Ask questions in plain English, powered by AI',
    category: 'analytics',
    author: 'AI Labs',
    downloads: 1567,
    rating: 4.7,
    reviews: 89,
    version: '1.0.0',
    tags: ['ai', 'nlp', 'chatbot'],
    icon: '🤖',
    price: '$299/mo',
    featured: true,
    dependencies: ['openai-api'],
    config: {
      type: 'NLPQueryEngine',
      props: {
        model: 'gpt-4',
        context: ['positions', 'trades', 'performance']
      }
    }
  },
  {
    id: 'portfolio-optimizer',
    name: 'Portfolio Optimizer',
    description: 'Mean-variance optimization with constraints',
    category: 'analytics',
    author: 'Quant Team',
    downloads: 654,
    rating: 4.8,
    reviews: 38,
    version: '2.3.0',
    tags: ['optimization', 'portfolio', 'analytics'],
    icon: '🎯',
    price: 'free',
    featured: false,
    dependencies: ['numeric.js', 'scipy'],
    config: {
      type: 'PortfolioOptimizer',
      props: {
        algorithm: 'efficient-frontier',
        constraints: ['position-limits', 'sector-limits']
      }
    }
  }
];
