
cube(`RealtimeOrders`, {
  sql_table: `fact_orders`, // This MV is in RisingWave
  data_source: `risingwave`,

  measures: {
    count: {
      type: `count`
    }
  },
  dimensions: {
    orderId: {
      sql: `order_id`,
      type: `number`,
      primary_key: true
    },
    orderDate: {
      sql: `order_date`,
      type: `time`
    }
  }
});
