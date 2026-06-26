
cube(`HistoricalSales`, {
  sql_table: `sales_performance_mv`, // This MV is in StarRocks
  data_source: `starrocks`,

  measures: {
    totalRevenue: {
      sql: `total_sale_amount`,
      type: `sum`,
      format: `currency`
    }
  },
  dimensions: {
    orderDate: {
      sql: `order_date`,
      type: `time`
    },
    customerCountry: {
      sql: `customer_country`,
      type: `string`
    }
  }
});
