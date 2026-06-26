/**
 * Transactions Cube - Hot/Cold Tiered Storage
 * 
 * Data Sources:
 *   - Hot (StarRocks): Last 90 days, real-time
 *   - Cold (StarRocks on Parquet): Historical data
 * 
 * Pre-aggregations stored in StarRocks (cube_preagg database)
 */

cube('Transactions', {
  // Use the unified view that combines hot + cold
  sql: `SELECT * FROM cube_hot.transactions_all`,
  
  dataSource: 'starrocks',

  // ============================================================================
  // PRE-AGGREGATIONS (Stored in StarRocks, NOT Redis)
  // ============================================================================
  
  preAggregations: {
    // Daily rollup by tenant - most common query pattern
    dailyByTenant: {
      measures: [
        CUBE.count,
        CUBE.totalAmount,
        CUBE.avgAmount,
      ],
      dimensions: [
        CUBE.tenantId,
        CUBE.transactionType,
        CUBE.category,
      ],
      timeDimension: CUBE.transactionDate,
      granularity: 'day',
      partitionGranularity: 'month',
      refreshKey: {
        every: '5 minute',
      },
      // Store in StarRocks, not Redis!
      external: true,
      // Indexes for fast lookups
      indexes: {
        tenantDate: {
          columns: [CUBE.tenantId, CUBE.transactionDate],
        },
      },
    },

    // Hourly for real-time dashboards (hot data only)
    hourlyRealtime: {
      measures: [
        CUBE.count,
        CUBE.totalAmount,
      ],
      dimensions: [
        CUBE.tenantId,
        CUBE.transactionType,
      ],
      timeDimension: CUBE.transactionDate,
      granularity: 'hour',
      partitionGranularity: 'day',
      refreshKey: {
        every: '1 minute',
      },
      external: true,
      // Only last 7 days
      buildRangeStart: {
        sql: `DATE_SUB(CURRENT_DATE(), INTERVAL 7 DAY)`,
      },
      buildRangeEnd: {
        sql: `CURRENT_DATE()`,
      },
    },

    // Monthly summary for reports
    monthlyByCategory: {
      measures: [
        CUBE.count,
        CUBE.totalAmount,
        CUBE.avgAmount,
        CUBE.minAmount,
        CUBE.maxAmount,
      ],
      dimensions: [
        CUBE.tenantId,
        CUBE.transactionType,
        CUBE.category,
        CUBE.currency,
      ],
      timeDimension: CUBE.transactionDate,
      granularity: 'month',
      partitionGranularity: 'year',
      refreshKey: {
        every: '1 hour',
      },
      external: true,
    },
  },

  // ============================================================================
  // MEASURES
  // ============================================================================
  
  measures: {
    count: {
      type: 'count',
    },
    
    totalAmount: {
      sql: `amount`,
      type: 'sum',
      format: 'currency',
    },
    
    avgAmount: {
      sql: `amount`,
      type: 'avg',
      format: 'currency',
    },
    
    minAmount: {
      sql: `amount`,
      type: 'min',
      format: 'currency',
    },
    
    maxAmount: {
      sql: `amount`,
      type: 'max',
      format: 'currency',
    },
    
    netAmount: {
      sql: `CASE WHEN ${CUBE.transactionType} IN ('credit', 'deposit', 'dividend') THEN amount ELSE -amount END`,
      type: 'sum',
      format: 'currency',
    },
  },

  // ============================================================================
  // DIMENSIONS
  // ============================================================================
  
  dimensions: {
    id: {
      sql: `transaction_id`,
      type: 'string',
      primaryKey: true,
    },
    
    tenantId: {
      sql: `tenant_id`,
      type: 'string',
    },
    
    accountId: {
      sql: `account_id`,
      type: 'string',
    },
    
    transactionDate: {
      sql: `transaction_date`,
      type: 'time',
    },
    
    transactionType: {
      sql: `transaction_type`,
      type: 'string',
    },
    
    amount: {
      sql: `amount`,
      type: 'number',
      format: 'currency',
    },
    
    currency: {
      sql: `currency`,
      type: 'string',
    },
    
    category: {
      sql: `category`,
      type: 'string',
    },
    
    merchant: {
      sql: `merchant`,
      type: 'string',
    },
    
    status: {
      sql: `status`,
      type: 'string',
    },
    
    dataTier: {
      sql: `data_tier`,
      type: 'string',
      description: 'hot or cold - indicates data storage tier',
    },
    
    createdAt: {
      sql: `created_at`,
      type: 'time',
    },
  },

  // ============================================================================
  // SEGMENTS (Row-level filtering)
  // ============================================================================
  
  segments: {
    hotData: {
      sql: `${CUBE}.data_tier = 'hot'`,
    },
    
    coldData: {
      sql: `${CUBE}.data_tier = 'cold'`,
    },
    
    credits: {
      sql: `${CUBE}.transaction_type IN ('credit', 'deposit', 'dividend')`,
    },
    
    debits: {
      sql: `${CUBE}.transaction_type IN ('debit', 'withdrawal', 'fee', 'payment')`,
    },
  },
});
