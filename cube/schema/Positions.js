/**
 * Positions Cube - Current Holdings from Tenant PostgreSQL
 * 
 * This cube reads from isolated tenant PostgreSQL databases
 * for financial data that requires strict tenant isolation.
 * 
 * Pre-aggregations cached in StarRocks for cross-tenant analytics.
 */

cube('Positions', {
  sql: `SELECT * FROM positions`,
  
  // Dynamic tenant database (resolved at query time)
  dataSource: 'tenant_financial',

  // ============================================================================
  // PRE-AGGREGATIONS
  // ============================================================================
  
  preAggregations: {
    // Portfolio summary - refreshed frequently
    portfolioSummary: {
      measures: [
        CUBE.positionCount,
        CUBE.totalMarketValue,
        CUBE.totalCostBasis,
        CUBE.totalUnrealizedPnl,
      ],
      dimensions: [
        CUBE.portfolioId,
        CUBE.currency,
      ],
      timeDimension: CUBE.asOfDate,
      granularity: 'day',
      refreshKey: {
        every: '5 minute',
      },
      external: true,
    },

    // Security breakdown
    bySecurityType: {
      measures: [
        CUBE.positionCount,
        CUBE.totalMarketValue,
      ],
      dimensions: [
        CUBE.portfolioId,
        CUBE.securityId,
      ],
      refreshKey: {
        every: '15 minute',
      },
      external: true,
    },
  },

  // ============================================================================
  // JOINS
  // ============================================================================
  
  joins: {
    Portfolio: {
      relationship: 'belongsTo',
      sql: `${CUBE}.portfolio_id = ${Portfolio}.id`,
    },
    Security: {
      relationship: 'belongsTo',
      sql: `${CUBE}.security_id = ${Security}.id`,
    },
  },

  // ============================================================================
  // MEASURES
  // ============================================================================
  
  measures: {
    positionCount: {
      type: 'count',
    },
    
    totalQuantity: {
      sql: `quantity`,
      type: 'sum',
    },
    
    totalMarketValue: {
      sql: `market_value`,
      type: 'sum',
      format: 'currency',
    },
    
    totalCostBasis: {
      sql: `cost_basis`,
      type: 'sum',
      format: 'currency',
    },
    
    totalUnrealizedPnl: {
      sql: `unrealized_pnl`,
      type: 'sum',
      format: 'currency',
    },
    
    avgUnrealizedPnlPct: {
      sql: `unrealized_pnl_pct`,
      type: 'avg',
      format: 'percent',
    },
    
    weightedAvgCost: {
      sql: `SUM(average_cost * quantity) / NULLIF(SUM(quantity), 0)`,
      type: 'number',
    },
  },

  // ============================================================================
  // DIMENSIONS
  // ============================================================================
  
  dimensions: {
    id: {
      sql: `id`,
      type: 'string',
      primaryKey: true,
    },
    
    portfolioId: {
      sql: `portfolio_id`,
      type: 'string',
    },
    
    securityId: {
      sql: `security_id`,
      type: 'string',
    },
    
    quantity: {
      sql: `quantity`,
      type: 'number',
    },
    
    averageCost: {
      sql: `average_cost`,
      type: 'number',
      format: 'currency',
    },
    
    costBasis: {
      sql: `cost_basis`,
      type: 'number',
      format: 'currency',
    },
    
    marketValue: {
      sql: `market_value`,
      type: 'number',
      format: 'currency',
    },
    
    unrealizedPnl: {
      sql: `unrealized_pnl`,
      type: 'number',
      format: 'currency',
    },
    
    unrealizedPnlPct: {
      sql: `unrealized_pnl_pct`,
      type: 'number',
      format: 'percent',
    },
    
    currency: {
      sql: `currency`,
      type: 'string',
    },
    
    asOfDate: {
      sql: `as_of_date`,
      type: 'time',
    },
  },
});

// ============================================================================
// PORTFOLIO CUBE
// ============================================================================

cube('Portfolio', {
  sql: `SELECT * FROM portfolios`,
  dataSource: 'tenant_financial',
  
  measures: {
    count: {
      type: 'count',
    },
  },
  
  dimensions: {
    id: {
      sql: `id`,
      type: 'string',
      primaryKey: true,
    },
    
    externalId: {
      sql: `external_id`,
      type: 'string',
    },
    
    portfolioName: {
      sql: `portfolio_name`,
      type: 'string',
    },
    
    portfolioType: {
      sql: `portfolio_type`,
      type: 'string',
    },
    
    benchmark: {
      sql: `benchmark`,
      type: 'string',
    },
    
    status: {
      sql: `status`,
      type: 'string',
    },
  },
});

// ============================================================================
// SECURITY CUBE
// ============================================================================

cube('Security', {
  sql: `SELECT * FROM securities`,
  dataSource: 'tenant_financial',
  
  measures: {
    count: {
      type: 'count',
    },
  },
  
  dimensions: {
    id: {
      sql: `id`,
      type: 'string',
      primaryKey: true,
    },
    
    symbol: {
      sql: `symbol`,
      type: 'string',
    },
    
    cusip: {
      sql: `cusip`,
      type: 'string',
    },
    
    isin: {
      sql: `isin`,
      type: 'string',
    },
    
    securityName: {
      sql: `security_name`,
      type: 'string',
    },
    
    securityType: {
      sql: `security_type`,
      type: 'string',
    },
    
    exchange: {
      sql: `exchange`,
      type: 'string',
    },
    
    currency: {
      sql: `currency`,
      type: 'string',
    },
    
    sector: {
      sql: `sector`,
      type: 'string',
    },
    
    industry: {
      sql: `industry`,
      type: 'string',
    },
    
    country: {
      sql: `country`,
      type: 'string',
    },
  },
});
