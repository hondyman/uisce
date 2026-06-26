# Cube.js Project Template for Semlayer

> **Quick Start Guide** for creating new tenant Cube.js integrations

## 📁 Directory Structure

```
tenant-cube-project/
├── cube/
│   ├── schema/
│   │   ├── Accounts.js              # Core account cube
│   │   ├── Transactions.js          # Transaction cube with pre-aggs
│   │   ├── Portfolios.js            # Portfolio cube
│   │   └── _shared/
│   │       ├── dimensions.js        # Reusable dimension definitions
│   │       └── measures.js          # Reusable measure definitions
│   ├── preaggregations/
│   │   ├── daily_rollups.yaml       # Daily pre-aggregation configs
│   │   └── monthly_rollups.yaml     # Monthly pre-aggregation configs
│   └── cube.js                      # Cube.js configuration
├── config/
│   ├── tenant.yaml                  # Tenant-specific configuration
│   └── datasources.yaml             # Datasource connection config
├── tests/
│   ├── schema_test.go               # Schema validation tests
│   └── preagg_test.go               # Pre-aggregation tests
├── .env.example                     # Environment variable template
└── README.md                        # Project-specific readme
```

## 🚀 Quick Setup

### 1. Copy Template

```bash
# Clone the template
cp -r templates/cube-project ./tenants/<tenant-name>

# Navigate to new project
cd tenants/<tenant-name>
```

### 2. Configure Environment

```bash
# Copy environment template
cp .env.example .env

# Required variables:
export CUBE_DB_HOST=<starrocks-host>
export CUBE_DB_PORT=9030
export CUBE_DB_NAME=<tenant_database>
export CUBE_DB_USER=<service_account>
export CUBE_DB_PASS=<password>
export CUBE_CACHE_HOST=<redis-host>
export TENANT_ID=<uuid>
export DATASOURCE_ID=<uuid>
```

### 3. Configure Tenant

Edit `config/tenant.yaml`:

```yaml
tenant:
  id: "00000000-0000-0000-0000-000000000000"
  name: "Acme Wealth"
  product: "wealth-management"
  
datasource:
  id: "11111111-1111-1111-1111-111111111111"
  type: "starrocks"
  catalog: "tenant_acme"
  
cube:
  refresh_key_interval: "1 hour"
  max_pre_aggregation_partitions: 500
  scheduled_refresh: true
```

## 📊 Schema Templates

### Basic Cube Definition

```javascript
// cube/schema/Accounts.js
cube('Accounts', {
  sql: `SELECT * FROM ${CUBE}.accounts WHERE tenant_id = '${TENANT_ID}'`,
  
  // Enable multi-tenancy
  multiTenantCompile: true,
  
  dimensions: {
    id: {
      sql: `account_id`,
      type: `string`,
      primaryKey: true
    },
    accountType: {
      sql: `account_type`,
      type: `string`
    },
    createdAt: {
      sql: `created_at`,
      type: `time`
    }
  },
  
  measures: {
    count: {
      type: `count`
    },
    totalBalance: {
      sql: `balance`,
      type: `sum`,
      format: `currency`
    },
    avgBalance: {
      sql: `balance`,
      type: `avg`,
      format: `currency`
    }
  },
  
  // Pre-aggregations for performance
  preAggregations: {
    accountsByTypeDaily: {
      type: `rollup`,
      measureReferences: [count, totalBalance],
      dimensionReferences: [accountType],
      timeDimensionReference: createdAt,
      granularity: `day`,
      partitionGranularity: `month`,
      refreshKey: {
        every: `1 hour`
      }
    }
  }
});
```

### Transaction Cube with Relationships

