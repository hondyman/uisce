package graphql

// Portfolio GraphQL Operations

// ============================================================================
// Portfolio Queries
// ============================================================================

// GetPortfolioByID fetches a portfolio with accounts and positions
const GetPortfolioWithPositions = `
query GetPortfolioWithPositions($id: String!) {
  portfolios_by_pk(id: $id) {
    id
    name
    code
    type
    strategy
    currency
    inception_date
    target_allocation
    risk_profile
    status
    accounts {
      id
      account_number
      account_type
      status
      positions(order_by: { market_value: desc }) {
        id
        quantity
        cost_basis
        average_cost
        market_value
        unrealized_gain_loss
        day_change
        weight
        as_of_date
        security {
          id
          ticker
          name
          type
          asset_class
          sector
          price
          day_change
          day_change_percent
        }
      }
    }
    benchmark {
      id
      name
      code
      ytd_return
    }
  }
}
`

// ListPortfolios fetches all portfolios for a tenant
const ListPortfolios = `
query ListPortfolios($tenant_id: String!) {
  portfolios(
    where: { tenant_id: { _eq: $tenant_id }, status: { _eq: "active" } }
    order_by: { name: asc }
  ) {
    id
    name
    code
    type
    strategy
    currency
    market_value
    day_change
    ytd_return
    position_count: accounts_aggregate {
      aggregate {
        count
      }
    }
  }
}
`

// GetPortfolioSummary fetches aggregated portfolio data
const GetPortfolioSummary = `
query GetPortfolioSummary($id: String!) {
  portfolios_by_pk(id: $id) {
    id
    name
    currency
    market_value
    cost_basis
    unrealized_gain_loss
    day_change
    ytd_return
    inception_return
  }
  
  positions_aggregate(where: { account: { portfolio_id: { _eq: $id } } }) {
    aggregate {
      count
      sum {
        market_value
        cost_basis
        unrealized_gain_loss
      }
    }
  }
}
`

// ============================================================================
// Position Queries
// ============================================================================

// GetPositionsByPortfolio fetches all positions
const GetPositionsByPortfolio = `
query GetPositionsByPortfolio($portfolio_id: String!) {
  positions(
    where: { account: { portfolio_id: { _eq: $portfolio_id } } }
    order_by: { market_value: desc }
  ) {
    id
    account_id
    security_id
    quantity
    cost_basis
    average_cost
    market_value
    unrealized_gain_loss
    day_change
    weight
    as_of_date
    security {
      id
      ticker
      name
      type
      asset_class
      sector
      industry
      price
    }
    tax_lots {
      id
      acquisition_date
      quantity
      cost_per_share
      holding_period
    }
  }
}
`

// ============================================================================
// Performance Queries
// ============================================================================

// GetPortfolioPerformance fetches performance metrics
const GetPortfolioPerformance = `
query GetPortfolioPerformance($portfolio_id: String!, $periods: [String!]!) {
  performance(
    where: { portfolio_id: { _eq: $portfolio_id }, period: { _in: $periods } }
    order_by: { as_of_date: desc }
    limit: 10
  ) {
    id
    period
    as_of_date
    return_twr
    return_mwr
    benchmark_return
    alpha
    beta
    sharpe_ratio
    sortino_ratio
    volatility
    max_drawdown
    tracking_error
    beginning_value
    ending_value
    net_contributions
    income
    fees
  }
}
`

// GetPortfolioHistory fetches historical valuations
const GetPortfolioHistory = `
query GetPortfolioHistory($portfolio_id: String!, $start: date!, $end: date!) {
  portfolio_valuations(
    where: {
      portfolio_id: { _eq: $portfolio_id }
      as_of_date: { _gte: $start, _lte: $end }
    }
    order_by: { as_of_date: asc }
  ) {
    as_of_date
    market_value
    cash_flow
    twr_return
    cumulative_return
  }
}
`

// ============================================================================
// Allocation Queries
// ============================================================================

// GetAssetAllocation fetches current allocation breakdown
const GetAssetAllocation = `
query GetAssetAllocation($portfolio_id: String!) {
  allocation(
    where: { portfolio_id: { _eq: $portfolio_id }, dimension: { _eq: "asset_class" } }
    order_by: { weight: desc }
  ) {
    id
    category
    market_value
    weight
    target_weight
    drift
  }
}
`

// GetSectorAllocation fetches sector breakdown
const GetSectorAllocation = `
query GetSectorAllocation($portfolio_id: String!) {
  allocation(
    where: { portfolio_id: { _eq: $portfolio_id }, dimension: { _eq: "sector" } }
    order_by: { weight: desc }
  ) {
    id
    category
    market_value
    weight
  }
}
`

// ============================================================================
// Transaction Queries
// ============================================================================

// GetRecentTransactions fetches recent transactions
const GetRecentTransactions = `
query GetRecentTransactions($account_ids: [String!]!, $limit: Int!) {
  transactions(
    where: { account_id: { _in: $account_ids } }
    order_by: { trade_date: desc }
    limit: $limit
  ) {
    id
    account_id
    security_id
    type
    trade_date
    settlement_date
    quantity
    price
    amount
    fees
    net_amount
    description
    security {
      ticker
      name
    }
  }
}
`

// ============================================================================
// Benchmark Queries
// ============================================================================

// GetBenchmarks fetches available benchmarks
const GetBenchmarks = `
query GetBenchmarks {
  benchmarks(where: { is_active: { _eq: true } }, order_by: { name: asc }) {
    id
    name
    code
    type
    currency
    mtd_return
    qtd_return
    ytd_return
    one_year_return
    three_year_return
    five_year_return
  }
}
`

// CompareToBenchmark compares portfolio to benchmark
const CompareToBenchmark = `
query CompareToBenchmark($portfolio_id: String!, $benchmark_id: String!, $start: date!, $end: date!) {
  portfolio: portfolio_valuations(
    where: { portfolio_id: { _eq: $portfolio_id }, as_of_date: { _gte: $start, _lte: $end } }
    order_by: { as_of_date: asc }
  ) {
    as_of_date
    cumulative_return
  }
  
  benchmark: benchmark_returns(
    where: { benchmark_id: { _eq: $benchmark_id }, as_of_date: { _gte: $start, _lte: $end } }
    order_by: { as_of_date: asc }
  ) {
    as_of_date
    cumulative_return
  }
}
`