```javascript
// cube/schema/Transactions.js
cube('Transactions', {
  sql: `SELECT * FROM ${CUBE}.transactions WHERE tenant_id = '${TENANT_ID}'`,
  
  joins: {
    Accounts: {
      relationship: `belongsTo`,
      sql: `${CUBE}.account_id = ${Accounts}.id`
    }
  },
  
  dimensions: {
    id: {
      sql: `transaction_id`,
      type: `string`,
      primaryKey: true
    },
    type: {
      sql: `transaction_type`,
      type: `string`
    },
    category: {
      sql: `category`,
      type: `string`
    },
    transactionDate: {
      sql: `transaction_date`,
      type: `time`
    }
  },
  
  measures: {
    count: {
      type: `count`
    },
    totalAmount: {
      sql: `amount`,
      type: `sum`,
      format: `currency`
    },
    avgAmount: {
      sql: `amount`,
      type: `avg`,
      format: `currency`
    },
    netFlow: {
      sql: `CASE WHEN transaction_type = 'credit' THEN amount ELSE -amount END`,
      type: `sum`,
      format: `currency`
    }
  },
  
  preAggregations: {
    dailyByCategory: {
      type: `rollup`,
      measureReferences: [count, totalAmount, netFlow],
      dimensionReferences: [type, category],
      timeDimensionReference: transactionDate,
      granularity: `day`,
      partitionGranularity: `month`,
      refreshKey: {
        every: `30 minutes`
      },
      indexes: {
        categoryIndex: {
          columns: [category]
        }
      }
    },
    monthlyRollup: {
      type: `rollup`,
      measureReferences: [count, totalAmount, netFlow],
      dimensionReferences: [type],
      timeDimensionReference: transactionDate,
      granularity: `month`,
      refreshKey: {
        every: `6 hours`
      }
    }
  }
});
```

## 📈 Pre-Aggregation Best Practices

### YAML Configuration

```yaml
# cube/preaggregations/daily_rollups.yaml
preAggregations:
  - cube: Transactions
    name: daily_category_rollup
    type: rollup
    measures:
      - count
      - totalAmount
      - netFlow
    dimensions:
      - type
      - category
    timeDimension: transactionDate
    granularity: day
    partitionGranularity: month
    refreshKey:
      every: 30 minutes
    indexes:
      - columns: [category]
      - columns: [type, category]

  - cube: Accounts
    name: daily_account_summary
    type: rollup
    measures:
      - count
      - totalBalance
    dimensions:
      - accountType
    timeDimension: createdAt
    granularity: day
    partitionGranularity: month
    refreshKey:
      every: 1 hour
```

### Granularity Guidelines

| Query Pattern | Granularity | Partition | Refresh |
|---------------|-------------|-----------|---------|
| Real-time dashboards | hour | day | 5 min |
| Daily reports | day | month | 30 min |
| Monthly trends | month | year | 6 hours |
| Historical analysis | quarter | year | 24 hours |

## 🔧 Configuration Reference

### cube.js Configuration

```javascript
// cube/cube.js
module.exports = {
  contextToAppId: ({ securityContext }) => {
    return `CUBEJS_APP_${securityContext.tenant_id}`;
  },
  
  contextToOrchestratorId: ({ securityContext }) => {
    return `CUBEJS_ORCHESTRATOR_${securityContext.datasource_id}`;
  },
  
  scheduledRefreshContexts: async () => {
    // Return all tenant contexts for scheduled refresh
    const tenants = await getTenantList();
    return tenants.map(t => ({
      securityContext: {
        tenant_id: t.id,
        datasource_id: t.datasource_id
      }
    }));
  },
  
  preAggregationsSchema: ({ securityContext }) => {
    return `pre_aggregations_${securityContext.tenant_id.replace(/-/g, '_')}`;
  },
  
  driverFactory: ({ securityContext }) => {
    return {
      type: 'starrocks',
      database: `tenant_${securityContext.tenant_id.replace(/-/g, '_')}`,
      host: process.env.CUBE_DB_HOST,
      port: parseInt(process.env.CUBE_DB_PORT),
      user: process.env.CUBE_DB_USER,
      password: process.env.CUBE_DB_PASS
    };
  }
};
```

## ✅ Validation Checklist

Before deploying, verify:

- [ ] All cubes have `multiTenantCompile: true`
- [ ] SQL includes `WHERE tenant_id = '${TENANT_ID}'`
- [ ] Pre-aggregations use appropriate granularity
- [ ] Refresh keys align with data freshness requirements
- [ ] Joins are defined correctly
- [ ] Indexes cover common query patterns
- [ ] Tests pass: `go test ./tests/...`

## 🔗 Related Documentation

- [Cube.js Schema Reference](https://cube.dev/docs/schema/reference)
- [Pre-Aggregations Guide](https://cube.dev/docs/caching/pre-aggregations)
- [Semlayer Tenant Setup](../runbooks/tenant-onboarding.md)
- [StarRocks Integration](../runbooks/starrocks-setup.md)

## 📞 Support

- Slack: #semlayer-cube
- On-call: See PagerDuty rotation
- Docs: https://docs.semlayer.io/cube
